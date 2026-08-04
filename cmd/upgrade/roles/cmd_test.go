package roles

import (
	"errors"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	awsmock "github.com/openshift/rosa/pkg/aws"
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
