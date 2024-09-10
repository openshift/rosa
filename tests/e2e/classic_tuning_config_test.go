package e2e

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Tuning Config(s) on Classic cluster", labels.Feature.TuningConfigs, func() {

	var rosaClient *rosacli.Client
	var clusterService rosacli.ClusterService

	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()
		clusterService = rosaClient.Cluster

		By("Skip testing if the cluster is not a Classic cluster")
		hosted, err := clusterService.IsHostedCPCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())
		if hosted {
			SkipNotClassic()
		}
	})

	AfterEach(func() {
		By("Clean remaining resources")
		err := rosaClient.CleanResources(clusterID)
		Expect(err).ToNot(HaveOccurred())
	})

	It("is not supported - [id:76056]",
		labels.Medium, labels.Runtime.Day2,
		func() {
			tuningConfigService := rosaClient.TuningConfig
			tcName := common.GenerateRandomName("tuned01", 2)
			tcSpec := rosacli.NewTuningConfigSpecRootStub(tcName, 25, 10)

			By("Create tuning config should fail")
			tcJSON, err := json.Marshal(tcSpec)
			Expect(err).ToNot(HaveOccurred())
			_, err = tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				tcName,
				string(tcJSON))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("This command is only supported for Hosted Control Planes"))
		})
})
