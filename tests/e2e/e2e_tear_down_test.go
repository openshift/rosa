package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("Cluster destroy", labels.Feature.Cluster, func() {
	It("by profile",
		labels.Runtime.Destroy,
		labels.Critical,
		func() {
			client := rosacli.NewClient()
			profile := profilehandler.LoadProfileYamlFileByENV()
			var errs = profilehandler.DestroyResourceByProfile(profile, client)
			Expect(len(errs)).To(Equal(0))
		})
})
