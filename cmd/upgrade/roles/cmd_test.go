// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package roles

import (
	"errors"
	"net/http"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	awsmock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("resolvePolicyVersionForUpgrade", func() {
	It("derives policy version from cluster upgrade version when policy is empty", func() {
		policyVersion, chosen, err := resolvePolicyVersionForUpgrade("", "4.21.3")
		Expect(err).ToNot(HaveOccurred())
		Expect(policyVersion).To(Equal("4.21"))
		Expect(chosen).To(BeTrue())
	})

	It("returns an error for a malformed cluster upgrade version", func() {
		_, _, err := resolvePolicyVersionForUpgrade("", "not-a-version")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("error parsing cluster upgrade version"))
	})

	It("preserves explicit policy version even when cluster version differs", func() {
		policyVersion, chosen, err := resolvePolicyVersionForUpgrade("4.20", "4.21.3")
		Expect(err).ToNot(HaveOccurred())
		Expect(policyVersion).To(Equal("4.20"))
		Expect(chosen).To(BeTrue())
	})

	It("returns empty and not chosen when both inputs are empty", func() {
		policyVersion, chosen, err := resolvePolicyVersionForUpgrade("", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(policyVersion).To(BeEmpty())
		Expect(chosen).To(BeFalse())
	})
})

var _ = Describe("getPolicyVersion", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("requests HCP versions for hosted control plane clusters", func() {
		version, err := cmv1.NewVersion().
			ID("openshift-v5.0.0-candidate").
			RawID("5.0.0").
			Enabled(true).
			ROSAEnabled(true).
			ChannelGroup("candidate").
			Build()
		Expect(err).NotTo(HaveOccurred(), "expected the HCP version fixture to build")
		t.ApiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/versions"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("product")).To(Equal(ocm.HcpProduct),
						"expected hosted role upgrades to request product=hcp")
				},
				RespondWithJSON(http.StatusOK, test.FormatVersionList([]*cmv1.Version{version})),
			),
		)

		policyVersion, err := getPolicyVersion(t.RosaRuntime.OCMClient, "5.0", "candidate", true)

		Expect(err).NotTo(HaveOccurred(), "expected HCP policy-version lookup to succeed")
		Expect(policyVersion).To(Equal("5.0"), "expected HCP policy version 5.0")
	})

	It("returns version lookup errors", func() {
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
			`{"kind":"Error","code":"CLUSTERS-MGMT-500","reason":"internal error"}`))

		_, err := getPolicyVersion(t.RosaRuntime.OCMClient, "5.0", "candidate", true)

		Expect(err).To(HaveOccurred(), "expected HCP policy-version lookup errors to be returned")
	})
})

var _ = Describe("generateClusterUpgradeInfo", func() {
	It("OK: Returns the cluster upgrade info string successfully", func() {
		info := generateClusterUpgradeInfo("cluster-key-01", "4.15.0", "auto")

		expected := "Account/Operator Role policies are not valid with upgrade version 4.15.0. " +
			"Run the following command(s) to upgrade the roles:\n" +
			"\trosa upgrade roles -c cluster-key-01 --cluster-version=4.15.0 --mode=auto\n\n" +
			", then run the upgrade command again:\n" +
			"\trosa upgrade cluster -c cluster-key-01\n"

		Expect(info).To(Equal(expected))
	})
})

var _ = Describe("syncAccountRoleVersionTagsForCluster", func() {
	It("updates all account role tags when all account roles are present", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		mockAwsClient := awsmock.NewMockClient(ctrl)
		cluster := buildClusterForRoleTagSync(
			"arn:aws:iam::123456789012:role/test-prefix-Installer-Role",
			"arn:aws:iam::123456789012:role/test-prefix-Support-Role",
			"arn:aws:iam::123456789012:role/test-prefix-ControlPlane-Role",
			"arn:aws:iam::123456789012:role/test-prefix-Worker-Role",
		)

		updatedRoles := map[string]bool{}
		mockAwsClient.EXPECT().UpdateTag(gomock.Any(), "4.17").DoAndReturn(
			func(roleName, _ string) error {
				updatedRoles[roleName] = true
				return nil
			},
		).Times(4)

		err := syncAccountRoleVersionTagsForCluster(mockAwsClient, cluster, "4.17")
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedRoles).To(HaveKey("test-prefix-Installer-Role"))
		Expect(updatedRoles).To(HaveKey("test-prefix-Support-Role"))
		Expect(updatedRoles).To(HaveKey("test-prefix-ControlPlane-Role"))
		Expect(updatedRoles).To(HaveKey("test-prefix-Worker-Role"))
	})

	It("skips missing account roles", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		mockAwsClient := awsmock.NewMockClient(ctrl)
		cluster := buildClusterForRoleTagSync(
			"arn:aws:iam::123456789012:role/test-prefix-Installer-Role",
			"",
			"",
			"",
		)

		mockAwsClient.EXPECT().UpdateTag("test-prefix-Installer-Role", "4.17").Return(nil).Times(1)

		err := syncAccountRoleVersionTagsForCluster(mockAwsClient, cluster, "4.17")
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns a wrapped error when role tag update fails", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		mockAwsClient := awsmock.NewMockClient(ctrl)
		cluster := buildClusterForRoleTagSync(
			"arn:aws:iam::123456789012:role/test-prefix-Installer-Role",
			"",
			"",
			"",
		)

		expectedErr := errors.New("failed to update tag")
		mockAwsClient.EXPECT().UpdateTag("test-prefix-Installer-Role", "4.17").Return(expectedErr).Times(1)

		err := syncAccountRoleVersionTagsForCluster(mockAwsClient, cluster, "4.17")
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring(
			"failed to update account role 'test-prefix-Installer-Role' version tag",
		)))
		Expect(errors.Is(err, expectedErr)).To(BeTrue())
	})

	It("returns an error when account role ARN cannot be parsed", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		mockAwsClient := awsmock.NewMockClient(ctrl)
		cluster := buildClusterForRoleTagSync(
			"invalid-arn-installer",
			"",
			"",
			"",
		)

		err := syncAccountRoleVersionTagsForCluster(mockAwsClient, cluster, "4.17")
		Expect(err).To(HaveOccurred())
	})
})

func buildClusterForRoleTagSync(installerRoleArn, supportRoleArn, masterRoleArn,
	workerRoleArn string) *cmv1.Cluster {

	clusterBuilder := cmv1.NewCluster().ID("test-cluster")
	clusterBuilder.AWS(
		cmv1.NewAWS().STS(
			cmv1.NewSTS().
				RoleARN(installerRoleArn).
				SupportRoleARN(supportRoleArn).
				InstanceIAMRoles(
					cmv1.NewInstanceIAMRoles().
						MasterRoleARN(masterRoleArn).
						WorkerRoleARN(workerRoleArn),
				),
		),
	)

	cluster, err := clusterBuilder.Build()
	Expect(err).ToNot(HaveOccurred())

	return cluster
}
