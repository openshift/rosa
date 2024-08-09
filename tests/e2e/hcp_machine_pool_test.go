package e2e

import (
	"fmt"
	"strings"

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
		rosaClient  *rosacli.Client
		profile     *ph.Profile
		isMultiArch bool
	)

	checkReplicas := func(mpName string, isAutoscale bool) bool {
		if isAutoscale {
			replicas, err := rosaClient.MachinePool.GetNodePoolAutoScaledReplicas(clusterID, mpName)
			if err != nil {
				return false
			}
			return replicas["Current replicas"] == replicas["Min replicas"]

		} else {
			mpDesc, err := rosaClient.MachinePool.DescribeAndReflectNodePool(clusterID, mpName)
			if err != nil {
				return false
			}
			return mpDesc.CurrentReplicas == fmt.Sprintf("%v", mpDesc.DesiredReplicas)
		}
	}

	scaleNodePool := func(mpName string, scaleFlag string, updateReplicas int, waitForNPInstancesReady bool) {
		_, err := rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
			"--replicas", fmt.Sprintf("%v", updateReplicas),
			"-y",
		)
		Expect(err).ToNot(HaveOccurred())

		By("Check the machinepool replicas after scale " + scaleFlag)
		mpDesc, err := rosaClient.MachinePool.DescribeAndReflectNodePool(clusterID, mpName)
		Expect(err).ToNot(HaveOccurred())
		Expect(mpDesc.DesiredReplicas).Should(Equal(updateReplicas))

		if waitForNPInstancesReady && isNodePoolGlobalCheck {
			By("Check current replicas reach the desired replicas after scale " + scaleFlag)
			Eventually(checkReplicas).
				WithArguments(mpName, false).
				WithTimeout(constants.NodePoolCheckTimeout).
				WithPolling(constants.NodePoolCheckPoll).
				Should(BeTrue())
		}

	}

	scaleAutoScaledNodePool := func(
		mpName string, scaleFlag string, minReplicas int, maxReplicas int, waitForNPInstancesReady bool,
	) {
		_, err := rosaClient.MachinePool.EditMachinePool(clusterID, mpName,
			"--enable-autoscaling",
			"--min-replicas", fmt.Sprintf("%v", minReplicas),
			"--max-replicas", fmt.Sprintf("%v", maxReplicas),
			"-y",
		)
		Expect(err).ToNot(HaveOccurred())

		By("Check the machinepool min_replica and max_replica after scale" + scaleFlag)
		desiredReplicas, err := rosaClient.MachinePool.GetNodePoolAutoScaledReplicas(clusterID, mpName)
		Expect(err).ToNot(HaveOccurred())
		Expect(desiredReplicas["Min replicas"]).Should(Equal(minReplicas))
		Expect(desiredReplicas["Max replicas"]).Should(Equal(maxReplicas))

		if waitForNPInstancesReady && isNodePoolGlobalCheck {
			By("Check current replicas reach the min_replica in desired replicas after scale " + scaleFlag)
			Eventually(checkReplicas).
				WithArguments(mpName, true).
				WithTimeout(constants.NodePoolCheckTimeout).
				WithPolling(constants.NodePoolCheckPoll).
				Should(BeTrue())
		}
	}

	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()

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
		When("with additional security group IDs", func() {
			It("should succeed to create/delete/view the machine pool[id:72195]", labels.Critical, labels.Runtime.Day2, func() {
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
				_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
					"--additional-security-group-ids", strings.Join(sgIDs, ","),
					"--replicas", "1",
					"-y",
				)
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
				By("Check current replicas reach the desired replicas after creating a machine pool")
				Eventually(checkReplicas).
					WithArguments(mpName, false).
					WithTimeout(constants.NodePoolCheckTimeout).
					WithPolling(constants.NodePoolCheckPoll).
					Should(BeTrue())
			}

			By("Scale a machine pool with unchanged parameters")
			scaleNodePool(mpName, "unchanged", desiredReplicas, true)

			By("Scale up a machine pool replicas from 1 to 2")
			upReplicas := 2
			scaleNodePool(mpName, "up", upReplicas, true)

			By("Scale down a machine pool replicas from 2 to 1")
			downReplicas := 1
			scaleNodePool(mpName, "down", downReplicas, true)

			By("Scale down a machine pool replicas to 0")
			zeroReplica := 0
			scaleNodePool(mpName, "down", zeroReplica, true)
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
		When("with the valid min replica and max replica", func() {
			It("should succeed to scale up/down the machine pool[id:60278]", labels.Medium, labels.Runtime.Day2, func() {
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

				// TODO There's issue here https://issues.redhat.com/browse/OCM-10147
				//By("Scale a machine pool with unchanged parameters")
				//scaleAutoScaledNodePool(mpName, "unchanged", minReplica, maxReplica, true)

				By("Scale up a machine pool replicas from 1~3 to 2~5")
				upMinReplica := 2
				upMaxReplica := 5
				scaleAutoScaledNodePool(mpName, "up", upMinReplica, upMaxReplica, true)

				// Don't check the current replicas when scale down, because it won't change after reducing min_replica
				// It only depends on the autoscale strategy when reducing min_replica
				By("Scale down a machine pool replicas from 2~5 to 1~2")
				downMinReplica := 1
				downMaxReplica := 2
				scaleAutoScaledNodePool(mpName, "down", downMinReplica, downMaxReplica, false)
			})
		})

		When("with the invalid min replica and max replica", func() {
			It("should raise error message[id:60278]", labels.Critical, labels.Runtime.Day2, func() {
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
})
