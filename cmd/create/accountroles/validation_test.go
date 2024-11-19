package accountroles

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/arguments"
)

var _ = Describe("Validate Shared VPC Inputs", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("validateSharedVpcInputs", func() {
		When("Validate flags properly for shared VPC for HCP op roles", func() {
			It("OK: Should pass with no error, for classic (return false)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(false, "", "")
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})
			It("OK: Should pass with no error, for HCP (return false)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(true, "", "")
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})
			It("OK: Should pass with no error, for HCP (return true)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(true, "123", "123")
				Expect(usingSharedVpc).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})
			It("KO: Should error when using HCP and the first flag but not the second (return false)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(true, "123", "")
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(arguments.MustUseBothFlagsErrorMessage,
					route53RoleArnFlag,
					vpcEndpointRoleArnFlag,
				))
			})
			It("KO: Should error when using HCP and the second flag but not the first (return false)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(true, "", "123")
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
