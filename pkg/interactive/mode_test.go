package interactive

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Mode Test", func() {
	var (
		cmd  *cobra.Command
		mode string
	)

	BeforeEach(func() {
		cmd = &cobra.Command{}
		AddModeFlag(cmd)
	})

	Context("GetMode", func() {
		It("should return the correct mode", func() {
			SetModeKey(ModeAuto)
			result, err := GetMode()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ModeAuto))
		})

		It("should return an error for an invalid mode", func() {
			SetModeKey("invalid_mode")
			_, err := GetMode()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Invalid mode. Allowed values are %v", Modes)))
		})
	})

	Context("GetOptionMode", func() {
		It("should return an error for an invalid mode", func() {
			cmd.Flags().Parse([]string{"--mode=invalid_mode"})
			_, err := GetOptionMode(cmd, mode, "Question")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid mode"))
		})
	})
})
