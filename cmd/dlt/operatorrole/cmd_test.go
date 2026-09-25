// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package operatorrole

import (
	"errors"
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

	It("returns interactive mode errors", func() {
		args.prefix = "test-prefix"
		Cmd.Flag(PrefixFlag).Changed = true
		interactive.SetEnabled(true)
		DeferCleanup(func() {
			interactive.SetEnabled(false)
			Cmd.Flag(PrefixFlag).Changed = false
		})

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("expected a valid operator role deletion mode")),
			"expected interactive mode input errors to be returned")
	})

	It("returns cluster role discovery errors", func() {
		Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed())
		Expect(Cmd.Flags().Set("cluster", "cluster1")).To(Succeed())
		DeferCleanup(func() {
			Cmd.Flag("mode").Changed = false
			Cmd.Flag("cluster").Changed = false
		})
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK,
				`{"kind":"SubscriptionList","items":[{"id":"sub1","cluster_id":"cluster1","plan":{"id":"MOA"}}],"total":1}`),
			RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","items":[],"total":0}`),
			RespondWithJSON(http.StatusOK, `{"kind":"STSOperatorList","items":[],"total":0}`),
		)

		mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
		mockAWS.EXPECT().GetOperatorRolesFromAccountByClusterID("cluster1", gomock.Any()).
			Return(nil, errors.New("list roles failed"))

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("list roles failed")),
			"expected cluster role discovery failure to be returned")
	})

	It("rejects installed clusters", func() {
		Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
			"expected manual mode to be configured")
		Expect(Cmd.Flags().Set("cluster", "cluster1")).To(Succeed(),
			"expected the cluster key to be configured")
		DeferCleanup(func() {
			interactive.SetModeKey("")
			Cmd.Flag("mode").Changed = false
			Cmd.Flag("cluster").Changed = false
		})
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK,
				`{"kind":"SubscriptionList","items":[{"id":"sub1","cluster_id":"cluster1","plan":{"id":"MOA"}}],"total":1}`),
			RespondWithJSON(http.StatusOK,
				`{"kind":"ClusterList","items":[{"id":"cluster1","state":"ready"}],"total":1}`),
		)

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("operator roles can be deleted only for the uninstalled clusters")),
			"expected installed clusters to prevent operator-role deletion")
	})

	It("returns credential request lookup errors", func() {
		Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
			"expected manual mode to be configured")
		Expect(Cmd.Flags().Set("cluster", "cluster1")).To(Succeed(),
			"expected the cluster key to be configured")
		DeferCleanup(func() {
			interactive.SetModeKey("")
			Cmd.Flag("mode").Changed = false
			Cmd.Flag("cluster").Changed = false
		})
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK,
				`{"kind":"SubscriptionList","items":[{"id":"sub1","cluster_id":"cluster1","plan":{"id":"MOA"}}],"total":1}`),
			RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","items":[],"total":0}`),
			RespondWithJSON(http.StatusInternalServerError,
				`{"kind":"Error","id":"500","reason":"credential requests unavailable"}`),
		)

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("error getting operator credential request from OCM")),
			"expected credential request lookup errors to be returned")
	})

	It("rejects ambiguous cluster names", func() {
		setClusterDeletionFlags()
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"kind":"SubscriptionList","items":[{"id":"sub1","cluster_id":"cluster1"}],"total":2}`))

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("more than one cluster found with the same name")),
			"expected ambiguous cluster names to be rejected")
	})

	It("returns subscription lookup errors", func() {
		setClusterDeletionFlags()
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
			`{"kind":"Error","id":"500","reason":"subscriptions unavailable"}`))

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("error validating cluster 'cluster1'")),
			"expected subscription lookup errors to be returned")
	})

	It("returns cluster lookup errors", func() {
		setClusterDeletionFlags()
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK,
				`{"kind":"SubscriptionList","items":[{"id":"sub1","cluster_id":"cluster1"}],"total":1}`),
			RespondWithJSON(http.StatusInternalServerError,
				`{"kind":"Error","id":"500","reason":"cluster unavailable"}`),
		)

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("error validating cluster 'cluster1'")),
			"expected cluster lookup errors to be returned")
	})

	It("returns not-found errors without a deleted subscription", func() {
		setClusterDeletionFlags()
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{"kind":"SubscriptionList","items":[],"total":0}`),
			RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","items":[],"total":0}`),
		)

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("failed to get cluster 'cluster1'")),
			"expected a missing cluster without a deleted subscription to be rejected")
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

		It("returns prefix usage lookup errors", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
				`{"kind":"Error","id":"500","reason":"clusters unavailable"}`))

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("problem checking if any clusters")),
				"expected prefix usage lookup errors to be returned")
		})

		It("returns credential request list errors", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","items":[],"total":0}`),
				RespondWithJSON(http.StatusInternalServerError,
					`{"kind":"Error","id":"500","reason":"credential requests unavailable"}`),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("error getting operator credential request from OCM")),
				"expected credential request list errors to be returned")
		})

		It("returns AWS operator-role lookup errors", func() {
			appendPrefixLookupResponses(t)
			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().GetOperatorRolesFromAccountByPrefix("test-prefix", gomock.Any()).
				Return(nil, errors.New("operator roles unavailable"))

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("operator roles unavailable")),
				"expected AWS operator-role lookup errors to be returned")
		})

		Context("with a matching operator role", func() {
			var mockAWS *awsClient.MockClient

			BeforeEach(func() {
				appendPrefixLookupResponses(t)
				mockAWS = t.RosaRuntime.AWSClient.(*awsClient.MockClient)
				mockAWS.EXPECT().GetOperatorRolesFromAccountByPrefix("test-prefix", gomock.Any()).
					Return([]string{"test-prefix-role"}, nil)
			})

			It("returns role ARN lookup errors", func() {
				mockAWS.EXPECT().CheckRoleExists("test-prefix-role").
					Return(false, "", errors.New("role unavailable"))

				err := runWithRuntime(t.RosaRuntime, Cmd)
				Expect(err).To(MatchError("failed to get 'test-prefix-role' role ARN"),
					"expected role ARN lookup errors to be returned")
			})

			It("returns managed-policy lookup errors", func() {
				mockAWS.EXPECT().CheckRoleExists("test-prefix-role").Return(true, "role-arn", nil)
				mockAWS.EXPECT().HasManagedPolicies("role-arn").
					Return(false, errors.New("managed policies unavailable"))

				err := runWithRuntime(t.RosaRuntime, Cmd)
				Expect(err).To(MatchError(ContainSubstring("managed policies unavailable")),
					"expected managed-policy lookup errors to be returned")
			})

			It("returns attached-policy lookup errors", func() {
				mockAWS.EXPECT().CheckRoleExists("test-prefix-role").Return(true, "role-arn", nil)
				mockAWS.EXPECT().HasManagedPolicies("role-arn").Return(false, nil)
				mockAWS.EXPECT().GetOperatorRolePolicies([]string{"test-prefix-role"}).
					Return(nil, nil, errors.New("attached policies unavailable"))

				err := runWithRuntime(t.RosaRuntime, Cmd)
				Expect(err).To(MatchError(ContainSubstring("attached policies unavailable")),
					"expected attached-policy lookup errors to be returned")
			})

			It("returns command generation errors", func() {
				mockAWS.EXPECT().CheckRoleExists("test-prefix-role").Return(true, "role-arn", nil)
				mockAWS.EXPECT().HasManagedPolicies("role-arn").Return(false, nil)
				mockAWS.EXPECT().GetOperatorRolePolicies([]string{"test-prefix-role"}).Return(
					map[string][]string{"test-prefix-role": {"policy-arn"}}, nil, nil,
				)
				mockAWS.EXPECT().GetPolicyDetailsFromRole(gomock.Any()).Return(nil, nil)
				mockAWS.EXPECT().ListPolicyVersions("policy-arn").
					Return(nil, errors.New("policy versions unavailable"))

				err := runWithRuntime(t.RosaRuntime, Cmd)
				Expect(err).To(MatchError("policy versions unavailable"),
					"expected command generation errors to be returned")
			})

			It("prints manual deletion commands", func() {
				mockAWS.EXPECT().CheckRoleExists("test-prefix-role").Return(true, "role-arn", nil)
				mockAWS.EXPECT().HasManagedPolicies("role-arn").Return(false, nil)
				mockAWS.EXPECT().GetOperatorRolePolicies([]string{"test-prefix-role"}).Return(
					map[string][]string{}, map[string][]string{}, nil,
				)
				mockAWS.EXPECT().GetPolicyDetailsFromRole(gomock.Any()).Return(nil, nil)

				err := runWithRuntime(t.RosaRuntime, Cmd)
				Expect(err).NotTo(HaveOccurred(), "expected manual operator-role deletion commands to be generated")
			})

			It("rejects an empty deletion mode", func() {
				interactive.SetModeKey("")
				mockAWS.EXPECT().CheckRoleExists("test-prefix-role").Return(true, "role-arn", nil)
				mockAWS.EXPECT().HasManagedPolicies("role-arn").Return(false, nil)

				err := runWithRuntime(t.RosaRuntime, Cmd)
				Expect(err).To(MatchError(ContainSubstring("invalid mode")),
					"expected an empty operator-role deletion mode to be rejected")
			})
		})
	})
})

