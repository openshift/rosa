package ocm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
)

var _ = Describe("New Operator Iam Role From Cmv1", func() {
	const (
		fakeOperatorRoleArn = "arn:aws:iam::765374464689:role/fake-arn-openshift-cluster-csi-drivers-ebs-cloud-credentials"
	)
	It("OK: Converts cmv1 operator iam role", func() {
		cmv1OperatorIamRole, err := cmv1.NewOperatorIAMRole().
			Name("openshift").Namespace("operator").RoleARN(fakeOperatorRoleArn).Build()
		Expect(err).NotTo(HaveOccurred())
		ocmOperatorIamRole, err := NewOperatorIamRoleFromCmv1(cmv1OperatorIamRole)
		Expect(err).NotTo(HaveOccurred())
		Expect(ocmOperatorIamRole.Name).To(Equal(cmv1OperatorIamRole.Name()))
		Expect(ocmOperatorIamRole.Namespace).To(Equal(cmv1OperatorIamRole.Namespace()))
		Expect(ocmOperatorIamRole.RoleARN).To(Equal(cmv1OperatorIamRole.RoleARN()))
		path, err := aws.GetPathFromARN(cmv1OperatorIamRole.RoleARN())
		Expect(err).NotTo(HaveOccurred())
		Expect(ocmOperatorIamRole.Path).To(Equal(path))
	})
})

var _ = Context("Generate a query", func() {
	Describe("getClusterFilter", func() {
		It("Should return the default value", func() {
			output := getClusterFilter(nil)
			Expect(output).To(Equal("product.id = 'rosa'"))
		})

		It("Should construct a proper string if creator is not nil", func() {
			creator := &aws.Creator{AccountID: "test-account-id"}
			output := getClusterFilter(creator)
			Expect(output).To(Equal(
				"product.id = 'rosa' AND (properties.rosa_creator_arn LIKE '%:test-account-id:%' OR " +
					"aws.sts.role_arn LIKE '%:test-account-id:%')"))
		})
	})

})

var _ = Context("List Clusters", func() {
	Describe("List Clusters using Account Role", func() {

		creator := &aws.Creator{
			AccountID: "12345678",
		}
		role := aws.Role{}

		It("Correctly builds the query for Installer Account Role", func() {

			role.RoleType = aws.InstallerAccountRoleType
			role.RoleARN = "arn:aws:iam::765374464689:role/test-Installer-Role"

			query, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("product.id = 'rosa' AND " +
				"(properties.rosa_creator_arn LIKE '%:12345678:%' OR aws.sts.role_arn LIKE '%:12345678:%') AND " +
				"aws.sts.role_arn='arn:aws:iam::765374464689:role/test-Installer-Role'"))
		})

		It("Correctly builds the query for Support Account Role", func() {

			role.RoleType = aws.SupportAccountRoleType
			role.RoleARN = "arn:aws:iam::765374464689:role/test-Support-Role"

			query, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("product.id = 'rosa' AND " +
				"(properties.rosa_creator_arn LIKE '%:12345678:%' OR aws.sts.role_arn LIKE '%:12345678:%') AND " +
				"aws.sts.support_role_arn='arn:aws:iam::765374464689:role/test-Support-Role'"))
		})

		It("Correctly builds the query for Control Plane Account Role", func() {

			role.RoleType = aws.ControlPlaneAccountRoleType
			role.RoleARN = "arn:aws:iam::765374464689:role/test-ControlPlane-Role"

			query, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("product.id = 'rosa' AND " +
				"(properties.rosa_creator_arn LIKE '%:12345678:%' OR aws.sts.role_arn LIKE '%:12345678:%') AND " +
				"aws.sts.instance_iam_roles.master_role_arn='arn:aws:iam::765374464689:role/test-ControlPlane-Role'"))
		})

		It("Correctly builds the query for Worker Account Role", func() {

			role.RoleType = aws.WorkerAccountRoleType
			role.RoleARN = "arn:aws:iam::765374464689:role/test-Worker-Role"

			query, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("product.id = 'rosa' AND " +
				"(properties.rosa_creator_arn LIKE '%:12345678:%' OR aws.sts.role_arn LIKE '%:12345678:%') AND " +
				"aws.sts.instance_iam_roles.worker_role_arn='arn:aws:iam::765374464689:role/test-Worker-Role'"))
		})

		It("Fails to build query for unknown Role Type", func() {
			role.RoleType = "foo"
			role.RoleARN = "arn:aws:iam::765374464689:role/test-Worker-Role"

			_, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).To(HaveOccurred())
		})

	})
})
