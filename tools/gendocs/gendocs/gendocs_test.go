package gendocs_test

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	. "github.com/openshift/rosa/tools/gendocs/gendocs"
)

var _ = Describe("Gendocs", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "gendocs-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("GenDocs", func() {
		Context("when generating documentation", func() {
			It("generates documentation for a command with examples", func() {
				// Create a simple template
				templatePath := filepath.Join(tmpDir, "template")
				templateContent := `{{range .Items}}{{.FullName}}: {{.Description}}
{{end}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				// Create test command
				rootCmd := &cobra.Command{
					Use:     "test",
					Short:   "Test command",
					Example: "test example",
				}

				// Generate docs
				outputPath := filepath.Join(tmpDir, "output.adoc")

				// Change directory to tmpDir so template path works
				oldDir, err := os.Getwd()
				Expect(err).ToNot(HaveOccurred())
				defer os.Chdir(oldDir)

				err = os.Chdir(tmpDir)
				Expect(err).ToNot(HaveOccurred())

				// Create templates/clibyexample directory structure
				err = os.MkdirAll("templates/clibyexample", 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.WriteFile("templates/clibyexample/template", []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				err = GenDocs(rootCmd, outputPath)
				Expect(err).ToNot(HaveOccurred())

				// Verify output file exists
				_, err = os.Stat(outputPath)
				Expect(err).ToNot(HaveOccurred())

				// Verify content
				content, err := os.ReadFile(outputPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("test: Test command"))
			})

			It("generates documentation for commands with subcommands", func() {
				// Create template
				templatePath := filepath.Join(tmpDir, "template")
				templateContent := `{{range .Items}}{{.FullName}}
{{end}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				// Create test command with subcommands
				rootCmd := &cobra.Command{
					Use:     "test",
					Short:   "Test command",
					Example: "test example",
				}
				subCmd := &cobra.Command{
					Use:     "sub",
					Short:   "Sub command",
					Example: "test sub example",
				}
				rootCmd.AddCommand(subCmd)

				// Generate docs
				outputPath := filepath.Join(tmpDir, "output.adoc")

				oldDir, err := os.Getwd()
				Expect(err).ToNot(HaveOccurred())
				defer os.Chdir(oldDir)

				err = os.Chdir(tmpDir)
				Expect(err).ToNot(HaveOccurred())

				err = os.MkdirAll("templates/clibyexample", 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.WriteFile("templates/clibyexample/template", []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				err = GenDocs(rootCmd, outputPath)
				Expect(err).ToNot(HaveOccurred())

				// Verify content includes both commands
				content, err := os.ReadFile(outputPath)
				Expect(err).ToNot(HaveOccurred())
				contentStr := string(content)
				Expect(contentStr).To(ContainSubstring("test"))
				Expect(contentStr).To(ContainSubstring("test sub"))
			})

			It("skips hidden commands", func() {
				templatePath := filepath.Join(tmpDir, "template")
				templateContent := `{{range .Items}}{{.FullName}}
{{end}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				// Create command with hidden subcommand
				rootCmd := &cobra.Command{
					Use:     "test",
					Short:   "Test command",
					Example: "test example",
				}
				hiddenCmd := &cobra.Command{
					Use:     "hidden",
					Short:   "Hidden command",
					Example: "test hidden example",
					Hidden:  true,
				}
				rootCmd.AddCommand(hiddenCmd)

				outputPath := filepath.Join(tmpDir, "output.adoc")

				oldDir, err := os.Getwd()
				Expect(err).ToNot(HaveOccurred())
				defer os.Chdir(oldDir)

				err = os.Chdir(tmpDir)
				Expect(err).ToNot(HaveOccurred())

				err = os.MkdirAll("templates/clibyexample", 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.WriteFile("templates/clibyexample/template", []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				err = GenDocs(rootCmd, outputPath)
				Expect(err).ToNot(HaveOccurred())

				// Verify hidden command is not in output
				content, err := os.ReadFile(outputPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).ToNot(ContainSubstring("hidden"))
			})

			It("skips commands without examples", func() {
				templatePath := filepath.Join(tmpDir, "template")
				templateContent := `{{range .Items}}{{.FullName}}
{{end}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				// Create command without examples
				rootCmd := &cobra.Command{
					Use:     "test",
					Short:   "Test command",
					Example: "", // No example
				}

				outputPath := filepath.Join(tmpDir, "output.adoc")

				oldDir, err := os.Getwd()
				Expect(err).ToNot(HaveOccurred())
				defer os.Chdir(oldDir)

				err = os.Chdir(tmpDir)
				Expect(err).ToNot(HaveOccurred())

				err = os.MkdirAll("templates/clibyexample", 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.WriteFile("templates/clibyexample/template", []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				err = GenDocs(rootCmd, outputPath)
				Expect(err).ToNot(HaveOccurred())

				// Verify output is essentially empty (just template header)
				content, err := os.ReadFile(outputPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.TrimSpace(string(content))).To(BeEmpty())
			})

			It("returns error when template file does not exist", func() {
				rootCmd := &cobra.Command{
					Use:     "test",
					Short:   "Test command",
					Example: "test example",
				}

				outputPath := filepath.Join(tmpDir, "output.adoc")

				oldDir, err := os.Getwd()
				Expect(err).ToNot(HaveOccurred())
				defer os.Chdir(oldDir)

				err = os.Chdir(tmpDir)
				Expect(err).ToNot(HaveOccurred())

				err = GenDocs(rootCmd, outputPath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to read template file"))
			})

			It("returns error when template is invalid", func() {
				templatePath := filepath.Join(tmpDir, "template")
				// Invalid template syntax
				templateContent := `{{range .Items}{{.FullName}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				rootCmd := &cobra.Command{
					Use:     "test",
					Short:   "Test command",
					Example: "test example",
				}

				outputPath := filepath.Join(tmpDir, "output.adoc")

				oldDir, err := os.Getwd()
				Expect(err).ToNot(HaveOccurred())
				defer os.Chdir(oldDir)

				err = os.Chdir(tmpDir)
				Expect(err).ToNot(HaveOccurred())

				err = os.MkdirAll("templates/clibyexample", 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.WriteFile("templates/clibyexample/template", []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				err = GenDocs(rootCmd, outputPath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to parse template"))
			})

			It("trims whitespace from examples", func() {
				templatePath := filepath.Join(tmpDir, "template")
				templateContent := `{{range .Items}}Examples: {{.Examples}}
{{end}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				// Create command with whitespace in examples
				rootCmd := &cobra.Command{
					Use:     "test",
					Short:   "Test command",
					Example: "  \n  test example  \n  ",
				}

				outputPath := filepath.Join(tmpDir, "output.adoc")

				oldDir, err := os.Getwd()
				Expect(err).ToNot(HaveOccurred())
				defer os.Chdir(oldDir)

				err = os.Chdir(tmpDir)
				Expect(err).ToNot(HaveOccurred())

				err = os.MkdirAll("templates/clibyexample", 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.WriteFile("templates/clibyexample/template", []byte(templateContent), 0644)
				Expect(err).ToNot(HaveOccurred())

				err = GenDocs(rootCmd, outputPath)
				Expect(err).ToNot(HaveOccurred())

				content, err := os.ReadFile(outputPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal("Examples: test example\n"))
			})
		})
	})
})
