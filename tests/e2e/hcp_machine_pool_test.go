package e2e

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	ciConfig "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	"github.com/openshift/rosa/tests/utils/helper"
	. "github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("HCP Machine Pool", labels.Feature.Machinepool, func() {
	// It doesn't check whether node pool instances ready in default.
	// If needed for verify hcp node pool's changes, pls set the ENV CLUSTER_NODE_POOL_GLOBAL_CHECK to true,
	// which will wait for node pool instances ready until timeout.
	isNodePoolGlobalCheck := config.IsNodePoolGlobalCheck()

	var (
		rosaClient         *rosacli.Client
		machinePoolService rosacli.MachinePoolService
		profile            *handler.Profile
		isMultiArch        bool
	)

	BeforeEach(func() {
		var err error

		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()
		machinePoolService = rosaClient.MachinePool

		By("Skip testing if the cluster is not a HCP cluster")
		hostedCluster, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())

		By("Check whether the cluster is multi arch")
		isMultiArch, err = rosaClient.Cluster.IsMultiArch(clusterID)
		Expect(err).ToNot(HaveOccurred())

		if !hostedCluster {
			SkipNotHosted()
		}

		profile = handler.LoadProfileYamlFileByENV()
	})

	Describe("Create/delete/view a machine pool", func() {
		It("should succeed with additional security group IDs [id:72195]", labels.Critical, labels.Runtime.Day2, labels.FedRAMP, func() {
			By("check the throttle version")
			throttleVersion, _ := semver.NewVersion("4.15.0-a.0")
			clusterDescription, err := rosaClient.Cluster.DescribeClusterAndReflect(clusterID)
			Expect(err).ToNot(HaveOccurred())
			clusterVersion, err := semver.NewVersion(clusterDescription.OpenshiftVersion)
			Expect(err).ToNot(HaveOccurred())

			By("Load the vpc client of the machinepool")
			mps, err := rosaClient.MachinePool.ListAndReflectNodePools(clusterID)
			Expect(err).ToNot(HaveOccurred())

			subnetID := mps.NodePools[0].Subnet
			vpcClient, err := vpc_client.GenerateVPCBySubnet(subnetID, profile.Region)
			Expect(err).ToNot(HaveOccurred())

			By("Prepare security groups")
			// Configure with a random str, which can solve the rerun failure
			sgPrefix := helper.GenerateRandomName("72195", 2)
			sgIDs, err := vpcClient.CreateAdditionalSecurityGroups(3, sgPrefix, "testing for case 72195")
			Expect(err).ToNot(HaveOccurred())
			defer func(sgs []string) {
				for _, sg := range sgs {
					vpcClient.AWSClient.DeleteSecurityGroup(sg)
				}
			}(sgIDs)

			By("Create machinepool with security groups set")
			mpName := "mp-72195"
			output, err := rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
				"--additional-security-group-ids", strings.Join(sgIDs, ","),
				"--replicas", "1",
				"-y",
			)
			if clusterVersion.LessThan(throttleVersion) {
				// Low version cannot support security group set
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring(
					"Additional security groups are not supported for '%s'",
					clusterDescription.OpenshiftVersion))
				return
			}
			Expect(err).ToNot(HaveOccurred())

			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

			By("Check the machinepool detail by describe")
			mpDescription, err := rosaClient.MachinePool.DescribeAndReflectNodePool(clusterID, mpName)
			Expect(err).ToNot(HaveOccurred())

			Expect(mpDescription.AdditionalSecurityGroupIDs).To(Equal(strings.Join(sgIDs, ", ")))

			By("Create another machinepool without security groups and describe it")
			mpName = "mp-72195-nsg"
			_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
				"--replicas", "1",
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)
			By("Check the machinepool detail by describe")
			mpDescription, err = rosaClient.MachinePool.DescribeAndReflectNodePool(clusterID, mpName)
			Expect(err).ToNot(HaveOccurred())
			Expect(mpDescription.AdditionalSecurityGroupIDs).To(BeEmpty())
		})
	})

	DescribeTable("create machinepool with volume size set - [id:66872]",
		labels.Runtime.Day2, labels.FedRAMP,
		labels.Critical,
		func(diskSize, instanceType, expectedDiskSize string) {
			npID := helper.GenerateRandomName("np-66872", 2)

			By("Create a nodepool with the disk size")
			output, err := machinePoolService.CreateMachinePool(clusterID, npID,
				"--replicas", "0",
				"--disk-size", diskSize,
				"--instance-type", instanceType,
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, npID)

			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).
				Should(ContainSubstring(
					"Machine pool '%s' created successfully on hosted cluster '%s'",
					npID,
					clusterID))

			By("Check the machinepool list")
			output, err = machinePoolService.ListMachinePool(clusterID)
			Expect(err).ToNot(HaveOccurred())

			nplist, err := machinePoolService.ReflectNodePoolList(output)
			Expect(err).ToNot(HaveOccurred())

			np := nplist.Nodepool(npID)
			Expect(np).ToNot(BeNil(), "node pool is not found for the cluster")
			Expect(np.DiskSize).To(Equal(expectedDiskSize))
			Expect(np.InstanceType).To(Equal(instanceType))

			By("Check the node pool description")
			output, err = machinePoolService.DescribeMachinePool(clusterID, npID)
			Expect(err).ToNot(HaveOccurred())
			npD, err := machinePoolService.ReflectNodePoolDescription(output)
			Expect(err).ToNot(HaveOccurred())
			Expect(npD.DiskSize).To(Equal(expectedDiskSize))
		},
		Entry("Disk size 200GB", "200GB", constants.R5XLarge, "186 GiB"),
		Entry("Disk size 0.5TiB", "0.5TiB", constants.M52XLarge, "512 GiB"),
	)

	It("machinepool AWS preflight tag validation[id:73638]",
		labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
		func() {

			By("Check the help message of machinepool creation")
			mpID := helper.GenerateRandomName("mp-73638", 2)
			out, err := machinePoolService.CreateMachinePool(clusterID, mpID, "-h")
			Expect(err).ToNot(HaveOccurred(), out.String())
			Expect(out.String()).Should(ContainSubstring("--tags strings"))

			By("Create a machinepool with tags set")
			tags := []string{
				"test:testvalue",
				"test2:testValue/openshift",
			}
			out, err = machinePoolService.CreateMachinePool(clusterID, mpID,
				"--replicas", "3",
				"--tags", strings.Join(tags, ","),
			)
			Expect(err).ToNot(HaveOccurred(), out.String())
			defer machinePoolService.DeleteMachinePool(clusterID, mpID)

			By("Describe the machinepool")
			description, err := machinePoolService.DescribeAndReflectMachinePool(clusterID, mpID)
			Expect(err).ToNot(HaveOccurred(), out.String())

			for _, tag := range tags {
				Expect(description.Tags).Should(ContainSubstring(strings.Replace(tag, ":", "=", -1)))
			}

			By("Create machinepool with too many tags")
			maxTags := 25
			var tooManyTags []string
			for i := 0; i < maxTags+1; i++ {
				t := strconv.Itoa(i)
				key := "foo" + t
				kvp := key + ":testValue"
				tooManyTags = append(tooManyTags, kvp)
			}

			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", strings.Join(tooManyTags, ","),
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("Invalid Node Pool AWS tags: Resource has too many AWS tags"))

			By("Create machinepool with a tag too long")
			maxKeyTagLength := 128
			maxValueTagLength := 256

			tooLongKeyTag := strings.Repeat("z", maxKeyTagLength+1)
			tag := tooLongKeyTag + ":testValue"

			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: expected a valid user tag key 'zzz"))

			tooLongValueTag := strings.Repeat("z", maxValueTagLength+1)
			tag = "testKey:" + tooLongValueTag

			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: expected a valid user tag value 'zzz"))

			By("Create machinepool using aws as a prefix")
			tag = "aws:testKey:" + "testValue"
			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: invalid tag format for tag '[aws testKey testValue]'"))

			By("Create machinepool with an invalid tag")
			tag = "#" + ":testValue"
			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: expected a valid user tag key '#'"))

			tag = "testKey:" + "#"
			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: expected a valid user tag value '#'"))
		})

	DescribeTable("Scale up/down a machine pool", labels.Critical, labels.Runtime.Day2, labels.FedRAMP,
		func(instanceType string, amdOrArm string) {
			if !isMultiArch && amdOrArm == constants.ARM {
				SkipNotMultiArch()
			}

			By("Create machinepool with " + amdOrArm + " instance " + instanceType)
			mpPrefix := fmt.Sprintf("%v-60278", amdOrArm)
			mpName := helper.GenerateRandomName(mpPrefix, 2)
			desiredReplicas := 1
			_, err := rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
				"--instance-type", instanceType,
				"--replicas", fmt.Sprintf("%v", desiredReplicas),
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

			mpDesc, err := rosaClient.MachinePool.DescribeAndReflectNodePool(clusterID, mpName)
			Expect(err).ToNot(HaveOccurred())
			Expect(mpDesc.DesiredReplicas).Should(Equal(desiredReplicas))
			Expect(mpDesc.InstanceType).Should(Equal(instanceType))

			if isNodePoolGlobalCheck {
				By("Check if current replicas reach the desired replicas after creating a machine pool")
				err = rosaClient.MachinePool.WaitForNodePoolReplicasReady(
					clusterID,
					mpName,
					false,
					constants.NodePoolCheckPoll,
					constants.NodePoolCheckTimeout,
				)
				Expect(err).ToNot(HaveOccurred())
			}

			By("Scale a machine pool with unchanged parameters")
			err = rosaClient.MachinePool.ScaleNodePool(clusterID, mpName, desiredReplicas, true)
			Expect(err).ToNot(HaveOccurred())

			By("Scale up a machine pool replicas from 1 to 2")
			upReplicas := 2
			err = rosaClient.MachinePool.ScaleNodePool(clusterID, mpName, upReplicas, true)
			Expect(err).ToNot(HaveOccurred())

			By("Scale down a machine pool replicas from 2 to 1")
			downReplicas := 1
			err = rosaClient.MachinePool.ScaleNodePool(clusterID, mpName, downReplicas, true)
			Expect(err).ToNot(HaveOccurred())

			By("Scale down a machine pool replicas to 0")
			zeroReplica := 0
			err = rosaClient.MachinePool.ScaleNodePool(clusterID, mpName, zeroReplica, true)
			Expect(err).ToNot(HaveOccurred())
		},
		Entry("For amd64 cpu architecture [id:60278]", constants.M5XLarge, constants.AMD),
		Entry("For arm64 cpu architecture [id:60278]", constants.M6gXLarge, constants.ARM),
	)

	DescribeTable("Scale up/down a machine pool with invalid replica", labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
		func(instanceType string, updatedReplicas string, expectedErrMsg string) {
			By("Create machinepool with instance " + instanceType)
			mpName := helper.GenerateRandomName("mp-60278", 2)
			desiredReplicas := 1
			_, err := rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
				"--instance-type", instanceType,
				"--replicas", fmt.Sprintf("%v", desiredReplicas),
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--replicas", fmt.Sprintf("%v", updatedReplicas),
				"-y",
			)
			Expect(err.Error()).Should(ContainSubstring(expectedErrMsg))
		},

		Entry("Scale replica too large [id:60278]", constants.M52XLarge, "1000", "exceeds the maximum allowed"),
		Entry("Scale replica to -1 [id:60278]", constants.M52XLarge, "-1", "must be a non-negative number"),
		Entry("Scale replica to a char [id:60278]", constants.M52XLarge, "a", "invalid syntax"),
	)

	Describe("Scale up/down a machine pool enabling autoscale", func() {
		It("should succeed to scale with valid parameters [id:60278]", labels.Medium, labels.Runtime.Day2, labels.FedRAMP, func() {
			instanceType := constants.M52XLarge
			By("Create machinepool with " + " instance " + instanceType + " and enable autoscale")

			mpPrefix := "autoscale"
			mpName := helper.GenerateRandomName(mpPrefix, 2)
			minReplica := 1
			maxReplica := 3
			_, err := rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
				"--instance-type", instanceType,
				"--enable-autoscaling",
				"--min-replicas", fmt.Sprintf("%v", minReplica),
				"--max-replicas", fmt.Sprintf("%v", maxReplica),
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

			if isNodePoolGlobalCheck {
				By("Check current replicas reach the min replicas after creating a autoscaled machine pool")
				err = rosaClient.MachinePool.WaitForNodePoolReplicasReady(
					clusterID,
					mpName,
					true,
					constants.NodePoolCheckPoll,
					constants.NodePoolCheckTimeout,
				)
				Expect(err).ToNot(HaveOccurred())
			}

			// TODO There's an issue here, uncomment when solved
			//By("Scale a machine pool with unchanged parameters")
			//err = rosaClient.MachinePool.ScaleAutoScaledNodePool(clusterID, mpName, minReplica, maxReplica, true)
			//Expect(err).ToNot(HaveOccurred())

			By("Scale up a machine pool replicas from 1~3 to 2~5")
			upMinReplica := 2
			upMaxReplica := 5
			err = rosaClient.MachinePool.ScaleAutoScaledNodePool(clusterID, mpName, upMinReplica, upMaxReplica, true)
			Expect(err).ToNot(HaveOccurred())

			// Don't check the current replicas when scale down, because it won't change after reducing min_replica
			// It only depends on the autoscale strategy when reducing min_replica
			By("Scale down a machine pool replicas from 2~5 to 1~2")
			downMinReplica := 1
			downMaxReplica := 2
			err = rosaClient.MachinePool.ScaleAutoScaledNodePool(clusterID, mpName, downMinReplica, downMaxReplica, false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should raise error message with the invalid parameters [id:60278]", labels.Medium, labels.Runtime.Day2, labels.FedRAMP, func() {
			instanceType := constants.M52XLarge
			By("Create machinepool with" + " instance " + instanceType + " and enable autoscale")

			mpPrefix := "autoscale"
			mpName := helper.GenerateRandomName(mpPrefix, 2)
			minReplica := 1
			maxReplica := 2
			_, err := rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
				"--instance-type", instanceType,
				"--enable-autoscaling",
				"--min-replicas", fmt.Sprintf("%v", minReplica),
				"--max-replicas", fmt.Sprintf("%v", maxReplica),
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

			By("Scale down a machine pool min replica to 0")
			zeroMinReplica := 0
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--min-replicas", fmt.Sprintf("%v", zeroMinReplica),
				"-y",
			)
			expectErrMsg := "ERR: min-replicas must be greater than zero"
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))

			By("Scale up a machine pool max replica too large")
			moreMaxReplica := 1000
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--max-replicas", fmt.Sprintf("%v", moreMaxReplica),
				"-y",
			)
			expectErrMsg = "exceeds the maximum allowed"
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))

			By("Scale a machine pool min replica > max replica")
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--min-replicas", fmt.Sprintf("%v", "5"),
				"--max-replicas", fmt.Sprintf("%v", "3"),
				"-y",
			)
			expectErrMsg = "min-replicas needs to be less than the number of machine pool max-replicas"
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))

			By("Scale down a machine pool min replica to -1")
			downMinReplica := -1
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--min-replicas", fmt.Sprintf("%v", downMinReplica),
				"-y",
			)
			expectErrMsg = "must be a non-negative number when autoscaling is set"
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))

			By("Scale a machine pool with min replica and max replica a char")
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--min-replicas", fmt.Sprintf("%v", "a"),
				"--max-replicas", fmt.Sprintf("%v", "b"),
				"-y",
			)
			expectErrMsg = "invalid syntax"
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))
		})
	})

	Describe("Validate machinepool", func() {
		It("creation - [id:56786]", labels.Medium, labels.Runtime.Day2, func() {
			By("with negative replicas number")
			_, err := machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "-9")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("replicas must be a non-negative integer"))

			By("with replicas > the maximum")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "501")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).
				Should(
					ContainSubstring("should provide an integer number less than or equal to"))

			By("with invalid name")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything%^#@", "--replicas", "2")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("expected a valid name for the machine pool"))

			By("with replicas and enable-autoscaling at the same time")
			_, err = machinePoolService.CreateMachinePool(
				clusterID,
				"anything",
				"--replicas", "2",
				"--enable-autoscaling",
				"--min-replicas", "3",
				"--max-replicas", "3")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("replicas can't be set when autoscaling is enabled"))

			By("with min-replicas larger than max-replicas")
			_, err = machinePoolService.CreateMachinePool(
				clusterID,
				"anything",
				"--enable-autoscaling",
				"--min-replicas", "6",
				"--max-replicas", "3")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).
				Should(
					ContainSubstring("max-replicas must be greater or equal to min-replicas"))

			By("with min-replicas and max-replicas but without enable-autoscaling")
			_, err = machinePoolService.CreateMachinePool(
				clusterID,
				"anything",
				"--min-replicas", "3",
				"--max-replicas", "3")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("autoscaling must be enabled in order to set min and max replicas"))

			By("with max-replicas > the maximum")
			_, err = machinePoolService.CreateMachinePool(
				clusterID,
				"anything",
				"--enable-autoscaling",
				"--min-replicas", "3",
				"--max-replicas", "501")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).
				Should(
					ContainSubstring("should provide an integer number less than or equal to"))

			By("with wrong instance-type")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--instance-type", "wrong")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("is not supported in availability zone"))

			By("with non existing subnet")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--subnet", "subnet-xxx")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("The subnet ID 'subnet-xxx' does not exist"))

			By("with subnet not in VPC")
			vpcPrefix := helper.TrimNameByLength("c56786", 20)
			resourcesHandler, err := handler.NewTempResourcesHandler(rosaClient, profile.Region,
				ciConfig.Test.GlobalENV.AWSCredetialsFile,
				ciConfig.Test.GlobalENV.SVPC_CREDENTIALS_FILE)
			Expect(err).ToNot(HaveOccurred())
			defer resourcesHandler.DestroyResources()

			vpc, err := resourcesHandler.PrepareVPC(vpcPrefix, constants.DefaultVPCCIDRValue, false, false)
			Expect(err).ToNot(HaveOccurred())
			zones, err := vpc.AWSClient.ListAvaliableZonesForRegion(profile.Region, "availability-zone")
			Expect(err).ToNot(HaveOccurred())
			subnet, err := vpc.CreateSubnet(zones[0])
			Expect(err).ToNot(HaveOccurred())
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--subnet", subnet.ID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("found but expected on VPC"))

			By("with label no key")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--labels", "v")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("expected key=value format for labels"))

			By("with taint no key")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--taints", "v")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("expected key=value:scheduleType format for taints. Got 'v'"))

			By("with taint no schedule type")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--taints", "k=v")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("expected key=value:scheduleType format for taints. Got 'k=v'"))

			By("with taint empty schedule type")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--taints", "k=v:")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("expected a not empty effect"))

			By("with auto-repair wrong value")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--autorepair=v")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("strconv.ParseBool: parsing \"v\": invalid syntax"))

			By("with unsupported version")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--version", "4.12.1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("expected a valid OpenShift version"))

			By("with unsupported flag")
			_, err = machinePoolService.CreateMachinePool(clusterID, "anything", "--replicas", "2", "--multi-availability-zone")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).
				Should(
					ContainSubstring("setting `multi-availability-zone` flag is not supported for HCP clusters"))
		})

		It("deletion - [id:56783]", labels.Medium, labels.Runtime.Day2, labels.FedRAMP, func() {
			By("with no machinepool id")
			_, err := machinePoolService.DeleteMachinePool(clusterID, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("You need to specify a machine pool name"))

			By("with non existing machinepool id")
			_, err = machinePoolService.DeleteMachinePool(clusterID, "anything")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("machine pool 'anything' does not exist"))

			By("with invalid machinepool id")
			_, err = machinePoolService.DeleteMachinePool(clusterID, "anything%^")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("Expected a valid identifier for the machine pool"))

			By("with unknown flag --interactive")
			_, err = machinePoolService.DeleteMachinePool(clusterID, "anything", "--interactive")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("unknown flag: --interactive"))

			if !profile.ClusterConfig.MultiAZ {
				By("Delete last remaining machinepool")
				_, err = machinePoolService.DeleteMachinePool(clusterID, constants.DefaultHostedWorkerPool)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).
					Should(
						ContainSubstring(
							fmt.Sprintf("failed to delete machine pool '%s' on hosted cluster", constants.DefaultHostedWorkerPool)))
				Expect(err.Error()).
					Should(
						ContainSubstring("The last node pool can not be deleted from a cluster"))
			}
		})

		It("creation in local zone subnet - [id:71319]",
			labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
			func() {
				if profile.ClusterConfig.SharedVPC {
					Skip("This test only run on the cluster not using shared-vpc")
				}
				var vpcClient *vpc_client.VPC
				var err error

				By("Retrieve cluster config")
				clusterConfig, err := config.ParseClusterProfile()
				Expect(err).ToNot(HaveOccurred())

				By("Prepare a subnet out of the cluster creation subnet")
				subnets := helper.ParseCommaSeparatedStrings(clusterConfig.Subnets.PrivateSubnetIds)

				By("Build vpc client to find a local zone for subnet preparation")
				vpcClient, err = vpc_client.GenerateVPCBySubnet(subnets[0], clusterConfig.Region)
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
				for _, subnet := range subNetMap {
					_, err := vpcClient.AWSClient.TagResource(
						subnet.ID,
						map[string]string{
							"kubernetes.io/cluster/unmanaged": "true",
						},
					)
					Expect(err).ToNot(HaveOccurred())
				}

				By("Create machinepool into the local zone subnet")
				name := helper.GenerateRandomName("mp-71319", 2)
				_, err = machinePoolService.CreateMachinePool(
					clusterID,
					name,
					"--replicas", "2",
					"--subnet", privateSubnet.ID,
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).
					To(
						ContainSubstring(
							fmt.Sprintf("Creating a node pool a in local zone '%s' isn't supported", localZone)))
			})

		It("upgrade - [id:67419]", labels.Medium, labels.Runtime.Day2, labels.FedRAMP, func() {
			var err error

			clusterService := rosaClient.Cluster
			machinePoolUpgradeService := rosaClient.MachinePoolUpgrade
			versionService := rosaClient.Version

			By("Retrieve cluster config")
			clusterConfig, err := config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())

			By("without machinepool")
			_, err = machinePoolUpgradeService.CreateManualUpgrade(
				clusterID,
				"",
				clusterConfig.Version.RawID,
				"",
				"")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid identifier for the machine pool"))

			By("with wrong machinepool")
			_, err = machinePoolUpgradeService.CreateManualUpgrade(
				clusterID,
				"anything",
				clusterConfig.Version.RawID,
				"",
				"")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Machine pool 'anything' does not exist for hosted cluster"))

			By("for not-ready cluster")
			var notReadyClusterID string
			rosaClient.Runner.UnsetArgs()
			output, err := clusterService.List()
			Expect(err).To(BeNil())
			clusterList, err := clusterService.ReflectClusterList(output)
			Expect(err).To(BeNil())
			for _, c := range clusterList.Clusters {
				if c.State != constants.Ready && c.Topology == constants.HostedCP {
					notReadyClusterID = c.ID
					break
				}
			}
			if notReadyClusterID != "" {
				_, err = machinePoolUpgradeService.CreateManualUpgrade(
					notReadyClusterID,
					"anything",
					clusterConfig.Version.RawID,
					"",
					"")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Cluster '%s' is not yet ready", notReadyClusterID)))
			} else {
				Logger.Info("No cluster found in state not-ready. Skipping this step")
			}

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
			nodePoolAutoName := helper.GenerateRandomName("np-67419", 2)
			_, err = machinePoolService.CreateMachinePool(clusterID, nodePoolAutoName,
				"--replicas", "0",
				"--version", upgradableVersion.Version)
			Expect(err).ToNot(HaveOccurred())

			By("with wrong schedule value")
			_, err = machinePoolUpgradeService.CreateAutomaticUpgrade(
				clusterID,
				nodePoolAutoName,
				"anything")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Schedule 'anything' is not a valid cron expression"))

			By("with wrong schedule-time value")
			_, err = machinePoolUpgradeService.CreateManualUpgrade(
				clusterID,
				nodePoolAutoName,
				clusterConfig.Version.RawID,
				"2154-12-31",
				"wrong")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("schedule date should use the format 'yyyy-mm-dd'"))

			By("with wrong schedule-date value")
			_, err = machinePoolUpgradeService.CreateManualUpgrade(
				clusterID,
				nodePoolAutoName,
				clusterConfig.Version.RawID,
				"wrong",
				"10:34")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Schedule time should use the format 'HH:mm'"))

			By("with wrong version value")
			_, err = machinePoolUpgradeService.CreateManualUpgrade(
				clusterID,
				nodePoolAutoName,
				"wrong",
				"",
				"")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected a valid machine pool version"))

			By("with schedule and version")
			_, err = machinePoolUpgradeService.CreateAutomaticUpgrade(
				clusterID,
				nodePoolAutoName,
				"2 5 * * *",
				"--version", clusterConfig.Version.RawID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The '--schedule' option is mutually exclusive with '--version'"))

			By("with schedule and schedule-date")
			_, err = machinePoolUpgradeService.CreateAutomaticUpgrade(
				clusterID,
				nodePoolAutoName,
				"2 5 * * *",
				"--schedule-date", "2154-12-31")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring(
					"The '--schedule-date' and '--schedule-time' options are mutually exclusive with '--schedule'"))

			By("with schedule and schedule-time")
			_, err = machinePoolUpgradeService.CreateAutomaticUpgrade(
				clusterID,
				nodePoolAutoName,
				"2 5 * * *",
				"--schedule-time", "10:34")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring(
					"The '--schedule-date' and '--schedule-time' options are mutually exclusive with '--schedule'"))

			By("with already existing upgrade")
			availableUpgradeVersions := helper.ParseCommaSeparatedStrings(upgradableVersion.AvailableUpgrades)
			Expect(len(availableUpgradeVersions)).NotTo(BeEquivalentTo(0))

			_, err = machinePoolUpgradeService.CreateManualUpgrade(
				clusterID,
				nodePoolAutoName,
				availableUpgradeVersions[0],
				"",
				"")
			Expect(err).ToNot(HaveOccurred())
			defer machinePoolUpgradeService.DeleteUpgrade(clusterID, nodePoolAutoName)

			output, err = machinePoolUpgradeService.CreateManualUpgrade(
				clusterID,
				nodePoolAutoName,
				availableUpgradeVersions[0],
				"",
				"")
			Expect(err).ToNot(HaveOccurred())
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("WARN: There is already a"))
		})

		It("will validate root volume size - [id:66874]",
			labels.Runtime.Day2, labels.Medium, labels.FedRAMP,
			func() {
				npName := helper.GenerateRandomName("np-66874", 2)
				By("Create with too small disk size will fail")
				output, err := rosaClient.MachinePool.CreateMachinePool(clusterID, npName,
					"--replicas", "0",
					"--disk-size", "2GiB",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring(
					fmt.Sprintf(constants.DiskSizeErrRangeMsg, 2, constants.MinHCPDiskSize, constants.MaxDiskSize)))

				By("Create with large disk size will fail")
				output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, npName,
					"--replicas", "0",
					"--disk-size", "17594GB",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring(
					fmt.Sprintf(
						constants.DiskSizeErrRangeMsg,
						16385, // 17594GB --> 16385GiB
						constants.MinHCPDiskSize,
						constants.MaxDiskSize)))

				By("Create with un-known unit will fail")
				output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, npName,
					"--replicas", "0",
					"--disk-size", "2GiiiB",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("invalid disk size format: '2GiiiB'." +
					" accepted units are Giga or Tera in the form of g, G, GB, GiB, Gi, t, T, TB, TiB, Ti"))

				By("Create with too large value will fail")
				output, err = rosaClient.MachinePool.CreateMachinePool(clusterID, npName,
					"--replicas", "0",
					"--disk-size", "25678987654567898765456789087654GiB",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("invalid disk size: '25678987654567898765456789087654Gi'." +
					" maximum size exceeded"))
			})

		It("validate maximum number of nodes - [id:78277]", labels.Medium, labels.Runtime.Day2, labels.FedRAMP, func() {
			By("Prepare testing machinepool")
			instanceType := constants.M5XLarge
			mpPrefix := "mp78277na"
			autoScaleMpPrefix := "mp78277a"
			mpName := helper.GenerateRandomName(mpPrefix, 2)
			autoScaleMpName := helper.GenerateRandomName(autoScaleMpPrefix, 2)
			minReplica := 3
			maxReplica := 6
			testReplicas := 3
			_, err := rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
				"--instance-type", instanceType,
				"--replicas", fmt.Sprintf("%v", testReplicas),
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

			_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, autoScaleMpName,
				"--instance-type", instanceType,
				"--enable-autoscaling",
				"--min-replicas", fmt.Sprintf("%v", minReplica),
				"--max-replicas", fmt.Sprintf("%v", maxReplica),
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, autoScaleMpName)

			By("Edit node pool mini and max replicas with the numner execceed maximum number")
			out, err := rosaClient.MachinePool.EditMachinePool(clusterID, autoScaleMpName,
				"--min-replicas", fmt.Sprintf("%v", 6000),
				"--max-replicas", fmt.Sprintf("%v", 6001),
				"-y",
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).To(ContainSubstring("Replicas+Autoscaling.Min: The total number of compute nodes"))
			Expect(out.String()).To(ContainSubstring("Reduce the total compute nodes requested to be within the maximum allowed"))

			out, err = rosaClient.MachinePool.EditMachinePool(clusterID, autoScaleMpName,
				"--min-replicas", fmt.Sprintf("%v", 1),
				"--max-replicas", fmt.Sprintf("%v", 6001),
				"-y",
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).To(ContainSubstring(" Replicas+Autoscaling.Max: The total number of compute nodes"))
			Expect(out.String()).To(ContainSubstring("Reduce the total compute nodes requested to be within the maximum allowed"))

			By("Edit node pool replicas with the numner execceed maximum number")
			out, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--replicas", fmt.Sprintf("%v", 6001),
				"-y",
			)
			Expect(out.String()).To(ContainSubstring("Reduce the total compute nodes requested to be within the maximum allowed"))

			By("Create node pool mini and max replicas with the numner execceed maximum number")

			mpNameNegative := helper.GenerateRandomName("mp78277n", 2)
			out, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpNameNegative,
				"--instance-type", instanceType,
				"--enable-autoscaling",
				"--min-replicas", fmt.Sprintf("%v", 1),
				"--max-replicas", fmt.Sprintf("%v", 6000),
				"-y",
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).To(ContainSubstring("should provide an integer number less than or equal to"))

			out, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpNameNegative,
				"--instance-type", instanceType,
				"--enable-autoscaling",
				"--min-replicas", fmt.Sprintf("%v", 6000),
				"--max-replicas", fmt.Sprintf("%v", 6001),
				"-y",
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).To(ContainSubstring("should provide an integer number less than or equal to"))

			By("Create node pool replicas with the numner execceed maximum number")
			out, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpNameNegative,
				"--instance-type", instanceType,
				"--replicas", fmt.Sprintf("%v", 6000),
				"-y",
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).To(ContainSubstring("should provide an integer number less than or equal to"))
		})
	})
})
