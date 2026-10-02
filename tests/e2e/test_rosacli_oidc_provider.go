package e2e

import (
	"fmt"
	"os"

	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
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
			labels.Medium, labels.Runtime.Day2, labels.FedRAMP, labels.Hyperfleet.Validated,
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
				fmt.Println(textData)

				Expect(textData).To(SatisfyAny(
					ContainSubstring("There is no cluster with identifier or name"),
					ContainSubstring("cluster '"+notExistedClusterID+"' not found"),
				))
			})
	})

var _ = Describe("OIDC provider by OIDC config ID",
	labels.Feature.OIDCProvider, labels.Hyperfleet.Validated,
	func() {
		var (
			rosaClient         *rosacli.Client
			ocmResourceService rosacli.OCMResourceService
		)

		BeforeEach(func() {
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
		})

		It("can resolve an existing provider using the oidc config id",
			labels.Medium, labels.Runtime.Day2,
			func() {
				var oidcConfigID string
				DeferCleanup(func() {
					if oidcConfigID == "" {
						return
					}

					By("Delete the OIDC config created for this test")
					_, err := ocmResourceService.DeleteOIDCConfig(
						"--oidc-config-id", oidcConfigID,
						"--mode", "auto",
						"-y",
					)
					Expect(err).NotTo(HaveOccurred())
				})

				isHyperfleet := os.Getenv("HYPERFLEET_URL") != "" || hyperfleet.Enabled()
				if isHyperfleet {
					By("Create a managed OIDC config through Platform API")
					output, err := ocmResourceService.CreateOIDCConfig(
						"--mode", "auto",
						"-o", "json",
						"-y",
					)
					Expect(err).NotTo(HaveOccurred())

					createdConfig := rosaClient.Parser.JsonData.Input(output).Parse()
					oidcConfigID = createdConfig.DigString("id")
					if oidcConfigID == "" {
						oidcConfigID = createdConfig.DigString("metadata", "name")
					}
				} else {
					By("Create a managed OIDC config through OCM")
					output, err := ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
					Expect(err).NotTo(HaveOccurred())

					providerARN := helper.ExtractOIDCProviderARN(output.String())
					providerID := helper.ExtractOIDCProviderIDFromARN(providerARN)
					Expect(providerID).NotTo(BeEmpty(), "create response should include the OIDC provider ARN")

					oidcConfigID, err = ocmResourceService.GetOIDCIdFromList(providerID)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(oidcConfigID).NotTo(BeEmpty(), "create response should include the OIDC config ID")

				By("Verify the provider using the OIDC config ID")
				providerOutput, providerErr := ocmResourceService.CreateOIDCProvider(
					"--oidc-config-id", oidcConfigID,
					"--mode", "auto",
					"-y",
				)
				Expect(providerErr).NotTo(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(providerOutput).Parse().Tip()
				Expect(textData).To(ContainSubstring("OIDC provider already exists"))
			})
	})
