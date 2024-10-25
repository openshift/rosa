package e2e

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ciConfig "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("HCP cluster testing",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()

		var (
			clusterID          string
			rosaClient         *rosacli.Client
			clusterService     rosacli.ClusterService
			clusterConfig      *config.ClusterConfig
			profile            *profilehandler.Profile
			machinePoolService rosacli.MachinePoolService
			ocmResourceService rosacli.OCMResourceService
			customProfile      *profilehandler.Profile
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			profile = profilehandler.LoadProfileYamlFileByENV()
			var err error
			clusterConfig, err = config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())
			machinePoolService = rosaClient.MachinePool
			ocmResourceService = rosaClient.OCMResource

			By("Skip testing if the cluster is not a HCP cluster")
			hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if !hostedCluster {
				SkipNotHosted()
			}

			By("Prepare custom profile")
			customProfile = &profilehandler.Profile{
				ClusterConfig: &profilehandler.ClusterConfig{
					HCP:           true,
					MultiAZ:       true,
					STS:           true,
					OIDCConfig:    "managed",
					NetworkingSet: true,
					BYOVPC:        true,
					Zones:         "",
				},
				AccountRoleConfig: &profilehandler.AccountRoleConfig{
					Path:               "",
					PermissionBoundary: "",
				},
				Version:      "latest",
				ChannelGroup: "candidate",
				Region:       constants.CommonAWSRegion,
			}
			customProfile.NamePrefix = constants.DefaultNamePrefix
		})

		AfterEach(func() {
			By("Clean the cluster")
			rosaClient.CleanResources(clusterID)
		})

		It("create and edit hosted-cp cluster with AuditLog Forwarding enabled/disabled via rosacli - [id:64491]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Get cluster description")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				if clusterConfig.AuditLogArn == "" {
					SkipTestOnFeature("audit log")
				}
				role := clusterDetail.AuditLogRoleARN
				Expect(clusterConfig.AuditLogArn).To(Equal(role))
				Expect(role).ToNot(BeEmpty())

				By("Edit the cluster to disable audit log forwarding")
				_, err = clusterService.EditCluster(
					clusterID,
					"--audit-log-arn", "",
					"-y",
				)
				Expect(err).To(BeNil())

				By("Get cluster description")
				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				_, err = clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				By("Edit the cluster to enable audit log forwarding")
				_, err = clusterService.EditCluster(
					clusterID,
					"--audit-log-arn", role,
					"-y",
				)
				Expect(err).To(BeNil())

				By("Get cluster description")
				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetail, err = clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				Expect(clusterDetail.AuditLogForwarding).To(Equal("Enabled"))
				Expect(role).To(Equal(role))

			})

		It("create cluster with the KMS and etcd encryption for hypershift clusters by rosa-cli - [id:60083]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check the help message of 'rosa create cluster -h'")
				output, err, _ := clusterService.Create("", "-h")
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("--kms-key-arn"))
				Expect(output.String()).To(ContainSubstring("--etcd-encryption"))
				Expect(output.String()).To(ContainSubstring("--enable-customer-managed-key"))

				By("Get cluster description")
				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				if clusterConfig.EtcdEncryption {
					Expect(clusterDetail.EnableEtcdEncryption).To(Equal("Enabled"))
					Expect(clusterDetail.EtcdKmsKeyARN).To(Equal(clusterConfig.Encryption.EtcdEncryptionKmsArn))
				} else {
					Expect(clusterDetail.EnableEtcdEncryption).To(Equal("Disabled"))
				}

				By("Get cluster description in JSON format")
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())

				enableEtcdEncryption := jsonData.DigBool("etcd_encryption")
				Expect(clusterConfig.EtcdEncryption).To(Equal(enableEtcdEncryption))

				ectdKMS := jsonData.DigString("aws", "etcd_encryption", "kms_key_arn")
				npKMS := jsonData.DigString("aws", "kms_key_arn")

				if clusterConfig.EtcdEncryption {
					Expect(clusterConfig.Encryption.EtcdEncryptionKmsArn).To(Equal(ectdKMS))
				}
				if clusterConfig.EnableCustomerManagedKey {
					Expect(clusterConfig.Encryption.KmsKeyArn).To(Equal(npKMS))
				}

			})

		It("create HCP cluster with network type can work well via rosa cli - [id:71050]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check the help message of 'rosa create cluster -h'")
				//It is hiddened now
				helpOutput, err, _ := clusterService.Create("", "-h")
				Expect(err).To(BeNil())
				Expect(helpOutput.String()).To(ContainSubstring("--no-cni"))

				By("Get cluster description")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				networkLine := clusterDetail.Network[0]

				By("Get cluster description via json")
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())
				Expect(networkLine["Type"]).To(Equal(jsonData.DigString("network", "type")))
				if clusterConfig.Networking != nil {
					networkType := clusterConfig.Networking.Type
					if networkType != "" && networkType == "Other" {
						Expect(networkLine["Type"]).To(Equal("Other"))
					}
				}
			})

		It("create ROSA HCP cluster with external_auth_config config should work well via rosa client - [id:71945]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check the help message of 'rosa create cluster -h'")
				helpOutput, err, _ := clusterService.Create("", "-h")
				Expect(err).To(BeNil())
				Expect(helpOutput.String()).To(ContainSubstring("--external-auth-providers-enabled"))

				By("Check if cluster enable external_auth_config")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				if !clusterConfig.ExternalAuthentication {
					Skip("It is only for external_auth_config enabled clusters")
				}
				Expect(clusterDetail.ExternalAuthentication).To(Equal("Enabled"))

				By("Check some cmds that are not supportted")
				output, err = rosaClient.User.CreateAdmin(clusterID)
				Expect(err).ToNot(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"ERR: Creating the 'cluster-admin' user is not supported for clusters with external authentication configured"))

				_, output, err = rosaClient.IDP.ListIDP(clusterID)
				Expect(err).ToNot(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"ERR: Listing identity providers is not supported for clusters with external authentication configured"))
			})

		It("can edit ROSA HCP cluster with additional allowed principals - [id:74556]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check the help message of 'rosa edit cluster -h'")
				helpOutput, err := clusterService.EditCluster("", "-h")
				Expect(err).To(BeNil())
				Expect(helpOutput.String()).To(ContainSubstring("--additional-allowed-principals"))

				By("Check if cluster profile is enabled with additional allowed principals")
				if !profile.ClusterConfig.AdditionalPrincipals {
					SkipTestOnFeature("additional allowed principals")
				}

				output, err := clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.AdditionalPrincipals).To(ContainSubstring(clusterConfig.AdditionalPrincipals))

				By("Get the installer role arn")
				rosaClient.Runner.JsonFormat()
				jsonOutput, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()
				installRoleArn := jsonData.DigString("aws", "sts", "role_arn")

				By("Get additional principal credentials")
				awsSharedCredentialFile := ciConfig.Test.GlobalENV.SVPC_CREDENTIALS_FILE

				By("Create additional account roles")
				accrolePrefix := "arPrefix74556"

				additionalPrincipalRoleName := fmt.Sprintf("%s-%s", accrolePrefix, "additional-principal-role")
				additionalPrincipalRoleArn, err := profilehandler.PrepareAdditionalPrincipalsRole(
					additionalPrincipalRoleName,
					installRoleArn,
					profile.Region, awsSharedCredentialFile)
				Expect(err).To(BeNil())
				defer func() {
					By("Delete the additional principal account-roles")
					err = profilehandler.DeleteAdditionalPrincipalsRole(additionalPrincipalRoleName,
						true, profile.Region, awsSharedCredentialFile)
					Expect(err).To(BeNil())
				}()

				additionalPrincipalsFlag := fmt.Sprintf(
					"%s,%s", clusterConfig.AdditionalPrincipals, additionalPrincipalRoleArn)

				By("Edit the cluster with additional allowed principals")
				out, err := clusterService.EditCluster(clusterID,
					"--additional-allowed-principals",
					additionalPrincipalsFlag)
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
				Expect(textData).To(ContainSubstring("Updated cluster '%s'", clusterID))

				By("Confirm additional principals is edited successfully")
				output, err = clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).To(BeNil())
				Expect(output.AdditionalPrincipals).
					To(
						ContainSubstring(
							"%s,%s", clusterConfig.AdditionalPrincipals, additionalPrincipalRoleArn))

				By("Edit the cluster with additional allowed principals")
				out, err = clusterService.EditCluster(clusterID,
					"--additional-allowed-principals",
					clusterConfig.AdditionalPrincipals)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
				Expect(textData).To(ContainSubstring("Updated cluster '%s'", clusterID))
			})

		It("rosacli can show the details of HCP cluster well when describe - [id:54869]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				By("Get cluster description")
				clusterDesc, err := clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).To(BeNil())

				By("Check values of description")
				Expect(clusterDesc.Name).ToNot(BeEmpty())
				Expect(clusterDesc.ID).ToNot(BeEmpty())
				Expect(clusterDesc.ControlPlane).To(Equal("ROSA Service Hosted"))
				Expect(clusterDesc.OpenshiftVersion).ToNot(BeEmpty())
				Expect(clusterDesc.DNS).ToNot(BeEmpty())
				Expect(clusterDesc.APIURL).ToNot(BeEmpty())
				Expect(clusterDesc.Region).ToNot(BeEmpty())
				Expect(clusterDesc.Availability[0]).To(HaveKeyWithValue("Control Plane", MatchRegexp("MultiAZ")))

				By("List nodepools")
				npList, err := machinePoolService.ListAndReflectNodePools(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if len(npList.NodePools) == 1 {
					Expect(clusterDesc.Availability[1]).To(HaveKeyWithValue("Data Plane", MatchRegexp("SingleAZ")))
				} else {
					Expect(clusterDesc.Availability[1]).To(HaveKeyWithValue("Data Plane", MatchRegexp("MultiAZ")))
				}

				Expect(clusterDesc.Nodes).To(HaveEach(HaveKey(MatchRegexp("Compute.*"))))
				for _, v := range clusterDesc.Nodes {
					for _, value := range v {
						switch value.(type) {
						case int:
							Expect(value).To(BeNumerically(">=", 0))
						case string:
							Expect(value).To(MatchRegexp("^[0-9]+-[0-9]+$"))
						default:
							Expect(value).To(BeNil())
						}
					}
				}
				Expect(clusterDesc.Network).To(ContainElements(HaveKey("Type"), HaveKey("Service CIDR"), HaveKey("Machine CIDR"),
					HaveKey("Pod CIDR"), HaveKey("Host Prefix"), HaveKeyWithValue("Subnets", MatchRegexp("^subnet-.{17}"))))
				Expect(clusterDesc.STSRoleArn).To(MatchRegexp("arn:aws:iam::[0-9]{12}:role/.+-HCP-ROSA-Installer-Role"))
				Expect(clusterDesc.SupportRoleARN).To(MatchRegexp("arn:aws:iam::[0-9]{12}:role/.+-HCP-ROSA-Support-Role"))
				Expect(clusterDesc.InstanceIAMRoles[0]).To(HaveKeyWithValue("Worker",
					MatchRegexp("arn:aws:iam::[0-9]{12}:role/.+-HCP-ROSA-Worker-Role")))

				By("List Operator roles")
				roles, err := ocmResourceService.ListOperatorRoles("--prefix", clusterConfig.Aws.Sts.OperatorRolesPrefix)
				Expect(err).ToNot(HaveOccurred())
				operRolesList, err := ocmResourceService.ReflectOperatorRoleList(roles)
				Expect(err).ToNot(HaveOccurred())
				for _, role := range operRolesList.OperatorRoleList {
					Expect(clusterDesc.OperatorIAMRoles).To(ContainElement(ContainSubstring(role.RoleName)))
				}

				Expect(clusterDesc.OperatorIAMRoles).To(HaveEach(MatchRegexp("arn:aws:iam::[0-9]{12}:role/.+")))
				Expect(clusterDesc.State).To(Equal(constants.Ready))
			})

		It("Changing billing account for rosa hcp cluster in rosa cli - [id:75921]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check the help message of 'rosa edit cluster -h'")
				helpOutput, err := clusterService.EditCluster("", "-h")
				Expect(err).To(BeNil())
				Expect(helpOutput.String()).To(ContainSubstring("--billing-account"))

				By("Change the billing account for the cluster")
				output, err := clusterService.EditCluster(clusterID, "--billing-account", constants.ChangedBillingAccount)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("Updated cluster"))

				By("Check if billing account is changed")
				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).ToNot(HaveOccurred())
				Expect(CD.AWSBillingAccount).To(Equal(constants.ChangedBillingAccount))

				By("Create another machinepool without security groups and describe it")
				mpName := "mp-75921"
				_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
					"--replicas", "1",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					By("Remove the machine pool")
					rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

					By("Change the billing account back")
					output, err := clusterService.EditCluster(clusterID, "--billing-account", constants.BillingAccount)
					Expect(err).ToNot(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("Updated cluster"))
				}()
			})

		It("Changing invalid billing account for rosa hcp cluster in rosa cli - [id:75922]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Change the billing account with invalid value")
				output, err := clusterService.EditCluster(clusterID, "--billing-account", "qweD3")
				Expect(err).ToNot(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"not valid. Rerun the command with a valid billing account number"))

				output, err = clusterService.EditCluster(clusterID, "--billing-account", "123")
				Expect(err).ToNot(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"not valid. Rerun the command with a valid billing account number"))

				By("Change the billing account with an empty string")
				output, err = clusterService.EditCluster(clusterID, "--billing-account", " ")
				Expect(err).ToNot(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"not valid. Rerun the command with a valid billing account number"))

				By("Check the billing account is NOT changed")
				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				Expect(CD.AWSBillingAccount).To(Equal(constants.BillingAccount))
			})

		It("create ROSA HCP with registry config can work well via rosa cli  - [id:76394]",
			labels.High, labels.Runtime.Day1Post,
			func() {
				By("Check the help message of 'rosa create cluster -h'")
				helpOutput, err, _ := clusterService.Create("", "-h")
				Expect(err).To(BeNil())
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-allowed-registries"))
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-insecure-registries"))
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-blocked-registries"))
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-allowed-registries-for-import"))
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-additional-trusted-ca"))

				By("Check if cluster enable registry config")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				if clusterConfig.RegistryConfig {
					Skip("It is only for registry config enabled clusters")
				}
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())

				for _, v := range clusterDetail.RegistryConfiguration {
					if v["Allowed Registries"] != nil {
						allowedList := jsonData.DigString("registry_config", "registry_sources", "allowed_registries")
						if len(allowedList) > 2 {
							result := strings.Replace(allowedList, " ", ",", 1)
							Expect(result).To(Equal(fmt.Sprintf("[%s]", v["Allowed Registries"])))
						}
					}
					if v["Blocked Registries"] != nil {
						blockedList := jsonData.DigString("registry_config", "registry_sources", "blocked_registries")
						if len(blockedList) > 2 {
							result := strings.Replace(blockedList, " ", ",", 1)
							Expect(result).To(Equal(fmt.Sprintf("[%s]", v["Blocked Registries"])))
						}
					}
					if v["Insecure Registries"] != nil {
						insecureList := jsonData.DigString("registry_config", "registry_sources", "insecure_registries")
						if len(insecureList) > 2 {
							result := strings.Replace(insecureList, " ", ",", 1)
							Expect(result).To(Equal(fmt.Sprintf("[%s]", v["Insecure Registries"])))
						}
					}
					if v["Allowed Registries for Import"] != nil {
						clusterData := jsonData.DigObject("registry_config", "allowed_registries_for_import")
						if clusterData != "" {
							allowedImport := v["Allowed Registries for Import"].([]interface{})
							for _, a := range clusterData.([]interface{}) {
								importListFromJson := a.(map[string]interface{})
								insecureValue := false
								if importListFromJson["insecure"] != nil {
									insecureValue = importListFromJson["insecure"].(bool)
								}
								value1 := map[string]interface{}{
									"Domain Name": importListFromJson["domain_name"],
								}
								value2 := map[string]interface{}{
									"Insecure": insecureValue,
								}
								Expect(allowedImport).To(ContainElement(value1))
								Expect(allowedImport).To(ContainElement(value2))
							}
						}
					}
					if v["Platform Allowlist"] != nil {
						platformListID := jsonData.DigString("registry_config", "platform_allowlist", "id")
						pList := v["Platform Allowlist"].([]interface{})
						for _, p := range pList {
							pMap := p.(map[string]interface{})
							if pMap["ID"] != nil {
								Expect(pMap["ID"].(string)).To(Equal(platformListID))
							}
						}
					}

					if v["Additional Trusted CA"] != nil {
						caContent := jsonData.DigObject("registry_config", "additional_trusted_ca")
						if caContent != "" {
							caFromc := caContent.(map[string]interface{})
							for _, ca := range v["Additional Trusted CA"].([]interface{}) {
								Expect(caFromc).To(Equal(ca))
							}
						}
					}
				}
			})

		It("edit ROSA HCP with registry config can work well via rosa cli  - [id:76395]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check the help message of 'rosa edit cluster -h'")
				helpOutput, err := clusterService.EditCluster("", "-h")
				Expect(err).To(BeNil())
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-allowed-registries"))
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-insecure-registries"))
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-blocked-registries"))
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-allowed-registries-for-import"))
				Expect(helpOutput.String()).To(ContainSubstring("--registry-config-additional-trusted-ca"))

				By("Edit hcp cluster with registry configs")
				if clusterConfig.RegistryConfig {
					Skip("It is only for registry config enabled clusters")
				}

				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				for _, v := range clusterDetail.RegistryConfiguration {
					if v["Allowed Registries"] != nil {
						By("Remove allowed registry config")
						originValue := v["Allowed Registries"].(string)
						out, err := clusterService.EditCluster(clusterID,
							"--registry-config-allowed-registries", "",
							"-y",
						)
						Expect(err).ToNot(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
						Expect(textData).To(ContainSubstring("Updated cluster '%s'", clusterID))

						By("Describe cluster to check the value")
						output, err := clusterService.DescribeCluster(clusterID)
						Expect(err).To(BeNil())
						clusterDetail, err = clusterService.ReflectClusterDescription(output)
						Expect(err).To(BeNil())
						Expect(clusterDetail.RegistryConfiguration[0]["Allowed Registries"]).To(BeNil())

						By("Add blocked registry config")
						blockedValue := "test.blocked.com,*.example.com"
						out, err = clusterService.EditCluster(clusterID,
							"--registry-config-blocked-registries", blockedValue,
							"-y",
						)
						Expect(err).ToNot(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
						Expect(textData).To(ContainSubstring("Updated cluster '%s'", clusterID))

						By("Describe cluster to check the value")
						output, err = clusterService.DescribeCluster(clusterID)
						Expect(err).To(BeNil())
						clusterDetail, err = clusterService.ReflectClusterDescription(output)
						Expect(err).To(BeNil())
						Expect(clusterDetail.RegistryConfiguration[1]["Blocked Registries"]).To(Equal(blockedValue))

						By("Update it back")
						out, err = clusterService.EditCluster(clusterID,
							"--registry-config-blocked-registries", "",
							"--registry-config-allowed-registries", originValue,
							"-y",
						)
						Expect(err).ToNot(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
						Expect(textData).To(ContainSubstring("Updated cluster '%s'", clusterID))

						By("Describe cluster to check the value")
						output, err = clusterService.DescribeCluster(clusterID)
						Expect(err).To(BeNil())
						clusterDetail, err = clusterService.ReflectClusterDescription(output)
						Expect(err).To(BeNil())
						Expect(clusterDetail.RegistryConfiguration[0]["Allowed Registries"]).To(Equal(originValue))
						Expect(clusterDetail.RegistryConfiguration[1]["Blocked Registries"]).To(BeNil())

					}
					if v["Blocked Registries"] != nil {
						By("Remove blocked registry config")
						originValue := v["Blocked Registries"].(string)
						out, err := clusterService.EditCluster(clusterID,
							"--registry-config-blocked-registries", "",
							"-y",
						)
						Expect(err).ToNot(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
						Expect(textData).To(ContainSubstring("Updated cluster '%s'", clusterID))

						By("Describe cluster to check the value")
						output, err := clusterService.DescribeCluster(clusterID)
						Expect(err).To(BeNil())
						clusterDetail, err = clusterService.ReflectClusterDescription(output)
						Expect(err).To(BeNil())
						Expect(clusterDetail.RegistryConfiguration[0]["Blocked Registries"]).To(BeNil())

						By("Add allowed registry config")
						allowedValue := "test.allowed.com,*.example.com"
						out, err = clusterService.EditCluster(clusterID,
							"--registry-config-allowed-registries", allowedValue,
							"-y",
						)
						Expect(err).ToNot(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
						Expect(textData).To(ContainSubstring("Updated cluster '%s'", clusterID))

						By("Describe cluster to check the value")
						output, err = clusterService.DescribeCluster(clusterID)
						Expect(err).To(BeNil())
						clusterDetail, err = clusterService.ReflectClusterDescription(output)
						Expect(err).To(BeNil())
						Expect(clusterDetail.RegistryConfiguration[0]["Allowed Registries"]).To(Equal(allowedValue))

						By("Update it back")
						out, err = clusterService.EditCluster(clusterID,
							"--registry-config-blocked-registries", originValue,
							"--registry-config-allowed-registries", "",
							"-y",
						)
						Expect(err).ToNot(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
						Expect(textData).To(ContainSubstring("Updated cluster '%s'", clusterID))

						By("Describe cluster to check the value")
						output, err = clusterService.DescribeCluster(clusterID)
						Expect(err).To(BeNil())
						clusterDetail, err = clusterService.ReflectClusterDescription(output)
						Expect(err).To(BeNil())
						Expect(clusterDetail.RegistryConfiguration[0]["Allowed Registries"]).To(BeNil())
						Expect(clusterDetail.RegistryConfiguration[1]["Blocked Registries"]).To(Equal(originValue))
					}
				}
			})
	})
