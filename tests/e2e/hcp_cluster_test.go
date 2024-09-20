package e2e

import (
	"fmt"

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
	})
