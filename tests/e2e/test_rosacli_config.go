package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("rosacli config",
	labels.Feature.Config,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			originalConfig map[string]string
			configName     []string
			ocmService     rosacli.OCMResourceService
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmService = rosaClient.OCMResource

			originalConfig = make(map[string]string)
		})
		AfterEach(func() {
			By("Restore the original config")
			if len(originalConfig) > 0 {
				for key, value := range originalConfig {
					if key != "scopes" {
						_, err := ocmService.SetConfig(key, value)
						Expect(err).To(BeNil())
					}
				}
			}
		})

		It("to set and get config via rosacli - [id:72166]", labels.High, labels.Runtime.OCMResources, func() {
			By("Get config via rosacli")
			configName = []string{
				"access_token",
				"client_id",
				"client_secret",
				"insecure",
				"refresh_token",
				"scopes",
				"token_url",
				"url",
				"fedramp",
			}
			for _, name := range configName {
				config, err := ocmService.GetConfig(name)
				Expect(err).To(BeNil())
				if name == "client_secret" {
					Expect(strings.TrimSuffix(config.String(), "\n")).To(Equal(""))
					originalConfig[name] = ""
				} else {
					Expect(config).ToNot(Equal(""))
					originalConfig[name] = strings.TrimSuffix(config.String(), "\n")
				}
			}
			By("Set some config via rosacli")
			testingConfig := map[string]string{
				"access_token":  "test_token",
				"client_id":     "test_client_id",
				"client_secret": "test_client_secret",
				"insecure":      "true",
				"refresh_token": "test_refresh_token",
				"token_url":     "test_token_url",
				"url":           "test_url",
				"fedramp":       "true",
			}
			for key, value := range testingConfig {
				_, err := ocmService.SetConfig(key, value)
				Expect(err).To(BeNil())
			}
			By("Check if the set operation works")
			for key, value := range testingConfig {
				config, err := ocmService.GetConfig(key)
				configString := strings.TrimSuffix(config.String(), "\n")
				Expect(err).To(BeNil())
				Expect(configString).To(Equal(value))
			}
			By("Set not supported config via rosacli")
			out, err := ocmService.SetConfig("scopes", "test_scopes")
			Expect(err).ToNot(BeNil())
			Expect(out.String()).To(ContainSubstring("Setting scopes is unsupported"))
		})
	})
