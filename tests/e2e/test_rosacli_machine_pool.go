package e2e

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/common/constants"
	con "github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("Create machinepool",
	labels.Feature.Machinepool,
	func() {
		defer GinkgoRecover()
		var (
			clusterID          string
			rosaClient         *rosacli.Client
			machinePoolService rosacli.MachinePoolService
			ocmResourceService rosacli.OCMResourceService
			clusterConfig      *config.ClusterConfig
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

			By("Parse the cluster config")
			clusterConfig, err = config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			By("Clean remaining resources")
			rosaClient.CleanResources(clusterID)

		})

		It("can create machinepool with volume size set - [id:66872]",
			labels.Runtime.Day2,
			labels.Critical,
			func() {
				mpID := "mp-66872"
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
				mpID = "mp-66872-2"
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
			labels.Runtime.Day2,
			labels.Medium,
			func() {
				By("List the available instance-types and verify the presence of newly added instance-types")
				newlyAddedTypes := []string{
					"c7a.xlarge",
					"c7a.48xlarge",
					"c7a.metal-48xl",
					"r7a.xlarge",
					"r7a.48xlarge",
					"r7a.metal-48xl",
					"hpc6a.48xlarge",
					"hpc6id.32xlarge",
					"hpc7a.96xlarge",
					"c7i.48xlarge",
					"c7i.metal-24xl",
					"c7i.metal-48xl",
					"r7i.xlarge",
					"r7i.48xlarge"}
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

		It("can create spot machinepool - [id:43251]",
			labels.Runtime.Day2,
			labels.High,
			func() {
				By("Create a spot machinepool on the cluster")
				// nolint:goconst
				machinePoolName := "spotmp"
				output, err := machinePoolService.CreateMachinePool(
					clusterID,
					machinePoolName,
					"--spot-max-price", "10.2",
					"--use-spot-instances",
					"--replicas", "0")
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Machine pool '%s' created successfully on cluster '%s'",
						machinePoolName,
						clusterID))

				By("Create another machinepool without spot instances")
				// nolint:goconst
				machinePoolName = "nospotmp"
				output, err = machinePoolService.CreateMachinePool(
					clusterID,
					machinePoolName,
					"--replicas", "0")
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Machine pool '%s' created successfully on cluster '%s'",
						machinePoolName,
						clusterID))

				By("Create another machinepool with use-spot-instances but no spot-max-price set")
				machinePoolName = "nopricemp"
				output, err = machinePoolService.CreateMachinePool(
					clusterID,
					machinePoolName,
					"--use-spot-instances",
					"--replicas", "0")
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Machine pool '%s' created successfully on cluster '%s'",
						machinePoolName,
						clusterID))

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
			labels.Runtime.Day2,
			labels.Medium,
			func() {
				By("Create a spot machinepool with negative price")
				machinePoolName := "spotmp"
				output, err := machinePoolService.CreateMachinePool(
					clusterID,
					machinePoolName,
					"--spot-max-price", "-10.2",
					"--use-spot-instances",
					"--replicas", "3")
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Spot max price must be positive"))

				By("Create a machinepool without spot instances, but with spot price")
				machinePoolName = "nospotmp"
				output, err = machinePoolService.CreateMachinePool(
					clusterID,
					machinePoolName,
					"--replicas", "3",
					"--spot-max-price", "10.2",
					"--use-spot-instances=false")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Can't set max price when not using spot instances"))
			})

		It("can create machinepool with tags - [id:73469]",
			labels.Runtime.Day2,
			labels.High,
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
						Expect(out.String()).
							Should(ContainSubstring(
								"invalid tag format for tag"))
					case "noTagValue":
						Expect(out.String()).
							Should(ContainSubstring(
								"invalid tag format, tag key or tag value can not be empty"))
					case "noTagKey":
						Expect(out.String()).
							Should(ContainSubstring(
								"invalid tag format, tag key or tag value can not be empty"))
					case "nonAscii":
						Expect(out.String()).
							Should(ContainSubstring(
								"Invalid Machine Pool AWS tags"))
					}
				}
			})

		It("can create machinepool with availibility zone - [id:52352]",
			labels.Runtime.Day2, labels.High,
			func() {
				By("Check the help message of create machinepool")
				output, err := machinePoolService.CreateMachinePool(clusterID, "help", "-h")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("--availability-zone"))

				By("Create a machinepool without availibility zone set will work")
				mpName := "naz-52352"
				_, err = machinePoolService.CreateMachinePool(clusterID, mpName,
					"--replicas", "0",
				)
				Expect(err).ToNot(HaveOccurred())
				mpDescription, err := machinePoolService.DescribeAndReflectMachinePool(clusterID, mpName)
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.DeleteMachinePool(clusterID, mpName)

				azs := common.ParseCommaSeparatedStrings(mpDescription.AvailablityZones)
				Expect(len(azs)).ToNot(Equal(0), "the azs of the machinepool is 0 for the cluster, it's a bug")
				indicatedZone := azs[0]

				By("Create a single AZ machinepool to the cluster")
				mpName = "sz-52352"
				_, err = machinePoolService.CreateMachinePool(clusterID, mpName,
					"--replicas", "1",
					"--availability-zone", indicatedZone,
				)
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.DeleteMachinePool(clusterID, mpName)

				By("List the machinepool to verify")
				mpList, err := machinePoolService.ListAndReflectMachinePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				mp := mpList.Machinepool(mpName)
				Expect(mp).ToNot(BeNil())
				Expect(mp.AvalaiblityZones).To(Equal(indicatedZone))

				By("Scale up the machinepool replicas to 2")
				_, err = machinePoolService.EditMachinePool(clusterID, mpName,
					"--replicas", "2",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Create another machinepool with autoscaling enabled")
				mpName = "szau-52352"
				_, err = machinePoolService.CreateMachinePool(clusterID, mpName,
					"--enable-autoscaling",
					"--min-replicas", "1",
					"--max-replicas", "2",
					"--availability-zone", indicatedZone,
				)
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.DeleteMachinePool(clusterID, mpName)

				By("Describe the machinepool to verify")
				mpDescription, err = machinePoolService.DescribeAndReflectMachinePool(clusterID, mpName)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpDescription.AvailablityZones).To(Equal(indicatedZone))
				Expect(mpDescription.Replicas).To(Equal("1-2"))
			})

		It("can create machinepool with another subnet - [id:52764]",
			labels.Runtime.Day2, labels.High,
			func() {
				By("Check if cluster is BYOVPC cluster")
				if clusterConfig.Subnets == nil {
					SkipTestOnFeature("This testing only work for byovpc cluster")
				}

				By("Prepare a subnet out of the cluster creation subnet")
				subnets := common.ParseCommaSeparatedStrings(clusterConfig.Subnets.PrivateSubnetIds)

				By("Create another machinepool with the subnet cluster creation used will succeed")
				szName := "sz-52764"
				_, err := machinePoolService.CreateMachinePool(clusterID, szName,
					"--replicas", "2",
					"--subnet", subnets[0],
				)
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.DeleteMachinePool(clusterID, szName)

				By("Describe the machinepool will work")
				mpDescription, err := machinePoolService.DescribeAndReflectMachinePool(clusterID, szName)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpDescription.Replicas).To(Equal("2"))
				Expect(mpDescription.Subnets).To(Equal(subnets[0]))

				By("Scale down the machine pool to 0/1 and check")
				for _, newReplica := range []string{
					"0", "1",
				} {
					_, err = machinePoolService.EditMachinePool(clusterID, szName,
						"--replicas", newReplica,
					)
					Expect(err).ToNot(HaveOccurred())
				}

				By("Build vpc client to find another zone for subnet preparation")
				vpcClient, err := vpc_client.GenerateVPCBySubnet(subnets[0], clusterConfig.Region)
				Expect(err).ToNot(HaveOccurred())

				zones, err := vpcClient.AWSClient.ListAvaliableZonesForRegion(clusterConfig.Region, "availability-zone")
				Expect(err).ToNot(HaveOccurred())

				var nonClusterUsedZone string
				for _, zone := range zones {
					for _, subnet := range vpcClient.SubnetList {
						if subnet.Zone == zone {
							continue
						}
					}
					nonClusterUsedZone = zone
					break
				}
				if nonClusterUsedZone == "" {
					log.Logger.Warnf("Didn't find a zone can be used for new subnet creation, skip the below steps")
					return
				}

				By("Prepare the subnet for the picked zone")
				subNetMap, err := vpcClient.PreparePairSubnetByZone(nonClusterUsedZone)
				Expect(err).ToNot(HaveOccurred())
				Expect(subNetMap).ToNot(BeNil())
				privateSubnet := subNetMap["private"]

				By("Describe the cluster to get the infra ID for tagging")
				clusterDescription, err := rosaClient.Cluster.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				tagKey := fmt.Sprintf("kubernetes.io/cluster/%s", clusterDescription.InfraID)
				vpcClient.AWSClient.TagResource(privateSubnet.ID, map[string]string{
					tagKey: "shared",
				})

				By("Create machinepool with the subnet specified will succeed")
				diffzName := "dissz-52764"
				_, err = machinePoolService.CreateMachinePool(clusterID, diffzName,
					"--enable-autoscaling",
					"--max-replicas", "2",
					"--min-replicas", "1",
					"--subnet", privateSubnet.ID,
				)
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.DeleteMachinePool(clusterID, szName)

				By("List the machinepools and check")
				mpList, err := machinePoolService.ListAndReflectMachinePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				mp := mpList.Machinepool(diffzName)
				Expect(mp.Replicas).To(Equal("1-2"))
				Expect(mp.Subnets).To(Equal(privateSubnet.ID))
			})

		It("can create machinepool with additional security groups - [id:68173]",
			labels.Runtime.Day2, labels.High,
			func() {
				By("Load the vpc client of the machinepool")
				mps, err := rosaClient.MachinePool.ListAndReflectMachinePools(clusterID)
				Expect(err).ToNot(HaveOccurred())

				subnetIDs := common.ParseCommaSeparatedStrings(mps.MachinePools[0].Subnets)
				if len(subnetIDs) == 0 {
					SkipTestOnFeature("Only check BYOVPC cluster") // validation will be covered by 68219
				}
				vpcClient, err := vpc_client.GenerateVPCBySubnet(subnetIDs[0], clusterConfig.Region)
				Expect(err).ToNot(HaveOccurred())

				By("Prepare security groups")
				sgIDs, err := vpcClient.CreateAdditionalSecurityGroups(3, "68173", "testing for case 68173")
				Expect(err).ToNot(HaveOccurred())
				defer func(sgs []string) {
					for _, sg := range sgs {
						vpcClient.AWSClient.DeleteSecurityGroup(sg)
					}
				}(sgIDs)

				By("Create machinepool with security groups set")
				mpName := "mp-68173"
				_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
					"--additional-security-group-ids", strings.Join(sgIDs, ","),
					"--replicas", "0",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

				By("Check the machinepool details by describe")
				mpDescription, err := rosaClient.MachinePool.DescribeAndReflectMachinePool(clusterID, mpName)
				Expect(err).ToNot(HaveOccurred())

				Expect(mpDescription.SecurityGroupIDs).To(Equal(strings.Join(sgIDs, ", ")))

				By("Create another machinepool without security groups and describe it")
				mpName = "mp-68173-nsg"
				_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
					"--replicas", "0",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)
				By("Check the machinepool detail by describe")
				mpDescription, err = rosaClient.MachinePool.DescribeAndReflectMachinePool(clusterID, mpName)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpDescription.SecurityGroupIDs).To(BeEmpty())
			})

		It("can create local zone machinepool - [id:55979]", labels.Runtime.Day2, labels.High,
			func() {
				By("Check if cluster is BYOVPC cluster")
				if clusterConfig.Subnets == nil {
					SkipTestOnFeature("This testing only work for byovpc cluster")
				}

				By("Prepare a subnet out of the cluster creation subnet")
				subnets := common.ParseCommaSeparatedStrings(clusterConfig.Subnets.PrivateSubnetIds)

				By("Build vpc client to find a local zone for subnet preparation")
				vpcClient, err := vpc_client.GenerateVPCBySubnet(subnets[0], clusterConfig.Region)
				Expect(err).ToNot(HaveOccurred())

				zones, err := vpcClient.AWSClient.ListAvaliableZonesForRegion(clusterConfig.Region, "local-zone")
				Expect(err).ToNot(HaveOccurred())
				if len(zones) == 0 {
					SkipTestOnFeature("No local zone found in the region skip the testing")
				}
				localZone := zones[0]

				By("Prepare the subnet for the picked zone")
				subNetMap, err := vpcClient.PreparePairSubnetByZone(localZone)
				Expect(err).ToNot(HaveOccurred())
				Expect(subNetMap).ToNot(BeNil())
				privateSubnet := subNetMap["private"]

				By("Describe the cluster to get the infra ID for tagging")
				clusterDescription, err := rosaClient.Cluster.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				tagKey := fmt.Sprintf("kubernetes.io/cluster/%s", clusterDescription.InfraID)
				vpcClient.AWSClient.TagResource(privateSubnet.ID, map[string]string{
					tagKey: "shared",
				})

				By("Create machinepool with the subnet specified will succeed")
				localZoneMpName := "localz-55979"
				_, err = machinePoolService.CreateMachinePool(clusterID, localZoneMpName,
					"--enable-autoscaling",
					"--max-replicas", "2",
					"--min-replicas", "1",
					"--subnet", privateSubnet.ID,
				)
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.DeleteMachinePool(clusterID, localZoneMpName)

				By("List the machinepools and check")
				mpList, err := machinePoolService.ListAndReflectMachinePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				mp := mpList.Machinepool(localZoneMpName)
				Expect(mp.Replicas).To(Equal("1-2"))
				Expect(mp.Subnets).To(Equal(privateSubnet.ID))
			})
		Context("validation", func() {
			It("will validate name/replicas/labels/taints  - [id:67057]",
				labels.Runtime.Day2, labels.Medium,
				func() {
					mpName := "mp-67057"

					By("Create with invalid name will fail")
					output, err := rosaClient.MachinePool.CreateMachinePool(clusterID, "Inv@lid",
						"--replicas", "0",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("Expected a valid name for the machine pool"))

					By("Create with invalid replicas will fail")
					output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
						"--replicas", "-1",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("min-replicas must be a non-negative integer"))

					By("Create with invalid labels will fail")
					output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
						"--replicas", "0",
						"--labels", "invalidformat",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("Expected key=value format for labels"))

					By("Create with invalid taints fmt will fail")
					output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
						"--replicas", "0",
						"--taints", "aaabbb:Invalid",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(
						ContainSubstring("Expected key=value:scheduleType format for taints. Got 'aaabbb:Invalid'"))

					By("Create with invalid taints effect will fail")
					output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
						"--replicas", "0",
						"--taints", "aaa=bbb:Invalid",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(
						ContainSubstring("Unrecognized taint effect 'Invalid'," +
							" only the following effects are supported: 'NoExecute', 'NoSchedule', 'PreferNoSchedule'"))

				})

			It("will validate root volume size - [id:66874]",
				labels.Runtime.Day2, labels.Medium,
				func() {
					mpName := "mp-66874"
					By("Create with too small disk size will fail")
					output, err := rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
						"--replicas", "0",
						"--disk-size", "2GiB",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("Invalid root disk size: 2 GiB." +
						" Must be between 128 GiB and 16384 GiB"))

					By("Create with large disk size will fail")
					output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
						"--replicas", "0",
						"--disk-size", "17594GB",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("Invalid root disk size: 16385 GiB." +
						" Must be between 128 GiB and 16384 GiB"))

					By("Create with un-known unit will fail")
					output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
						"--replicas", "0",
						"--disk-size", "2GiiiB",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("invalid disk size format: '2GiiiB'." +
						" accepted units are Giga or Tera in the form of g, G, GB, GiB, Gi, t, T, TB, TiB, Ti"))

					By("Create with too large value will fail")
					output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
						"--replicas", "0",
						"--disk-size", "25678987654567898765456789087654GiB",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("invalid disk size: '25678987654567898765456789087654Gi'." +
						" maximum size exceeded"))

				})
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

			verifyClusterComputeNodesMatched func(clusterID string) = func(clusterID string) {
				By("List the machinepools and calculate")
				mpList, err := rosaClient.MachinePool.ListAndReflectMachinePools(clusterID)
				Expect(err).ToNot(HaveOccurred())

				var autoscaling bool
				var minReplicas int
				var maxReplicas int
				for _, mp := range mpList.MachinePools {
					if mp.AutoScaling == "true" {
						autoscaling = true
						min, max := strings.Split(mp.Replicas, "-")[0], strings.Split(mp.Replicas, "-")[1]
						minNum, err := strconv.Atoi(min)
						Expect(err).ToNot(HaveOccurred())
						maxNum, err := strconv.Atoi(max)
						Expect(err).ToNot(HaveOccurred())
						minReplicas += minNum
						maxReplicas += maxNum

					} else {
						replicasNum, err := strconv.Atoi(mp.Replicas)
						Expect(err).ToNot(HaveOccurred())
						minReplicas += replicasNum
						maxReplicas += replicasNum
					}
				}
				expectedClusterTotalNode := strconv.Itoa(minReplicas)
				computeKey := "Compute"
				if autoscaling {
					computeKey = "Compute (Autoscaled)"
					expectedClusterTotalNode = fmt.Sprintf("%d-%d", minReplicas, maxReplicas)
				}

				By("Describe the rosa cluster and check the cluster nodes")
				clusterDescription, err := rosaClient.Cluster.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				var clusterTotalNumber string
				for _, nodesMap := range clusterDescription.Nodes {
					if computeNumber, ok := nodesMap[computeKey]; ok {
						clusterTotalNumber = fmt.Sprintf("%v", computeNumber)
						break
					}
				}
				Expect(clusterTotalNumber).To(Equal(expectedClusterTotalNode))
			}
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
			rosaClient.CleanResources(clusterID)
		})

		It("will succeed - [id:38838]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check help message")
				output, err := machinePoolService.EditMachinePool(clusterID, "", "-h")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).
					Should(ContainSubstring(
						"machinepool, machinepools, machine-pool, machine-pools"))

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
				Expect(mp.AutoScaling).To(Equal(constants.Yes))
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
				Expect(mp.AutoScaling).To(Equal(constants.Yes))
				Expect(mp.Replicas).To(Equal("0-3"))
				Expect(mp.Labels).To(Equal(strings.Join(common.ParseCommaSeparatedStrings(labels), ", ")))
				Expect(mp.Taints).To(Equal(strings.Join(common.ParseCommaSeparatedStrings(taints), ", ")))
			})

		It("can list/edit/delete the default worker pool - [id:66750]", labels.Runtime.Destructive, labels.High,
			func() {
				By("List the machinepools of the cluster")
				mpList, err := rosaClient.MachinePool.ListAndReflectMachinePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				workerPool := mpList.Machinepool(con.DefaultClassicWorkerPool)
				Expect(workerPool).ToNot(BeNil())

				By("Scale up the default machinepool worker")
				var updatedValue string
				var flags []string
				if workerPool.AutoScaling == constants.Yes {
					flags = []string{
						"--enable-autoscaling=false",
						"--replicas", "6",
					}
					updatedValue = "6"
				} else {
					originalNodeNumber, err := strconv.Atoi(workerPool.Replicas)
					Expect(err).ToNot(HaveOccurred())
					updatedValue = fmt.Sprintf("%d", originalNodeNumber+3)
					flags = []string{
						"--replicas", updatedValue,
					}
				}
				_, err = rosaClient.MachinePool.EditMachinePool(clusterID, con.DefaultClassicWorkerPool,
					flags...,
				)
				Expect(err).ToNot(HaveOccurred())
				workerPoolDescription, err := rosaClient.MachinePool.DescribeAndReflectMachinePool(
					clusterID,
					con.DefaultClassicWorkerPool,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(workerPoolDescription.Replicas).To(Equal(updatedValue))

				verifyClusterComputeNodesMatched(clusterID)

				By("Create an additional machinepool of the cluster")
				_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, "worker-replace",
					"--replicas", "3",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Scale down the worker pool nodes to 0")

				_, err = rosaClient.MachinePool.EditMachinePool(clusterID, con.DefaultClassicWorkerPool,
					"--replicas", "0",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Edit the cluster to UWM enabled/disabled")
				output, err := rosaClient.Cluster.EditCluster(clusterID,
					"--disable-workload-monitoring",
					"-y")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("Updated cluster"))

				By("Delete the default machinepool")
				_, err = rosaClient.MachinePool.DeleteMachinePool(clusterID, con.DefaultClassicWorkerPool)
				Expect(err).ToNot(HaveOccurred())

				By("List the machinepool again")
				mpList, err = rosaClient.MachinePool.ListAndReflectMachinePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				workerPool = mpList.Machinepool(con.DefaultClassicWorkerPool)
				Expect(workerPool).To(BeNil())

				By("Check the cluster nodes")
				verifyClusterComputeNodesMatched(clusterID)
			})

		It("enable/disable/update autoscaling will work well - [id:38194]", labels.Runtime.Day2, labels.High,
			func() {
				By("Record the original info of default worker pool")
				mpDescription, err := machinePoolService.DescribeAndReflectMachinePool(clusterID, con.DefaultClassicWorkerPool)
				Expect(err).ToNot(HaveOccurred())
				recoverFlags := []string{}
				if mpDescription.AutoScaling == constants.Yes {
					minReplicas, maxReplicas := strings.Split(mpDescription.Replicas, "-")[0],
						strings.Split(mpDescription.Replicas, "-")[1]
					recoverFlags = append(recoverFlags,
						"--enable-autoscaling=true",
						"--min-replicas", minReplicas,
						"--max-replicas", maxReplicas,
					)
				} else {
					recoverFlags = append(recoverFlags,
						"--enable-autoscaling=false",
						"--replicas", mpDescription.Replicas,
					)
				}

				By("Update the worker pool to autoscaling will work")
				_, err = machinePoolService.EditMachinePool(clusterID, con.DefaultClassicWorkerPool,
					"--enable-autoscaling",
					"--min-replicas", "3",
					"--max-replicas", "9",
				)
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.EditMachinePool(clusterID, con.DefaultClassicWorkerPool, recoverFlags...)

				By("Describe the machinepool and check the editing")
				mpDescription, err = machinePoolService.DescribeAndReflectMachinePool(clusterID, con.DefaultClassicWorkerPool)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpDescription.AutoScaling).To(Equal(constants.Yes))
				Expect(mpDescription.Replicas).To(Equal("3-9"))

				By("Scale up the worker pool with autoscaling will work")
				_, err = machinePoolService.EditMachinePool(clusterID, con.DefaultClassicWorkerPool,
					"--max-replicas", "12",
				)
				Expect(err).ToNot(HaveOccurred())

				mpDescription, err = machinePoolService.DescribeAndReflectMachinePool(clusterID, con.DefaultClassicWorkerPool)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpDescription.AutoScaling).To(Equal(constants.Yes))
				Expect(mpDescription.Replicas).To(Equal("3-12"))
			})
	})
