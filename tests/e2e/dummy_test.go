package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	TC "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/log"
	PH "github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("ROSA CLI Test", func() {
	Describe("Dummy test", func() {
		It("Dummy", func() {
			str := "dummy string"
			Expect(str).ToNot(BeEmpty())
			log.Logger.Infof("This is a dummy test to check everything is fine by executing jobs. Please remove me once other tests are added")
		})
	})
	Describe("Profile test", func() {
		It("ProfileParserTest", func() {
			profile := PH.LoadProfileYamlFileByENV()
			log.Logger.Infof("Got configured profile: %v", *profile)
			log.Logger.Infof("Got configured cluster profile: %v", *profile.ClusterConfig)
			log.Logger.Infof("Got configured account role profile: %v", *profile.AccountRoleConfig)
		})
		It("TestENVSetup", func() {
			log.Logger.Infof("Got dir of out: %v", TC.Test.OutputDir)
		})
	})
})
