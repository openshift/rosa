package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Create machinepool",
	labels.Day2,
	labels.FeatureMachinepool,
	labels.NonHCPCluster,
	func() {
		defer GinkgoRecover()
		var (
			clusterID          string
			rosaClient         *rosacli.Client
			machinePoolService rosacli.MachinePoolService
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			machinePoolService = rosaClient.MachinePool

		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("can create machinepool with volume size set - [id:66872]",
			labels.Critical,
			func() {
				mpID := "mp-66359"
				expectedDiskSize := "186 GiB" // it is 200GB
				machineType := "r5.xlarge"

				By("Create a machinepool with the disk size")
				_, err := machinePoolService.CreateMachinePool(clusterID, mpID,
					"--replicas", "0",
					"--disk-size", "200GB",
					"--instance-type", machineType,
				)
				Expect(err).ToNot(HaveOccurred())

				By("Check the machinepool list")
				output, err := machinePoolService.ListMachinePool(clusterID)
				Expect(err).ToNot(HaveOccurred())

				mplist, err := machinePoolService.ReflectMachinePoolList(output)
				Expect(err).ToNot(HaveOccurred())

				mp := mplist.Machinepool(mpID)
				Expect(mp).ToNot(BeNil(), "machine pool is not found for the cluster")
				Expect(mp.DiskSize).To(Equal(expectedDiskSize))
				Expect(mp.InstanceType).To(Equal(machineType))

				By("Check the default worker pool description")
				output, err = machinePoolService.DescribeMachinePool(clusterID, mpID)
				Expect(err).ToNot(HaveOccurred())
				mpD, err := machinePoolService.ReflectMachinePoolDescription(output)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpD.DiskSize).To(Equal(expectedDiskSize))

				By("Create another machinepool with volume size 0.5TiB")
				mpID = "mp-66359-2"
				expectedDiskSize = "512 GiB" // it is 0.5TiB
				machineType = "m5.2xlarge"
				_, err = machinePoolService.CreateMachinePool(clusterID, mpID,
					"--replicas", "0",
					"--disk-size", "0.5TiB",
					"--instance-type", machineType,
				)

				Expect(err).ToNot(HaveOccurred())

				By("Check the machinepool list")
				output, err = machinePoolService.ListMachinePool(clusterID)
				Expect(err).ToNot(HaveOccurred())

				mplist, err = machinePoolService.ReflectMachinePoolList(output)
				Expect(err).ToNot(HaveOccurred())

				mp = mplist.Machinepool(mpID)
				Expect(mp).ToNot(BeNil(), "machine pool is not found for the cluster")
				Expect(mp.DiskSize).To(Equal(expectedDiskSize))
				Expect(mp.DiskSize).To(Equal(expectedDiskSize))
				Expect(mp.InstanceType).To(Equal(machineType))

			})

		It("can create spot machinepool - [id: 43251]",
			labels.High,
			func() {
				By("Create a spot machinepool on the cluster")
				machinePoolName := "spotmp"
				output, err := machinePoolService.CreateMachinePool(clusterID, machinePoolName, "--spot-max-price", "10.2", "--use-spot-instances",
					"--replicas", "0")
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Machine pool '%s' created successfully on cluster '%s'", machinePoolName, clusterID))

				By("Create another machinepool without spot instances")
				machinePoolName = "nospotmp"
				output, err = machinePoolService.CreateMachinePool(clusterID, machinePoolName, "--replicas", "0")
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Machine pool '%s' created successfully on cluster '%s'", machinePoolName, clusterID))

				By("Create another machinepool with use-spot-instances but no spot-max-price set")
				machinePoolName = "nopricemp"
				output, err = machinePoolService.CreateMachinePool(clusterID, machinePoolName, "--use-spot-instances", "--replicas", "0")
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Machine pool '%s' created successfully on cluster '%s'", machinePoolName, clusterID))

				By("Confirm list of machinepools contains all created machinepools with SpotInstance field set appropriately")
				output, err = machinePoolService.ListMachinePool(clusterID)
				Expect(err).To(BeNil())
				mpTab, err := machinePoolService.ReflectMachinePoolList(output)
				Expect(err).To(BeNil())
				for _, mp := range mpTab.MachinePools {
					switch mp.ID {
					case "spotmp":
						Expect(mp.SpotInstances).To(Equal("Yes (max $10.2)"))
					case "nospotmp":
						Expect(mp.SpotInstances).To(Equal("No"))
					case "nopricemp":
						Expect(mp.SpotInstances).To(Equal("Yes (on-demand)"))
					default:
						continue
					}
				}

			})
	})

var _ = Describe("Edit machinepool",
	labels.Day2,
	labels.FeatureMachinepool,
	labels.NonHCPCluster,
	func() {
		defer GinkgoRecover()
		var (
			clusterID          string
			rosaClient         *rosacli.Client
			machinePoolService rosacli.MachinePoolService
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			machinePoolService = rosaClient.MachinePool

		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("will succeed - [id:38838]", func() {
			By("Check help message")
			output, err := machinePoolService.EditMachinePool(clusterID, "", "-h")
			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).Should(ContainSubstring("machinepool, machinepools, machine-pool, machine-pools"))

			By("Create an additional machinepool")
			machinePoolName := "mp-38838"
			_, err = machinePoolService.CreateMachinePool(clusterID, machinePoolName,
				"--replicas", "3")
			Expect(err).ToNot(HaveOccurred())

			By("Edit the additional machinepool to autoscaling")
			_, err = machinePoolService.EditMachinePool(clusterID, machinePoolName,
				"--enable-autoscaling",
				"--min-replicas", "3",
				"--max-replicas", "3",
			)
			Expect(err).ToNot(HaveOccurred())

			By("Check the edited machinePool")
			mp, err := machinePoolService.DescribeAndReflectMachinePool(clusterID, machinePoolName)
			Expect(err).ToNot(HaveOccurred())
			Expect(mp.AutoScaling).To(Equal("Yes"))
			Expect(mp.Replicas).To(Equal("3-3"))

			By("Edit the the machinepool to min-replicas 0, taints and labels")
			taints := "k38838=v:NoSchedule,k38838-2=:NoExecute"
			labels := "test38838="
			_, err = machinePoolService.EditMachinePool(clusterID, machinePoolName,
				"--min-replicas", "0",
				"--taints", taints,
				"--labels", labels,
			)
			Expect(err).ToNot(HaveOccurred())
			mp, err = machinePoolService.DescribeAndReflectMachinePool(clusterID, machinePoolName)
			Expect(err).ToNot(HaveOccurred())
			Expect(mp.AutoScaling).To(Equal("Yes"))
			Expect(mp.Replicas).To(Equal("0-3"))
			Expect(mp.Labels).To(Equal(strings.Join(common.ParseCommaSeparatedStrings(labels), ", ")))
			Expect(mp.Taints).To(Equal(strings.Join(common.ParseCommaSeparatedStrings(taints), ", ")))
		})
	})
