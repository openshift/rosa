package config

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/properties"
)

type mockSpy struct {
	calledUpsert bool
	calledRemove bool
	calledGet    bool
	testCfg      []byte
	upsertErr    error
	removeErr    error
	getErr       error
}

func (m *mockSpy) MockUpsertConfigToKeyring(keyring string, data []byte) error {
	m.calledUpsert = true
	return m.upsertErr
}

func (m *mockSpy) MockRemoveConfigFromKeyring(keyring string) error {
	m.calledRemove = true
	return m.removeErr
}

func (m *mockSpy) MockGetConfigFromKeyring(keyring string) ([]byte, error) {
	m.calledGet = true
	return m.testCfg, m.getErr
}

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
		"user_agent":    "OCM clients UserAgent. Default value is used if not set.",
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
			Expect(err).NotTo(HaveOccurred())
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
			Expect(err).NotTo(HaveOccurred())
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
			Expect(err).NotTo(HaveOccurred())
			Expect(myconf).To(BeNil())
		})
	})

	When("Persisting tokens", Ordered, func() {
		var tmpdir string
		var err error

		BeforeAll(func() {
			tmpdir, err = os.MkdirTemp("/tmp", ".ocm-config-*")
			Expect(err).NotTo(HaveOccurred())
			os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")
		})

		AfterAll(func() {
			os.Setenv("OCM_CONFIG", "")
		})

		It("Uses existing config and saves", func() {
			cfg := &Config{}
			err := PersistTokens(cfg, "foo", "bar")
			Expect(err).NotTo(HaveOccurred())

			myconf, err := Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(myconf.AccessToken).To(Equal("foo"))
			Expect(myconf.RefreshToken).To(Equal("bar"))
		})

		It("Loads config and saves", func() {
			err := PersistTokens(nil, "foo", "bar")
			Expect(err).NotTo(HaveOccurred())

			myconf, err := Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(myconf.AccessToken).To(Equal("foo"))
			Expect(myconf.RefreshToken).To(Equal("bar"))
		})
	})

})
var _ = Describe("Config Keyring", func() {
	When("Load()", func() {
		Context(properties.KeyringEnvKey+" is set", func() {
			BeforeEach(func() {
				os.Setenv(properties.KeyringEnvKey, "keyring")
			})

			AfterEach(func() {
				os.Setenv(properties.KeyringEnvKey, "")
			})

			It("Returns a valid config", func() {
				data := generateConfigBytes(Config{
					AccessToken: "access_token",
				})
				mockSpy := &mockSpy{testCfg: data}
				GetConfigFromKeyring = mockSpy.MockGetConfigFromKeyring

				cfg, err := Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).ToNot(BeNil())
				Expect(cfg.AccessToken).To(Equal("access_token"))
				Expect(mockSpy.calledGet).To(BeTrue())
			})

			It("Returns nil for no config content", func() {
				mockSpy := &mockSpy{testCfg: nil}
				GetConfigFromKeyring = mockSpy.MockGetConfigFromKeyring

				cfg, err := Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).To(BeNil())
				Expect(mockSpy.calledGet).To(BeTrue())
			})

			It("Returns nil for invalid config content", func() {
				data := generateInvalidConfigBytes()
				mockSpy := &mockSpy{testCfg: data}
				GetConfigFromKeyring = mockSpy.MockGetConfigFromKeyring

				cfg, err := Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).To(BeNil())
				Expect(mockSpy.calledGet).To(BeTrue())
			})

			It("Handles Error", func() {
				data := generateInvalidConfigBytes()
				mockSpy := &mockSpy{testCfg: data}
				mockSpy.getErr = fmt.Errorf("error")
				GetConfigFromKeyring = mockSpy.MockGetConfigFromKeyring

				cfg, err := Load()
				Expect(err).NotTo(BeNil())
				Expect(cfg).To(BeNil())
				Expect(mockSpy.calledGet).To(BeTrue())
			})
		})
	})

	When("Save()", func() {
		Context(properties.KeyringEnvKey+" is set", func() {
			BeforeEach(func() {
				os.Setenv(properties.KeyringEnvKey, "keyring")
			})

			AfterEach(func() {
				os.Setenv(properties.KeyringEnvKey, "")
			})

			It("Saves a valid config", func() {
				data := &Config{
					AccessToken: "access_token",
				}
				mockSpy := &mockSpy{}
				UpsertConfigToKeyring = mockSpy.MockUpsertConfigToKeyring

				err := Save(data)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockSpy.calledUpsert).To(BeTrue())
			})

			It("Handles Error", func() {
				data := &Config{
					AccessToken: "access_token",
				}
				mockSpy := &mockSpy{}
				mockSpy.upsertErr = fmt.Errorf("error")
				UpsertConfigToKeyring = mockSpy.MockUpsertConfigToKeyring

				err := Save(data)
				Expect(err).NotTo(BeNil())
				Expect(mockSpy.calledUpsert).To(BeTrue())
			})
		})
	})

	When("Remove()", func() {
		Context(properties.KeyringEnvKey+" is set", func() {
			BeforeEach(func() {
				os.Setenv(properties.KeyringEnvKey, "keyring")
			})

			AfterEach(func() {
				os.Setenv(properties.KeyringEnvKey, "")
			})

			It("Removes a config", func() {
				mockSpy := &mockSpy{}
				RemoveConfigFromKeyring = mockSpy.MockRemoveConfigFromKeyring

				err := Remove()
				Expect(err).NotTo(HaveOccurred())
				Expect(mockSpy.calledRemove).To(BeTrue())
			})

			It("Handles Error", func() {
				mockSpy := &mockSpy{}
				mockSpy.removeErr = fmt.Errorf("error")
				RemoveConfigFromKeyring = mockSpy.MockRemoveConfigFromKeyring

				err := Remove()
				Expect(err).NotTo(BeNil())
				Expect(mockSpy.calledRemove).To(BeTrue())
			})
		})
	})
})

func generateInvalidConfigBytes() []byte {
	return []byte("foo")
}

func generateConfigBytes(config Config) []byte {
	data := &config
	jsonData, err := json.Marshal(data)
	Expect(err).NotTo(HaveOccurred())

	return jsonData
}
