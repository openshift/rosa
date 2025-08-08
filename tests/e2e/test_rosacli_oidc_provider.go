package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("OIDC provider",
	labels.Feature.OIDCProvider,
	func() {
		defer GinkgoRecover()

		var (
			clusterID          string
			rosaClient         *rosacli.Client
			clusterService     rosacli.ClusterService
			ocmResourceService rosacli.OCMResourceService
		)

		BeforeEach(func() {
			By("Get the cluster id")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource
		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validate when user create oidc-provider to cluster - [id:43046]",
			labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
			func() {
				By("Check if cluster is sts cluster")
				StsCluster, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).To(BeNil())

				By("Check if cluster is using reusable oidc config")
				Expect(err).To(BeNil())

				notExistedClusterID := "notexistedclusterid111"

				switch StsCluster {
				case true:
					By("Create oidc-provider on sts cluster which status is not pending")
					output, err := ocmResourceService.CreateOIDCProvider(
						"--mode", "auto",
						"-c", clusterID,
						"-y")
					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("OIDC provider already exists"))
				case false:
					By("Create oidc-provider on classic non-sts cluster")
					output, err := ocmResourceService.CreateOIDCProvider(
						"--mode", "auto",
						"-c", clusterID,
						"-y")
					Expect(err).NotTo(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("is not an STS cluster"))
				}
				By("Create oidc-provider on not-existed cluster")
				output, err := ocmResourceService.CreateOIDCProvider(
					"--mode", "auto",
					"-c", notExistedClusterID,
					"-y")
				Expect(err).NotTo(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("There is no cluster with identifier or name"))
			})
	})
