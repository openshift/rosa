package e2e

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("Create machinepool",
	labels.Feature.Machinepool,
	func() {
		defer GinkgoRecover()
		var (
			clusterID              string
			rosaClient             *rosacli.Client
			machinePoolService     rosacli.MachinePoolService
			ocmResourceService     rosacli.OCMResourceService
			permissionsBoundaryArn string = "arn:aws:iam::aws:policy/AdministratorAccess"
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			machinePoolService = rosaClient.MachinePool
			ocmResourceService = rosaClient.OCMResource

			By("Skip testing if the cluster is not a Classic cluster")
			isHostedCP, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if isHostedCP {
				SkipNotClassic()
			}
		})

		AfterEach(func() {
			By("Clean remaining resources")
			rosaClient.CleanResources(clusterID)

		})

		It("can create machinepool with volume size set - [id:66872]",
			labels.Critical, labels.Runtime.Day2,
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

		It("List newly added instance-types - [id:73308]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("List the available instance-types and verify the presence of newly added instance-types")
				newlyAddedTypes := []string{"c7a.xlarge", "c7a.48xlarge", "c7a.metal-48xl", "r7a.xlarge", "r7a.48xlarge", "r7a.metal-48xl", "hpc6a.48xlarge", "hpc6id.32xlarge", "hpc7a.96xlarge", "c7i.48xlarge", "c7i.metal-24xl", "c7i.metal-48xl", "r7i.xlarge", "r7i.48xlarge"}
				availableMachineTypes, _, err := ocmResourceService.ListInstanceTypes()

				if err != nil {
					log.Logger.Errorf("Failed to fetch instance types: %v", err)
				} else {
					var availableMachineTypesIDs []string
					for _, it := range availableMachineTypes.InstanceTypesList {
						availableMachineTypesIDs = append(availableMachineTypesIDs, it.ID)
					}
					Expect(availableMachineTypesIDs).To(ContainElements(newlyAddedTypes))
				}
			})

		It("List instance-types with region flag - [id:72174]",
			labels.Low, labels.Runtime.Day2,
			func() {
				By("List the available instance-types with the region flag")
				typesList := []string{"dl1.24xlarge", "g4ad.16xlarge", "c5.xlarge"}
				region := "us-east-1"
				accountRolePrefix := fmt.Sprintf("QEAuto-accr72174-%s", time.Now().UTC().Format("20060102"))
				_, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accountRolePrefix,
					"--permissions-boundary", permissionsBoundaryArn,
					"-y")
				Expect(err).To(BeNil())
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				classicInstallerRoleArn := accountRoleList.InstallerRole(accountRolePrefix, false).RoleArn
				availableMachineTypes, _, err := ocmResourceService.ListInstanceTypesByRegion("--region", region, "--role-arn", classicInstallerRoleArn)

				if err != nil {
					log.Logger.Errorf("Failed to fetch instance types: %v", err)
				} else {
					var availableMachineTypesIDs []string
					for _, it := range availableMachineTypes.InstanceTypesList {
						availableMachineTypesIDs = append(availableMachineTypesIDs, it.ID)
					}
					Expect(availableMachineTypesIDs).To(ContainElements(typesList))
				}
			})

		It("can create spot machinepool - [id:43251]",
			labels.High, labels.Runtime.Day2,
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

		It("validate inputs for create spot machinepool - [id:43252]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Create a spot machinepool with negative price")
				machinePoolName := "spotmp"
				output, err := machinePoolService.CreateMachinePool(clusterID, machinePoolName, "--spot-max-price", "-10.2", "--use-spot-instances",
					"--replicas", "3")
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Spot max price must be positive"))

				By("Create a machinepool without spot instances, but with spot price")
				machinePoolName = "nospotmp"
				output, err = machinePoolService.CreateMachinePool(clusterID, machinePoolName, "--replicas", "3", "--spot-max-price", "10.2", "--use-spot-instances=false")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Can't set max price when not using spot instances"))
			})

		It("can create machinepool with tags - [id:73469]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check the help message of machinepool creation")
				out, err := machinePoolService.CreateMachinePool(clusterID, "mp-73469", "-h")
				Expect(err).ToNot(HaveOccurred(), out.String())
				Expect(out.String()).Should(ContainSubstring("--tags strings"))

				By("Create a machinepool with tags set")
				tags := []string{
					"test:testvalue",
					"test2:testValue/openshift",
				}
				out, err = machinePoolService.CreateMachinePool(clusterID, "mp-73469",
					"--replicas", "3",
					"--tags", strings.Join(tags, ","),
				)
				Expect(err).ToNot(HaveOccurred(), out.String())

				By("Describe the machinepool")
				description, err := machinePoolService.DescribeAndReflectMachinePool(clusterID, "mp-73469")
				Expect(err).ToNot(HaveOccurred(), out.String())

				for _, tag := range tags {
					Expect(description.Tags).Should(ContainSubstring(strings.Replace(tag, ":", "=", -1)))
				}

				By("Create with invalid tags")
				invalidTagMap := map[string]string{
					"invalidFmt": "invalid",
					"noTagValue": "notagvalue:",
					"noTagKey":   ":notagkey",
					"nonAscii":   "non-ascii:值",
				}
				for errorType, tag := range invalidTagMap {
					out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
						"--replicas", "3",
						"--tags", tag,
					)
					Expect(err).To(HaveOccurred())
					switch errorType {
					case "invalidFmt":
						Expect(out.String()).Should(ContainSubstring("invalid tag format for tag"))
					case "noTagValue":
						Expect(out.String()).Should(ContainSubstring("invalid tag format, tag key or tag value can not be empty"))
					case "noTagKey":
						Expect(out.String()).Should(ContainSubstring("invalid tag format, tag key or tag value can not be empty"))
					case "nonAscii":
						Expect(out.String()).Should(ContainSubstring("Invalid Machine Pool AWS tags"))
					}
				}
			})
	})

var _ = Describe("Edit machinepool",
	labels.Feature.Machinepool,
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

			By("Skip testing if the cluster is not a Classic cluster")
			isHostedCP, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if isHostedCP {
				SkipNotClassic()
			}

		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("will succeed - [id:38838]",
			labels.High, labels.Runtime.Day2,
			func() {
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
