package operatorroles

import (
	"net/http"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Upgrade operator-roles", func() {
	var (
		t          *test.TestingRuntime
		mockClient *aws.MockClient
	)

	BeforeEach(func() {
		t = test.NewTestRuntime()
		mockClient = t.RosaRuntime.AWSClient.(*aws.MockClient)
		args.upgradeVersion = ""
		interactive.SetEnabled(false)
		interactive.SetModeKey("")
	})

	Context("runWithRuntime", func() {
		It("returns error when cluster has no operator roles", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role"),
				))
				c.Version(cmv1.NewVersion().ID("openshift-v4.14.0").RawID("4.14.0").
					ChannelGroup("stable"))
			})
			t.SetCluster("test-cluster", cluster)

			// GetLatestVersion calls GetVersions which hits versions API
			v := cmv1.NewVersion().ID("openshift-v4.14.0").RawID("4.14.0").
				Enabled(true).ROSAEnabled(true).ChannelGroup("stable")
			versionObj, err := v.Build()
			Expect(err).NotTo(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatVersionList([]*cmv1.Version{versionObj})))

			_, _, err = test.RunWithOutputCapture(runWithRuntime, t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("doesn't have any operator roles"))
		})

		It("returns error when account roles need upgrade first", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/test-prefix-Installer-Role").
						OperatorRolePrefix("test-prefix").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
						),
				))
				c.Version(cmv1.NewVersion().ID("openshift-v4.14.0").RawID("4.14.0").
					ChannelGroup("stable"))
			})
			t.SetCluster("test-cluster", cluster)

			v := cmv1.NewVersion().ID("openshift-v4.14.0").RawID("4.14.0").
				Enabled(true).ROSAEnabled(true).ChannelGroup("stable")
			versionObj, err := v.Build()
			Expect(err).NotTo(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatVersionList([]*cmv1.Version{versionObj})))

			// GetCredRequests hits /api/clusters_mgmt/v1/aws_inquiries/sts_credential_requests
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				`{"kind":"STSCredentialRequestList","page":1,"size":0,"total":0,"items":[]}`))
			// GetPolicies hits /api/clusters_mgmt/v1/aws_inquiries/sts_policies
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				`{"kind":"AWSSTSPolicyList","page":1,"size":0,"total":0,"items":[]}`))

			mockClient.EXPECT().IsUpgradedNeededForAccountRolePolicies("test-prefix", "4.14").
				Return(true, nil)

			_, _, err = test.RunWithOutputCapture(runWithRuntime, t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("need to be upgraded before operator roles"))
			Expect(err.Error()).To(ContainSubstring("rosa upgrade account-roles --prefix"))
		})

		It("returns no error when policies are already up-to-date", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/test-prefix-Installer-Role").
						OperatorRolePrefix("test-prefix").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
						),
				))
				c.Version(cmv1.NewVersion().ID("openshift-v4.14.0").RawID("4.14.0").
					ChannelGroup("stable"))
			})
			t.SetCluster("test-cluster", cluster)

			v := cmv1.NewVersion().ID("openshift-v4.14.0").RawID("4.14.0").
				Enabled(true).ROSAEnabled(true).ChannelGroup("stable")
			versionObj, err := v.Build()
			Expect(err).NotTo(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatVersionList([]*cmv1.Version{versionObj})))

			// GetCredRequests
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				`{"kind":"STSCredentialRequestList","page":1,"size":0,"total":0,"items":[]}`))
			// GetPolicies
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				`{"kind":"AWSSTSPolicyList","page":1,"size":0,"total":0,"items":[]}`))

			mockClient.EXPECT().IsUpgradedNeededForAccountRolePolicies("test-prefix", "4.14").
				Return(false, nil)
			mockClient.EXPECT().IsUpgradedNeededForOperatorRolePoliciesUsingPrefix(
				"test-prefix", gomock.Any(), gomock.Any(), "4.14",
				gomock.Any(), gomock.Any()).
				Return(false, nil)

			// FindMissingOperatorRolesForUpgrade hits the versions API again
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatVersionList([]*cmv1.Version{versionObj})))

			_, _, err = test.RunWithOutputCapture(runWithRuntime, t.RosaRuntime, Cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error for invalid hidden --version flag", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/test-prefix-Installer-Role").
						OperatorRolePrefix("test-prefix").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
						),
				))
				c.Version(cmv1.NewVersion().ID("openshift-v4.14.0").RawID("4.14.0").
					ChannelGroup("stable").AvailableUpgrades("4.14.1", "4.14.2"))
			})
			t.SetCluster("test-cluster", cluster)

			v := cmv1.NewVersion().ID("openshift-v4.14.0").RawID("4.14.0").
				Enabled(true).ROSAEnabled(true).ChannelGroup("stable")
			versionObj, err := v.Build()
			Expect(err).NotTo(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatVersionList([]*cmv1.Version{versionObj})))

			args.upgradeVersion = "9.99.99"

			_, _, err = test.RunWithOutputCapture(runWithRuntime, t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid version"))
		})

		It("returns error when GetLatestVersion fails", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/test-prefix-Installer-Role").
						OperatorRolePrefix("test-prefix").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
						),
				))
				c.Version(cmv1.NewVersion().ID("openshift-v4.14.0").RawID("4.14.0").
					ChannelGroup("stable"))
			})
			t.SetCluster("test-cluster", cluster)

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
				`{"kind":"Error","id":"500","href":"/api/clusters_mgmt/v1/errors/500","code":"CLUSTERS-MGMT-500","reason":"internal error"}`))

			_, _, err := test.RunWithOutputCapture(runWithRuntime, t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting latest version"))
		})
	})

	Context("handleModeFlag", func() {
		DescribeTable("passes the mode through when the mode flag is set explicitly",
			func(mode string) {
				interactive.SetModeKey(mode)
				interactive.SetEnabled(false)
				cmd := &cobra.Command{Use: "test"}
				interactive.AddModeFlag(cmd)
				Expect(cmd.Flags().Set("mode", mode)).To(Succeed(),
					"setting the mode flag should not return an error")

				resultMode, err := handleModeFlag(cmd, mode)
				Expect(err).NotTo(HaveOccurred())
				Expect(resultMode).To(Equal(mode))
				Expect(interactive.Enabled()).To(BeFalse(),
					"interactive should stay disabled when mode flag is explicitly set")
			},
			Entry("auto", interactive.ModeAuto),
			Entry("manual", interactive.ModeManual),
		)

		It("enables interactive mode when the mode flag is not set", func() {
			interactive.SetEnabled(false)
			cmd := &cobra.Command{Use: "test"}
			interactive.AddModeFlag(cmd)

			_, _ = handleModeFlag(cmd, interactive.ModeAuto)
			Expect(interactive.Enabled()).To(BeTrue(),
				"interactive should be enabled when mode flag is not explicitly changed")
		})
	})
})
