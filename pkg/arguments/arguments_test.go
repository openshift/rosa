package arguments

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Client", func() {
	var (
		cmd *cobra.Command
	)

	Context("Test PreprocessUnknownFlagsWithId func", func() {
		BeforeEach(func() {
			cmd = &cobra.Command{
				Use:   "test",
				Short: "Test command used for testing non-positional args",
				Long: "This test command is being used specifically for testing non-positional args, " +
					"so we do not confuse users with hard rules for where, for example, the ID in " +
					"`rosa edit addon ID` must be. For example, we want to be able to do `rosa edit addon " +
					"-c test <ADDON_ID>` as well as `rosa edit addon <ADDON_ID> -c test`.",
				Args: func(cmd *cobra.Command, argv []string) error {

					return nil
				},
			}
			cmd.Flags().BoolP("help", "h", false, "")
			s := ""
			cmd.Flags().StringVarP(
				&s,
				"cluster",
				"c",
				"",
				"Name or ID of the cluster.",
			)
		})
		It("Returns without error", func() {
			err := PreprocessUnknownFlagsWithId(cmd, []string{"test", "-c", "test-cluster"})
			Expect(err).ToNot(HaveOccurred())
		})
		It("Returns error with no ID", func() {
			err := PreprocessUnknownFlagsWithId(cmd, []string{"-c", "test-cluster"})
			Expect(err).To(HaveOccurred())
			Expect(fmt.Sprint(err)).To(Equal("ID argument not found in list of arguments passed to command"))
		})
		It("Returns error with flag that has no value", func() {
			err := PreprocessUnknownFlagsWithId(cmd, []string{"test", "-c", "-c", "-c", "-c"})
			Expect(err).To(HaveOccurred())
			Expect(fmt.Sprint(err)).To(Equal("No value given for flag '-c'"))
		})
	})
})
