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

var _ = Describe("Cluster preparation", labels.Feature.Cluster, func() {
	It("by profile",
		labels.Runtime.Day1,
		func() {
			client := rosacli.NewClient()
			profile := profilehandler.LoadProfileYamlFileByENV()
			cluster, err := profilehandler.CreateClusterByProfile(profile, client, config.Test.GlobalENV.WaitSetupClusterReady)
			Expect(err).ToNot(HaveOccurred())
			log.Logger.Infof("Cluster prepared successfully with id %s", cluster.ID)
		})

	It("to wait for cluster ready",
		labels.Runtime.Day1Readiness,
		func() {
			clusterDetail, err := profilehandler.ParserClusterDetail()
			Expect(err).ToNot(HaveOccurred())
			client := rosacli.NewClient()
			profilehandler.WaitForClusterReady(client, clusterDetail.ClusterID, config.Test.GlobalENV.ClusterWaitingTime)
		})
})
