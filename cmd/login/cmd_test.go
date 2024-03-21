package login_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/cmd/login"
	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("Validate login command", func() {

	AfterEach(func() {
		fedramp.Disable()
		os.Setenv("AWS_REGION", "")
	})

	Context("login command", func() {
		When("logging into FedRAMP", func() {
			It("only 'region' is FedRAMP", func() {
				os.Setenv("AWS_REGION", "us-gov-west-1")
				// Load the configuration file:
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())
				if cfg == nil {
					cfg = new(config.Config)
				}
				err = login.CheckAndLogIntoFedramp(false, false, cfg, "", "staging", rosa.NewRuntime())
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
				err = login.CheckAndLogIntoFedramp(true, false, cfg, "", "staging", rosa.NewRuntime())
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
				err = login.CheckAndLogIntoFedramp(false, false, cfg, "", "staging", rosa.NewRuntime())
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
				err = login.CheckAndLogIntoFedramp(false, false, cfg, "", "staging", rosa.NewRuntime())
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
