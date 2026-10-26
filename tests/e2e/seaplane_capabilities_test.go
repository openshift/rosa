package e2e

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/openshift/rosa/pkg/seaplane"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
)

var _ = Describe("Seaplane HCP cluster creation with disabled cluster capabilities",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			clusterHandler handler.ClusterHandler
		)

		BeforeEach(func() {
			if !isSeaplaneMode() {
				Skip("This case is only for Seaplane")
			}
			profile := handler.LoadProfileYamlFileByENV()
			if profile.Name != "rosa-seaplane-disable-capabilities" {
				Skip("This case requires TEST_PROFILE=rosa-seaplane-disable-capabilities")
			}

			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			var err error
			profile.Region, err = seaplane.ExtractRegion(seaplaneEndpointForTests())
			Expect(err).NotTo(HaveOccurred())
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, profile)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if clusterHandler != nil {
				errs := clusterHandler.Destroy()
				Expect(errs).To(BeEmpty())
			}
		})

		It("creates an HCP cluster with the ImageRegistry capability disabled and reports it in describe output id:ROSAENG-68175",
			labels.High,
			labels.Runtime.Day1,
			labels.Seaplane.Validated,
			labels.Seaplane.One,
			func() {
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				Expect(err).NotTo(HaveOccurred())
				Expect(flags).To(ContainElement("--disable-cluster-capabilities"))
				Expect(flags).To(ContainElement("ImageRegistry"))

				By("Create the HCP cluster with the ImageRegistry capability disabled")
				Expect(clusterHandler.CreateCluster(true)).To(Succeed())
				clusterID := clusterHandler.GetClusterDetail().ClusterID
				Expect(clusterID).NotTo(BeEmpty())

				By("Verify describe cluster reports the disabled capability in JSON output")
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).NotTo(HaveOccurred())

				capabilities, ok := jsonData.DigObject("capabilities").(map[string]interface{})
				Expect(ok).To(BeTrue(),
					"describe JSON must report capabilities at the top level, alongside id")
				disabled, ok := capabilities["disabled"].([]interface{})
				Expect(ok).To(BeTrue(), "describe JSON must report capabilities.disabled")
				Expect(disabled).To(ContainElement("ImageRegistry"))
				Expect(capabilities).NotTo(HaveKey("enabled"),
					"no capabilities were enabled, so capabilities.enabled must be absent")

				By("Verify describe cluster keeps capabilities outside of spec")
				spec, ok := jsonData.DigObject("spec").(map[string]interface{})
				Expect(ok).To(BeTrue(), "describe JSON must report spec")
				Expect(spec).NotTo(HaveKey("capabilities"),
					"capabilities must be reported at the same level as id, not nested under spec")
			})
	})
