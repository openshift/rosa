package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/strings/slices"

	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	. "github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("Edit nodepool",
	labels.Feature.Machinepool,
	func() {
		defer GinkgoRecover()

		var (
			clusterID                 string
			rosaClient                *rosacli.Client
			clusterService            rosacli.ClusterService
			machinePoolService        rosacli.MachinePoolService
			machinePoolUpgradeService rosacli.MachinePoolUpgradeService
			versionService            rosacli.VersionService
		)

		const (
			defaultNodePoolReplicas = "2"
		)

		BeforeEach(func() {
			var err error

			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			machinePoolService = rosaClient.MachinePool
			machinePoolUpgradeService = rosaClient.MachinePoolUpgrade
			versionService = rosaClient.Version

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

		It("can create/edit/list/delete nodepool - [id:56782]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				nodePoolName := common.GenerateRandomName("np-56782", 2)
				labels := "label1=value1,label2=value2"
				taints := "t1=v1:NoSchedule,l2=:NoSchedule"
				instanceType := "m5.2xlarge"

				By("Retrieve cluster initial information")
				cluster, err := clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				cpVersion := cluster.OpenshiftVersion

				By("Create new nodepool")
				output, err := machinePoolService.CreateMachinePool(clusterID, nodePoolName,
					"--replicas", "0",
					"--instance-type", instanceType,
					"--labels", labels,
					"--taints", taints)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						nodePoolName,
						clusterID))

				By("Check created nodepool")
				npList, err := machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				np := npList.Nodepool(nodePoolName)
				Expect(np).ToNot(BeNil())
				Expect(np.AutoScaling).To(Equal("No"))
				Expect(np.Replicas).To(Equal("0/0"))
				Expect(np.InstanceType).To(Equal(instanceType))
				Expect(np.AvalaiblityZones).ToNot(BeNil())
				Expect(np.Subnet).ToNot(BeNil())
				Expect(np.Version).To(Equal(cpVersion))
				Expect(np.AutoRepair).To(Equal("Yes"))
				Expect(len(common.ParseLabels(np.Labels))).To(Equal(len(common.ParseLabels(labels))))
				Expect(common.ParseLabels(np.Labels)).To(ContainElements(common.ParseLabels(labels)))
				Expect(len(common.ParseTaints(np.Taints))).To(Equal(len(common.ParseTaints(taints))))
				Expect(common.ParseTaints(np.Taints)).To(ContainElements(common.ParseTaints(taints)))

				By("Edit nodepool")
				newLabels := "l3=v3"
				newTaints := "t3=value3:NoExecute"
				replicasNb := "3"
				output, err = machinePoolService.EditMachinePool(clusterID, nodePoolName,
					"--replicas", replicasNb,
					"--labels", newLabels,
					"--taints", newTaints)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Updated machine pool '%s' on hosted cluster '%s'",
						nodePoolName,
						clusterID))

				By("Check edited nodepool")
				npList, err = machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				np = npList.Nodepool(nodePoolName)
				Expect(np).ToNot(BeNil())
				Expect(np.Replicas).To(Equal(fmt.Sprintf("0/%s", replicasNb)))
				Expect(len(common.ParseLabels(np.Labels))).To(Equal(len(common.ParseLabels(newLabels))))
				Expect(common.ParseLabels(np.Labels)).To(BeEquivalentTo(common.ParseLabels(newLabels)))
				Expect(len(common.ParseTaints(np.Taints))).To(Equal(len(common.ParseTaints(newTaints))))
				Expect(common.ParseTaints(np.Taints)).To(BeEquivalentTo(common.ParseTaints(newTaints)))

				By("Check describe nodepool")
				npDesc, err := machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
				Expect(err).ToNot(HaveOccurred())

				Expect(npDesc).ToNot(BeNil())
				Expect(npDesc.AutoScaling).To(Equal("No"))
				Expect(npDesc.DesiredReplicas).To(Equal(replicasNb))
				Expect(npDesc.CurrentReplicas).To(Equal("0"))
				Expect(npDesc.InstanceType).To(Equal(instanceType))
				Expect(npDesc.AvalaiblityZones).ToNot(BeNil())
				Expect(npDesc.Subnet).ToNot(BeNil())
				Expect(npDesc.Version).To(Equal(cpVersion))
				Expect(npDesc.AutoRepair).To(Equal("Yes"))
				Expect(len(common.ParseLabels(npDesc.Labels))).To(Equal(len(common.ParseLabels(newLabels))))
				Expect(common.ParseLabels(npDesc.Labels)).To(BeEquivalentTo(common.ParseLabels(newLabels)))
				Expect(len(common.ParseTaints(npDesc.Taints))).To(Equal(len(common.ParseTaints(newTaints))))
				Expect(common.ParseTaints(npDesc.Taints)).To(BeEquivalentTo(common.ParseTaints(newTaints)))

				By("Delete nodepool")
				output, err = machinePoolService.DeleteMachinePool(clusterID, nodePoolName)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Successfully deleted machine pool '%s' from hosted cluster '%s'",
						nodePoolName,
						clusterID))

				By("Nodepool does not appear anymore")
				npList, err = machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(npList.Nodepool(nodePoolName)).To(BeNil())
			})

		It("can create nodepool with defined subnets - [id:60202]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				var subnets []string
				nodePoolName := common.GenerateRandomName("np-60202", 2)
				replicasNumber := 3
				maxReplicasNumber := 6

				By("Retrieve cluster nodes information")
				CD, err := clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				initialNodesNumber, err := rosacli.RetrieveDesiredComputeNodes(CD)
				Expect(err).ToNot(HaveOccurred())

				By("List nodepools")
				npList, err := machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				for _, np := range npList.NodePools {
					Expect(np.ID).ToNot(BeNil())
					if strings.HasPrefix(np.ID, constants.DefaultHostedWorkerPool) {
						Expect(np.AutoScaling).ToNot(BeNil())
						Expect(np.Subnet).ToNot(BeNil())
						Expect(np.AutoRepair).ToNot(BeNil())
					}

					if !slices.Contains(subnets, np.Subnet) {
						subnets = append(subnets, np.Subnet)
					}
				}

				By("Create new nodepool with defined subnet")
				output, err := machinePoolService.CreateMachinePool(clusterID, nodePoolName,
					"--replicas", strconv.Itoa(replicasNumber),
					"--subnet", subnets[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						nodePoolName,
						clusterID))

				npList, err = machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				np := npList.Nodepool(nodePoolName)
				Expect(np).ToNot(BeNil())
				Expect(np.AutoScaling).To(Equal("No"))
				Expect(np.Replicas).To(Equal("0/3"))
				Expect(np.AvalaiblityZones).ToNot(BeNil())
				Expect(np.Subnet).To(Equal(subnets[0]))
				Expect(np.AutoRepair).To(Equal("Yes"))

				By("Check cluster nodes information with new replicas")
				CD, err = clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				newNodesNumber, err := rosacli.RetrieveDesiredComputeNodes(CD)
				Expect(err).ToNot(HaveOccurred())
				Expect(newNodesNumber).To(Equal(initialNodesNumber + replicasNumber))

				By("Add autoscaling to nodepool")
				output, err = machinePoolService.EditMachinePool(clusterID, nodePoolName,
					"--enable-autoscaling",
					"--min-replicas", strconv.Itoa(replicasNumber),
					"--max-replicas", strconv.Itoa(maxReplicasNumber),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Updated machine pool '%s' on hosted cluster '%s'",
						nodePoolName,
						clusterID))
				npList, err = machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				np = npList.Nodepool(nodePoolName)
				Expect(np).ToNot(BeNil())
				Expect(np.AutoScaling).To(Equal("Yes"))

				// Change autorepair
				output, err = machinePoolService.EditMachinePool(clusterID, nodePoolName,
					"--autorepair=false",

					// Temporary fix until https://issues.redhat.com/browse/OCM-5186 is corrected
					"--enable-autoscaling",
					"--min-replicas", strconv.Itoa(replicasNumber),
					"--max-replicas", strconv.Itoa(maxReplicasNumber),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Updated machine pool '%s' on hosted cluster '%s'",
						nodePoolName,
						clusterID))
				npList, err = machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				np = npList.Nodepool(nodePoolName)
				Expect(np).ToNot(BeNil())
				Expect(np.AutoRepair).To(Equal("No"))

				By("Delete nodepool")
				output, err = machinePoolService.DeleteMachinePool(clusterID, nodePoolName)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Successfully deleted machine pool '%s' from hosted cluster '%s'",
						nodePoolName,
						clusterID))

				By("Check cluster nodes information after deletion")
				CD, err = clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				newNodesNumber, err = rosacli.RetrieveDesiredComputeNodes(CD)
				Expect(err).ToNot(HaveOccurred())
				Expect(newNodesNumber).To(Equal(initialNodesNumber))

				By("Create new nodepool with replicas 0")
				replicas0NPName := common.GenerateRandomName(nodePoolName, 2)
				_, err = machinePoolService.CreateMachinePool(
					clusterID,
					replicas0NPName,
					"--replicas", strconv.Itoa(0),
					"--subnet", subnets[0])
				Expect(err).ToNot(HaveOccurred())
				npList, err = machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				np = npList.Nodepool(replicas0NPName)
				Expect(np).ToNot(BeNil())
				Expect(np.Replicas).To(Equal("0/0"))

				By("Create new nodepool with min replicas 0")
				minReplicas0NPName := common.GenerateRandomName(nodePoolName, 2)
				_, err = machinePoolService.CreateMachinePool(
					clusterID,
					minReplicas0NPName,
					"--enable-autoscaling",
					"--min-replicas", strconv.Itoa(0),
					"--max-replicas", strconv.Itoa(3),
					"--subnet", subnets[0],
				)
				Expect(err).To(HaveOccurred())
			})

		It("can create nodepool with tuning config - [id:63178]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				tuningConfigService := rosaClient.TuningConfig
				nodePoolName := common.GenerateRandomName("np-63178", 2)
				tuningConfig1Name := common.GenerateRandomName("tuned01", 2)
				tc1Spec := rosacli.NewTuningConfigSpecRootStub(tuningConfig1Name, 25, 10)
				tuningConfig2Name := common.GenerateRandomName("tuned02", 2)
				tc2Spec := rosacli.NewTuningConfigSpecRootStub(tuningConfig2Name, 25, 10)
				tuningConfig3Name := common.GenerateRandomName("tuned03", 2)
				tc3Spec := rosacli.NewTuningConfigSpecRootStub(tuningConfig2Name, 25, 10)
				allTuningConfigNames := []string{tuningConfig1Name, tuningConfig2Name, tuningConfig3Name}

				By("Prepare tuning configs")
				tc1JSON, err := json.Marshal(tc1Spec)
				Expect(err).ToNot(HaveOccurred())
				_, err = tuningConfigService.CreateTuningConfig(
					clusterID,
					tuningConfig1Name,
					string(tc1JSON))
				Expect(err).ToNot(HaveOccurred())

				tc2JSON, err := json.Marshal(tc2Spec)
				Expect(err).ToNot(HaveOccurred())
				_, err = tuningConfigService.CreateTuningConfig(
					clusterID,
					tuningConfig2Name,
					string(tc2JSON))
				Expect(err).ToNot(HaveOccurred())

				tc3JSON, err := json.Marshal(tc3Spec)
				Expect(err).ToNot(HaveOccurred())
				_, err = tuningConfigService.CreateTuningConfig(
					clusterID,
					tuningConfig3Name,
					string(tc3JSON))
				Expect(err).ToNot(HaveOccurred())

				By("Create nodepool with tuning configs")
				_, err = machinePoolService.CreateMachinePool(clusterID, nodePoolName,
					"--replicas", "3",
					"--tuning-configs", strings.Join(allTuningConfigNames, ","),
				)
				Expect(err).ToNot(HaveOccurred())

				By("Describe nodepool")
				np, err := machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(common.ParseTuningConfigs(np.TuningConfigs))).To(Equal(3))
				Expect(common.ParseTuningConfigs(np.TuningConfigs)).To(ContainElements(allTuningConfigNames))

				By("Update nodepool with only one tuning config")
				_, err = machinePoolService.EditMachinePool(clusterID, nodePoolName,
					"--tuning-configs", tuningConfig1Name,
				)
				Expect(err).ToNot(HaveOccurred())
				np, err = machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(common.ParseTuningConfigs(np.TuningConfigs))).To(Equal(1))
				Expect(common.ParseTuningConfigs(np.TuningConfigs)).To(ContainElements([]string{tuningConfig1Name}))

				By("Update nodepool with no tuning config")
				_, err = machinePoolService.EditMachinePool(clusterID, nodePoolName,
					"--tuning-configs", "",
				)
				Expect(err).ToNot(HaveOccurred())
				np, err = machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(common.ParseTuningConfigs(np.TuningConfigs))).To(Equal(0))
			})

		It("create nodepool with tuning config will validate well - [id:63179]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				tuningConfigService := rosaClient.TuningConfig
				nodePoolName := common.GenerateRandomName("np-63179", 2)
				tuningConfigName := common.GenerateRandomName("tuned01", 2)
				tcSpec := rosacli.NewTuningConfigSpecRootStub(tuningConfigName, 25, 10)
				nonExistingTuningConfigName := common.GenerateRandomName("fake_tuning_config", 2)

				By("Prepare tuning configs")
				tcJSON, err := json.Marshal(tcSpec)
				Expect(err).ToNot(HaveOccurred())
				_, err = tuningConfigService.CreateTuningConfig(
					clusterID,
					tuningConfigName,
					string(tcJSON))
				Expect(err).ToNot(HaveOccurred())

				By("Create nodepool with the non-existing tuning configs")
				output, err := machinePoolService.CreateMachinePool(
					clusterID,
					nodePoolName,
					"--replicas", "3",
					"--tuning-configs", nonExistingTuningConfigName,
				)
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(err).To(HaveOccurred())
				Expect(textData).
					To(ContainSubstring(
						fmt.Sprintf("Failed to add machine pool to hosted cluster '%s': "+
							"Tuning config with name '%s' does not exist for cluster '%s'",
							clusterID,
							nonExistingTuningConfigName,
							clusterID)))

				By("Create nodepool with duplicate tuning configs")
				output, err = machinePoolService.CreateMachinePool(
					clusterID,
					nodePoolName,
					"--replicas", "3",
					"--tuning-configs", fmt.Sprintf("%s,%s", tuningConfigName, tuningConfigName),
				)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(err).To(HaveOccurred())
				Expect(textData).
					To(ContainSubstring(
						fmt.Sprintf("Failed to add machine pool to hosted cluster '%s': "+
							"Tuning config with name '%s' is duplicated",
							clusterID,
							tuningConfigName)))

				By("Create a nodepool")
				_, err = machinePoolService.CreateMachinePool(clusterID, nodePoolName,
					"--replicas", "3",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Update nodepool with non-existing tuning config")
				output, err = machinePoolService.EditMachinePool(clusterID, nodePoolName,
					"--tuning-configs", nonExistingTuningConfigName,
				)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(err).To(HaveOccurred())
				Expect(textData).
					To(ContainSubstring(
						fmt.Sprintf("Failed to update machine pool '%s' on hosted cluster '%s': "+
							"Tuning config with name '%s' does not exist for cluster '%s'",
							nodePoolName,
							clusterID,
							nonExistingTuningConfigName,
							clusterID)))
			})

		It("does support 'version' parameter on nodepool - [id:61138]",
			labels.High, labels.Runtime.Day2,
			func() {
				nodePoolName := common.GenerateRandomName("np-61138", 2)

				By("Get previous version")
				clusterVersionInfo, err := clusterService.GetClusterVersion(clusterID)
				Expect(err).ToNot(HaveOccurred())
				clusterVersion := clusterVersionInfo.RawID
				clusterChannelGroup := clusterVersionInfo.ChannelGroup
				versionList, err := versionService.ListAndReflectVersions(clusterChannelGroup, true)
				Expect(err).ToNot(HaveOccurred())

				previousVersionsList, err := versionList.FilterVersionsLowerThan(clusterVersion)
				Expect(err).ToNot(HaveOccurred())
				if previousVersionsList.Len() <= 1 {
					Skip("Skipping as no previous version is available for testing")
				}
				previousVersionsList.Sort(true)
				previousVersion := previousVersionsList.OpenShiftVersions[0].Version

				By("Check create nodepool version help parameter")
				help, err := machinePoolService.RetrieveHelpForCreate()
				Expect(err).ToNot(HaveOccurred())
				Expect(help.String()).To(ContainSubstring("--version"))

				By("Check version is displayed in list")
				nps, err := machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				for _, np := range nps.NodePools {
					Expect(np.Version).To(Not(BeEmpty()))
				}

				By("Create NP with previous version")
				_, err = machinePoolService.CreateMachinePool(clusterID, nodePoolName,
					"--replicas", defaultNodePoolReplicas,
					"--version", previousVersion,
				)
				Expect(err).ToNot(HaveOccurred())

				By("Check NodePool was correctly created")
				np, err := machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
				Expect(err).ToNot(HaveOccurred())
				Expect(np.Version).To(Equal(previousVersion))

				By("Wait for NodePool replicas to be available")
				err = wait.PollUntilContextTimeout(
					context.Background(),
					30*time.Second,
					20*time.Minute,
					false,
					func(context.Context) (bool, error) {
						npDesc, err := machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
						if err != nil {
							return false, err
						}
						return npDesc.CurrentReplicas == defaultNodePoolReplicas, nil
					})
				common.AssertWaitPollNoErr(err, "Replicas are not ready after 600")

				nodePoolVersion, err := versionList.FindNearestBackwardMinorVersion(clusterVersion, 1, true)
				Expect(err).ToNot(HaveOccurred())
				if nodePoolVersion != nil {
					By("Create NodePool with version minor - 1")
					nodePoolName = common.GenerateRandomName("np-61138-m1", 2)
					_, err = machinePoolService.CreateMachinePool(clusterID,
						nodePoolName,
						"--replicas", defaultNodePoolReplicas,
						"--version", nodePoolVersion.Version,
					)
					Expect(err).ToNot(HaveOccurred())
					np, err = machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
					Expect(err).ToNot(HaveOccurred())
					Expect(np.Version).To(Equal(nodePoolVersion.Version))
				}

				nodePoolVersion, err = versionList.FindNearestBackwardMinorVersion(clusterVersion, 2, true)
				Expect(err).ToNot(HaveOccurred())
				if nodePoolVersion != nil {
					By("Create NodePool with version minor - 2")
					nodePoolName = common.GenerateRandomName("np-61138-m1", 2)
					_, err = machinePoolService.CreateMachinePool(clusterID,
						nodePoolName,
						"--replicas", defaultNodePoolReplicas,
						"--version", nodePoolVersion.Version,
					)
					Expect(err).ToNot(HaveOccurred())
					np, err = machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
					Expect(err).ToNot(HaveOccurred())
					Expect(np.Version).To(Equal(nodePoolVersion.Version))
				}

				nodePoolVersion, err = versionList.FindNearestBackwardMinorVersion(clusterVersion, 3, true)
				Expect(err).ToNot(HaveOccurred())
				if nodePoolVersion != nil {
					By("Create NodePool with version minor - 3 should fail")
					_, err = machinePoolService.CreateMachinePool(clusterID,
						common.GenerateRandomName("np-61138-m3", 2),
						"--replicas", defaultNodePoolReplicas,
						"--version", nodePoolVersion.Version,
					)
					Expect(err).To(HaveOccurred())
				}
			})

		It("can validate the version parameter on nodepool creation/editing - [id:61139]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				testVersionFailFunc := func(flags ...string) {
					Logger.Infof("Creating nodepool with flags %v", flags)
					output, err := machinePoolService.CreateMachinePool(
						clusterID,
						common.GenerateRandomName("np-61139", 2),
						flags...)
					Expect(err).To(HaveOccurred())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).
						Should(ContainSubstring(
							`ERR: Expected a valid OpenShift version: A valid version number must be specified`))
					textData = rosaClient.Parser.TextData.Input(output).Parse().Output()
					Expect(textData).Should(ContainSubstring(`Valid versions:`))
				}

				By("Get cluster version")
				clusterVersionInfo, err := clusterService.GetClusterVersion(clusterID)
				Expect(err).ToNot(HaveOccurred())
				clusterVersion := clusterVersionInfo.RawID
				clusterChannelGroup := clusterVersionInfo.ChannelGroup
				clusterSemVer, err := semver.NewVersion(clusterVersion)
				Expect(err).ToNot(HaveOccurred())
				clusterVersionList, err := versionService.ListAndReflectVersions(clusterChannelGroup, true)
				Expect(err).ToNot(HaveOccurred())

				By("Create a nodepool with version greater than cluster's version should fail")
				testVersion := fmt.Sprintf("%d.%d.%d",
					clusterSemVer.Major()+100,
					clusterSemVer.Minor()+100,
					clusterSemVer.Patch()+100)
				testVersionFailFunc("--replicas",
					defaultNodePoolReplicas,
					"--version",
					testVersion)

				if clusterChannelGroup != rosacli.VersionChannelGroupNightly {
					versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupNightly, true)
					Expect(err).ToNot(HaveOccurred())
					lowerVersionsList, err := versionList.FilterVersionsLowerThan(clusterVersion)
					Expect(err).ToNot(HaveOccurred())
					if lowerVersionsList.Len() > 0 {
						By("Create a nodepool with version from incompatible channel group should fail")
						lowerVersionsList.Sort(true)
						testVersion := lowerVersionsList.OpenShiftVersions[0].Version
						testVersionFailFunc("--replicas",
							defaultNodePoolReplicas,
							"--version",
							testVersion)
					}
				}

				By("Create a nodepool with major different from cluster's version should fail")
				testVersion = fmt.Sprintf("%d.%d.%d",
					clusterSemVer.Major()-1,
					clusterSemVer.Minor(),
					clusterSemVer.Patch())
				testVersionFailFunc("--replicas",
					defaultNodePoolReplicas,
					"--version",
					testVersion)

				foundVersion, err := clusterVersionList.FindNearestBackwardMinorVersion(clusterVersion, 3, false)
				Expect(err).ToNot(HaveOccurred())
				if foundVersion != nil {
					By("Create a nodepool with minor lower than cluster's 'minor - 3' should fail")
					testVersion = foundVersion.Version
					testVersionFailFunc("--replicas",
						defaultNodePoolReplicas,
						"--version",
						testVersion)
				}

				By("Create a nodepool with non existing version should fail")
				testVersion = "24512.5632.85"
				testVersionFailFunc("--replicas",
					defaultNodePoolReplicas,
					"--version",
					testVersion)
			})

		It("can list/describe/delete nodepool upgrade policies - [id:67414]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				currentDateTimeUTC := time.Now().UTC()

				By("Check help(s) for node pool upgrade")
				helpMessageFuncs := []func() (bytes.Buffer, error){
					machinePoolUpgradeService.RetrieveHelpForCreate,
					machinePoolUpgradeService.RetrieveHelpForDescribe,
					machinePoolUpgradeService.RetrieveHelpForList,
					machinePoolUpgradeService.RetrieveHelpForDelete,
				}
				for index, funcName := range helpMessageFuncs {
					help, err := funcName()
					Expect(err).ToNot(HaveOccurred())
					if index == 0 {
						continue
					}
					Expect(help.String()).To(ContainSubstring("--machinepool"))
				}

				By("Get a lower version")
				clusterVersionInfo, err := clusterService.GetClusterVersion(clusterID)
				Expect(err).ToNot(HaveOccurred())
				clusterVersion := clusterVersionInfo.RawID
				clusterChannelGroup := clusterVersionInfo.ChannelGroup
				versionList, err := versionService.ListAndReflectVersions(clusterChannelGroup, false)
				Expect(err).ToNot(HaveOccurred())

				var lVersion string = clusterVersion
				var upgradeVersion string
				for {
					lowerVersion, err := versionList.FindNearestBackwardOptionalVersion(lVersion, 1, false)
					Expect(err).ToNot(HaveOccurred())
					lVersion = lowerVersion.Version
					if lowerVersion.AvailableUpgrades != "" {
						upgrades := common.ParseCommaSeparatedStrings(lowerVersion.AvailableUpgrades)
						upgradeVersion = upgrades[len(upgrades)-1]
						break
					}
					Logger.Debugf("The lower version %s has no available upgrades continue to find next one\n", lVersion)
				}
				if upgradeVersion == "" {
					Logger.Warn("Cannot find a version with available upgrades")
					return
				}
				Logger.Infof("Using previous version %s", lVersion)

				By("Prepare a node pool with optional-1 version with manual upgrade")
				nodePoolManualName := common.GenerateRandomName("np-67414", 2)
				output, err := machinePoolService.CreateMachinePool(clusterID, nodePoolManualName,
					"--replicas", "2",
					"--version", lVersion)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						nodePoolManualName,
						clusterID))
				output, err = machinePoolUpgradeService.CreateManualUpgrade(clusterID, nodePoolManualName, "", "", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Upgrade successfully scheduled for the machine pool '%s' on cluster '%s'",
						nodePoolManualName,
						clusterID))

				By("Prepare a node pool with lower version with automatic upgrade")
				nodePoolAutoName := common.GenerateRandomName("np-67414", 2)
				output, err = machinePoolService.CreateMachinePool(clusterID, nodePoolAutoName,
					"--replicas", "2",
					"--version", lVersion)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						nodePoolAutoName,
						clusterID))
				output, err = machinePoolUpgradeService.CreateAutomaticUpgrade(clusterID, nodePoolAutoName, "2 5 * * *")
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"Upgrade successfully scheduled for the machine pool '%s' on cluster '%s'",
						nodePoolAutoName,
						clusterID))

				analyzeUpgrade := func(nodePoolName string, scheduleType string) {
					By(fmt.Sprintf("Describe node pool in json format (%s upgrade)", scheduleType))
					rosaClient.Runner.JsonFormat()
					jsonOutput, err := machinePoolService.DescribeMachinePool(clusterID, nodePoolName)
					Expect(err).To(BeNil())
					rosaClient.Runner.UnsetFormat()
					jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()
					var npAvailableUpgrades []string
					for _, value := range jsonData.DigObject("version", "available_upgrades").([]interface{}) {
						npAvailableUpgrades = append(npAvailableUpgrades, fmt.Sprint(value))
					}

					By(fmt.Sprintf("Describe node pool upgrade (%s upgrade)", scheduleType))
					npuDesc, err := machinePoolUpgradeService.DescribeAndReflectUpgrade(clusterID, nodePoolName)
					Expect(err).ToNot(HaveOccurred())
					Expect(npuDesc.ScheduleType).To(Equal(scheduleType))
					Expect(npuDesc.NextRun).ToNot(BeNil())
					nextRunDT, err := time.Parse("2006-01-02 15:04 UTC", npuDesc.NextRun)
					Expect(err).ToNot(HaveOccurred())
					Expect(nextRunDT.After(currentDateTimeUTC)).To(BeTrue())
					Expect(npuDesc.UpgradeState).To(BeElementOf("pending", "scheduled"))
					Expect(npuDesc.Version).To(Equal(upgradeVersion))

					nextRun := npuDesc.NextRun

					By(fmt.Sprintf("Describe node pool should contain upgrade (%s upgrade)", scheduleType))
					npDesc, err := machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
					Expect(err).ToNot(HaveOccurred())
					Expect(npDesc.ScheduledUpgrade).To(ContainSubstring(upgradeVersion))
					Expect(npDesc.ScheduledUpgrade).To(ContainSubstring(nextRun))
					Expect(npDesc.ScheduledUpgrade).To(Or(ContainSubstring("pending"), ContainSubstring("scheduled")))

					By(fmt.Sprintf("List the upgrade policies (%s upgrade)", scheduleType))
					npuList, err := machinePoolUpgradeService.ListAndReflectUpgrades(clusterID, nodePoolName)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(npuList.MachinePoolUpgrades)).To(Equal(len(npAvailableUpgrades)))
					var upgradeMPU rosacli.MachinePoolUpgrade
					for _, mpu := range npuList.MachinePoolUpgrades {
						Expect(mpu.Version).To(BeElementOf(npAvailableUpgrades))
						if mpu.Version == upgradeVersion {
							upgradeMPU = mpu
						}
					}
					Expect(upgradeMPU.Notes).To(Or(ContainSubstring("pending"), ContainSubstring("scheduled")))
					Expect(upgradeMPU.Notes).To(ContainSubstring(nextRun))

					By(fmt.Sprintf("Delete the upgrade policy (%s upgrade)", scheduleType))
					output, err = machinePoolUpgradeService.DeleteUpgrade(clusterID, nodePoolName)
					Expect(err).ToNot(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).Should(
						ContainSubstring("Successfully canceled scheduled upgrade for machine pool '%s' for cluster '%s'",
							nodePoolName, clusterID))
				}

				analyzeUpgrade(nodePoolManualName, "manual")
				analyzeUpgrade(nodePoolAutoName, "automatic")
			})

		It("can upgrade machinepool of hosted cluster - [id:67412]", labels.Critical, labels.Runtime.Upgrade,
			func() {

				By("Find a version has available upgrade versions")
				clusterVersionInfo, err := clusterService.GetClusterVersion(clusterID)
				Expect(err).ToNot(HaveOccurred())
				clusterVersion := clusterVersionInfo.RawID
				clusterChannelGroup := clusterVersionInfo.ChannelGroup
				versionList, err := versionService.ListAndReflectVersions(clusterChannelGroup, false)
				Expect(err).ToNot(HaveOccurred())

				upgradableVersion, err := versionList.FindZStreamUpgradableVersion(clusterVersion, 1)
				Expect(err).ToNot(HaveOccurred())

				By("Create a machinepool with the upgradable version")
				nodePoolAutoName := common.GenerateRandomName("np-67412", 2)
				_, err = machinePoolService.CreateMachinePool(clusterID, nodePoolAutoName,
					"--replicas", "0",
					"--version", upgradableVersion.Version)
				Expect(err).ToNot(HaveOccurred())
				availableUpgradeVersions := common.ParseCommaSeparatedStrings(upgradableVersion.AvailableUpgrades)

				By("Schedule the upgrade without time scheduled")
				_, err = machinePoolUpgradeService.CreateManualUpgrade(
					clusterID,
					nodePoolAutoName,
					availableUpgradeVersions[0],
					"",
					"")
				Expect(err).ToNot(HaveOccurred())

				By("Wait for the upgrade finished ")
				err = machinePoolUpgradeService.WaitForUpgradeFinished(clusterID, nodePoolAutoName, 30)
				Expect(err).ToNot(HaveOccurred())

				By("Verify the upgrade result")
				npDescription, err := machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolAutoName)
				Expect(err).ToNot(HaveOccurred())
				Expect(npDescription.Version).To(Equal(availableUpgradeVersions[0]))
				Expect(npDescription.ScheduledUpgrade).To(BeEmpty())
			})

		It("create/edit nodepool with node_drain_grace_period to HCP cluster via ROSA cli can work well - [id:72715]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("check help message for create/edit machinepool")
				help, err := machinePoolService.RetrieveHelpForCreate()
				Expect(err).ToNot(HaveOccurred())
				Expect(help.String()).To(ContainSubstring("--node-drain-grace-period"))
				help, err = machinePoolService.RetrieveHelpForEdit()
				Expect(err).ToNot(HaveOccurred())
				Expect(help.String()).To(ContainSubstring("--node-drain-grace-period"))

				By("Create nodepool with different node-drain-grace-periods")
				nodeDrainGracePeriodsReqAndRes := []map[string]string{{
					"20":         "20 minutes",
					"20 hours":   "1200 minutes",
					"20 minutes": "20 minutes",
				}}
				for _, nodnodeDrainGracePeriod := range nodeDrainGracePeriodsReqAndRes {
					for req, res := range nodnodeDrainGracePeriod {

						nodePoolName := common.GenerateRandomName("np-72715", 2)
						By("Create nodepool with node-drain-grace-period")
						_, err = machinePoolService.CreateMachinePool(clusterID, nodePoolName,
							"--replicas", "3",
							"--node-drain-grace-period", req,
						)
						Expect(err).ToNot(HaveOccurred())

						By("Describe nodepool")
						output, err := machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
						Expect(err).ToNot(HaveOccurred())
						Expect(output.NodeDrainGracePeriod).To(Equal(res))
					}
				}

				By("Create nodepool without node-drain-grace-period")
				nodePoolName := common.GenerateRandomName("np-72715", 3)
				_, err = machinePoolService.CreateMachinePool(clusterID, nodePoolName,
					"--replicas", "3",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Describe cluster in json format")
				rosaClient.Runner.JsonFormat()
				jsonOutput, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()
				value := jsonData.DigFloat("node_drain_grace_period", "value")
				nodeDrainGracePeriodForCluster := strconv.FormatFloat(value, 'f', -1, 64)

				By("Describe nodepool")
				output, err := machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
				Expect(err).ToNot(HaveOccurred())
				if nodeDrainGracePeriodForCluster == "0" {
					Expect(output.NodeDrainGracePeriod).To(Equal(""))
				} else {
					Expect(output.NodeDrainGracePeriod).To(Equal(nodeDrainGracePeriodForCluster))
				}

				By("Edit nodepool with different node-drain-grace-periods")
				nodeDrainGracePeriodsReqAndRes = []map[string]string{{
					"10":         "10 minutes",
					"10 hours":   "600 minutes",
					"10 minutes": "10 minutes",
				}}
				for _, nodnodeDrainGracePeriod := range nodeDrainGracePeriodsReqAndRes {
					for req, res := range nodnodeDrainGracePeriod {

						By("Edit nodepool with node-drain-grace-period")
						_, err = machinePoolService.EditMachinePool(clusterID, nodePoolName,
							"--node-drain-grace-period", req,
							"--replicas", "3",
						)
						Expect(err).ToNot(HaveOccurred())

						By("Describe nodepool")
						output, err := machinePoolService.DescribeAndReflectNodePool(clusterID, nodePoolName)
						Expect(err).ToNot(HaveOccurred())
						Expect(output.NodeDrainGracePeriod).To(Equal(res))
					}
				}
			})

		It("validations will work for editing machinepool via rosa cli - [id:73391]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				nonExistingMachinepoolName := common.GenerateRandomName("mp-fake", 2)
				machinepoolName := common.GenerateRandomName("mp-73391", 2)

				By("Try to edit machinepool with the name not present in cluster")
				output, err := machinePoolService.EditMachinePool(clusterID, nonExistingMachinepoolName, "--replicas", "3")
				Expect(err).To(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Machine pool '%s' does not exist for hosted cluster '%s'",
						nonExistingMachinepoolName,
						clusterID))

				By("Create a new machinepool to the cluster")
				output, err = machinePoolService.CreateMachinePool(clusterID, machinepoolName, "--replicas", "3")
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						machinepoolName,
						clusterID))

				By("Try to edit the replicas of the machinepool with negative value")
				output, err = machinePoolService.EditMachinePool(clusterID, machinepoolName, "--replicas", "-9")
				Expect(err).To(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"The number of machine pool replicas needs to be a non-negative integer"))

				By("Try to edit the machinepool with --min-replicas flag when autoscaling is disabled for the machinepool.")
				output, err = machinePoolService.EditMachinePool(clusterID, machinepoolName, "--min-replicas", "2")
				Expect(err).To(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Failed to get autoscaling or replicas: 'Autoscaling is not enabled on machine pool '%s'. "+
							"can't set min or max replicas'",
						machinepoolName))

				By("Try to edit the machinepool with --max-replicas flag when autoscaling is disabled for the machinepool.")
				output, err = machinePoolService.EditMachinePool(clusterID, machinepoolName, "--max-replicas", "5")
				Expect(err).To(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Failed to get autoscaling or replicas: 'Autoscaling is not enabled on machine pool '%s'. "+
							"can't set min or max replicas'",
						machinepoolName))

				By("Edit the machinepool to autoscaling mode.")
				output, err = machinePoolService.EditMachinePool(
					clusterID,
					machinepoolName,
					"--enable-autoscaling",
					"--min-replicas", "2",
					"--max-replicas", "6")
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Updated machine pool '%s' on hosted cluster '%s'",
						machinepoolName,
						clusterID))

				By("Try to edit machinepool with negative min_replicas value.")
				output, err = machinePoolService.EditMachinePool(clusterID, machinepoolName, "--min-replicas", "-3")
				Expect(err).To(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"The number of machine pool min-replicas needs to be greater than zero"))

				By("Try to edit machinepool with --replicas flag when the autoscaling is enabled for the machinepool.")
				output, err = machinePoolService.EditMachinePool(clusterID, machinepoolName, "--replicas", "3")
				Expect(err).To(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Failed to get autoscaling or replicas: 'Autoscaling enabled on machine pool '%s'. can't set replicas'",
						machinepoolName))
			})

		It("create/describe machinepool with user tags for HCP - [id:73492]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Get the Organization Id")
				rosaClient.Runner.JsonFormat()
				userInfo, err := rosaClient.OCMResource.UserInfo()
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				organizationID := userInfo.OCMOrganizationID

				By("Get OCM Env")
				OCMEnv := common.ReadENVWithDefaultValue("OCM_LOGIN_ENV", "staging")

				By("Get the cluster informations")
				rosaClient.Runner.JsonFormat()
				jsonOutput, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()
				clusterName := jsonData.DigString("display_name")
				clusterProductID := jsonData.DigString("product", "id")
				var clusterNamePrefix string
				if jsonData.DigString("domain_prefix") != "" {
					clusterNamePrefix = jsonData.DigString("domain_prefix")
				} else {
					clusterNamePrefix = clusterName
				}
				clusterTagsString := jsonData.DigString("aws", "tags")
				clusterTags := common.ParseTagsFronJsonOutput(clusterTagsString)

				By("Get the managed tags for the nodepool")
				var managedTags = func(npID string) map[string]interface{} {
					npLabelName := clusterNamePrefix + "-" + npID
					managedTags := map[string]interface{}{
						"api.openshift.com/environment":         OCMEnv,
						"api.openshift.com/id":                  clusterID,
						"api.openshift.com/legal-entity-id":     organizationID,
						"api.openshift.com/name":                clusterName,
						"api.openshift.com/nodepool-hypershift": npLabelName,
						"api.openshift.com/nodepool-ocm":        npID,
						"red-hat-clustertype":                   clusterProductID,
						"red-hat-managed":                       "true",
					}
					return managedTags
				}

				By("Create a machinepool without tags to the cluster")
				machinePoolName_1 := common.GenerateRandomName("np-73492", 2)
				requiredTags := managedTags(machinePoolName_1)
				if len(clusterTags) > 0 {
					By("Attach cluster AWS tags")
					for k, v := range clusterTags {
						requiredTags[k] = v
					}
				}
				output, err := machinePoolService.CreateMachinePool(clusterID, machinePoolName_1, "--replicas", "3")
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						machinePoolName_1,
						clusterID))

				By("Describe the machinepool in json format")
				rosaClient.Runner.JsonFormat()
				jsonOutput, err = machinePoolService.DescribeMachinePool(clusterID, machinePoolName_1)
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				jsonData = rosaClient.Parser.JsonData.Input(jsonOutput).Parse()
				tagsString := jsonData.DigString("aws_node_pool", "tags")
				tags := common.ParseTagsFronJsonOutput(tagsString)
				for k, v := range requiredTags {
					Expect(tags[k]).To(Equal(v))
				}

				By("Create a machinepool with tags to the cluster")
				machinePoolName_2 := common.GenerateRandomName("mp-73492-1", 2)
				tagsReq := "foo:bar, testKey:testValue"
				tagsRequestMap := map[string]interface{}{
					"foo":     "bar",
					"testKey": "testValue",
				}
				requiredTags = managedTags(machinePoolName_2)
				if len(clusterTags) > 0 {
					By("Attach cluster AWS tags")
					for k, v := range clusterTags {
						requiredTags[k] = v
					}
				}
				output, err = machinePoolService.CreateMachinePool(
					clusterID,
					machinePoolName_2,
					"--replicas", "3",
					"--tags", tagsReq)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						machinePoolName_2,
						clusterID))

				By("Describe the machinepool in json format")
				rosaClient.Runner.JsonFormat()
				jsonOutput, err = machinePoolService.DescribeMachinePool(clusterID, machinePoolName_2)
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				jsonData = rosaClient.Parser.JsonData.Input(jsonOutput).Parse()
				tagsString = jsonData.DigString("aws_node_pool", "tags")
				tags = common.ParseTagsFronJsonOutput(tagsString)
				for k, v := range requiredTags {
					Expect(tags[k]).To(Equal(v))
				}
				for k, v := range tagsRequestMap {
					Expect(tags[k]).To(Equal(v))
				}

				By("Create machinepool with invalid tags")
				machinePoolName_3 := common.GenerateRandomName("mp-73492-2", 2)
				output, err = machinePoolService.CreateMachinePool(
					clusterID,
					machinePoolName_3,
					"--replicas", "3",
					"--tags", "#.bar")
				Expect(err).To(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"ERR: invalid tag format for tag '[#.bar]'. Expected tag format: 'key value'"))
			})

		It("create/edit/describe maxunavailable/maxsurge for HCP nodepools - [id:74387]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				By("Retrieve help for create/edit machinepool")
				output, err := machinePoolService.RetrieveHelpForCreate()
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("--max-surge"))
				Expect(output.String()).To(ContainSubstring("--max-unavailable"))

				output, err = machinePoolService.RetrieveHelpForEdit()
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("--max-surge"))
				Expect(output.String()).To(ContainSubstring("--max-unavailable"))

				reqBody := []map[string]string{
					{
						"max surge":       "5%",
						"max unavailable": "10%",
					},
					{
						"max surge":       "3",
						"max unavailable": "2",
					},
					{
						"max surge":       "",
						"max unavailable": "",
					},
					{
						"max surge":       "0%",
						"max unavailable": "10%",
					},
					{
						"max surge":       "10%",
						"max unavailable": "0%",
					},
					{
						"max surge":       "100%",
						"max unavailable": "10%",
					},
					{
						"max surge":       "10%",
						"max unavailable": "100%",
					},
					{
						"max surge":       "0",
						"max unavailable": "1",
					},
					{
						"max surge":       "1",
						"max unavailable": "0",
					},
				}

				for _, flags := range reqBody {

					By("Create nodepool with max-surge/max-unavailable set with different values")
					machinePoolName := common.GenerateRandomName("ocp-74387", 2)
					output, err = machinePoolService.CreateMachinePool(
						clusterID,
						machinePoolName,
						"--replicas", "3",
						"--max-surge", flags["max surge"],
						"--max-unavailable", flags["max unavailable"])
					Expect(err).ToNot(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(
							"Machine pool '%s' created successfully on hosted cluster '%s'",
							machinePoolName,
							clusterID))

					By("Describe the nodepool to see max surge and max unavailable is set correctly")
					res, err := machinePoolService.DescribeAndReflectNodePool(clusterID, machinePoolName)
					Expect(err).ToNot(HaveOccurred())
					Expect(res.ManagementUpgrade[0]["Type"]).To(Equal("Replace"))
					if flags["max surge"] == "" && flags["max unavailable"] == "" {
						Expect(res.ManagementUpgrade[1]["Max surge"]).To(Equal("1"))
						Expect(res.ManagementUpgrade[2]["Max unavailable"]).To(Equal("0"))
					} else {
						Expect(res.ManagementUpgrade[1]["Max surge"]).To(Equal(flags["max surge"]))
						Expect(res.ManagementUpgrade[2]["Max unavailable"]).To(Equal(flags["max unavailable"]))
					}
				}

				By("Create a nodepool with just max surge set")
				machinePool_Name := common.GenerateRandomName("ocp-74387", 2)
				output, err = machinePoolService.CreateMachinePool(
					clusterID,
					machinePool_Name,
					"--replicas", "3",
					"--max-surge", "2")
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						machinePool_Name,
						clusterID))

				By("Describe the nodepool to see max surge and max unavailable is set correctly")
				out, err := machinePoolService.DescribeAndReflectNodePool(clusterID, machinePool_Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(out.ManagementUpgrade[0]["Type"]).To(Equal("Replace"))
				Expect(out.ManagementUpgrade[1]["Max surge"]).To(Equal("2"))
				Expect(out.ManagementUpgrade[2]["Max unavailable"]).To(Equal("0"))

				By("Create a nodepool with just max unavailable set")
				machinePool_Name = common.GenerateRandomName("ocp-74387", 2)
				output, err = machinePoolService.CreateMachinePool(
					clusterID,
					machinePool_Name,
					"--replicas", "3",
					"--max-unavailable", "2")
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						machinePool_Name,
						clusterID))

				By("Describe the nodepool to see max surge and max unavailable is set correctly")
				out, err = machinePoolService.DescribeAndReflectNodePool(clusterID, machinePool_Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(out.ManagementUpgrade[0]["Type"]).To(Equal("Replace"))
				Expect(out.ManagementUpgrade[1]["Max surge"]).To(Equal("1"))
				Expect(out.ManagementUpgrade[2]["Max unavailable"]).To(Equal("2"))

				By("Get a nodepool to edit")
				res, err := machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.NodePools).ToNot(BeNil())
				var machinePoolName string
				for _, nodepool := range res.NodePools {
					if !strings.Contains(nodepool.ID, "workers") {
						machinePoolName = nodepool.ID
						break
					}
				}

				for _, flags := range reqBody {

					By("Describe the nodepool to see max surge and max unavailable prev value")
					out, err := machinePoolService.DescribeAndReflectNodePool(clusterID, machinePoolName)
					Expect(err).ToNot(HaveOccurred())

					By("Edit nodepool with max-surge/max-unavailable set with different values")
					output, err = machinePoolService.EditMachinePool(
						clusterID,
						machinePoolName,
						"--max-surge", flags["max surge"],
						"--max-unavailable", flags["max unavailable"])
					Expect(err).ToNot(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(
							"Updated machine pool '%s' on hosted cluster '%s'",
							machinePoolName,
							clusterID))

					By("Describe the nodepool to see max surge and max unavailable is set correctly")
					res, err := machinePoolService.DescribeAndReflectNodePool(clusterID, machinePoolName)
					Expect(err).ToNot(HaveOccurred())
					Expect(res.ManagementUpgrade[0]["Type"]).To(Equal("Replace"))
					if flags["max surge"] == "" && flags["max unavailable"] == "" {
						Expect(res.ManagementUpgrade[1]["Max surge"]).
							To(
								Equal(out.ManagementUpgrade[1]["Max surge"]))
						Expect(res.ManagementUpgrade[2]["Max unavailable"]).
							To(
								Equal(out.ManagementUpgrade[2]["Max unavailable"]))
					} else {
						Expect(res.ManagementUpgrade[1]["Max surge"]).
							To(
								Equal(flags["max surge"]))
						Expect(res.ManagementUpgrade[2]["Max unavailable"]).
							To(
								Equal(flags["max unavailable"]))
					}
				}

				By("Edit a nodepool with just max surge set")
				output, err = machinePoolService.EditMachinePool(
					clusterID,
					machinePoolName,
					"--max-surge", "7")
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Updated machine pool '%s' on hosted cluster '%s'",
						machinePoolName,
						clusterID))

				By("Describe the nodepool to see max surge and max unavailable is set correctly")
				out, err = machinePoolService.DescribeAndReflectNodePool(clusterID, machinePoolName)
				Expect(err).ToNot(HaveOccurred())
				Expect(out.ManagementUpgrade[1]["Max surge"]).To(Equal("7"))

				By("Edit a nodepool with just max unavailable set")
				output, err = machinePoolService.EditMachinePool(
					clusterID,
					machinePoolName,
					"--max-unavailable", "7")
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(ContainSubstring(
						"Updated machine pool '%s' on hosted cluster '%s'",
						machinePoolName,
						clusterID))

				By("Describe the nodepool to see max surge and max unavailable is set correctly")
				out, err = machinePoolService.DescribeAndReflectNodePool(clusterID, machinePoolName)
				Expect(err).ToNot(HaveOccurred())
				Expect(out.ManagementUpgrade[2]["Max unavailable"]).To(Equal("7"))
			})

		It("validation for create/edit HCP nodepool with maxunavailable/maxsurge - [id:74430]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				rangeofNumbers := "must be between 0 and 100"
				parseMsg := "machine pool to hosted cluster '%s': Failed to parse percentage "
				attributeMsg := "machine pool to hosted cluster '%s': The value of attribute "
				bothMsg := "'management_upgrade.max_unavailable' and 'management_upgrade.max_surge' "
				eitherMsg := "'management_upgrade.max_unavailable' or 'management_upgrade.max_surge', "
				createFailMsg := "ERR: Failed to add "
				zeroMsg := "could be zero, not both"
				sameUnitMsg := "must both use the same units (absolute value or percentage)"
				integerMsg := "'1.1' to integer"

				reqCreateBody := []map[string]string{
					{
						"max surge":       "0",
						"max unavailable": "0",
						"errMsg": fmt.Sprintf(createFailMsg+
							"machine pool to hosted cluster '%s': The value of only one attribute, "+
							eitherMsg+
							zeroMsg,
							clusterID),
					},
					{
						"max surge":       "0%",
						"max unavailable": "0%",
						"errMsg": fmt.Sprintf(createFailMsg+
							"machine pool to hosted cluster '%s': The value of only one attribute, "+
							eitherMsg+
							zeroMsg,
							clusterID),
					},
					{
						"max surge":       "0",
						"max unavailable": "1%",
						"errMsg": fmt.Sprintf(createFailMsg+
							"machine pool to hosted cluster '%s': Attribute "+
							bothMsg+
							sameUnitMsg,
							clusterID),
					},
					{
						"max surge":       "1%",
						"max unavailable": "0",
						"errMsg": fmt.Sprintf(createFailMsg+
							"machine pool to hosted cluster '%s': Attribute "+
							bothMsg+
							sameUnitMsg,
							clusterID),
					},
					{
						"max surge":       "-1",
						"max unavailable": "1",
						"errMsg": fmt.Sprintf(createFailMsg+
							attributeMsg+
							"'management_upgrade.max_surge' cannot be a negative integer",
							clusterID),
					},
					{
						"max surge":       "1",
						"max unavailable": "-1",
						"errMsg": fmt.Sprintf(createFailMsg+
							attributeMsg+
							"'management_upgrade.max_unavailable' cannot be a negative integer",
							clusterID),
					},
					{
						"max surge":       "0%",
						"max unavailable": "-1%",
						"errMsg": fmt.Sprintf(createFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_unavailable': Value -1 "+
							rangeofNumbers,
							clusterID),
					},
					{
						"max surge":       "-1%",
						"max unavailable": "0%",
						"errMsg": fmt.Sprintf(createFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_surge': Value -1 "+
							rangeofNumbers,
							clusterID),
					},
					{
						"max surge":       "0%",
						"max unavailable": "101%",
						"errMsg": fmt.Sprintf(createFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_unavailable': Value 101 "+
							rangeofNumbers,
							clusterID),
					},
					{
						"max surge":       "101%",
						"max unavailable": "0%",
						"errMsg": fmt.Sprintf(createFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_surge': Value 101 "+
							rangeofNumbers,
							clusterID),
					},
					{
						"max surge":       "0%",
						"max unavailable": "1.1%",
						"errMsg": fmt.Sprintf(createFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_unavailable': Error converting "+
							integerMsg,
							clusterID),
					},
					{
						"max surge":       "1.1%",
						"max unavailable": "0%",
						"errMsg": fmt.Sprintf(createFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_surge': Error converting "+
							integerMsg,
							clusterID),
					},
					{
						"max surge":       "1.1",
						"max unavailable": "0",
						"errMsg": fmt.Sprintf(createFailMsg+
							attributeMsg+
							"'management_upgrade.max_surge' must be an integer",
							clusterID),
					},
					{
						"max surge":       "0",
						"max unavailable": "1.1",
						"errMsg": fmt.Sprintf(createFailMsg+
							attributeMsg+
							"'management_upgrade.max_unavailable' must be an integer",
							clusterID),
					},
				}

				for _, flags := range reqCreateBody {

					By("Create nodepool with max-surge/max-unavailable set with different inavlid values")
					machinePoolName := common.GenerateRandomName("ocp-74387", 2)
					output, err := machinePoolService.CreateMachinePool(
						clusterID,
						machinePoolName,
						"--replicas", "3",
						"--max-surge", flags["max surge"],
						"--max-unavailable", flags["max unavailable"])
					Expect(err).To(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(flags["errMsg"]))
				}

				By("Create a nodepool to check edit validation")
				machinePoolName := common.GenerateRandomName("ocp-74430", 2)
				res, err := machinePoolService.CreateMachinePool(clusterID, machinePoolName, "--replicas", "3")
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.DeleteMachinePool(clusterID, machinePoolName)
				Expect(rosaClient.Parser.TextData.Input(res).Parse().Tip()).
					To(ContainSubstring(
						"Machine pool '%s' created successfully on hosted cluster '%s'",
						machinePoolName,
						clusterID))

				parseMsg = "machine pool '%s' on hosted cluster '%s': Failed to parse percentage "
				attributeMsg = "machine pool '%s' on hosted cluster '%s': The value of attribute "
				editFailMsg := "ERR: Failed to update "
				reqEditBody := []map[string]string{
					{
						"max surge":       "0",
						"max unavailable": "0",
						"errMsg": fmt.Sprintf(editFailMsg+
							"machine pool '%s' on hosted cluster '%s': The value of only one attribute, "+
							eitherMsg+
							zeroMsg,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "0%",
						"max unavailable": "0%",
						"errMsg": fmt.Sprintf(editFailMsg+
							"machine pool '%s' on hosted cluster '%s': The value of only one attribute, "+
							eitherMsg+
							zeroMsg,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "0",
						"max unavailable": "1%",
						"errMsg": fmt.Sprintf(editFailMsg+
							"machine pool '%s' on hosted cluster '%s': Attribute "+
							bothMsg+
							sameUnitMsg,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "1%",
						"max unavailable": "0",
						"errMsg": fmt.Sprintf(editFailMsg+
							"machine pool '%s' on hosted cluster '%s': Attribute "+
							bothMsg+
							sameUnitMsg,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "-1",
						"max unavailable": "1",
						"errMsg": fmt.Sprintf(editFailMsg+
							attributeMsg+
							"'management_upgrade.max_surge' cannot be a negative integer",
							machinePoolName, clusterID),
					},
					{
						"max surge":       "1",
						"max unavailable": "-1",
						"errMsg": fmt.Sprintf(editFailMsg+
							attributeMsg+
							"'management_upgrade.max_unavailable' cannot be a negative integer",
							machinePoolName, clusterID),
					},
					{
						"max surge":       "0%",
						"max unavailable": "-1%",
						"errMsg": fmt.Sprintf(editFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_unavailable': Value -1 "+
							rangeofNumbers,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "-1%",
						"max unavailable": "0%",
						"errMsg": fmt.Sprintf(editFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_surge': Value -1 "+
							rangeofNumbers,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "0%",
						"max unavailable": "101%",
						"errMsg": fmt.Sprintf(editFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_unavailable': Value 101 "+
							rangeofNumbers,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "101%",
						"max unavailable": "0%",
						"errMsg": fmt.Sprintf(editFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_surge': Value 101 "+
							rangeofNumbers,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "0%",
						"max unavailable": "1.1%",
						"errMsg": fmt.Sprintf(editFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_unavailable': Error converting "+
							integerMsg,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "1.1%",
						"max unavailable": "0%",
						"errMsg": fmt.Sprintf(editFailMsg+
							parseMsg+
							"value for attribute 'management_upgrade.max_surge': Error converting "+
							integerMsg,
							machinePoolName, clusterID),
					},
					{
						"max surge":       "1.1",
						"max unavailable": "0",
						"errMsg": fmt.Sprintf(editFailMsg+
							attributeMsg+
							"'management_upgrade.max_surge' must be an integer",
							machinePoolName, clusterID),
					},
					{
						"max surge":       "0",
						"max unavailable": "1.1",
						"errMsg": fmt.Sprintf(editFailMsg+
							attributeMsg+
							"'management_upgrade.max_unavailable' must be an integer",
							machinePoolName, clusterID),
					},
				}

				for _, flags := range reqEditBody {

					By("Edit nodepool with max-surge/max-unavailable set with different invalid values")
					output, err := machinePoolService.EditMachinePool(
						clusterID,
						machinePoolName,
						"--max-surge", flags["max surge"],
						"--max-unavailable", flags["max unavailable"])
					Expect(err).To(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(flags["errMsg"]))
				}
			})

		It("can enable/disable/update autoscaling - [id:59430]", func() {
			By("Check help message in edit machinepool")
			output, err := machinePoolService.EditMachinePool(clusterID, "", "-h")
			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).Should(
				MatchRegexp(`--enable-autoscaling[\s\t]*Enable autoscaling for the machine pool.`))
			Expect(output.String()).Should(
				MatchRegexp(`--max-replicas int[\s\t]*Maximum number of machines for the machine pool.`))
			Expect(output.String()).Should(
				MatchRegexp(`--min-replicas int[\s\t]*Minimum number of machines for the machine pool.`))

			By("Prepare a machinepool")
			mpName := common.GenerateRandomName("np-59430", 2)
			_, err = machinePoolService.CreateMachinePool(
				clusterID, mpName,
				"--replicas", "0",
			)
			Expect(err).ToNot(HaveOccurred())
			defer machinePoolService.DeleteMachinePool(clusterID, mpName)

			By("Update the machinepool to autoscaling")
			output, err = machinePoolService.EditMachinePool(
				clusterID, mpName,
				"--enable-autoscaling",
				"--min-replicas", "1",
				"--max-replicas", "3",
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).Should(ContainSubstring("Updated machine pool "))

			By("Describe the machinepool and check the value")
			mpDescription, err := machinePoolService.DescribeMachinePool(clusterID, mpName)
			Expect(err).ToNot(HaveOccurred())
			Expect(mpDescription.String()).To(MatchRegexp(`Autoscaling:[\s\t]*Yes`))
			Expect(mpDescription.String()).To(
				ContainSubstring(`Min replicas: 1`))
			Expect(mpDescription.String()).To(
				ContainSubstring(`Max replicas: 3`))

			By("Edit the machinepool and min-replicas to another value")
			output, err = machinePoolService.EditMachinePool(
				clusterID, mpName,
				"--min-replicas", "2",
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).Should(ContainSubstring("Updated machine pool "))

			By("Describe the machinepool and check the value")
			mpDescription, err = machinePoolService.DescribeMachinePool(clusterID, mpName)
			Expect(err).ToNot(HaveOccurred())
			Expect(mpDescription.String()).To(MatchRegexp(`Autoscaling:[\s\t]*Yes`))
			Expect(mpDescription.String()).To(
				ContainSubstring(`Min replicas: 2`))
			Expect(mpDescription.String()).To(
				ContainSubstring(`Max replicas: 3`))

			By("Disable the autoscaling")
			_, err = machinePoolService.EditMachinePool(
				clusterID, mpName,
				"--enable-autoscaling=false",
				"--replicas", "0",
			)
			Expect(err).ToNot(HaveOccurred())

			mpDescription, err = machinePoolService.DescribeMachinePool(clusterID, mpName)
			Expect(err).ToNot(HaveOccurred())
			Expect(mpDescription.String()).To(MatchRegexp(`Autoscaling:[\s\t]*No`))
			Expect(mpDescription.String()).ToNot(
				ContainSubstring(`Min replicas`))
			Expect(mpDescription.String()).ToNot(
				ContainSubstring(`Max replicas`))
			Expect(mpDescription.String()).Should(MatchRegexp(`Desired replicas:[\s\t]*0`))
		})
	})
