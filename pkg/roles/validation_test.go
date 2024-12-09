package roles

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/arguments"
)

var _ = Describe("Validate Shared VPC Inputs", func() {
	var ctrl *gomock.Controller

	var route53RoleArnFlag = "route53-role-arn"
	var vpcEndpointRoleArnFlag = "vpc-endpoint-role-arn"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("validateSharedVpcInputs", func() {
		When("Validate flags properly for shared VPC for HCP op roles", func() {
			It("OK: Should pass with no error, no flag usage (return false)", func() {
				usingSharedVpc, err := ValidateSharedVpcInputs("", "",
					route53RoleArnFlag, vpcEndpointRoleArnFlag)
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})
			It("OK: Should pass with no error, for HCP (return true)", func() {
				usingSharedVpc, err := ValidateSharedVpcInputs("123", "123",
					route53RoleArnFlag, vpcEndpointRoleArnFlag)
				Expect(usingSharedVpc).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})
			It("KO: Should error when using the first flag but not the second (return false)", func() {
				usingSharedVpc, err := ValidateSharedVpcInputs("123", "",
					route53RoleArnFlag, vpcEndpointRoleArnFlag)
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(arguments.MustUseBothFlagsErrorMessage,
					route53RoleArnFlag,
					vpcEndpointRoleArnFlag,
				))
			})
			It("KO: Should error when using the second flag but not the first (return false)", func() {
				usingSharedVpc, err := ValidateSharedVpcInputs("", "123",
					route53RoleArnFlag, vpcEndpointRoleArnFlag)
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(arguments.MustUseBothFlagsErrorMessage,
					vpcEndpointRoleArnFlag,
					route53RoleArnFlag,
				))
			})
		})
	})
})
