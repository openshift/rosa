package e2e

import (
	"fmt"
	"path"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	utilConfig "github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/occli"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("Cluster preparation", labels.Feature.Cluster, func() {
	It("by profile",
		labels.Runtime.Day1,
		labels.Critical,
		func() {
			client := rosacli.NewClient()
			profile := handler.LoadProfileYamlFileByENV()
			clusterHandler, err := handler.NewClusterHandler(client, profile)
			Expect(err).ToNot(HaveOccurred())
			err = clusterHandler.CreateCluster(config.Test.GlobalENV.WaitSetupClusterReady)
			Expect(err).ToNot(HaveOccurred())
			clusterID := clusterHandler.GetClusterDetail().ClusterID
			log.Logger.Infof("Cluster prepared successfully with id %s", clusterID)

		})

	It("to wait for cluster ready",
		labels.Runtime.Day1Readiness,
		func() {
			profile := handler.LoadProfileYamlFileByENV()
			client := rosacli.NewClient()
			clusterHandler, err := handler.NewClusterHandlerFromFilesystem(client, profile)
			Expect(err).ToNot(HaveOccurred())
			err = clusterHandler.WaitForClusterReady(config.Test.GlobalENV.ClusterWaitingTime)
			Expect(err).ToNot(HaveOccurred())

			clusterID := clusterHandler.GetClusterDetail().ClusterID
			clusterService := client.Cluster
			output, err := clusterService.DescribeCluster(clusterID)
			Expect(err).To(BeNil())
			clusterDetails, err := clusterService.ReflectClusterDescription(output)
			Expect(err).To(BeNil())

			// Workaround for public HCP with proxy
			if profile.ClusterConfig.HCP && profile.ClusterConfig.ProxyEnabled && !profile.ClusterConfig.Private {
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())
				clusterNoProxy := jsonData.DigString("proxy", "no_proxy")
				clusterDNS := fmt.Sprintf("%s.%s", clusterDetails.Name, jsonData.DigString("dns", "base_domain"))
				_, err = clusterService.EditCluster(clusterID, "--no-proxy",
					fmt.Sprintf("%s,.%s", clusterNoProxy, clusterDNS))
				Expect(err).To(BeNil())
			}
			// For HCP cluster with other network type,it is required to set one configure:cilium
			if profile.ClusterConfig.HCP && profile.ClusterConfig.NetworkType == "other" {
				if clusterDetails.ExternalAuthentication == "Enabled" {
					By("Create a fake external auth provider to avoid failure of console operator")
					secretValue := fmt.Sprintf("%s~%s~%s",
						helper.GenerateRandomStringWithSymbols(5),
						helper.GenerateRandomStringWithSymbols(10),
						helper.GenerateRandomStringWithSymbols(23))
					value := []string{
						"--name", helper.GenerateRandomName("provider", 2),
						"--issuer-url", "https://login.microsoftonline.com/fa5d3dd8-b8ec-4407-a55c-ced639f1c8c5/v2.0",
						"--issuer-audiences", "8a769b34-13c9-4f5b-9933-ec439700ec67",
						"--claim-mapping-username-claim", "email",
						"--claim-mapping-groups-claim", "groups",
						"--console-client-id", "8a769b34-13c9-4f5b-9933-ec439700ec67",
						"--console-client-secret", secretValue,
					}
					_, err = client.ExternalAuthProvider.CreateExternalAuthProvider(clusterID, value...)
					Expect(err).ToNot(HaveOccurred())

					// it is not support to create htpasswd for cluster with xternal auth enabled
					// create break-glass-credential to get kubeconfig
					By("Create a break glass credential")
					_, err := client.BreakGlassCredential.CreateBreakGlassCredential(clusterID)
					Expect(err).To(BeNil())
					breakGlassCredList, err := client.BreakGlassCredential.ListBreakGlassCredentialsAndReflect(clusterID)
					Expect(err).To(BeNil())
					kubeconfigFile := path.Join(config.Test.OutputDir, "kubeconfig")

					By("Get the issued credential")
					for _, i := range breakGlassCredList.BreakGlassCredentials {
						Eventually(
							client.BreakGlassCredential.WaitForBreakGlassCredentialToStatus(
								clusterID,
								"issued",
								i.Username),
							time.Minute*1,
						).Should(BeTrue())
						output, err := client.BreakGlassCredential.GetIssuedCredential(clusterID, i.ID)
						Expect(err).ToNot(HaveOccurred())
						Expect(output.String()).ToNot(BeEmpty())
						_, err = helper.CreateFileWithContent(kubeconfigFile, output.String())
						Expect(err).ToNot(HaveOccurred())
						break
					}
					hostPrefix, podCIDR := "", ""
					for _, networkLine := range clusterDetails.Network {
						if value, containsKey := networkLine["Host Prefix"]; containsKey {
							hostPrefix = value[1:]
						}
						if value, containsKey := networkLine["Pod CIDR"]; containsKey {
							podCIDR = value
						}
					}
					By("Deploy cilium configures")
					ocClient, err := occli.NewOCClient(kubeconfigFile)
					Expect(err).ToNot(HaveOccurred())
					err = utilConfig.DeployCilium(ocClient, podCIDR, hostPrefix,
						config.Test.OutputDir, kubeconfigFile)
					Expect(err).ToNot(HaveOccurred())
					log.Logger.Infof("Deploy cilium for HCP cluster: %s successfully ", clusterID)
				} else {
					By("Create IDP to get kubeconfig")
					idpType := "htpasswd"
					idpName := "myhtpasswdKubeconf"
					name, password := clusterHandler.GetResourcesHandler().PrepareAdminUser()
					usersValue := name + ":" + password
					_, err := client.IDP.CreateIDP(clusterID, idpName,
						"--type", idpType,
						"--users", usersValue,
						"-y")
					Expect(err).ToNot(HaveOccurred())

					_, err = client.User.GrantUser(clusterID, "dedicated-admins", name)
					Expect(err).ToNot(HaveOccurred())

					helper.CreateFileWithContent(config.Test.ClusterIDPAdminUsernamePassword, usersValue)
				}
			}

		})

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

			if profile.ClusterConfig.BYOVPC == false {
				Skip("This day2 config only work for byovpc cluster")
			}

			//Create local zone for cluster
			if profile.Day2Config != nil && profile.Day2Config.LocalZoneMP {
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
				vpcClient.AWSClient.TagResource(privateSubnet.ID, map[string]string{
					tagKey: "shared",
				})

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

				defer client.OCMResource.DeleteAccountRole(
					"--prefix", namePrefix,
					"--mode", "auto",
					"-y")

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
			}
		})
})
