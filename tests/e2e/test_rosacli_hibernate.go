package e2e

import (
	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
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
			isLimitedSupport, err := clusterService.IsLimitedSupport(clusterID)
			Expect(err).To(BeNil())
			if isLimitedSupport {
				Skip("You can't hibernate a cluster in limited support")
			}

			By("resuming a cluster that's already running should fail")
			_, err = clusterService.ResumeCluster(clusterID, "-y")
			rosaClient.Runner.UnsetArgs()
			helper.ExpectErrorWithMessage(
				err,
				"Resuming a cluster from hibernation is only supported for clusters in 'Hibernating' state",
			)

			By("hibernate cluster")
			out, err := clusterService.HibernateCluster(clusterID, "-y")
			rosaClient.Runner.UnsetArgs()
			Expect(err).To(BeNil())
			Expect(out.String()).To(ContainSubstring("is hibernating"))

			err = clusterService.WaitClusterStatus(clusterID, "hibernating", 3, 30)
			Expect(err).To(BeNil(), "It met error or timeout when waiting cluster to hibernating status")

			By("hibernating a cluster that's already hibernating should fail")
			_, err = clusterService.HibernateCluster(clusterID, "-y")
			rosaClient.Runner.UnsetArgs()
			helper.ExpectErrorWithMessage(
				err,
				"Hibernating a cluster is only supported for 'Ready' clusters",
			)

			By("resume cluster")
			out, err = clusterService.ResumeCluster(clusterID, "-y")
			rosaClient.Runner.UnsetArgs()
			Expect(err).To(BeNil())
			Expect(out.String()).To(ContainSubstring("is resuming"))

			err = clusterService.WaitClusterStatus(clusterID, "ready", 3, 30)
			Expect(err).To(BeNil(), "It met error or timeout when waiting cluster to ready status")
		})
})
