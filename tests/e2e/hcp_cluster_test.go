package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("HCP cluster testing",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()

		var (
			clusterID      string
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			clusterConfig  *config.ClusterConfig
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			var err error
			clusterConfig, err = config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())

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
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())

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
				Expect(helpOutput.String()).ToNot(ContainSubstring("--no-cni"))

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
	})
