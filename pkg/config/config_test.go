package config

import (
	"encoding/base64"
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
		"access_token":   "Bearer access token.",
		"client_id":      "OpenID client identifier.",
		"client_secret":  "OpenID client secret.",
		"insecure":       "Enables insecure communication with the server.",
		"refresh_token":  "Offline or refresh token.",
		"scopes":         "OpenID scope.",
		"token_url":      "OpenID token URL.",
		"url":            "URL of the API gateway.",
		"user_agent":     "OCM client UserAgent. Default value is used if not set.",
		"version":        "OCM client version. Default value is used if not set.",
		"fedramp":        "Indicates FedRAMP.",
		"hyperfleet_url": "Platform API v2 endpoint URL.",
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

var _ = Describe("Config error paths", func() {
	var tmpdir string

	// buildJWT creates a minimal unsigned JWT from the given claims payload.
	buildJWT := func(claims map[string]interface{}) string {
		headerJSON, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
		payloadJSON, _ := json.Marshal(claims)
		h := base64.RawURLEncoding.EncodeToString(headerJSON)
		p := base64.RawURLEncoding.EncodeToString(payloadJSON)
		return h + "." + p + "."
	}

	BeforeEach(func() {
		var err error
		tmpdir, err = os.MkdirTemp("/tmp", ".ocm-config-err-*")
		Expect(err).NotTo(HaveOccurred())
		os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")
	})

	AfterEach(func() {
		os.Setenv("OCM_CONFIG", "")
		os.RemoveAll(tmpdir)
	})

	It("loadFromFile returns error for invalid JSON content", func() {
		err := os.WriteFile(tmpdir+"/ocm_config.json", []byte("{not valid json"), 0600)
		Expect(err).NotTo(HaveOccurred())

		_, err = Load()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to parse config file"))
	})

	It("GetData returns error for unparseable access token", func() {
		cfg := &Config{AccessToken: "not-a-valid-jwt"}
		_, err := cfg.GetData("username")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to parse token"))
	})

	It("GetData returns error when claim is missing from token", func() {
		token := buildJWT(map[string]interface{}{"sub": "test", "exp": 9999999999})
		cfg := &Config{AccessToken: token}
		_, err := cfg.GetData("nonexistent_claim")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not contain"))
	})

	It("GetData returns error when claim value is not a string", func() {
		token := buildJWT(map[string]interface{}{"sub": "test", "exp": 9999999999, "num_claim": 42})
		cfg := &Config{AccessToken: token}
		_, err := cfg.GetData("num_claim")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("expected string"))
	})

	It("loadFromFile returns error when config path is a directory", func() {
		os.Setenv("OCM_CONFIG", tmpdir)

		_, err := Load()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to read config file"))
	})

	It("Armed returns error for invalid access token", func() {
		cfg := &Config{AccessToken: "not-a-valid-jwt"}
		_, err := cfg.Armed()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to parse token"))
	})

	It("Armed returns error for invalid refresh token", func() {
		cfg := &Config{RefreshToken: "not-a-valid-jwt"}
		_, err := cfg.Armed()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to parse token"))
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
