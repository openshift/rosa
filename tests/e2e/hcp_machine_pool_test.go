package e2e

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	ph "github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("HCP Machine Pool", labels.Feature.Machinepool, func() {
	// It doesn't check whether node pool instances ready in default.
	// If needed for verify hcp node pool's changes, pls set the ENV CLUSTER_NODE_POOL_GLOBAL_CHECK to true,
	// which will wait for node pool instances ready until timeout.
	isNodePoolGlobalCheck := config.IsNodePoolGlobalCheck()

	var (
		rosaClient         *rosacli.Client
		machinePoolService rosacli.MachinePoolService
		profile            *ph.Profile
		isMultiArch        bool
	)

	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()
		machinePoolService = rosaClient.MachinePool

		By("Skip testing if the cluster is not a HCP cluster")
		hostedCluster, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())

		By("Check whether the cluster is multi arch")
		isMultiArch, err = rosaClient.Cluster.IsMultiArch(clusterID)
		Expect(err).ToNot(HaveOccurred())

		profile = ph.LoadProfileYamlFileByENV()

		if !hostedCluster {
			SkipNotHosted()
		}
	})

	Describe("Create/delete/view a machine pool", func() {
		It("should succeed with additional security group IDs [id:72195]", labels.Critical, labels.Runtime.Day2, func() {
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
			sgPrefix := common.GenerateRandomName("72195", 2)
			sgIDs, err := vpcClient.CreateAdditionalSecurityGroups(3, sgPrefix, "testing for case 72195")
			Expect(err).ToNot(HaveOccurred())

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

	It("machinepool AWS preflight tag validation[id:73638]",
		labels.Medium, labels.Runtime.Day2,
		func() {

			By("Check the help message of machinepool creation")
			mpID := common.GenerateRandomName("mp-73638", 2)
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

	DescribeTable("Scale up/down a machine pool", labels.Critical, labels.Runtime.Day2,
		func(instanceType string, amdOrArm string) {
			if !isMultiArch && amdOrArm == constants.ARM {
				SkipNotMultiArch()
			}

			By("Create machinepool with " + amdOrArm + " instance " + instanceType)
			mpPrefix := fmt.Sprintf("%v-60278", amdOrArm)
			mpName := common.GenerateRandomName(mpPrefix, 2)
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
				err = rosaClient.MachinePool.WaitNodePoolReplicasReady(
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

	DescribeTable("Scale up/down a machine pool with invalid replica", labels.Critical, labels.Runtime.Day2,
		func(instanceType string, updatedReplicas string, expectedErrMsg string) {
			By("Create machinepool with instance " + instanceType)
			mpName := common.GenerateRandomName("mp-60278", 2)
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
		Entry("Scale replica to -1 [id:60278]", constants.M52XLarge, "-1", "Replicas must be a non-negative number"),
		Entry("Scale replica to a char [id:60278]", constants.M52XLarge, "a", "invalid syntax"),
	)

	Describe("Scale up/down a machine pool enabling autoscale", func() {
		It("should succeed to scale with valid parameters [id:60278]", labels.Medium, labels.Runtime.Day2, func() {
			instanceType := constants.M52XLarge
			By("Create machinepool with " + " instance " + instanceType + " and enable autoscale")

			mpPrefix := "autoscale"
			mpName := common.GenerateRandomName(mpPrefix, 2)
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
				err = rosaClient.MachinePool.WaitNodePoolReplicasReady(
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

		It("should raise error message with the invalid parameters [id:60278]", labels.Critical, labels.Runtime.Day2, func() {
			instanceType := constants.M52XLarge
			By("Create machinepool with" + " instance " + instanceType + " and enable autoscale")

			mpPrefix := "autoscale"
			mpName := common.GenerateRandomName(mpPrefix, 2)
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
			expectErrMsg := "The number of machine pool min-replicas needs to be a non-negative integer"
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))

			By("Scale up a machine pool max replica too large")
			moreMaxReplica := 1000
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--max-replicas", fmt.Sprintf("%v", moreMaxReplica),
				"-y",
			)
			expectErrMsg = "exceeds the maximum allowed"
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))

			By("Scale a machine pool min replica > max replica")
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--min-replicas", fmt.Sprintf("%v", "5"),
				"--max-replicas", fmt.Sprintf("%v", "3"),
				"-y",
			)
			expectErrMsg = "min-replicas needs to be less than the number of machine pool max-replicas"
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))

			By("Scale down a machine pool min replica to -1")
			downMinReplica := -1
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--min-replicas", fmt.Sprintf("%v", downMinReplica),
				"-y",
			)
			expectErrMsg = "Min replicas must be a non-negative number when autoscaling is set"
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))

			By("Scale a machine pool with min replica and max replica a char")
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
				"--min-replicas", fmt.Sprintf("%v", "a"),
				"--max-replicas", fmt.Sprintf("%v", "b"),
				"-y",
			)
			expectErrMsg = "invalid syntax"
			Expect(err.Error()).Should(ContainSubstring(expectErrMsg))
		})
	})
})
