package e2e

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Autoscaler", labels.Feature.Autoscaler, func() {

	var rosaClient *rosacli.Client
	var clusterService rosacli.ClusterService
	var clusterConfig *config.ClusterConfig

	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()
		clusterService = rosaClient.Cluster
	})

	Describe("creation testing", func() {
		BeforeEach(func() {
			hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())

			clusterConfig, err = config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())

			By("Skip testing if the cluster is not a Classic cluster")
			if hostedCluster {
				SkipNotClassic()
			}

		})

		It("create/describe/edit/delete cluster autoscaler by rosacli - [id:67275]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				By("Retrieve help for create/list/describe/delete autoscaler")
				_, err := rosaClient.AutoScaler.CreateAutoScaler(clusterID, "-h")
				Expect(err).ToNot(HaveOccurred())
				_, err = rosaClient.AutoScaler.EditAutoScaler(clusterID, "-h")
				Expect(err).ToNot(HaveOccurred())
				_, err = rosaClient.AutoScaler.RetrieveHelpForDescribe()
				Expect(err).ToNot(HaveOccurred())
				_, err = rosaClient.AutoScaler.RetrieveHelpForDelete()
				Expect(err).ToNot(HaveOccurred())

				By("Check if cluster is autoscaler enabled")
				if clusterConfig.Autoscaler != nil {

					By("Describe the autoscaler to the cluster")
					rosaClient.Runner.YamlFormat()
					yamlOutput, err := rosaClient.AutoScaler.DescribeAutoScaler(clusterID)
					Expect(err).ToNot(HaveOccurred())
					rosaClient.Runner.UnsetFormat()
					yamlData, err := rosaClient.Parser.TextData.Input(yamlOutput).Parse().YamlToMap()
					Expect(err).ToNot(HaveOccurred())
					logVerbosity := yamlData["log_verbosity"].(int)
					balancingIgnoredLabels := yamlData["balancing_ignored_labels"].([]interface{})[0]
					maxNodeProvisionTime := yamlData["max_node_provision_time"].(string)
					maxPodGracePeriod := yamlData["max_pod_grace_period"].(int)
					podPriorityThresold := yamlData["pod_priority_threshold"].(int)
					coreMin := yamlData["resource_limits"].(map[string]interface{})["cores"].(map[string]interface{})["min"].(int)
					coreMax := yamlData["resource_limits"].(map[string]interface{})["cores"].(map[string]interface{})["max"].(int)
					memoryMin := yamlData["resource_limits"].(map[string]interface{})["memory"].(map[string]interface{})["min"].(int)
					memoryMax := yamlData["resource_limits"].(map[string]interface{})["memory"].(map[string]interface{})["max"].(int)
					maxNodesTotal := yamlData["resource_limits"].(map[string]interface{})["max_nodes_total"].(int)
					sdDelayAfterAdd := yamlData["scale_down"].(map[string]interface{})["delay_after_add"].(string)
					sdDelayAfterDelete := yamlData["scale_down"].(map[string]interface{})["delay_after_delete"].(string)
					sdDelayAfterFailure := yamlData["scale_down"].(map[string]interface{})["delay_after_failure"].(string)
					sdUtilizationThreshold := yamlData["scale_down"].(map[string]interface{})["utilization_threshold"].(string)

					defer func() {
						By("Create the autoscaler to the cluster")
						resp, err := rosaClient.AutoScaler.CreateAutoScaler(clusterID, "--balance-similar-node-groups",
							"--skip-nodes-with-local-storage",
							"--log-verbosity", strconv.Itoa(logVerbosity),
							"--max-pod-grace-period", strconv.Itoa(maxPodGracePeriod),
							"--pod-priority-threshold", strconv.Itoa(podPriorityThresold),
							"--ignore-daemonsets-utilization",
							"--max-node-provision-time", maxNodeProvisionTime,
							"--balancing-ignored-labels", balancingIgnoredLabels.(string),
							"--max-nodes-total", strconv.Itoa(maxNodesTotal),
							"--min-cores", strconv.Itoa(coreMin),
							"--scale-down-delay-after-add", sdDelayAfterAdd,
							"--max-cores", strconv.Itoa(coreMax),
							"--min-memory", strconv.Itoa(memoryMin),
							"--max-memory", strconv.Itoa(memoryMax),
							"--scale-down-enabled",
							"--scale-down-utilization-threshold", sdUtilizationThreshold,
							"--scale-down-delay-after-delete", sdDelayAfterDelete,
							"--scale-down-delay-after-failure", sdDelayAfterFailure)
						Expect(err).ToNot(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("INFO: Successfully created autoscaler configuration for cluster '%s'", clusterID))
					}()
				} else {

					By("Create the autoscaler to the cluster")
					resp, err := rosaClient.AutoScaler.CreateAutoScaler(clusterID, "--balance-similar-node-groups",
						"--skip-nodes-with-local-storage",
						"--log-verbosity", "4",
						"--max-pod-grace-period", "0",
						"--pod-priority-threshold", "1",
						"--ignore-daemonsets-utilization",
						"--max-node-provision-time", "10m",
						"--balancing-ignored-labels", "aaa",
						"--max-nodes-total", "1000",
						"--min-cores", "0",
						"--scale-down-delay-after-add", "10s",
						"--max-cores", "100",
						"--min-memory", "0",
						"--max-memory", "4096",
						"--scale-down-enabled",
						"--scale-down-utilization-threshold", "1",
						"--scale-down-delay-after-delete", "10s",
						"--scale-down-delay-after-failure", "10s",
						"--gpu-limit", "nvidia.com/gpu,0,10",
						"--gpu-limit", "amd.com/gpu,1,5",
						"--scale-down-unneeded-time", "10s")
					Expect(err).ToNot(HaveOccurred())
					textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
					Expect(textData).To(ContainSubstring("INFO: Successfully created autoscaler configuration for cluster '%s'", clusterID))

					defer func() {
						By("Delete the autoscaler of the cluster")
						resp, err = rosaClient.AutoScaler.DeleteAutoScaler(clusterID)
						Expect(err).ToNot(HaveOccurred())
					}()

					By("Describe the autoscaler of the cluster")
					rosaClient.Runner.YamlFormat()
					yamlOutput, err := rosaClient.AutoScaler.DescribeAutoScaler(clusterID)
					Expect(err).ToNot(HaveOccurred())
					rosaClient.Runner.UnsetFormat()
					yamlData, err := rosaClient.Parser.TextData.Input(yamlOutput).Parse().YamlToMap()
					Expect(err).ToNot(HaveOccurred())
					Expect(yamlData["balance_similar_node_groups"]).To(Equal(true))
					Expect(yamlData["skip_nodes_with_local_storage"]).To(Equal(true))
					Expect(yamlData["log_verbosity"]).To(Equal(4))
					Expect(yamlData["balancing_ignored_labels"]).To(ContainElement("aaa"))
					Expect(yamlData["ignore_daemonsets_utilization"]).To(Equal(true))
					Expect(yamlData["max_node_provision_time"]).To(Equal("10m"))
					Expect(yamlData["max_pod_grace_period"]).To(Equal(0))
					Expect(yamlData["pod_priority_threshold"]).To(Equal(1))
					Expect(yamlData["resource_limits"].(map[string]interface{})["cores"].(map[string]interface{})["min"]).To(Equal(0))
					Expect(yamlData["resource_limits"].(map[string]interface{})["cores"].(map[string]interface{})["max"]).To(Equal(100))
					Expect(yamlData["resource_limits"].(map[string]interface{})["memory"].(map[string]interface{})["min"]).To(Equal(0))
					Expect(yamlData["resource_limits"].(map[string]interface{})["memory"].(map[string]interface{})["max"]).To(Equal(4096))
					Expect(yamlData["resource_limits"].(map[string]interface{})["gpus"].([]interface{})[0].(map[string]interface{})["range"].(map[string]interface{})["max"]).To(Equal(10))
					Expect(yamlData["resource_limits"].(map[string]interface{})["gpus"].([]interface{})[0].(map[string]interface{})["range"].(map[string]interface{})["min"]).To(Equal(0))
					Expect(yamlData["resource_limits"].(map[string]interface{})["gpus"].([]interface{})[0].(map[string]interface{})["type"]).To(Equal("nvidia.com/gpu"))
					Expect(yamlData["resource_limits"].(map[string]interface{})["gpus"].([]interface{})[1].(map[string]interface{})["range"].(map[string]interface{})["max"]).To(Equal(5))
					Expect(yamlData["resource_limits"].(map[string]interface{})["gpus"].([]interface{})[1].(map[string]interface{})["range"].(map[string]interface{})["min"]).To(Equal(1))
					Expect(yamlData["resource_limits"].(map[string]interface{})["gpus"].([]interface{})[1].(map[string]interface{})["type"]).To(Equal("amd.com/gpu"))
					Expect(yamlData["resource_limits"].(map[string]interface{})["max_nodes_total"]).To(Equal(1000))
					Expect(yamlData["scale_down"].(map[string]interface{})["delay_after_add"]).To(Equal("10s"))
					Expect(yamlData["scale_down"].(map[string]interface{})["delay_after_delete"]).To(Equal("10s"))
					Expect(yamlData["scale_down"].(map[string]interface{})["delay_after_failure"]).To(Equal("10s"))
					Expect(yamlData["scale_down"].(map[string]interface{})["enabled"]).To(Equal(true))
					Expect(yamlData["scale_down"].(map[string]interface{})["unneeded_time"]).To(Equal("10s"))
					Expect(yamlData["scale_down"].(map[string]interface{})["utilization_threshold"]).To(Equal("1.000000"))

				}

				By("Edit the autoscaler of the cluster")
				resp, err := rosaClient.AutoScaler.EditAutoScaler(clusterID, "--ignore-daemonsets-utilization",
					"--min-cores", "0",
					"--max-cores", "10",
					"--scale-down-delay-after-add", "0s",
					"--gpu-limit", "amd.com/gpu,1,5")
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).To(ContainSubstring("INFO: Successfully updated autoscaler configuration for cluster '%s'", clusterID))

				By("Describe autoscaler to check the edited value is correct")
				rosaClient.Runner.YamlFormat()
				yamlOutput, err := rosaClient.AutoScaler.DescribeAutoScaler(clusterID)
				Expect(err).ToNot(HaveOccurred())
				rosaClient.Runner.UnsetFormat()
				yamlData, err := rosaClient.Parser.TextData.Input(yamlOutput).Parse().YamlToMap()
				Expect(err).ToNot(HaveOccurred())
				Expect(yamlData["ignore_daemonsets_utilization"]).To(Equal(true))
				Expect(yamlData["resource_limits"].(map[string]interface{})["cores"].(map[string]interface{})["max"]).To(Equal(10))
				Expect(yamlData["resource_limits"].(map[string]interface{})["gpus"].([]interface{})[0].(map[string]interface{})["range"].(map[string]interface{})["max"]).To(Equal(5))
				Expect(yamlData["resource_limits"].(map[string]interface{})["gpus"].([]interface{})[0].(map[string]interface{})["range"].(map[string]interface{})["min"]).To(Equal(1))
				Expect(yamlData["resource_limits"].(map[string]interface{})["gpus"].([]interface{})[0].(map[string]interface{})["type"]).To(Equal("amd.com/gpu"))
				Expect(yamlData["scale_down"].(map[string]interface{})["delay_after_add"]).To(Equal("0s"))

				By("Delete the autoscaler of the cluster")
				resp, err = rosaClient.AutoScaler.DeleteAutoScaler(clusterID)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).To(ContainSubstring("INFO: Successfully deleted autoscaler configuration for cluster '%s'", clusterID))
			})
	})
})
