package logout

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/properties"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var _ = Describe("logout command", func() {
	var tmpdir string

	BeforeEach(func() {
		var err error
		tmpdir, err = os.MkdirTemp("", ".ocm-logout-test-*")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")).To(Succeed())
		DeferCleanup(os.RemoveAll, tmpdir)
		DeferCleanup(os.Unsetenv, "OCM_CONFIG")
	})

	It("Removes config file successfully when it exists", func() {
		cfg := &config.Config{
			AccessToken: "test-token",
			URL:         "https://api.example.com",
		}
		err := config.Save(cfg)
		Expect(err).NotTo(HaveOccurred())

		err = runLogout(rprtr.CreateReporter(), false)
		Expect(err).NotTo(HaveOccurred())

		_, statErr := os.Stat(tmpdir + "/ocm_config.json")
		Expect(os.IsNotExist(statErr)).To(BeTrue())
	})

	It("Returns nil when config file does not exist", func() {
		err := runLogout(rprtr.CreateReporter(), false)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Removes keyring-managed config successfully", func() {
		Expect(os.Setenv(properties.KeyringEnvKey, "test-keyring")).To(Succeed())
		DeferCleanup(os.Unsetenv, properties.KeyringEnvKey)

		called := false
		originalRemoveConfigFromKeyring := config.RemoveConfigFromKeyring
		DeferCleanup(func() {
			config.RemoveConfigFromKeyring = originalRemoveConfigFromKeyring
		})
		config.RemoveConfigFromKeyring = func(_ string) error {
			called = true
			return nil
		}

		err := runLogout(rprtr.CreateReporter(), false)
		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeTrue())
	})

	It("Returns an error when keyring removal fails", func() {
		Expect(os.Setenv(properties.KeyringEnvKey, "test-keyring")).To(Succeed())
		DeferCleanup(os.Unsetenv, properties.KeyringEnvKey)

		originalRemoveConfigFromKeyring := config.RemoveConfigFromKeyring
		DeferCleanup(func() {
			config.RemoveConfigFromKeyring = originalRemoveConfigFromKeyring
		})
		config.RemoveConfigFromKeyring = func(_ string) error {
			return fmt.Errorf("keyring locked")
		}

		err := runLogout(rprtr.CreateReporter(), false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("keyring"))
	})

	It("Rejects extra arguments via cobra.NoArgs", func() {
		Expect(Cmd.Args).NotTo(BeNil())
		err := Cmd.Args(Cmd, []string{"unexpected"})
		Expect(err).To(HaveOccurred())
	})

	Describe("--hyperfleet flag", func() {
		It("clears HyperfleetURL from config leaving other fields intact", func() {
			cfg := &config.Config{
				AccessToken:   "test-token",
				URL:           "https://api.example.com",
				HyperfleetURL: "https://abc.execute-api.us-east-1.amazonaws.com",
			}
			Expect(config.Save(cfg)).To(Succeed())

			err := runLogout(rprtr.CreateReporter(), true)
			Expect(err).NotTo(HaveOccurred())

			loaded, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.HyperfleetURL).To(BeEmpty())
			Expect(loaded.AccessToken).To(Equal("test-token"))
			Expect(loaded.URL).To(Equal("https://api.example.com"))
		})

		It("reports not logged in when HyperfleetURL is already empty", func() {
			cfg := &config.Config{AccessToken: "test-token"}
			Expect(config.Save(cfg)).To(Succeed())

			err := runLogout(rprtr.CreateReporter(), true)
			Expect(err).NotTo(HaveOccurred())

			loaded, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.HyperfleetURL).To(BeEmpty())
			Expect(loaded.AccessToken).To(Equal("test-token"))
		})

		It("reports not logged in when config file does not exist", func() {
			err := runLogout(rprtr.CreateReporter(), true)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
