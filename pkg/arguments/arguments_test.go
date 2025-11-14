package arguments

import (
	"fmt"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Client", func() {
	var (
		cmd                      *cobra.Command
		childCmd                 *cobra.Command
		o                        string
		disableRegionDeprecation bool
	)

	Context("Region deprecation test", func() {
		BeforeEach(func() {
			cmd = &cobra.Command{
				Use:   "test",
				Short: "Test command used for testing deprecation",
				Long: "This command is used for testing the deprecation of the 'region' flag in " +
					"arguments.go - it is used for nothing else.",
			}
			childCmd = &cobra.Command{
				Use:   "child",
				Short: "Child command used for testing deprecation",
				Long: "This child command is used for testing the deprecation of the 'region' flag in " +
					"arguments.go - it is used for nothing else.",
				Run: func(c *cobra.Command, a []string) {
					//nothing to be done
				},
			}

			cmd.AddCommand(childCmd)

			AddRegionFlag(cmd.PersistentFlags())
			AddDebugFlag(cmd.PersistentFlags())
			flagSet := cmd.PersistentFlags()
			flagSet.StringVarP(
				&o,
				"output",
				"o",
				"",
				"",
			)
			flagSet.BoolVarP(
				&disableRegionDeprecation,
				DisableRegionDeprecationFlagName,
				"",
				true,
				"",
			)
		})
		It("Without setting region", func() {
			MarkRegionDeprecated(cmd, []*cobra.Command{childCmd})
			original := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			childCmd.Run(childCmd, []string{})
			err := w.Close()
			Expect(err).ToNot(HaveOccurred())
			out, _ := io.ReadAll(r)
			os.Stdout = original
			Expect(string(out)).To(BeEmpty())
		})
		It("Setting region", func() {
			MarkRegionDeprecated(cmd, []*cobra.Command{childCmd})
			original := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			childCmd.Flag("region").Value.Set("us-east-1")
			childCmd.Run(childCmd, []string{})
			err := w.Close()
			Expect(err).ToNot(HaveOccurred())
			out, _ := io.ReadAll(r)
			os.Stdout = original
			Expect(string(out)).To(ContainSubstring(regionDeprecationMessage))
		})
		It("Setting output to json", func() {
			MarkRegionDeprecated(cmd, []*cobra.Command{childCmd})
			original := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			childCmd.Flag("region").Value.Set("us-east-1")
			childCmd.Flag("output").Value.Set("json")
			childCmd.Run(childCmd, []string{})
			err := w.Close()
			Expect(err).ToNot(HaveOccurred())
			out, _ := io.ReadAll(r)
			os.Stdout = original
			Expect(string(out)).To(BeEmpty())
		})
		It("Nested function (disableRegionDeprecation flag)", func() {
			MarkRegionDeprecated(cmd, []*cobra.Command{childCmd})
			original := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			childCmd.Flag(DisableRegionDeprecationFlagName).Value.Set("true")
			childCmd.Run(childCmd, []string{})
			err := w.Close()
			Expect(err).ToNot(HaveOccurred())
			out, _ := io.ReadAll(r)
			os.Stdout = original
			Expect(string(out)).To(BeEmpty())
			childCmd.Flag(DisableRegionDeprecationFlagName).Value.Set("false")
		})
	})

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
			Expect(fmt.Sprint(err)).To(Equal("no value given for flag '-c'"))
		})
	})
})
