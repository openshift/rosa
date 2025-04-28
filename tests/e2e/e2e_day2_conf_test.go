package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/tests/ci/labels"
	utilConfig "github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("Cluster Day2 preparation", labels.Feature.Cluster, func() {

	It("to prepare day2 conf for cluster",
		labels.Runtime.Day2Readiness,
		func() {
			profile := handler.LoadProfileYamlFileByENV()
			client := rosacli.NewClient()
			clusterHandler, err := handler.NewClusterHandlerFromFilesystem(client, profile)
			Expect(err).ToNot(HaveOccurred())

			clusterID := clusterHandler.GetClusterDetail().ClusterID
			clusterService := client.Cluster
			output, err := clusterService.DescribeCluster(clusterID)
			Expect(err).To(BeNil())
			clusterDetails, err := clusterService.ReflectClusterDescription(output)
			Expect(err).To(BeNil())

			clusterConfig, err := utilConfig.ParseClusterProfile()
			Expect(err).To(BeNil())

			//Edit cluster default autoscaler for HCP cluster
			if profile.Day2Config != nil && profile.Day2Config.ClusterAutoScaler && profile.ClusterConfig.HCP {
				By("Edit the autoscaler with custom values")
				podPriorityThreshold := "1000"
				maxNodeProvisionTime := "50m"
				maxPodGracePeriod := "700"
				maxNodesTotal := "100"
				_, err = client.AutoScaler.EditAutoScaler(clusterID,
					"--pod-priority-threshold", podPriorityThreshold,
					"--max-node-provision-time", maxNodeProvisionTime,
					"--max-pod-grace-period", maxPodGracePeriod,
					"--max-nodes-total", maxNodesTotal,
				)
				Expect(err).ToNot(HaveOccurred())
				log.Logger.Infof("Update cluster autoscaler successfully")
			}

			//Create machinepool with local zone for cluster
			if profile.Day2Config != nil && profile.Day2Config.LocalZoneMP && !profile.ClusterConfig.HCP {
				if profile.ClusterConfig.BYOVPC == false {
					Skip("This day2 config only work for byovpc cluster")
				}
				By("Prepare a subnet out of the cluster creation subnet")
				subnets := helper.ParseCommaSeparatedStrings(clusterConfig.Subnets.PrivateSubnetIds)

				By("Build vpc client to find a local zone for subnet preparation")
				vpcClient, err := vpc_client.GenerateVPCBySubnet(subnets[0], clusterConfig.Region)
				Expect(err).ToNot(HaveOccurred())

				zones, err := vpcClient.AWSClient.ListAvaliableZonesForRegion(clusterConfig.Region, "local-zone")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(zones)).ToNot(BeZero(), "No local zone found in the region")
				localZone := zones[0]

				By("Prepare the subnet for the picked zone")
				subNetMap, err := vpcClient.PreparePairSubnetByZone(localZone)
				Expect(err).ToNot(HaveOccurred())
				Expect(subNetMap).ToNot(BeNil())
				privateSubnet := subNetMap["private"]

				By("Describe the cluster to get the infra ID for tagging")
				tagKey := fmt.Sprintf("kubernetes.io/cluster/%s", clusterDetails.InfraID)
				_, err = vpcClient.AWSClient.TagResource(privateSubnet.ID, map[string]string{
					tagKey: "shared",
				})
				Expect(err).ToNot(HaveOccurred())

				By("Find a machinetype supported by the zone")
				instanceTypes, err := vpcClient.AWSClient.ListAvaliableInstanceTypesForRegion(
					clusterConfig.Region, localZone)
				Expect(err).ToNot(HaveOccurred())

				By("Create temporary account-roles for instance type list")
				namePrefix := helper.GenerateRandomName("test-day2-conf", 2)
				majorVersion := helper.SplitMajorVersion(clusterConfig.Version.RawID)
				_, err = client.OCMResource.CreateAccountRole("--mode", "auto",
					"--prefix", namePrefix,
					"--version", majorVersion,
					"--channel-group", clusterConfig.Version.ChannelGroup,
					"-y")
				Expect(err).ToNot(HaveOccurred())

				var accountRoles *rosacli.AccountRolesUnit
				accRoleList, _, err := client.OCMResource.ListAccountRole()
				Expect(err).ToNot(HaveOccurred())
				accountRoles = accRoleList.DigAccountRoles(namePrefix, false)

				defer func() {
					_, err = client.OCMResource.DeleteAccountRole(
						"--prefix", namePrefix,
						"--mode", "auto",
						"-y")
					Expect(err).ToNot(HaveOccurred())
				}()

				rosaSupported, _, err := client.OCMResource.ListInstanceTypes(
					"--region", clusterConfig.Region,
					"--role-arn", accountRoles.InstallerRole,
				)
				Expect(err).ToNot(HaveOccurred())

				bothSupported := []string{}
				for _, rosains := range rosaSupported.InstanceTypesList {
					if helper.SliceContains(instanceTypes, rosains.ID) {
						bothSupported = append(bothSupported, rosains.ID)
					}
				}
				Expect(bothSupported).ToNot(BeEmpty(), "There are no instance types supported in the zone")
				instanceType := bothSupported[0]

				By("Create machinepool with the subnet specified will succeed")
				localZoneMpName := "localz-day2-conf"
				_, err = client.MachinePool.CreateMachinePool(clusterID, localZoneMpName,
					"--replicas", "1",
					"--subnet", privateSubnet.ID,
					"--instance-type", instanceType,
					"--labels", "prowci=true,node-role.kubernetes.io/edge=",
					"--taints", "prowci=true:NoSchedule,node-role.kubernetes.io/edge=:NoSchedule",
				)
				Expect(err).ToNot(HaveOccurred())

				By("List the machinepools and check")
				mpList, err := client.MachinePool.ListAndReflectMachinePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				mp := mpList.Machinepool(localZoneMpName)
				Expect(mp.Replicas).To(Equal("1"))
				Expect(mp.Subnets).To(Equal(privateSubnet.ID))
				log.Logger.Infof("Create machine pool with local zone successfully")
			}
		})
})
