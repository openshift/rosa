package config

import (
	"os"
	"testing"

	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "config suite")
}

var _ = Describe("Config", Ordered, func() {
	propNamesAndDocs := map[string]string{
		"access_token":  "Bearer access token.",
		"client_id":     "OpenID client identifier.",
		"client_secret": "OpenID client secret.",
		"insecure":      "Enables insecure communication with the server.",
		"refresh_token": "Offline or refresh token.",
		"scopes":        "OpenID scope.",
		"token_url":     "OpenID token URL.",
		"url":           "URL of the API gateway.",
		"fedramp":       "Indicates FedRAMP.",
	}

	It("Shows properties and docs for config", func() {
		propNames, docs := ConfigPropertiesNamesAndDocs()
		for i := range propNames {
			val, ok := propNamesAndDocs[propNames[i]]
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(docs[i]))
		}
	})

	It("Shows all properties for getting config", func() {
		for _, prop := range GetAllConfigProperties() {
			_, ok := propNamesAndDocs[prop]
			Expect(ok).To(BeTrue())
		}
	})

	It("Does not show disallowed properties for setting config", func() {
		allowedProperties := GetAllowedConfigProperties()
		for _, prop := range DisallowedSetConfigProperties {
			Expect(slices.Contains(allowedProperties, prop)).To(BeFalse())
		}
	})

	When("Config is present", Ordered, func() {
		var tmpdir string
		var err error

		BeforeAll(func() {
			tmpdir, err = os.MkdirTemp("/tmp", ".ocm-config-*")
			Expect(err).To(BeNil())
			os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")
		})

		AfterAll(func() {
			os.Setenv("OCM_CONFIG", "")
		})

		It("Saves and loads config", func() {
			url := "mytesturl"
			cfg := &Config{
				URL: url,
			}
			Save(cfg)

			myconf, err := Load()
			Expect(err).To(BeNil())
			Expect(myconf.URL).To(Equal(url))
		})
	})

	When("Config is not present", Ordered, func() {
		BeforeAll(func() {
			os.Setenv("OCM_CONFIG", "invalid-config.json")
		})

		AfterAll(func() {
			os.Setenv("OCM_CONFIG", "")
		})

		It("Saves and loads config", func() {
			myconf, err := Load()
			Expect(err).To(BeNil())
			Expect(myconf).To(BeNil())
		})
	})
})
