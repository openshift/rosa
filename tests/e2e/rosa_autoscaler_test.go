package e2e

import (
	"fmt"
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
							"--balancing-ignored-labels", originalAutoscaler.BalancingIgnoredLabels[0],
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
						"--max-nodes-total", "100",
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
						rosaClient.AutoScaler.DeleteAutoScaler(clusterID)
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
					Expect(autoscaler.ResourcesLimits.Cores.Max).To(Equal(100))
					Expect(autoscaler.ResourcesLimits.Memory.Min).To(Equal(0))
					Expect(autoscaler.ResourcesLimits.Memory.Max).To(Equal(4096))
					Expect(autoscaler.ResourcesLimits.GPUs[0].Range.Max).To(Equal(10))
					Expect(autoscaler.ResourcesLimits.GPUs[0].Range.Min).To(Equal(0))
					Expect(autoscaler.ResourcesLimits.GPUs[0].Type).To(Equal("nvidia.com/gpu"))
					Expect(autoscaler.ResourcesLimits.GPUs[1].Range.Max).To(Equal(5))
					Expect(autoscaler.ResourcesLimits.GPUs[1].Range.Min).To(Equal(1))
					Expect(autoscaler.ResourcesLimits.GPUs[1].Type).To(Equal("amd.com/gpu"))
					Expect(autoscaler.ResourcesLimits.MaxNodesTotal).To(Equal(100))
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
				Expect(autoscaler.ResourcesLimits.Cores.Min).To(Equal(0))
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

	Describe("validation testing", func() {

		Context("create/describe/edit/delete autoscaler - [id:67348]",
			labels.Medium, labels.Runtime.Day2,
			func() {

				It("for hcp cluster",
					func() {
						hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
						Expect(err).ToNot(HaveOccurred())

						if !hostedCluster {
							SkipNotHosted()
						}

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
						Expect(err).To(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(
								ContainSubstring(
									"ERR: Hosted Control Plane clusters do not support cluster-autoscaler configuration"))

						By("Describe the autoscaler of the cluster")
						output, err := rosaClient.AutoScaler.DescribeAutoScaler(clusterID)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).
							To(
								ContainSubstring(
									"ERR: Hosted Control Plane clusters do not support cluster-autoscaler configuration"))

						By("Edit the autoscaler of the cluster")
						resp, err = rosaClient.AutoScaler.EditAutoScaler(clusterID, "--ignore-daemonsets-utilization",
							"--min-cores", "0",
							"--max-cores", "10",
							"--scale-down-delay-after-add", "0s",
							"--gpu-limit", "amd.com/gpu,1,5")
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(
								ContainSubstring(
									"ERR: Hosted Control Plane clusters do not support cluster-autoscaler configuration"))

						By("Delete the autoscaler of the cluster")
						resp, err = rosaClient.AutoScaler.DeleteAutoScaler(clusterID)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(
								ContainSubstring(
									"ERR: Hosted Control Plane clusters do not support cluster-autoscaler configuration"))
					})

				It("for classic non-autoscaler cluster",
					func() {
						clusterConfig, err := config.ParseClusterProfile()
						Expect(err).ToNot(HaveOccurred())

						hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
						Expect(err).ToNot(HaveOccurred())

						if hostedCluster {
							SkipNotClassic()
						}

						if clusterConfig.Autoscaler != nil {
							Skip("autoscaler should not be enabled for 67348")
						}

						By("Describe autoscaler when no autoscaler exists in cluster")
						output, err := rosaClient.AutoScaler.DescribeAutoScaler(clusterID)
						Expect(err).To(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).
							To(
								ContainSubstring("ERR: No autoscaler exists for cluster '%s'", clusterID))

						By("Edit autoscaler when no autoscaler exists in cluster")
						resp, err := rosaClient.AutoScaler.EditAutoScaler(clusterID, "--ignore-daemonsets-utilization",
							"--min-cores", "0",
							"--max-cores", "10",
							"--scale-down-delay-after-add", "0s",
							"--gpu-limit", "amd.com/gpu,1,5")
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(
								ContainSubstring(
									"ERR: No autoscaler for cluster '%s' has been found", clusterID))

						resp, err = rosaClient.AutoScaler.EditAutoScaler(clusterID)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(
								ContainSubstring(
									"ERR: No autoscaler for cluster '%s' has been found", clusterID))

						By("Delete autoscaler when no autoscaler exists in cluster")
						resp, err = rosaClient.AutoScaler.DeleteAutoScaler(clusterID)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(
								ContainSubstring(
									"ERR: Failed to delete autoscaler configuration for "+
										"cluster '%s': Autoscaler for cluster ID '%s' is not found",
									clusterID, clusterID))

						By("Create autoscaler without setting cluster id")
						resp, err = rosaClient.AutoScaler.CreateAutoScaler("")
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("Error: required flag(s) \"cluster\" not set"))

						By("Create the autoscaler with invalid value set one at a time")

						errAndFlagCreateMap := map[string][]string{
							"Error: unknown flag: --invalid": {"--invalid", "invalid"},
							"Error: invalid argument \"ty\" for \"--balance-similar-node-groups\" " +
								"flag: strconv.ParseBool: parsing \"ty\": " +
								"invalid syntax": {"--balance-similar-node-groups=ty"},
							"Error: invalid argument \"ty\" for \"--skip-nodes-with-local-storage\" flag" +
								": strconv.ParseBool: parsing \"ty\": " +
								"invalid syntax": {"--skip-nodes-with-local-storage=ty"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s'"+
								": Error validating log-verbosity: "+
								"Number must be greater or "+
								"equal to zero.", clusterID): {"--log-verbosity", "-1"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s'"+
								": Error validating max-pod-grace-period: "+
								"Number must be greater or equal to zero.",
								clusterID): {"--max-pod-grace-period", "-1"},
							"Error: invalid argument \"ss\" for \"--pod-priority-threshold\" " +
								"flag: strconv.ParseInt: parsing \"ss\": " +
								"invalid syntax": {"--pod-priority-threshold", "ss"},
							"Error: invalid argument \"ty\" for \"--ignore-daemonsets-utilization\" " +
								"flag: strconv.ParseBool: parsing \"ty\": " +
								"invalid syntax": {"--ignore-daemonsets-utilization=ty"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"time: unknown unit \"-\" in duration \"9-\"",
								clusterID): {"--max-node-provision-time", "9-"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating max-nodes-total: "+
								"Number must be greater or equal to zero",
								clusterID): {"--max-nodes-total", "-1"},
							"Error: if any flags in the group [min-cores max-cores] " +
								"are set they must all be set; " +
								"missing [max-cores]": {"--min-cores", "1"},
							"Error: if any flags in the group [min-cores max-cores] " +
								"are set they must all be set; " +
								"missing [min-cores]": {"--max-cores", "1"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating min-cores: Number must be greater or equal to zero.",
								clusterID): {"--min-cores", "-1", "--max-cores", "1"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating max-cores: Number must be greater or equal to zero.",
								clusterID): {"--min-cores", "1", "--max-cores", "-1"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating cores range: max value must be greater or equal than min value 10.",
								clusterID): {"--min-cores", "10", "--max-cores", "8"},
							"Error: if any flags in the group [min-memory max-memory] " +
								"are set they must all be set; " +
								"missing [max-memory]": {"--min-memory", "1"},
							"Error: if any flags in the group [min-memory max-memory] " +
								"are set they must all be set; " +
								"missing [min-memory]": {"--max-memory", "1"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating min-memory: Number must be greater or equal to zero.",
								clusterID): {"--min-memory", "-1", "--max-memory", "1"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating max-memory: Number must be greater or equal to zero.",
								clusterID): {"--min-memory", "1", "--max-memory", "-1"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating memory range: max value must be greater or equal than min value 10.",
								clusterID): {"--min-memory", "10", "--max-memory", "8"},
							"Error: invalid argument \"ty\" for \"--scale-down-enabled\" flag: " +
								"strconv.ParseBool: parsing \"ty\": " +
								"invalid syntax": {"--scale-down-enabled=ty"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating delay-after-add: time: "+
								"unknown unit \"-\" in duration \"20-\"",
								clusterID): {"--scale-down-delay-after-add", "20-"},
							"Error: invalid argument \"ss\" for \"--scale-down-utilization-threshold\" " +
								"flag: strconv.ParseFloat: parsing \"ss\": " +
								"invalid syntax": {"--scale-down-utilization-threshold", "ss"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating utilization-threshold: "+
								"Expecting a floating-point number between 0 and 1.",
								clusterID): {"--scale-down-utilization-threshold", "-1"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating utilization-threshold: "+
								"Expecting a floating-point number between 0 and 1.",
								clusterID): {"--scale-down-utilization-threshold", "2"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating delay-after-delete: time: "+
								"unknown unit \"-\" in duration \"20-\"",
								clusterID): {"--scale-down-delay-after-delete", "20-"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating delay-after-failure: time: "+
								"unknown unit \"-\" in duration \"20-\"",
								clusterID): {"--scale-down-delay-after-failure", "20-"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating unneeded-time: time: "+
								"unknown unit \"-\" in duration \"20-\"",
								clusterID): {"--scale-down-unneeded-time", "20-"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating GPU range: "+
								"max value must be greater or equal than min value 10.",
								clusterID): {"--gpu-limit", "nvidia.com/gpu,10,0"},
							fmt.Sprintf("ERR: Failed creating autoscaler configuration for cluster '%s': "+
								"Error validating GPU range: "+
								"max value must be greater or equal than min value 5.",
								clusterID): {"--gpu-limit", "amd.com/gpu,5,1"},
							"Error: invalid argument \"100000000000000000000000\" for " +
								"\"--max-cores\" flag: " +
								"strconv.ParseInt: parsing \"100000000000000000000000\": " +
								"value out " +
								"of range": {"--min-cores", "5", "--max-cores", "100000000000000000000000"},
							"Error: invalid argument \"100000000000000000000000\" for " +
								"\"--max-memory\" flag: strconv.ParseInt: parsing \"100000000000000000000000\": " +
								"value out " +
								"of range": {"--min-memory", "5", "--max-memory", "100000000000000000000000"},
						}

						for errMsg, flag := range errAndFlagCreateMap {
							resp, err = rosaClient.AutoScaler.CreateAutoScaler(clusterID, flag...)
							Expect(err).To(HaveOccurred())
							textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
							Expect(textData).To(ContainSubstring(errMsg))
						}

						By("Create the autoscaler to the cluster")
						resp, err = rosaClient.AutoScaler.CreateAutoScaler(clusterID, "--balance-similar-node-groups",
							"--skip-nodes-with-local-storage",
							"--log-verbosity", "4",
							"--max-pod-grace-period", "0",
							"--pod-priority-threshold", "1",
							"--ignore-daemonsets-utilization",
							"--max-node-provision-time", "10m",
							"--balancing-ignored-labels", "aaa",
							"--max-nodes-total", "100",
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
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
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

						By("Edit autoscaler without setting cluster id")
						resp, err = rosaClient.AutoScaler.EditAutoScaler("")
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("Error: required flag(s) \"cluster\" not set"))

						By("Edit the autoscaler with invalid value set one at a time")

						errAndFlagEditMap := map[string][]string{
							"Error: unknown flag: --invalid": {"--invalid", "invalid"},
							"Error: invalid argument \"ty\" for \"--balance-similar-node-groups\" " +
								"flag: strconv.ParseBool: parsing \"ty\": " +
								"invalid syntax": {"--balance-similar-node-groups=ty"},
							"Error: invalid argument \"ty\" for \"--skip-nodes-with-local-storage\" flag" +
								": strconv.ParseBool: parsing \"ty\": " +
								"invalid syntax": {"--skip-nodes-with-local-storage=ty"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s'"+
								": Error validating log-verbosity: "+
								"Number must be greater or "+
								"equal to zero.", clusterID): {"--log-verbosity", "-1"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s'"+
								": Error validating max-pod-grace-period: "+
								"Number must be greater or equal to zero.",
								clusterID): {"--max-pod-grace-period", "-1"},
							"Error: invalid argument \"ss\" for \"--pod-priority-threshold\" " +
								"flag: strconv.ParseInt: parsing \"ss\": " +
								"invalid syntax": {"--pod-priority-threshold", "ss"},
							"Error: invalid argument \"ty\" for \"--ignore-daemonsets-utilization\" " +
								"flag: strconv.ParseBool: parsing \"ty\": " +
								"invalid syntax": {"--ignore-daemonsets-utilization=ty"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"time: unknown unit \"-\" in duration \"9-\"",
								clusterID): {"--max-node-provision-time", "9-"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating max-nodes-total: "+
								"Number must be greater or equal to zero",
								clusterID): {"--max-nodes-total", "-1"},
							"Error: if any flags in the group [min-cores max-cores] " +
								"are set they must all be set; " +
								"missing [max-cores]": {"--min-cores", "1"},
							"Error: if any flags in the group [min-cores max-cores] " +
								"are set they must all be set; " +
								"missing [min-cores]": {"--max-cores", "1"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating min-cores: Number must be greater or equal to zero.",
								clusterID): {"--min-cores", "-1", "--max-cores", "1"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating max-cores: Number must be greater or equal to zero.",
								clusterID): {"--min-cores", "1", "--max-cores", "-1"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating cores range: max value must be greater or equal than min value 10.",
								clusterID): {"--min-cores", "10", "--max-cores", "8"},
							"Error: if any flags in the group [min-memory max-memory] " +
								"are set they must all be set; " +
								"missing [max-memory]": {"--min-memory", "1"},
							"Error: if any flags in the group [min-memory max-memory] " +
								"are set they must all be set; " +
								"missing [min-memory]": {"--max-memory", "1"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating min-memory: Number must be greater or equal to zero.",
								clusterID): {"--min-memory", "-1", "--max-memory", "1"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating max-memory: Number must be greater or equal to zero.",
								clusterID): {"--min-memory", "1", "--max-memory", "-1"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating memory range: max value must be greater or equal than min value 10.",
								clusterID): {"--min-memory", "10", "--max-memory", "8"},
							"Error: invalid argument \"ty\" for \"--scale-down-enabled\" flag: " +
								"strconv.ParseBool: parsing \"ty\": " +
								"invalid syntax": {"--scale-down-enabled=ty"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating delay-after-add: time: "+
								"unknown unit \"-\" in duration \"20-\"",
								clusterID): {"--scale-down-delay-after-add", "20-"},
							"Error: invalid argument \"ss\" for \"--scale-down-utilization-threshold\" " +
								"flag: strconv.ParseFloat: parsing \"ss\": " +
								"invalid syntax": {"--scale-down-utilization-threshold", "ss"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating utilization-threshold: "+
								"Expecting a floating-point number between 0 and 1.",
								clusterID): {"--scale-down-utilization-threshold", "-1"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating utilization-threshold: "+
								"Expecting a floating-point number between 0 and 1.",
								clusterID): {"--scale-down-utilization-threshold", "2"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating delay-after-delete: time: "+
								"unknown unit \"-\" in duration \"20-\"",
								clusterID): {"--scale-down-delay-after-delete", "20-"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating delay-after-failure: time: "+
								"unknown unit \"-\" in duration \"20-\"",
								clusterID): {"--scale-down-delay-after-failure", "20-"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating unneeded-time: time: "+
								"unknown unit \"-\" in duration \"20-\"",
								clusterID): {"--scale-down-unneeded-time", "20-"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating GPU range: "+
								"max value must be greater or equal than min value 10.",
								clusterID): {"--gpu-limit", "nvidia.com/gpu,10,0"},
							fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
								"Error validating GPU range: "+
								"max value must be greater or equal than min value 5.",
								clusterID): {"--gpu-limit", "amd.com/gpu,5,1"},
							"Error: invalid argument \"100000000000000000000000\" for " +
								"\"--max-cores\" flag: " +
								"strconv.ParseInt: parsing \"100000000000000000000000\": " +
								"value out " +
								"of range": {"--min-cores", "5", "--max-cores", "100000000000000000000000"},
							"Error: invalid argument \"100000000000000000000000\" for " +
								"\"--max-memory\" flag: strconv.ParseInt: parsing \"100000000000000000000000\": " +
								"value out " +
								"of range": {"--min-memory", "5", "--max-memory", "100000000000000000000000"},
						}

						for errMsg, flag := range errAndFlagEditMap {
							resp, err = rosaClient.AutoScaler.EditAutoScaler(clusterID, flag...)
							Expect(err).To(HaveOccurred())
							textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
							Expect(textData).To(ContainSubstring(errMsg))
						}
					})
			})

		It("create/describe/edit/delete autoscaler for autoscaler enabled cluster - [id:74468]",
			labels.Medium, labels.Runtime.Day1Post,
			func() {
				hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				clusterConfig, err := config.ParseClusterProfile()
				Expect(err).ToNot(HaveOccurred())

				if hostedCluster {
					SkipNotClassic()
				}

				if clusterConfig.Autoscaler == nil {
					SkipTestOnFeature("autoscaler")
				}

				By("Create the autoscaler to the cluster when already existing")
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
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).
					To(
						ContainSubstring(
							"ERR: Autoscaler for cluster '%s' already exists", clusterID))

				By("Edit autoscaler without setting cluster id")
				resp, err = rosaClient.AutoScaler.EditAutoScaler("")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).
					To(
						ContainSubstring("Error: required flag(s) \"cluster\" not set"))

				By("Edit the autoscaler with invalid value set one at a time")

				errAndFlagEditMap := map[string][]string{
					"Error: unknown flag: --invalid": {"--invalid", "invalid"},
					"Error: invalid argument \"ty\" for \"--balance-similar-node-groups\" " +
						"flag: strconv.ParseBool: parsing \"ty\": " +
						"invalid syntax": {"--balance-similar-node-groups=ty"},
					"Error: invalid argument \"ty\" for \"--skip-nodes-with-local-storage\" flag" +
						": strconv.ParseBool: parsing \"ty\": " +
						"invalid syntax": {"--skip-nodes-with-local-storage=ty"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s'"+
						": Error validating log-verbosity: "+
						"Number must be greater or "+
						"equal to zero.", clusterID): {"--log-verbosity", "-1"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s'"+
						": Error validating max-pod-grace-period: "+
						"Number must be greater or equal to zero.",
						clusterID): {"--max-pod-grace-period", "-1"},
					"Error: invalid argument \"ss\" for \"--pod-priority-threshold\" " +
						"flag: strconv.ParseInt: parsing \"ss\": " +
						"invalid syntax": {"--pod-priority-threshold", "ss"},
					"Error: invalid argument \"ty\" for \"--ignore-daemonsets-utilization\" " +
						"flag: strconv.ParseBool: parsing \"ty\": " +
						"invalid syntax": {"--ignore-daemonsets-utilization=ty"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"time: unknown unit \"-\" in duration \"9-\"",
						clusterID): {"--max-node-provision-time", "9-"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating max-nodes-total: "+
						"Number must be greater or equal to zero",
						clusterID): {"--max-nodes-total", "-1"},
					"Error: if any flags in the group [min-cores max-cores] " +
						"are set they must all be set; " +
						"missing [max-cores]": {"--min-cores", "1"},
					"Error: if any flags in the group [min-cores max-cores] " +
						"are set they must all be set; " +
						"missing [min-cores]": {"--max-cores", "1"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating min-cores: Number must be greater or equal to zero.",
						clusterID): {"--min-cores", "-1", "--max-cores", "1"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating max-cores: Number must be greater or equal to zero.",
						clusterID): {"--min-cores", "1", "--max-cores", "-1"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating cores range: max value must be greater or equal than min value 10.",
						clusterID): {"--min-cores", "10", "--max-cores", "8"},
					"Error: if any flags in the group [min-memory max-memory] " +
						"are set they must all be set; " +
						"missing [max-memory]": {"--min-memory", "1"},
					"Error: if any flags in the group [min-memory max-memory] " +
						"are set they must all be set; " +
						"missing [min-memory]": {"--max-memory", "1"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating min-memory: Number must be greater or equal to zero.",
						clusterID): {"--min-memory", "-1", "--max-memory", "1"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating max-memory: Number must be greater or equal to zero.",
						clusterID): {"--min-memory", "1", "--max-memory", "-1"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating memory range: max value must be greater or equal than min value 10.",
						clusterID): {"--min-memory", "10", "--max-memory", "8"},
					"Error: invalid argument \"ty\" for \"--scale-down-enabled\" flag: " +
						"strconv.ParseBool: parsing \"ty\": " +
						"invalid syntax": {"--scale-down-enabled=ty"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating delay-after-add: time: "+
						"unknown unit \"-\" in duration \"20-\"",
						clusterID): {"--scale-down-delay-after-add", "20-"},
					"Error: invalid argument \"ss\" for \"--scale-down-utilization-threshold\" " +
						"flag: strconv.ParseFloat: parsing \"ss\": " +
						"invalid syntax": {"--scale-down-utilization-threshold", "ss"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating utilization-threshold: "+
						"Expecting a floating-point number between 0 and 1.",
						clusterID): {"--scale-down-utilization-threshold", "-1"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating utilization-threshold: "+
						"Expecting a floating-point number between 0 and 1.",
						clusterID): {"--scale-down-utilization-threshold", "2"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating delay-after-delete: time: "+
						"unknown unit \"-\" in duration \"20-\"",
						clusterID): {"--scale-down-delay-after-delete", "20-"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating delay-after-failure: time: "+
						"unknown unit \"-\" in duration \"20-\"",
						clusterID): {"--scale-down-delay-after-failure", "20-"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating unneeded-time: time: "+
						"unknown unit \"-\" in duration \"20-\"",
						clusterID): {"--scale-down-unneeded-time", "20-"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating GPU range: "+
						"max value must be greater or equal than min value 10.",
						clusterID): {"--gpu-limit", "nvidia.com/gpu,10,0"},
					fmt.Sprintf("ERR: Failed updating autoscaler configuration for cluster '%s': "+
						"Error validating GPU range: "+
						"max value must be greater or equal than min value 5.",
						clusterID): {"--gpu-limit", "amd.com/gpu,5,1"},
					"Error: invalid argument \"100000000000000000000000\" for " +
						"\"--max-cores\" flag: " +
						"strconv.ParseInt: parsing \"100000000000000000000000\": " +
						"value out " +
						"of range": {"--min-cores", "5", "--max-cores", "100000000000000000000000"},
					"Error: invalid argument \"100000000000000000000000\" for " +
						"\"--max-memory\" flag: strconv.ParseInt: parsing \"100000000000000000000000\": " +
						"value out " +
						"of range": {"--min-memory", "5", "--max-memory", "100000000000000000000000"},
				}

				for errMsg, flag := range errAndFlagEditMap {
					resp, err = rosaClient.AutoScaler.EditAutoScaler(clusterID, flag...)
					Expect(err).To(HaveOccurred())
					textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
					Expect(textData).To(ContainSubstring(errMsg))
				}
			})
	})
})
