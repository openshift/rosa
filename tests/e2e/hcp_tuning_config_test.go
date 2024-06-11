package e2e

import (
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Create Tuning Config", labels.Feature.TuningConfigs, func() {

	var rosaClient *rosacli.Client
	var clusterService rosacli.ClusterService

	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()
		clusterService = rosaClient.Cluster

		By("Skip testing if the cluster is not a HCP cluster")
		hosted, err := clusterService.IsHostedCPCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())
		if !hosted {
			SkipNotHosted()
		}
	})

	AfterEach(func() {
		By("Clean remaining resources")
		err := rosaClient.CleanResources(clusterID)
		Expect(err).ToNot(HaveOccurred())
	})

	It("tuning config can be created/updated/deleted to hosted cluster - [id:63164]",
		labels.Critical, labels.Runtime.Day2,
		func() {
			tuningConfigService := rosaClient.TuningConfig
			tuningConfigName_1 := common.GenerateRandomName("tuned01", 2)
			tuningConfigName_2 := common.GenerateRandomName("tuned02", 2)

			tuningConfigPayload_1 := `{
			"profile": [
				{
				"data": "[main]\nsummary=Custom OpenShift profile\ninclude=openshift-node\n\n[sysctl]\nvm.dirty_ratio=\"25\"\n",
				"name": "%s-profile"
				}
			],
			"recommend": [
				{
				"priority": 10,
				"profile": "%s-profile"
				}
			]
		}`
			tuningConfigPayload_2 := `{
			"profile": [
				{
					"data": "[main]\nsummary=Custom OpenShift profile\ninclude=openshift-node\n\n[sysctl]\nvm.dirty_ratio=\"65\"\n",
					"name": "%s-profile"
				}
			],
			"recommend": [
				{
					"priority": 20,
					"profile": "%s-profile"
				}
			]
		}`
			By("Create tuning configs to the cluster")
			resp, err := tuningConfigService.CreateTuningConfig(
				clusterID,
				tuningConfigName_1,
				fmt.Sprintf(tuningConfigPayload_1, tuningConfigName_1, tuningConfigName_1))
			Expect(err).ToNot(HaveOccurred())
			textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
			Expect(textData).
				To(ContainSubstring(
					fmt.Sprintf("Tuning config '%s' has been created on cluster '%s'", tuningConfigName_1, clusterID)))
			resp, err = tuningConfigService.CreateTuningConfig(
				clusterID,
				tuningConfigName_2,
				fmt.Sprintf(tuningConfigPayload_1, tuningConfigName_2, tuningConfigName_2))
			Expect(err).ToNot(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
			Expect(textData).
				To(ContainSubstring(
					fmt.Sprintf("Tuning config '%s' has been created on cluster '%s'", tuningConfigName_2, clusterID)))

			By("List all tuning configs")
			tuningConfigList, err := tuningConfigService.ListTuningConfigsAndReflect(clusterID)
			Expect(err).ToNot(HaveOccurred())
			Expect(tuningConfigList.IsPresent(tuningConfigName_1)).
				To(BeTrue(), "the tuningconfig %s is not in output", tuningConfigName_1)
			Expect(tuningConfigList.IsPresent(tuningConfigName_2)).
				To(BeTrue(), "the tuningconfig %s is not in output", tuningConfigName_2)

			By("Update a tuning config of the cluster")
			specPath, err := common.CreateTempFileWithContent(
				fmt.Sprintf(tuningConfigPayload_2, tuningConfigName_2, tuningConfigName_2))
			defer os.Remove(specPath)
			Expect(err).ToNot(HaveOccurred())
			_, err = tuningConfigService.EditTuningConfig(clusterID, tuningConfigName_2, "--spec-path", specPath)
			Expect(err).ToNot(HaveOccurred())

			By("Describe the updated tuning config")
			output, err := tuningConfigService.DescribeTuningConfigAndReflect(clusterID, tuningConfigName_2)
			Expect(err).ToNot(HaveOccurred())
			tuningConfigPayload_2 = strings.ReplaceAll(tuningConfigPayload_2, "\t", "  ")
			Expect(output.Spec).To(Equal(fmt.Sprintf(tuningConfigPayload_2, tuningConfigName_2, tuningConfigName_2)))
			Expect(output.Name).To(Equal(tuningConfigName_2))

			By("Delete the tuning config")
			_, err = tuningConfigService.DeleteTuningConfig(clusterID, tuningConfigName_2)
			Expect(err).ToNot(HaveOccurred())

			By("List the tuning configs and check deleted tuning config should not be present]")
			tuningConfigList, err = tuningConfigService.ListTuningConfigsAndReflect(clusterID)
			Expect(err).ToNot(HaveOccurred())
			Expect(tuningConfigList.IsPresent(tuningConfigName_1)).
				To(BeTrue(), "the tuningconfig %s is not in output", tuningConfigName_1)
			Expect(tuningConfigList.IsPresent(tuningConfigName_2)).
				To(BeFalse(), "the tuningconfig %s is in the output", tuningConfigName_2)
		})
})
