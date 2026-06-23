package fedramp

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/config"
)

func TestFedramp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fedramp Suite")
}

var _ = Describe("Fedramp", func() {
	var (
		previousEnabled   bool
		previousOcmConfig string
	)

	BeforeEach(func() {
		previousEnabled = enabled
		previousOcmConfig = os.Getenv("OCM_CONFIG")
		enabled = false
		Expect(os.Setenv("OCM_CONFIG", "")).To(Succeed())
	})

	AfterEach(func() {
		enabled = previousEnabled
		Expect(os.Setenv("OCM_CONFIG", previousOcmConfig)).To(Succeed())
	})

	Describe("AddFlag and flag detection", func() {
		It("registers the govcloud and admin flags and detects when they are changed", func() {
			cmd := &cobra.Command{Use: "test"}

			AddFlag(cmd.Flags())

			Expect(cmd.Flags().Lookup("govcloud")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("admin")).NotTo(BeNil())
			Expect(HasFlag(cmd)).To(BeFalse())
			Expect(HasAdminFlag(cmd)).To(BeFalse())

			Expect(cmd.Flags().Set("govcloud", "true")).To(Succeed())
			Expect(HasFlag(cmd)).To(BeTrue())

			Expect(cmd.Flags().Set("admin", "true")).To(Succeed())
			Expect(HasAdminFlag(cmd)).To(BeTrue())
		})
	})

	Describe("Enabled", func() {
		It("returns true when the in-memory flag is already enabled", func() {
			Enable()

			Expect(Enabled()).To(BeTrue())
		})

		It("loads a valid FedRAMP config from disk and caches the enabled state", func() {
			tempDir := GinkgoT().TempDir()
			Expect(os.Setenv("OCM_CONFIG", filepath.Join(tempDir, "ocm.json"))).To(Succeed())
			Expect(config.Save(&config.Config{
				AccessToken: "token",
				ClientID:    "client",
				TokenURL:    "https://sso.example.com/token",
				URL:         "https://api.example.com",
				FedRAMP:     true,
			})).To(Succeed())

			Expect(Enabled()).To(BeTrue())
			Expect(enabled).To(BeTrue())
		})

		It("returns false when the loaded config is invalid", func() {
			tempDir := GinkgoT().TempDir()
			Expect(os.Setenv("OCM_CONFIG", filepath.Join(tempDir, "ocm.json"))).To(Succeed())
			Expect(config.Save(&config.Config{
				FedRAMP: true,
			})).To(Succeed())

			Expect(Enabled()).To(BeFalse())
		})

		It("returns false when the config file does not exist", func() {
			tempDir := GinkgoT().TempDir()
			Expect(os.Setenv("OCM_CONFIG", filepath.Join(tempDir, "missing.json"))).To(Succeed())

			Expect(Enabled()).To(BeFalse())
		})
	})

	Describe("Disable", func() {
		It("clears the in-memory flag and persists FedRAMP=false for a valid config", func() {
			tempDir := GinkgoT().TempDir()
			Expect(os.Setenv("OCM_CONFIG", filepath.Join(tempDir, "ocm.json"))).To(Succeed())
			Expect(config.Save(&config.Config{
				AccessToken: "token",
				ClientID:    "client",
				TokenURL:    "https://sso.example.com/token",
				URL:         "https://api.example.com",
				FedRAMP:     true,
			})).To(Succeed())
			Enable()

			Disable()

			Expect(enabled).To(BeFalse())
			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
			Expect(cfg.FedRAMP).To(BeFalse())
		})
	})

	Describe("IsGovRegion", func() {
		It("recognizes the GovCloud regions", func() {
			Expect(IsGovRegion("us-gov-west-1")).To(BeTrue())
			Expect(IsGovRegion("us-gov-east-1")).To(BeTrue())
		})

		It("rejects non-GovCloud regions", func() {
			Expect(IsGovRegion("us-east-1")).To(BeFalse())
			Expect(IsGovRegion("")).To(BeFalse())
		})
	})

	Describe("IsValidEnv", func() {
		It("recognizes known environments", func() {
			Expect(IsValidEnv("production")).To(BeTrue())
			Expect(IsValidEnv("staging")).To(BeTrue())
			Expect(IsValidEnv("integration")).To(BeTrue())
		})

		It("rejects unknown environments", func() {
			Expect(IsValidEnv("dev")).To(BeFalse())
			Expect(IsValidEnv("")).To(BeFalse())
		})
	})
})
