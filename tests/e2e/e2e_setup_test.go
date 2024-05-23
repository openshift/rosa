package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
	"github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("ROSA CLI Test", func() {
	It("PrepareClusterByProfile",
		labels.Day1Prepare,
		func() {
			client := rosacli.NewClient()
			profile := profilehandler.LoadProfileYamlFileByENV()
			cluster, err := profilehandler.CreateClusterByProfile(profile, client, config.Test.GlobalENV.WaitSetupClusterReady)
			Expect(err).ToNot(HaveOccurred())
			log.Logger.Infof("Cluster prepared successfully with id %s", cluster.ID)
		})

	It("WaitClusterReady", func() {
		clusterDetail, err := profilehandler.ParserClusterDetail()
		Expect(err).ToNot(HaveOccurred())
		client := rosacli.NewClient()
		profilehandler.WaitForClusterReady(client, clusterDetail.ClusterID, config.Test.GlobalENV.ClusterWaitingTime)
	})
})
