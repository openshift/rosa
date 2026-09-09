package operatorroles

import (
	"net/http"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

const (
	testOperatorRoleArn = "arn:aws:iam::765374464689:role/fake-arn-openshift-cluster-csi-drivers-ebs-cloud-credentials"
	clusterKey          = "cluster1"

	stsPoliciesResponse = `{
		"kind": "AWSSTSPolicyList",
		"page": 1,
		"size": 0,
		"total": 0,
		"items": []
	}`

	stsCredRequestsResponse = `{
		"kind": "STSCredentialRequestList",
		"page": 1,
		"size": 0,
		"total": 0,
		"items": []
	}`

	versionsResponse = `{
		"kind": "VersionList",
		"page": 1,
		"size": 1,
		"total": 1,
		"items": [
			{
				"id": "4.16.3",
				"raw_id": "4.16.3",
				"channel_group": "stable",
				"rosa_enabled": true
			}
		]
	}`
)

var _ = Describe("create operator-roles cmd", func() {
	var t *test.TestingRuntime

	resetCmdFlags := func() {
		Cmd.Flags().VisitAll(func(f *pflag.Flag) {
			f.Changed = false
			_ = f.Value.Set(f.DefValue)
		})
	}

	BeforeEach(func() {
		t = test.NewTestRuntime()
		resetCmdFlags()
		args = struct {
			prefix              string
			hostedCp            bool
			installerRoleArn    string
			permissionsBoundary string
			forcePolicyCreation bool
			oidcConfigId        string
			sharedVpcRoleArn    string
			channelGroup        string
			vpcEndpointRoleArn  string
		}{}
		interactive.SetEnabled(false)
		interactive.SetModeKey("")
		ocm.SetClusterKey("")
	})

	Context("convertV1OperatorIAMRoleIntoOcmOperatorIamRole", func() {
		It("converts valid operator IAM roles", func() {
			operatorIAMRole, err := cmv1.NewOperatorIAMRole().
				Name("openshift").
				Namespace("operator").
				RoleARN(testOperatorRoleArn).
				Build()
			Expect(err).NotTo(HaveOccurred())

			roles, err := convertV1OperatorIAMRoleIntoOcmOperatorIamRole([]*cmv1.OperatorIAMRole{operatorIAMRole})
			Expect(err).NotTo(HaveOccurred())
			Expect(roles).To(HaveLen(1))
			Expect(roles[0].Name).To(Equal(operatorIAMRole.Name()))
			Expect(roles[0].Namespace).To(Equal(operatorIAMRole.Namespace()))
			Expect(roles[0].RoleARN).To(Equal(operatorIAMRole.RoleARN()))
			path, err := aws.GetPathFromARN(operatorIAMRole.RoleARN())
			Expect(err).NotTo(HaveOccurred())
			Expect(roles[0].Path).To(Equal(path))
		})

		It("returns an error for malformed operator IAM roles", func() {
			operatorIAMRole, err := cmv1.NewOperatorIAMRole().
				Name("openshift").
				Namespace("operator").
				Build()
			Expect(err).NotTo(HaveOccurred())

			roles, err := convertV1OperatorIAMRoleIntoOcmOperatorIamRole([]*cmv1.OperatorIAMRole{operatorIAMRole})
			Expect(err).To(HaveOccurred())
			Expect(roles).To(BeEmpty())
		})
	})

	Context("runWithRuntime", func() {
		It("returns an error when neither cluster nor prefix is specified", func() {
			err := runWithRuntime(t.RosaRuntime, Cmd, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"either a cluster key for STS cluster or an operator roles prefix must be specified"))
		})

		It("dispatches to the prefix path when prefix is set", func() {
			Expect(Cmd.Flags().Set(PrefixFlag, "my-prefix")).To(Succeed())
			Expect(Cmd.Flags().Set("mode", interactive.ModeAuto)).To(Succeed())

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, stsPoliciesResponse),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(OidcConfigIdFlag))
			Expect(err.Error()).To(ContainSubstring(PrefixFlag))
		})

		It("dispatches to the cluster path when cluster is set", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.Version(cmv1.NewVersion().
					ID("4.16.3").
					RawID("openshift-4.16.3").
					ChannelGroup("stable"))
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role").
						OIDCEndpointURL("https://oidc.example.com").
						OperatorRolePrefix("ManagedOpenShift").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/ManagedOpenShift-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
						),
				))
			})
			t.SetCluster(clusterKey, cluster)
			Expect(Cmd.Flags().Set("cluster", clusterKey)).To(Succeed())
			Expect(Cmd.Flags().Set("mode", interactive.ModeAuto)).To(Succeed())

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, stsPoliciesResponse),
				RespondWithJSON(http.StatusOK, versionsResponse),
				RespondWithJSON(http.StatusOK, stsCredRequestsResponse),
			)

			mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
			mockClient.EXPECT().CheckRoleExists(gomock.Any()).Return(true, "", nil).AnyTimes()

			stdout, stderr, err := test.RunWithOutputCapture(
				func(r *rosa.Runtime, cmd *cobra.Command) error {
					return runWithRuntime(r, cmd, false)
				}, t.RosaRuntime, Cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring("Operator Roles already exists"))
		})

		It("returns an error when STS policy fetch fails", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.Version(cmv1.NewVersion().
					ID("4.16.3").
					RawID("openshift-4.16.3").
					ChannelGroup("stable"))
			})
			t.SetCluster(clusterKey, cluster)
			Expect(Cmd.Flags().Set("cluster", clusterKey)).To(Succeed())
			Expect(Cmd.Flags().Set("mode", interactive.ModeAuto)).To(Succeed())

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusInternalServerError, "{}"),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid role creation mode"))
		})
	})
})
