/*
Copyright (c) 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package login

import (
	"net/http"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

func TestCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Login Suite")
}

var _ = Describe("Validate login command", func() {

	AfterEach(func() {
		fedramp.Disable()
		os.Setenv("AWS_REGION", "")
	})

	Context("login command", func() {
		When("logging into FedRAMP", func() {
			env = "staging"
			It("only 'region' is FedRAMP", func() {
				os.Setenv("AWS_REGION", "us-gov-west-1")
				// Load the configuration file:
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())
				if cfg == nil {
					cfg = new(config.Config)
				}
				env = "staging"
				err = CheckAndLogIntoFedramp(false, false, cfg, "", rosa.NewRuntime())
				Expect(err).ToNot(HaveOccurred())
			})
			It("only 'govcloud' flag is true", func() {
				os.Setenv("AWS_REGION", "us-east-1")
				// Load the configuration file:
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())
				if cfg == nil {
					cfg = new(config.Config)
				}
				err = CheckAndLogIntoFedramp(true, false, cfg, "", rosa.NewRuntime())
				Expect(err).To(HaveOccurred())
			})
			It("only 'cfg' has FedRAMP", func() {
				os.Setenv("AWS_REGION", "us-east-1")
				// Load the configuration file:
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())
				if cfg == nil {
					cfg = new(config.Config)
				}
				cfg.FedRAMP = true
				err = CheckAndLogIntoFedramp(false, false, cfg, "", rosa.NewRuntime())
				Expect(err).To(HaveOccurred())
			})
			It("'cfg' has FedRAMP and region is govcloud", func() {
				os.Setenv("AWS_REGION", "us-gov-east-1")
				// Load the configuration file:
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())
				if cfg == nil {
					cfg = new(config.Config)
				}
				cfg.FedRAMP = true
				err = CheckAndLogIntoFedramp(false, false, cfg, "", rosa.NewRuntime())
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

var _ = Describe("Login Configuration", Ordered, func() {
	var testRuntime test.TestingRuntime
	var tmpdir string
	invalidKeyring := "not-a-keyring"

	BeforeAll(func() {
		tmpdir, _ = os.MkdirTemp("/tmp", ".ocm-config-*")
		os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")
	})

	AfterAll(func() {
		os.Setenv("OCM_CONFIG", "")
	})

	BeforeEach(func() {
		testRuntime.InitRuntime()
	})

	When("Using offline token", func() {
		It("Creates the configuration file", func() {
			// Create the token:
			claims := MakeClaims()
			claims["username"] = "test"
			accessTokenObj := MakeTokenObject(claims)

			// Run the command:
			args.token = accessTokenObj.Raw
			args.tokenURL = testRuntime.SsoServer.URL()
			_, _, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime, Cmd, &[]string{})
			Expect(err).To(BeNil())

			cfg, _ := config.Load()
			Expect(cfg.AccessToken).To(Equal(accessTokenObj.Raw))
			Expect(cfg.TokenURL).To(Equal(testRuntime.SsoServer.URL()))
		})
	})

	When("Using client credentials grant", func() {
		It("Creates the configuration file", func() {
			// Run the command:
			args.clientID = "my-client"
			args.clientSecret = "my-secret"
			args.tokenURL = testRuntime.SsoServer.URL()
			args.env = testRuntime.ApiServer.URL()

			// Create the token:
			claims := MakeClaims()
			claims["username"] = "test"
			accessTokenObj := MakeTokenObject(claims)

			testRuntime.SsoServer.AppendHandlers(
				RespondWithAccessToken(accessTokenObj.Raw),
			)

			testRuntime.SsoServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))

			_, _, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime, Cmd, &[]string{})
			Expect(err).To(BeNil())

			cfg, _ := config.Load()
			Expect(cfg.AccessToken).ToNot(BeEmpty())
			Expect(cfg.TokenURL).To(Equal(testRuntime.SsoServer.URL()))
			Expect(cfg.ClientID).To(Equal("my-client"))
			Expect(cfg.ClientSecret).To(Equal("my-secret"))
		})
	})

	When(properties.KeyringEnvKey+" is used", func() {
		AfterEach(func() {
			// reset keyring
			os.Setenv(properties.KeyringEnvKey, "")
		})

		It("Fails for an invalid keyring", func() {
			os.Setenv(properties.KeyringEnvKey, invalidKeyring)

			// Run the command:
			stdout, _, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime, Cmd, &[]string{})

			Expect(err).ToNot(BeNil())
			Expect(stdout).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("keyring is invalid"))
		})
	})
})
