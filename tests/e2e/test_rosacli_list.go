package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("List resources",
	labels.Day2,
	labels.FeatureMachinepool,
	labels.NonHCPCluster,
	func() {
		defer GinkgoRecover()
		var (
			clusterID  string
			rosaClient *rosacli.Client
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
		})

		AfterEach(func() {
			By("Clean remaining resources")
			rosaClient.CleanResources(clusterID)

		})

		It("List cluster via ROSA cli will work well - [id:38816]",
			labels.Medium,
			func() {
				var (
					clusterData    []string
					ClusterService = rosaClient.Cluster
				)

				By("List all clusters")
				clusterList, _, err := ClusterService.ListCluster()
				Expect(err).To(BeNil())
				if err != nil {
					log.Logger.Errorf("Failed to fetch clusters: %v", err)
					return
				}

				for _, it := range clusterList.ListCluster {
					clusterData = append(clusterData, it.ID)
				}
				Expect(err).To(BeNil())
				Expect(clusterData).Should(ContainElement(clusterID))

				clusterData = []string{}
				By("List clusters with '--all' flag")
				clusterList, _, err = ClusterService.ListCluster("--all")
				Expect(err).To(BeNil())
				if err != nil {
					log.Logger.Errorf("Failed to fetch clusters: %v", err)
					return
				}
				for _, it := range clusterList.ListCluster {
					clusterData = append(clusterData, it.ID)
				}
				Expect(err).To(BeNil())
				Expect(clusterData).Should(ContainElement(clusterID))
			})
	})
