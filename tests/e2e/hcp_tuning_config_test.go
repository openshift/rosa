package e2e

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
)

var _ = Describe("Tuning Config(s)", labels.Feature.TuningConfigs, func() {

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

	It("can be created/updated/deleted to hosted cluster - [id:63164]",
		labels.Critical, labels.Runtime.Day2, labels.FedRAMP,
		func() {
			tuningConfigService := rosaClient.TuningConfig
			tc1Name := helper.GenerateRandomName("tuned01", 2)
			firstPriority := 10
			firstVMDirtyRatio := 25
			tc2Name := helper.GenerateRandomName("tuned02", 2)
			secondPriority := 20
			secondVMDirtyRatio := 65

			tc1Spec := rosacli.NewTuningConfigSpecRootStub(tc1Name, firstVMDirtyRatio, firstPriority)
			tc2Spec := rosacli.NewTuningConfigSpecRootStub(tc2Name, firstVMDirtyRatio, firstPriority)

			By("Create tuning configs to the cluster")
			tc1JSON, err := json.Marshal(tc1Spec)
			Expect(err).ToNot(HaveOccurred())
			resp, err := tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				tc1Name,
				string(tc1JSON))
			Expect(err).ToNot(HaveOccurred())
			textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
			Expect(textData).
				To(ContainSubstring(
					fmt.Sprintf("Tuning config '%s' has been created on cluster '%s'", tc1Name, clusterID)))

			tc2YAML, err := yaml.Marshal(tc2Spec)
			Expect(err).ToNot(HaveOccurred())
			resp, err = tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				tc2Name,
				string(tc2YAML))
			Expect(err).ToNot(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
			Expect(textData).
				To(ContainSubstring(
					fmt.Sprintf("Tuning config '%s' has been created on cluster '%s'", tc2Name, clusterID)))

			By("List all tuning configs")
			tuningConfigList, err := tuningConfigService.ListTuningConfigsAndReflect(clusterID)
			Expect(err).ToNot(HaveOccurred())
			expectDescriptionTemplate := "the tuningconfig %s is not in output"
			Expect(tuningConfigList.IsPresent(tc1Name)).
				To(BeTrue(), expectDescriptionTemplate, tc1Name)
			Expect(tuningConfigList.IsPresent(tc2Name)).
				To(BeTrue(), expectDescriptionTemplate, tc2Name)

			By("Update a tuning config of the cluster")
			tc2Spec.Profile[0].Data = rosacli.NewTuningConfigSpecProfileData(secondVMDirtyRatio)
			tc2Spec.Recommend[0].Priority = secondPriority
			tc2JSON, err := json.Marshal(tc2Spec)
			Expect(err).ToNot(HaveOccurred())
			specPath, err := helper.CreateTempFileWithContent(string(tc2JSON))
			defer os.Remove(specPath)
			Expect(err).ToNot(HaveOccurred())
			_, err = tuningConfigService.EditTuningConfig(clusterID, tc2Name, "--spec-path", specPath)
			Expect(err).ToNot(HaveOccurred())

			By("Describe the updated tuning config")
			output, err := tuningConfigService.DescribeTuningConfigAndReflect(clusterID, tc2Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Name).To(Equal(tc2Name))
			var spec rosacli.TuningConfigSpecRoot
			err = json.Unmarshal([]byte(output.Spec), &spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(spec.Profile)).To(Equal(len(tc2Spec.Profile)))
			Expect(spec.Profile[0].Data).To(Equal(tc2Spec.Profile[0].Data))
			Expect(spec.Profile[0].Name).To(Equal(tc2Spec.Profile[0].Name))
			Expect(len(spec.Recommend)).To(Equal(len(tc2Spec.Recommend)))
			Expect(spec.Recommend[0].Priority).To(Equal(tc2Spec.Recommend[0].Priority))
			Expect(spec.Recommend[0].Profile).To(Equal(tc2Spec.Recommend[0].Profile))

			By("Delete the tuning config")
			_, err = tuningConfigService.DeleteTuningConfig(clusterID, tc2Name)
			Expect(err).ToNot(HaveOccurred())

			By("List the tuning configs and check deleted tuning config should not be present]")
			tuningConfigList, err = tuningConfigService.ListTuningConfigsAndReflect(clusterID)
			Expect(err).ToNot(HaveOccurred())
			Expect(tuningConfigList.IsPresent(tc1Name)).
				To(BeTrue(), "the tuningconfig %s is not in output", tc1Name)
			Expect(tuningConfigList.IsPresent(tc2Name)).
				To(BeFalse(), "the tuningconfig %s is in the output", tc2Name)
		})

	It("can validate creation - [id:63169]",
		labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
		func() {
			tuningConfigService := rosaClient.TuningConfig

			By("Create tuning config")
			tcName := helper.GenerateRandomName("c63169", 2)
			firstPriority := 10
			firstVMDirtyRatio := 25
			tcSpec := rosacli.NewTuningConfigSpecRootStub(tcName, firstVMDirtyRatio, firstPriority)
			tcSpecJSON, err := json.Marshal(tcSpec)
			Expect(err).ToNot(HaveOccurred())
			resp, err := tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				tcName,
				string(tcSpecJSON))
			Expect(err).ToNot(HaveOccurred())
			textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
			Expect(textData).
				To(ContainSubstring(
					fmt.Sprintf("Tuning config '%s' has been created on cluster '%s'", tcName, clusterID)))

			By("Create another tuning config with same name")
			_, err = tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				tcName,
				string(tcSpecJSON))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("A tuning config with name '%s'", tcName)))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("already exists for cluster '%s'", clusterID)))

			By("Create tuning config with no name")
			_, err = tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				"",
				string(tcSpecJSON))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected a valid name"))

			By("Create tuning config with no spec")
			_, err = tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				helper.GenerateRandomName("c63169", 2),
				"")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Attribute 'spec' must be set"))

			By("Create tuning config with invalid name")
			_, err = tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				"c63169%^",
				string(tcSpecJSON))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).
				To(
					ContainSubstring("Name 'c63169%^' is not valid. The name must be a lowercase RFC 1123 subdomain"))

			By("Create tuning config with invalid spec file")
			_, err = tuningConfigService.CreateTuningConfigFromSpecFile(
				clusterID,
				helper.GenerateRandomName("c63169", 2),
				"wrong_file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).
				To(
					ContainSubstring("Expected a valid TuneD spec file: open wrong_file: no such file or directory"))

			By("Create tuning config with invalid spec content")
			_, err = tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				helper.GenerateRandomName("c63169", 2),
				"Spec%^")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected a valid TuneD spec file: error unmarshaling"))

			By("Create more than 100 tuning configs")
			tcs, err := tuningConfigService.ListTuningConfigsAndReflect(clusterID)
			Expect(err).ToNot(HaveOccurred())
			maxTcNb := 100 - len(tcs.TuningConfigs)
			for i := 1; i <= maxTcNb; i++ {
				_, err = tuningConfigService.CreateTuningConfigFromSpecContent(
					clusterID,
					helper.GenerateRandomName(fmt.Sprintf("c63169%d", i), 2),
					string(tcSpecJSON))
				Expect(err).ToNot(HaveOccurred())
			}
			_, err = tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				helper.GenerateRandomName("c63169", 2),
				string(tcSpecJSON))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).
				To(
					ContainSubstring("Maximum number of TuningConfigs per cluster reached. " +
						"Maximum allowed is '100'. Current count is '100'."))
		})

	It("can validate update - [id:63174]",
		labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
		func() {
			tuningConfigService := rosaClient.TuningConfig

			By("Create tuning config")
			tcName := helper.GenerateRandomName("c63174", 2)
			firstPriority := 10
			firstVMDirtyRatio := 25
			tcSpec := rosacli.NewTuningConfigSpecRootStub(tcName, firstVMDirtyRatio, firstPriority)
			tcSpecJSON, err := json.Marshal(tcSpec)
			Expect(err).ToNot(HaveOccurred())
			resp, err := tuningConfigService.CreateTuningConfigFromSpecContent(
				clusterID,
				tcName,
				string(tcSpecJSON))
			Expect(err).ToNot(HaveOccurred())
			textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
			Expect(textData).
				To(ContainSubstring(
					fmt.Sprintf("Tuning config '%s' has been created on cluster '%s'", tcName, clusterID)))

			By("Update tuning config with incorrect name")
			_, err = tuningConfigService.EditTuningConfig(clusterID, "wrong_name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Tuning config 'wrong_name' does not exist"))

			By("Update tuning config with incorrect spec file")
			_, err = tuningConfigService.EditTuningConfig(clusterID, tcName, "--spec-path", "wrong_file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected a valid spec file: open wrong_file: no such file or directory"))

			By("Update tuning config with incorrect spec content")
			specPath, err := helper.CreateTempFileWithContent("Spec%^")
			defer os.Remove(specPath)
			Expect(err).ToNot(HaveOccurred())
			_, err = tuningConfigService.EditTuningConfig(clusterID, tcName, "--spec-path", specPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected a valid spec file: error unmarshaling"))
		})
})
