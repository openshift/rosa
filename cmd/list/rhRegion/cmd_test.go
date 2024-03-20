package rhRegion

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/ocm"
)

func TestRhRegionCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa list rh-regions command")
}

var _ = Describe("Token", Ordered, func() {
	var tmpdir string
	var err error
	var cfg *config.Config

	BeforeAll(func() {
		tmpdir, err = os.MkdirTemp("/tmp", ".ocm-config-*")
		Expect(err).To(BeNil())
		os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")
		cfg = &config.Config{}
		cfg.URL = ocm.URLAliases["staging"]
		err = config.Save(cfg)
		Expect(err).To(BeNil())
	})

	AfterAll(func() {
		os.Setenv("OCM_CONFIG", "")
	})

	When("Logged in", func() {
		It("Displays rh-regions", func() {
			err = ListRhRegions("", nil)
			Expect(err).To(BeNil())
		})
	})
})
