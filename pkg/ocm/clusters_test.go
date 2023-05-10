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
