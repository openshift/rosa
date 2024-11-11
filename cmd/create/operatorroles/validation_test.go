package operatorroles

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Create dns domain", func() {
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
			It("KO: Should error when using classic and the first flag (return false)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(false, "123", "")
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Can only use '%s' flag for Hosted Control Plane operator "+
					"roles", vpcEndpointRoleArnFlag,
				))
			})
			It("KO: Should error when using classic and the vpc endpoint flag (return false)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(false, "123", "")
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Can only use '%s' flag for Hosted Control Plane operator "+
					"roles", vpcEndpointRoleArnFlag,
				))
			})
			It("KO: Should error when using classic and both flags (return false)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(false, "123", "123")
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Can only use '%s' flag for Hosted Control Plane operator "+
					"roles", vpcEndpointRoleArnFlag,
				))
			})
			It("KO: Should error when using HCP and the first flag but not the second (return false)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(true, "123", "")
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Must supply '%s' flag when using the '%s' flag",
					hostedZoneRoleArnFlag,
					vpcEndpointRoleArnFlag,
				))
			})
			It("KO: Should error when using HCP and the second flag but not the first (return false)", func() {
				usingSharedVpc, err := validateSharedVpcInputs(true, "", "123")
				Expect(usingSharedVpc).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Must supply '%s' flag when using the '%s' flag",
					vpcEndpointRoleArnFlag,
					hostedZoneRoleArnFlag,
				))
			})
		})
	})
})
