package operatorrole

import (
	"net/http"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	awsClient "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("runWithRuntime", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		args = struct {
			prefix                     string
			deleteHcpSharedVpcPolicies bool
		}{}
		interactive.SetEnabled(false)
		interactive.SetModeKey("")
	})

	It("returns error when GetMode fails", func() {
		interactive.SetModeKey("invalid-mode")
		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Invalid mode"))
	})

	It("returns error when neither cluster nor prefix is provided", func() {
		interactive.SetModeKey("auto")
		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("either a cluster key or a prefix must be specified"))
	})

	Context("with prefix set", func() {
		BeforeEach(func() {
			Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed())
			args.prefix = "test-prefix"
			Cmd.Flag(PrefixFlag).Changed = true
		})

		AfterEach(func() {
			interactive.SetModeKey("")
			Cmd.Flag("mode").Changed = false
			Cmd.Flag(PrefixFlag).Changed = false
		})

		It("reports no operator roles when none found for prefix", func() {
			// HasAClusterUsingOperatorRolesPrefix -> clusters list -> empty
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","items":[],"total":0}`),
			)
			// GetAllCredRequests -> GetCredRequests(false) classic
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{"kind":"STSOperatorList","items":[],"total":0}`),
			)
			// GetAllCredRequests -> GetCredRequests(true) hcp
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{"kind":"STSOperatorList","items":[],"total":0}`),
			)

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().GetOperatorRolesFromAccountByPrefix(
				"test-prefix", gomock.Any(),
			).Return([]string{}, nil)

			err := t.StdOutReader.Record()
			Expect(err).NotTo(HaveOccurred())

			err = runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).NotTo(HaveOccurred())

			stdout, err := t.StdOutReader.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("no operator roles to delete"))
		})

		It("returns error when a cluster is using the prefix", func() {
			// HasAClusterUsingOperatorRolesPrefix -> clusters list -> found one
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","items":[{"id":"abc"}],"total":1}`),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("there are clusters using Operator Roles Prefix"))
		})
	})
})

var _ = Describe("buildCommand", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("generates detach, delete-version, delete-policy, and delete-role commands", func() {
		roleNames := []string{"my-operator-role"}
		policyMap := map[string][]string{
			"my-operator-role": {"arn:aws:iam::123:policy/op-policy"},
		}
		arbitraryPolicyMap := map[string][]string{}
		policiesOutput := []*iam.GetPolicyOutput{}

		mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
		mockAWS.EXPECT().ListPolicyVersions("arn:aws:iam::123:policy/op-policy").Return(
			[]awsClient.PolicyVersion{
				{VersionID: "v1", IsDefaultVersion: true},
				{VersionID: "v2", IsDefaultVersion: false},
			}, nil,
		)

		result := buildCommand(t.RosaRuntime, roleNames, policyMap, arbitraryPolicyMap, false, policiesOutput)
		Expect(result).To(ContainSubstring("detach-role-policy"))
		Expect(result).To(ContainSubstring("delete-policy-version"))
		Expect(result).To(ContainSubstring("v2"))
		Expect(result).To(ContainSubstring("delete-policy"))
		Expect(result).To(ContainSubstring("delete-role"))
		Expect(result).To(ContainSubstring("my-operator-role"))
	})

	It("uses shared-vpc policy ARN for managed policies", func() {
		roleNames := []string{"my-operator-role"}
		policyMap := map[string][]string{
			"my-operator-role": {"arn:aws:iam::123:policy/op-policy"},
		}
		arbitraryPolicyMap := map[string][]string{}
		policiesOutput := []*iam.GetPolicyOutput{}

		mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
		mockAWS.EXPECT().ListPolicyVersions("arn:aws:iam::123:policy/op-policy").Return(
			[]awsClient.PolicyVersion{}, nil,
		)

		result := buildCommand(t.RosaRuntime, roleNames, policyMap, arbitraryPolicyMap, true, policiesOutput)
		Expect(result).To(ContainSubstring("detach-role-policy"))
		Expect(result).To(ContainSubstring("delete-role"))
		Expect(result).To(ContainSubstring(awsClient.SharedVpcAssumeRolePrefix))
	})

	It("detaches arbitrary policies", func() {
		roleNames := []string{"my-operator-role"}
		policyMap := map[string][]string{}
		arbitraryPolicyMap := map[string][]string{
			"my-operator-role": {"arn:aws:iam::123:policy/extra-policy"},
		}
		policiesOutput := []*iam.GetPolicyOutput{}

		result := buildCommand(t.RosaRuntime, roleNames, policyMap, arbitraryPolicyMap, false, policiesOutput)
		Expect(result).To(ContainSubstring("detach-role-policy"))
		Expect(result).To(ContainSubstring("extra-policy"))
		Expect(result).To(ContainSubstring("delete-role"))
	})
})
