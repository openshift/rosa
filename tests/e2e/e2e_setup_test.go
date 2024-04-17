package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
	PH "github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("ROSA CLI Test", func() {
	It("PrepareClusterByProfile",
		labels.Critical,
		labels.Day1Prepare,
		func() {
			client := rosacli.NewClient()
			profile := PH.LoadProfileYamlFileByENV()
			cluster, err := PH.CreateClusterByProfile(profile, client, true)
			Expect(err).ToNot(HaveOccurred())
			log.Logger.Infof("Cluster prepared successfully with id %s", cluster.ID)
		})
})