func appendPrefixLookupResponses(t *test.TestingRuntime) {
	t.ApiServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","items":[],"total":0}`),
		RespondWithJSON(http.StatusOK, `{"kind":"STSOperatorList","items":[],"total":0}`),
		RespondWithJSON(http.StatusOK, `{"kind":"STSOperatorList","items":[],"total":0}`),
	)
}

func setClusterDeletionFlags() {
	Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
		"expected manual mode to be configured")
	Expect(Cmd.Flags().Set("cluster", "cluster1")).To(Succeed(),
		"expected the cluster key to be configured")
	DeferCleanup(func() {
		interactive.SetModeKey("")
		Cmd.Flag("mode").Changed = false
		Cmd.Flag("cluster").Changed = false
	})
}

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

		result, err := buildCommand(t.RosaRuntime, roleNames, policyMap, arbitraryPolicyMap, false, policiesOutput)
		Expect(err).NotTo(HaveOccurred())
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

		result, err := buildCommand(t.RosaRuntime, roleNames, policyMap, arbitraryPolicyMap, true, policiesOutput)
		Expect(err).NotTo(HaveOccurred())
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

		result, err := buildCommand(t.RosaRuntime, roleNames, policyMap, arbitraryPolicyMap, false, policiesOutput)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(ContainSubstring("detach-role-policy"))
		Expect(result).To(ContainSubstring("extra-policy"))
		Expect(result).To(ContainSubstring("delete-role"))
	})

	It("returns policy version lookup errors", func() {
		roleNames := []string{"my-operator-role"}
		policyMap := map[string][]string{
			"my-operator-role": {"arn:aws:iam::123:policy/op-policy"},
		}

		mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
		mockAWS.EXPECT().ListPolicyVersions("arn:aws:iam::123:policy/op-policy").
			Return(nil, errors.New("policy versions unavailable"))

		_, err := buildCommand(t.RosaRuntime, roleNames, policyMap, nil, false, nil)
		Expect(err).To(MatchError("policy versions unavailable"),
			"expected policy version lookup failure to be returned")
	})
})
