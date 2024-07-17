package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("hibernate and resume cluster testing", labels.Feature.Hibernation, func() {
	defer GinkgoRecover()
	var (
		clusterID      string
		rosaClient     *rosacli.Client
		clusterService rosacli.ClusterService
	)

	BeforeEach(func() {
		By("Init the client")
		rosaClient = rosacli.NewClient()
		clusterService = rosaClient.Cluster
		clusterID = config.GetClusterID()
	})

	AfterEach(func() {
		By("Clean remaining resources")
		rosaClient.CleanResources(clusterID)

	})

	It("to hibernate and resume then delete cluster via rosacli - [id:42832]",
		labels.Critical, labels.Runtime.Hibernate,
		func() {
			By("Skip testing if the cluster is a hosted-cp cluster")
			isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).To(BeNil())
			if isHostedCP {
				Skip("Skip this case as it only supports on not-hosted-cp clusters")
			}

			By("hibernate cluster")
			out, err := clusterService.HibernateCluster(clusterID, "-y")
			Expect(err).To(BeNil())
			Expect(out.String()).To(ContainSubstring("is hibernating"))
			rosaClient.Runner.UnsetArgs()

			err = clusterService.WaitClusterStatus(clusterID, "hibernating", 3, 30)
			Expect(err).To(BeNil(), "It met error or timeout when waiting cluster to hibernating status")

			By("resume cluster")
			out, err = clusterService.ResumeCluster(clusterID, "-y")
			Expect(err).To(BeNil())
			Expect(out.String()).To(ContainSubstring("is resuming"))
			rosaClient.Runner.UnsetArgs()

			err = clusterService.WaitClusterStatus(clusterID, "ready", 3, 30)
			Expect(err).To(BeNil(), "It met error or timeout when waiting cluster to ready status")

			By("hibernate the cluster again then delete the cluster in not hibernating status")
			out, err = clusterService.HibernateCluster(clusterID, "-y")
			Expect(err).To(BeNil())
			Expect(out.String()).To(ContainSubstring("is hibernating"))
			rosaClient.Runner.UnsetArgs()

			err = clusterService.WaitClusterStatus(clusterID, "hibernating", 3, 30)
			Expect(err).To(BeNil(), "It met error or timeout when waiting cluster to hibernating status")

			_, err = clusterService.DeleteCluster(clusterID, "-y")
			Expect(err).To(BeNil())

			rosaClient.Runner.UnsetArgs()
			err = clusterService.WaitClusterDeleted(clusterID, 3, 30)
			Expect(err).To(BeNil(), "It failed to delete the cluster or met timeout")
		})
})
