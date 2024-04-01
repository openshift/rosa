package login

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
			env = "staging"
			It("only 'region' is FedRAMP", func() {
				os.Setenv("AWS_REGION", "us-gov-west-1")
				// Load the configuration file:
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())
				if cfg == nil {
					cfg = new(config.Config)
				}
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
			It("env is empty", func() {
				os.Setenv("AWS_REGION", "us-gov-east-1")
				// Load the configuration file:
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())
				if cfg == nil {
					cfg = new(config.Config)
				}
				env = ""
				cfg.FedRAMP = true
				err = CheckAndLogIntoFedramp(false, false, cfg, "", rosa.NewRuntime())
				Expect(err).ToNot(HaveOccurred())
				Expect(env).To(Equal("production"))
			})
		})
	})
})
