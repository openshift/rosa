package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

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
			isHosted           bool
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			machinePoolService = rosaClient.MachinePool

			var err error
			isHosted, err = rosaClient.Cluster.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())

			if isHosted {
				Skip("This test case only work for classic cluster")
				return
			}
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
			labels.MigrationToVerify,
			labels.Exclude,
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
