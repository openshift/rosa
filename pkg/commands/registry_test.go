package commands

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("RegisterCommands", func() {
	var rootCmd *cobra.Command

	BeforeEach(func() {
		// Create a fresh root command for each test
		rootCmd = &cobra.Command{
			Use:   "test",
			Short: "Test command",
			Long:  "Test command for verifying command registration",
		}
	})

	Context("when registering all ROSA CLI commands", func() {
		It("successfully registers all expected commands", func() {
			// Call the function under test
			RegisterCommands(rootCmd)

			// Verify commands were registered
			commands := rootCmd.Commands()
			Expect(commands).ToNot(BeEmpty())

			// Verify the expected number of commands are registered
			// As of this test, there should be 29 top-level commands
			Expect(len(commands)).To(Equal(29))

			// Verify specific critical commands are present
			commandNames := make(map[string]bool)
			for _, cmd := range commands {
				commandNames[cmd.Name()] = true
			}

			// Check for essential commands
			expectedCommands := []string{
				"completion",
				"create",
				"describe",
				"delete",
				"docs",
				"download",
				"edit",
				"grant",
				"list",
				"init",
				"install",
				"login",
				"logout",
				"logs",
				"register",
				"revoke",
				"uninstall",
				"upgrade",
				"verify",
				"version",
				"whoami",
				"hibernate",
				"resume",
				"link",
				"unlink",
				"token",
				"config",
				"attach",
				"detach",
			}

			for _, cmdName := range expectedCommands {
				Expect(commandNames[cmdName]).To(BeTrue(), "Expected command '%s' to be registered", cmdName)
			}
		})

		It("does not modify the root command's basic properties", func() {
			originalUse := rootCmd.Use
			originalShort := rootCmd.Short
			originalLong := rootCmd.Long

			RegisterCommands(rootCmd)

			// Verify root command properties remain unchanged
			Expect(rootCmd.Use).To(Equal(originalUse))
			Expect(rootCmd.Short).To(Equal(originalShort))
			Expect(rootCmd.Long).To(Equal(originalLong))
		})

		It("can be called multiple times without error", func() {
			// First registration
			RegisterCommands(rootCmd)
			firstCount := len(rootCmd.Commands())

			// Create a new root command for second registration
			rootCmd2 := &cobra.Command{
				Use:   "test2",
				Short: "Test command 2",
				Long:  "Test command 2 for verifying command registration",
			}

			// Second registration on different root
			RegisterCommands(rootCmd2)
			secondCount := len(rootCmd2.Commands())

			// Both should have the same number of commands
			Expect(firstCount).To(Equal(secondCount))
			Expect(firstCount).To(Equal(29))
		})
	})
})
