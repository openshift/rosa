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
					By("Describe the original autoscaler of the cluster")
					rosaClient.Runner.YamlFormat()
					yamlOutput, err := rosaClient.AutoScaler.DescribeAutoScaler(clusterID)
					Expect(err).ToNot(HaveOccurred())
					rosaClient.Runner.UnsetFormat()

					originalAutoscaler := rosacli.Autoscaler{}
					err = rosaClient.Parser.TextData.Input(yamlOutput).Parse().YamlToObj(&originalAutoscaler)
					Expect(err).ToNot(HaveOccurred())

					defer func() {
						By("Create the autoscaler to the cluster")
						resp, err := rosaClient.AutoScaler.CreateAutoScaler(clusterID,
							"--balance-similar-node-groups",
							"--skip-nodes-with-local-storage",
							"--log-verbosity", strconv.Itoa(originalAutoscaler.LogVerbosity),
							"--max-pod-grace-period", strconv.Itoa(originalAutoscaler.MaxPodGracePeriod),
							"--pod-priority-threshold", strconv.Itoa(originalAutoscaler.PodPriorityThresold),
							"--ignore-daemonsets-utilization",
							"--max-node-provision-time", originalAutoscaler.MaxNodeProvisionTime,
							"--balancing-ignored-labels", originalAutoscaler.BalancingIgnoredLabels,
							"--max-nodes-total", strconv.Itoa(originalAutoscaler.ResourcesLimits.MaxNodesTotal),
							"--min-cores", strconv.Itoa(originalAutoscaler.ResourcesLimits.Cores.Min),
							"--scale-down-delay-after-add", originalAutoscaler.ScaleDown.DelayAfterAdd,
							"--max-cores", strconv.Itoa(originalAutoscaler.ResourcesLimits.Cores.Max),
							"--min-memory", strconv.Itoa(originalAutoscaler.ResourcesLimits.Memory.Min),
							"--max-memory", strconv.Itoa(originalAutoscaler.ResourcesLimits.Memory.Max),
							"--scale-down-enabled",
							"--scale-down-utilization-threshold", originalAutoscaler.ScaleDown.UtilizationThreshold,
							"--scale-down-delay-after-delete", originalAutoscaler.ScaleDown.DelayAfterDelete,
							"--scale-down-delay-after-failure", originalAutoscaler.ScaleDown.DelayAfterFailure)
						Expect(err).ToNot(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(
								ContainSubstring("INFO: Successfully created autoscaler configuration for cluster '%s'", clusterID))
					}()
				} else {

					By("Create the autoscaler to the cluster")
					resp, err := rosaClient.AutoScaler.CreateAutoScaler(clusterID,
						"--balance-similar-node-groups",
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
					Expect(textData).
						To(
							ContainSubstring(
								"INFO: Successfully created autoscaler configuration for cluster '%s'",
								clusterID))

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

					autoscaler := rosacli.Autoscaler{}
					err = rosaClient.Parser.TextData.Input(yamlOutput).Parse().YamlToObj(&autoscaler)
					Expect(err).ToNot(HaveOccurred())
					Expect(autoscaler.BalanceSimilarNodeGroups).To(Equal(true))
					Expect(autoscaler.SkipNodesWithLocalStorage).To(Equal(true))
					Expect(autoscaler.LogVerbosity).To(Equal(4))
					Expect(autoscaler.BalancingIgnoredLabels).To(ContainElement("aaa"))
					Expect(autoscaler.IgnoreDaemonSetsUtilization).To(Equal(true))
					Expect(autoscaler.MaxNodeProvisionTime).To(Equal("10m"))
					Expect(autoscaler.MaxPodGracePeriod).To(Equal(0))
					Expect(autoscaler.PodPriorityThresold).To(Equal(1))
					Expect(autoscaler.ResourcesLimits.Cores.Min).To(Equal(0))
					Expect(autoscaler.ResourcesLimits.Cores.Min).To(Equal(100))
					Expect(autoscaler.ResourcesLimits.Memory.Min).To(Equal(0))
					Expect(autoscaler.ResourcesLimits.Cores.Max).To(Equal(4096))
					Expect(autoscaler.ResourcesLimits.GPUs[0].Range.Max).To(Equal(10))
					Expect(autoscaler.ResourcesLimits.GPUs[0].Range.Min).To(Equal(0))
					Expect(autoscaler.ResourcesLimits.GPUs[0].Type).To(Equal("nvidia.com/gpu"))
					Expect(autoscaler.ResourcesLimits.GPUs[0].Range.Max).To(Equal(5))
					Expect(autoscaler.ResourcesLimits.GPUs[0].Range.Min).To(Equal(1))
					Expect(autoscaler.ResourcesLimits.GPUs[0].Type).To(Equal("amd.com/gpu"))
					Expect(autoscaler.ResourcesLimits.MaxNodesTotal).To(Equal(1000))
					Expect(autoscaler.ScaleDown.DelayAfterAdd).To(Equal("10s"))
					Expect(autoscaler.ScaleDown.DelayAfterDelete).To(Equal("10s"))
					Expect(autoscaler.ScaleDown.DelayAfterFailure).To(Equal("10s"))
					Expect(autoscaler.ScaleDown.Enabled).To(Equal(true))
					Expect(autoscaler.ScaleDown.UnneededTime).To(Equal("10s"))
					Expect(autoscaler.ScaleDown.UtilizationThreshold).To(Equal("1.000000"))

				}

				By("Edit the autoscaler of the cluster")
				resp, err := rosaClient.AutoScaler.EditAutoScaler(clusterID, "--ignore-daemonsets-utilization",
					"--min-cores", "0",
					"--max-cores", "10",
					"--scale-down-delay-after-add", "0s",
					"--gpu-limit", "amd.com/gpu,1,5")
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).
					To(
						ContainSubstring("INFO: Successfully updated autoscaler configuration for cluster '%s'",
							clusterID))

				By("Describe autoscaler to check the edited value is correct")
				rosaClient.Runner.YamlFormat()
				yamlOutput, err := rosaClient.AutoScaler.DescribeAutoScaler(clusterID)
				Expect(err).ToNot(HaveOccurred())
				rosaClient.Runner.UnsetFormat()

				autoscaler := rosacli.Autoscaler{}
				err = rosaClient.Parser.TextData.Input(yamlOutput).Parse().YamlToObj(&autoscaler)
				Expect(err).ToNot(HaveOccurred())
				Expect(autoscaler.IgnoreDaemonSetsUtilization).To(Equal(true))
				Expect(autoscaler.ResourcesLimits.Cores.Max).To(Equal(10))
				Expect(autoscaler.ResourcesLimits.GPUs[0].Range.Max).To(Equal(5))
				Expect(autoscaler.ResourcesLimits.GPUs[0].Range.Min).To(Equal(1))
				Expect(autoscaler.ResourcesLimits.GPUs[0].Type).To(Equal("amd.com/gpu"))
				Expect(autoscaler.ScaleDown.DelayAfterAdd).To(Equal("0s"))

				By("Delete the autoscaler of the cluster")
				resp, err = rosaClient.AutoScaler.DeleteAutoScaler(clusterID)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"INFO: Successfully deleted autoscaler configuration for cluster '%s'",
						clusterID))
			})
	})
})
