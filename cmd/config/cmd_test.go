package config

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/cmd/config/get"
	"github.com/openshift/rosa/cmd/config/set"
	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/test"
)

func TestConfigCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa config command")
}

var _ = Describe("Run Command", Ordered, func() {
	var testRuntime test.TestingRuntime
	var buf *bytes.Buffer
	var tmpdir string
	var err error

	When("Config file exists", Ordered, func() {
		BeforeAll(func() {
			buf = new(bytes.Buffer)
			get.Writer = buf
			testRuntime.InitRuntime()
			tmpdir, err = os.MkdirTemp("/tmp", ".ocm-config-*")
			os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")
		})

		AfterAll(func() {
			os.Setenv("OCM_CONFIG", "")
		})

		It("Saves config", func() {
			accessToken := "MyTestToken"
			err = set.SaveConfig("access_token", accessToken)
			Expect(err).To(BeNil())
			currentConfig, err := config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.AccessToken).To(Equal(accessToken))

			clientId := "MyClientId"
			err = set.SaveConfig("client_id", clientId)
			Expect(err).To(BeNil())
			currentConfig, err = config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.ClientID).To(Equal(clientId))

			insecure := "true"
			err = set.SaveConfig("insecure", insecure)
			Expect(err).To(BeNil())
			currentConfig, err = config.Load()
			Expect(err).To(BeNil())
			Expect(strconv.FormatBool(currentConfig.Insecure)).To(Equal(insecure))

			refreshToken := "MyRefreshToken"
			err = set.SaveConfig("refresh_token", refreshToken)
			Expect(err).To(BeNil())
			currentConfig, err = config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.RefreshToken).To(Equal(refreshToken))

			scopes := "MyScopes"
			err = set.SaveConfig("scopes", scopes)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("Setting scopes is unsupported"))

			tokenUrl := "MyTokenURL"
			err = set.SaveConfig("token_url", tokenUrl)
			Expect(err).To(BeNil())
			currentConfig, err = config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.TokenURL).To(Equal(tokenUrl))

			url := "MyURL"
			err = set.SaveConfig("url", url)
			Expect(err).To(BeNil())
			currentConfig, err = config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.URL).To(Equal(url))

			fedramp := "true"
			err = set.SaveConfig("fedramp", fedramp)
			Expect(err).To(BeNil())
			currentConfig, err = config.Load()
			Expect(err).To(BeNil())
			Expect(strconv.FormatBool(currentConfig.FedRAMP)).To(Equal(fedramp))

			insecure = "Incorrect"
			err = set.SaveConfig("insecure", insecure)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("Failed to set insecure"))

			randomField := "random value"
			err = set.SaveConfig("random_field", randomField)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("'random_field' is not a supported setting"))
		})

		It("Prints config", func() {
			currentConfig, err := config.Load()
			Expect(err).To(BeNil())
			err = get.PrintConfig("access_token")
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring(currentConfig.AccessToken))

			err = get.PrintConfig("client_id")
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring(currentConfig.ClientID))

			err = get.PrintConfig("client_secret")
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring(currentConfig.ClientSecret))

			err = get.PrintConfig("insecure")
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring(strconv.FormatBool(currentConfig.Insecure)))

			err = get.PrintConfig("refresh_token")
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring(currentConfig.RefreshToken))

			err = get.PrintConfig("scopes")
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring(fmt.Sprint(currentConfig.Scopes)))

			err = get.PrintConfig("token_url")
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring(currentConfig.TokenURL))

			err = get.PrintConfig("url")
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring(currentConfig.URL))

			err = get.PrintConfig("fedramp")
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring(strconv.FormatBool(currentConfig.FedRAMP)))

			err = get.PrintConfig("test")
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("'test' is not a supported setting"))
		})
	})

	When("Config file doesn't exist", func() {
		AfterEach(func() {
			os.Setenv("OCM_CONFIG", "")
		})

		It("Does not show error when config is empty", func() {
			os.Setenv("OCM_CONFIG", "//invalid/path")
			config.Save(nil)
			_, err := config.Load()
			Expect(err).To(BeNil())
			err = get.PrintConfig("access_token")
			Expect(err).To(BeNil())
		})
	})
})
