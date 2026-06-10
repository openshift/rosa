package region

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const regionTestExternalID = "223B9588-36A5-ECA4-BE8D-7C673B77CEC1"

var _ = Describe("validateChangedSTSExternalIDFlag", func() {
	var cmd *cobra.Command

	BeforeEach(func() {
		cmd = &cobra.Command{}
		cmd.Flags().String("external-id", "", "STS external ID")
	})

	It("skips validation when the external-id flag is unset", func() {
		err := validateChangedSTSExternalIDFlag(cmd, regionTestExternalID)
		Expect(err).NotTo(HaveOccurred(), "unset external-id flag should skip validation")
	})

	It("validates the external-id when the flag is set", func() {
		Expect(cmd.Flags().Set("external-id", regionTestExternalID)).To(Succeed())
		err := validateChangedSTSExternalIDFlag(cmd, regionTestExternalID)
		Expect(err).NotTo(HaveOccurred(), "valid external-id should pass validation")
	})

	It("rejects an invalid external-id when the flag is set", func() {
		Expect(cmd.Flags().Set("external-id", "x")).To(Succeed())
		err := validateChangedSTSExternalIDFlag(cmd, "x")
		Expect(err).To(HaveOccurred(), "invalid external-id should fail validation")
	})
})
