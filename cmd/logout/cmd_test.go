package logout

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/properties"
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

		err = runLogout()
		Expect(err).NotTo(HaveOccurred())

		_, statErr := os.Stat(tmpdir + "/ocm_config.json")
		Expect(os.IsNotExist(statErr)).To(BeTrue())
	})

	It("Returns nil when config file does not exist", func() {
		err := runLogout()
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

		err := runLogout()
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

		err := runLogout()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("keyring"))
	})

	It("Rejects extra arguments via cobra.NoArgs", func() {
		Expect(Cmd.Args).NotTo(BeNil())
		err := Cmd.Args(Cmd, []string{"unexpected"})
		Expect(err).To(HaveOccurred())
	})
})
