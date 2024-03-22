package ocm

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sdk "github.com/openshift-online/ocm-sdk-go"

	"github.com/openshift/rosa/pkg/config"
)

var _ = Describe("Gateway URL Resolution", func() {

	var nilConfig *config.Config = nil
	var emptyConfig = &config.Config{}
	var emptyURLConfig = &config.Config{URL: ""}
	var nonEmptyURLConfig = &config.Config{URL: "https://api.example.com"}
	validUrlOverrides := []string{
		"https://api.example.com", "http://api.example.com", "http://localhost",
		"http://localhost:8080", "https://localhost:8080/", "unix://my.server.com/tmp/api.socket",
		"unix+https://my.server.com/tmp/api.socket", "h2c://api.example.com",
		"unix+h2c://my.server.com/tmp/api.socket",
	}
	invalidUrlOverrides := []string{
		//nolint:misspell // intentional misspellings
		"productin", "PRod", //alias typo
		"localhost", "192.168.1.1", "api.openshift.com", //ip address/hostname without protocol
	}
	When("Resolving gatewayURL", func() {
		It("Priority 1 - cli arg valid url aliases", func() {
			for alias, url := range URLAliases {
				resolved, err := ResolveGatewayUrl(alias, nilConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolved).To(Equal(url))

				resolved, err = ResolveGatewayUrl(alias, emptyConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolved).To(Equal(url))

				resolved, err = ResolveGatewayUrl(alias, emptyURLConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolved).To(Equal(url))

				resolved, err = ResolveGatewayUrl(alias, nonEmptyURLConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolved).To(Equal(url))
			}
		})

		It("Priority 2 - cli arg valid url", func() {
			for _, urlOverride := range validUrlOverrides {
				resolved, err := ResolveGatewayUrl(urlOverride, nilConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolved).To(Equal(urlOverride))

				resolved, err = ResolveGatewayUrl(urlOverride, emptyConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolved).To(Equal(urlOverride))

				resolved, err = ResolveGatewayUrl(urlOverride, emptyURLConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolved).To(Equal(urlOverride))

				resolved, err = ResolveGatewayUrl(urlOverride, nonEmptyURLConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolved).To(Equal(urlOverride))
			}
		})

		It("Priority 3 - valid config url", func() {
			for _, urlOverride := range validUrlOverrides {
				resolved, err := ResolveGatewayUrl("", &config.Config{URL: urlOverride})
				Expect(err).ToNot(HaveOccurred())
				Expect(resolved).To(Equal(urlOverride))
			}
		})

		It("Priority 4 - api.openshift.com", func() {
			resolved, err := ResolveGatewayUrl("", nilConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(sdk.DefaultURL))

			resolved, err = ResolveGatewayUrl("", emptyConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(sdk.DefaultURL))

			resolved, err = ResolveGatewayUrl("", emptyURLConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(sdk.DefaultURL))
		})

		It("Invalid url alias throws an error", func() {
			for _, urlOverride := range invalidUrlOverrides {
				_, err := ResolveGatewayUrl(urlOverride, nilConfig)
				Expect(err).To(HaveOccurred())
			}
		})

		It("Invalid cfg.URL throws an error", func() {
			for _, urlOverride := range invalidUrlOverrides {
				_, err := ResolveGatewayUrl("", &config.Config{URL: urlOverride})
				Expect(err).To(HaveOccurred())
			}
		})
	})
	When("Getting env", Ordered, func() {
		BeforeAll(func() {
			tmpdir, err := os.MkdirTemp("/tmp", ".ocm-config-*")
			Expect(err).To(BeNil())
			os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")
		})

		AfterAll(func() {
			os.Setenv("OCM_CONFIG", "")
		})

		It("Returns a valid OCM stage env", func() {
			url := "https://api.stage.openshift.com"
			cfg := &config.Config{}
			cfg.URL = url
			err := config.Save(cfg)
			Expect(err).To(BeNil())
			currentConfig, err := config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.URL).To(Equal(url))

			env, err := GetEnv()
			Expect(err).To(BeNil())
			Expect(env).To(Equal("staging"))
		})

		It("Returns a valid local env", func() {
			url := "http://localhost:8000"
			cfg := &config.Config{}
			cfg.URL = url
			err := config.Save(cfg)
			Expect(err).To(BeNil())
			currentConfig, err := config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.URL).To(Equal(url))

			env, err := GetEnv()
			Expect(err).To(BeNil())
			Expect(env).To(Equal("local"))
		})

		It("Returns a valid fedRAMP env", func() {
			url := "https://api.int.openshiftusgov.com"
			cfg := &config.Config{}
			cfg.URL = url
			cfg.FedRAMP = true
			err := config.Save(cfg)
			Expect(err).To(BeNil())
			currentConfig, err := config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.URL).To(Equal(url))

			env, err := GetEnv()
			Expect(err).To(BeNil())
			Expect(env).To(Equal("integration"))
		})

		It("Returns a valid regionalized env", func() {
			url := "https://api.aws.ap-southeast-1.integration.openshift.com"
			cfg := &config.Config{}
			cfg.URL = url
			err := config.Save(cfg)
			Expect(err).To(BeNil())
			currentConfig, err := config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.URL).To(Equal(url))

			env, err := GetEnv()
			Expect(err).To(BeNil())
			Expect(env).To(Equal("integration"))

			url = "https://api.aws.ap-southeast-1.openshift.com"
			cfg = &config.Config{}
			cfg.URL = url
			err = config.Save(cfg)
			Expect(err).To(BeNil())
			currentConfig, err = config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.URL).To(Equal(url))

			env, err = GetEnv()
			Expect(err).To(BeNil())
			Expect(env).To(Equal("production"))
		})

		It("Fails for invalid URL", func() {
			url := "https://urlthatfails.com"
			cfg := &config.Config{}
			cfg.URL = url
			err := config.Save(cfg)
			Expect(err).To(BeNil())
			currentConfig, err := config.Load()
			Expect(err).To(BeNil())
			Expect(currentConfig.URL).To(Equal(url))

			env, err := GetEnv()
			Expect(err).NotTo(BeNil())
			Expect(env).To(BeEmpty())
		})
	})
})
