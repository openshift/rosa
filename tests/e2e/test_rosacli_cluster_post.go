package e2e

import (
	"context"
	"fmt"
	"io"
	nets "net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("Healthy check",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()
		var (
			clusterID          string
			rosaClient         *rosacli.Client
			clusterService     rosacli.ClusterService
			machinePoolService rosacli.MachinePoolService
			profile            *profilehandler.Profile
			isHosted           bool
			err                error
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init service")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			machinePoolService = rosaClient.MachinePool

			profile = profilehandler.LoadProfileYamlFileByENV()

			By("Retrieve whether hosted cluster")
			isHosted, err = clusterService.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())

		})

		AfterEach(func() {
			By("Clean the cluster")
			rosaClient.CleanResources(clusterID)
		})

		Context("using cluster configuraion file", func() {
			var clusterConfig *config.ClusterConfig
			BeforeEach(func() {
				By("Retrieve Cluster config and profile")
				clusterConfig, err = config.ParseClusterProfile()
				Expect(err).ToNot(HaveOccurred())
			})

			It("the creation of rosa cluster with volume size will work - [id:66359]",
				labels.Critical, labels.Runtime.Day1Post,
				func() {
					alignDiskSize := func(diskSize string) string {
						aligned := strings.Join(strings.Split(diskSize, " "), "")
						return aligned
					}

					By("Set expected worker pool size")
					expectedDiskSize := clusterConfig.WorkerDiskSize
					if expectedDiskSize == "" {
						expectedDiskSize = "300GiB" // if no worker disk size set, it will use default value
					}

					By("Check the machinepool list")
					output, err := machinePoolService.ListMachinePool(clusterID)
					Expect(err).ToNot(HaveOccurred())

					if isHosted {
						npList, err := machinePoolService.ReflectNodePoolList(output)
						Expect(err).ToNot(HaveOccurred())

						nodePool := npList.GetFirstNodePool()
						Expect(nodePool).ToNot(BeNil(), "node pool is not found for the cluster")
						Expect(alignDiskSize(nodePool.DiskSize)).To(Equal(expectedDiskSize))

						By("Check the default node pool description")
						output, err = machinePoolService.DescribeMachinePool(clusterID, nodePool.ID)
						Expect(err).ToNot(HaveOccurred())
						npD, err := machinePoolService.ReflectNodePoolDescription(output)
						Expect(err).ToNot(HaveOccurred())
						Expect(alignDiskSize(npD.DiskSize)).To(Equal(expectedDiskSize))

					} else {
						mplist, err := machinePoolService.ReflectMachinePoolList(output)
						Expect(err).ToNot(HaveOccurred())

						workPool := mplist.Machinepool(constants.DefaultClassicWorkerPool)
						Expect(workPool).ToNot(BeNil(), "worker pool is not found for the cluster")
						Expect(alignDiskSize(workPool.DiskSize)).To(Equal(expectedDiskSize))

						By("Check the default worker pool description")
						output, err = machinePoolService.DescribeMachinePool(clusterID, constants.DefaultClassicWorkerPool)
						Expect(err).ToNot(HaveOccurred())
						mpD, err := machinePoolService.ReflectMachinePoolDescription(output)
						Expect(err).ToNot(HaveOccurred())
						Expect(alignDiskSize(mpD.DiskSize)).To(Equal(expectedDiskSize))
					}
				})

			It("the creation of ROSA cluster with default-mp-labels option will succeed - [id:57056]",
				labels.Critical, labels.Runtime.Day1Post,
				func() {
					if isHosted {
						SkipTestOnFeature("default machinepool labels")
					}

					By("Check the cluster config")
					mpLables := strings.Split(clusterConfig.DefaultMpLabels, ",")

					By("Check the machinepool list")
					output, err := machinePoolService.ListMachinePool(clusterID)
					Expect(err).ToNot(HaveOccurred())

					mplist, err := machinePoolService.ReflectMachinePoolList(output)
					Expect(err).ToNot(HaveOccurred())

					workPool := mplist.Machinepool(constants.DefaultClassicWorkerPool)
					Expect(workPool).ToNot(BeNil(), "worker pool is not found for the cluster")
					for _, label := range mpLables {
						Expect(workPool.Labels).To(ContainSubstring(label))
					}

					By("Check the default worker pool description")
					output, err = machinePoolService.DescribeMachinePool(clusterID, constants.DefaultClassicWorkerPool)
					Expect(err).ToNot(HaveOccurred())

					mpD, err := machinePoolService.ReflectMachinePoolDescription(output)
					Expect(err).ToNot(HaveOccurred())
					for _, label := range mpLables {
						Expect(mpD.Labels).To(ContainSubstring(label))
					}

				})

			It("the additional security groups are working well - [id:68172]",
				labels.Critical, labels.Runtime.Day1Post,
				labels.Exclude, //Exclude it until day1 refactor support this part. It cannot be run with current day1
				func() {
					By("Run command to check help message of security groups")
					output, err, _ := clusterService.Create("", "-h")
					Expect(err).ToNot(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("--additional-compute-security-group-ids"))
					Expect(output.String()).Should(ContainSubstring("--additional-infra-security-group-ids"))
					Expect(output.String()).Should(ContainSubstring("--additional-control-plane-security-group-ids"))

					if isHosted {
						By("Describe the cluster to check the additional compute security groups")
						rosaClient.Runner.JsonFormat()
						output, err := clusterService.DescribeCluster(clusterID)
						Expect(err).ToNot(HaveOccurred())
						jsonData := rosaClient.Parser.JsonData.Input(output).Parse()
						computeSgIDs := jsonData.DigString("aws", "additional_compute_security_group_ids")
						computeSgIDs = computeSgIDs[1 : len(computeSgIDs)-1]
						Expect(strings.ReplaceAll(computeSgIDs, " ", ",")).To(
							Equal(clusterConfig.AdditionalSecurityGroups.WorkerSecurityGroups))

						By("Describe the workers nodepool and check the compute security groups")
						npList, err := machinePoolService.ListAndReflectNodePools(clusterID)
						Expect(err).ToNot(HaveOccurred())
						for _, np := range npList.NodePools {
							Expect(np.ID).ToNot(BeNil())
							if strings.HasPrefix(np.ID, constants.DefaultHostedWorkerPool) {
								npDesc, err := machinePoolService.DescribeAndReflectNodePool(clusterID, np.ID)
								Expect(err).ToNot(HaveOccurred())
								Expect(npDesc.AdditionalSecurityGroupIDs).To(
									Equal(helper.ReplaceCommaWithCommaSpace(
										clusterConfig.AdditionalSecurityGroups.WorkerSecurityGroups)))
							}
						}
					} else {

						By("Describe the cluster to check the control plane and infra additional security groups")
						des, err := clusterService.DescribeClusterAndReflect(clusterID)
						Expect(err).ToNot(HaveOccurred())
						var additionalMap []interface{}
						for _, item := range des.Nodes {
							if value, ok := item["Additional Security Group IDs"]; ok {
								additionalMap = value.([]interface{})
							}
						}
						if clusterConfig.AdditionalSecurityGroups == nil {
							Expect(additionalMap).To(BeNil())
						} else {
							Expect(additionalMap).ToNot(BeNil())
							for _, addSgGroups := range additionalMap {
								if value, ok := addSgGroups.(map[string]interface{})["Control Plane"]; ok {
									Expect(value).
										To(Equal(
											helper.ReplaceCommaWithCommaSpace(
												clusterConfig.AdditionalSecurityGroups.ControlPlaneSecurityGroups)))
								} else {
									value = addSgGroups.(map[string]interface{})["Infra"]
									Expect(value).
										To(Equal(
											helper.ReplaceCommaWithCommaSpace(
												clusterConfig.AdditionalSecurityGroups.InfraSecurityGroups)))
								}
							}
						}

						By("Describe the worker pool and check the compute security groups")
						mp, err := machinePoolService.DescribeAndReflectMachinePool(clusterID, constants.DefaultClassicWorkerPool)
						Expect(err).ToNot(HaveOccurred())
						if clusterConfig.AdditionalSecurityGroups == nil {
							Expect(mp.SecurityGroupIDs).To(BeEmpty())
						} else {
							Expect(mp.SecurityGroupIDs).
								To(Equal(
									helper.ReplaceCommaWithCommaSpace(
										clusterConfig.AdditionalSecurityGroups.WorkerSecurityGroups)))
						}
					}
				})

			It("bring your own kms key functionality works on cluster creation - [id:60082]",
				labels.Critical, labels.Runtime.Day1Post,
				func() {
					By("Confirm current cluster profile uses kms keys")
					if !clusterConfig.EnableCustomerManagedKey {
						SkipTestOnFeature("byo kms")
					}
					By("Check the help message of 'rosa create cluster -h'")
					output, err := clusterService.CreateDryRun(clusterID, "-h")
					Expect(err).To(BeNil())
					Expect(output.String()).To(ContainSubstring("--kms-key-arn"))
					Expect(output.String()).To(ContainSubstring("--enable-customer-managed-key"))

					By("Confirm KMS key is present")
					jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
					Expect(err).To(BeNil())
					kmsKey := jsonData.DigString("aws", "kms_key_arn")
					Expect(clusterConfig.Encryption.KmsKeyArn).To(Equal(kmsKey))
				})

			It("additional allowed principals work on cluster creation - [id:74408]",
				labels.Critical, labels.Runtime.Day1Post,
				func() {
					By("Confirm current cluster profile uses additional allowed principals")
					if !profile.ClusterConfig.AdditionalPrincipals {
						SkipTestOnFeature("additional allowed principals")
					}

					By("Check the help message of 'rosa create cluster -h'")
					output, err := clusterService.CreateDryRun(clusterID, "-h")
					Expect(err).To(BeNil())
					Expect(output.String()).
						To(
							ContainSubstring("--additional-allowed-principals"))

					By("Confirm additional principals is present")
					out, err := clusterService.DescribeClusterAndReflect(clusterID)
					Expect(err).To(BeNil())
					Expect(out.AdditionalPrincipals).To(ContainSubstring(clusterConfig.AdditionalPrincipals))
				})

			It("etcd encryption works on cluster creation - [id:42188]",
				labels.Critical, labels.Runtime.Day1Post,
				func() {
					By("Confirm current cluster profile uses etcd encryption")
					if !clusterConfig.EtcdEncryption {
						SkipTestOnFeature("etcd encryption")
					}
					By("Check the help message of 'rosa create cluster -h'")
					output, err := clusterService.CreateDryRun(clusterID, "-h")
					Expect(err).To(BeNil())
					Expect(output.String()).To(ContainSubstring("--etcd-encryption"))

					By("Confirm etcd encryption is enabled")
					jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
					Expect(err).To(BeNil())
					etcdEncryption := jsonData.DigBool("etcd_encryption")
					Expect(etcdEncryption).To(BeTrue())
				})

			It("with private_link will work - [id:41549]", labels.Runtime.Day1Post, labels.Critical,
				func() {
					private := constants.No
					ingressPrivate := "false"
					if clusterConfig.PrivateLink {
						private = constants.Yes
						ingressPrivate = "true" // nolint
					}
					By("Describe the cluster the cluster should be private")
					clusterDescription, err := clusterService.DescribeClusterAndReflect(clusterID)
					Expect(err).ToNot(HaveOccurred())
					Expect(clusterDescription.Private).To(Equal(private))

					By("Check the ingress should be private")
					ingress, err := rosaClient.Ingress.DescribeIngressAndReflect(clusterID, "apps")
					Expect(err).ToNot(HaveOccurred())
					Expect(ingress.Private).To(Equal(ingressPrivate))

				})

			It("with compute_machine_type will work - [id:75150]", labels.Runtime.Day1Post, labels.High,
				func() {
					By("Check compute machine type")
					jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
					Expect(err).To(BeNil())
					Expect(jsonData.DigString("nodes", "compute_machine_type", "id")).To(
						Equal(clusterConfig.Nodes.ComputeInstanceType))
				})

			It("with multiAZ will work - [id:75535]", labels.Runtime.Day1Post, labels.Critical,
				func() {
					if !isHosted {
						SkipNotHosted()
					}
					By("Retrieve cluster description")
					clusterDesc, err := clusterService.DescribeClusterAndReflect(clusterID)
					Expect(err).To(BeNil())

					By("Check Data plane")
					for _, plane := range clusterDesc.Availability {
						for planeKey, planeValue := range plane {
							if planeKey == "Data Plane" {
								if clusterConfig.MultiAZ {
									Expect(planeValue).To(Equal("MultiAZ"))
								} else {
									Expect(planeValue).To(Equal("SingleAZ"))
								}
							}
						}
					}

					By("Check the machinepool size and subnets")
					var mpSize int
					var subnets []string
					mps, err := machinePoolService.ListAndReflectNodePools(clusterID)
					Expect(err).To(BeNil())
					mpSize = len(mps.NodePools)
					for _, mp := range mps.NodePools {
						subnets = append(subnets, mp.Subnet)
					}

					if clusterConfig.MultiAZ {
						Expect(mpSize > 1).To(BeTrue(), fmt.Sprintf("MachinePool size is not greater than one: '%d'", mpSize))
						Expect(len(subnets) > 1).To(BeTrue(), fmt.Sprintf("Subnet size is not greater than one: '%d'", len(subnets)))
						Expect(len(subnets)).To(Equal(len(helper.UniqueStringValues(subnets))), "Subnet is duplicated")
					} else {
						Expect(mpSize).To(Equal(1))
						Expect(len(subnets)).To(Equal(1))
					}
				})

			It("with private will work - [id:75526]", labels.Runtime.Day1Post, labels.Critical,
				func() {
					private := constants.No
					ingressPrivate := "false"
					if clusterConfig.Private {
						private = constants.Yes
						ingressPrivate = "true" // nolint
					}

					By("Describe the cluster and check private field")
					clusterDescription, err := clusterService.DescribeClusterAndReflect(clusterID)
					Expect(err).ToNot(HaveOccurred())
					Expect(clusterDescription.Private).To(Equal(private))

					By("Check the ingress should be private")
					ingress, err := rosaClient.Ingress.DescribeIngressAndReflect(clusterID, "apps")
					Expect(err).ToNot(HaveOccurred())
					Expect(ingress.Private).To(Equal(ingressPrivate))
				})

			It("with autoscaling will work - [id:75527]", labels.Runtime.Day1Post, labels.High,
				func() {
					isAutoscale := clusterConfig.Autoscaling != nil && clusterConfig.Autoscaling.Enabled

					By("Retrieve cluster description")
					jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
					Expect(err).To(BeNil())

					if isAutoscale {
						Expect(jsonData.DigString("nodes", "autoscale_compute")).ToNot(BeNil())
					} else {
						Expect(jsonData.DigString("nodes", "autoscale_compute")).To(BeEmpty())
					}
				})

			It("with subnets will work - [id:37176]", labels.Runtime.Day1Post, labels.Critical,
				func() {
					By("Check the creation command help")
					output, err, _ := clusterService.Create("cluster-37176", "-h")
					Expect(err).ToNot(HaveOccurred())
					flag := "--subnet-ids strings"
					description := "The Subnet IDs to use when installing the cluster." +
						" Format should be a comma-separated list." +
						" Leave empty for installer provisioned subnet IDs"
					Expect(output.String()).Should(MatchRegexp(fmt.Sprintf(`%s[\s\t]*%s`, flag, description)))

					By("Check that cluster is created with the correct subnets")
					clusterDescription, err := clusterService.DescribeClusterAndReflect(clusterID)
					Expect(err).ToNot(HaveOccurred())
					clusterSubnets := []string{}

					for _, item := range clusterDescription.Network {
						if value, ok := item["Subnets"]; ok {
							clusterSubnets = helper.ParseCommaSeparatedStrings(value)
							break
						}
					}
					configuredPrivateSubnets := []string{}
					if !profile.ClusterConfig.BYOVPC {
						Expect(len(clusterSubnets)).To(BeZero())
					} else {
						configuredPrivateSubnets = helper.ParseCommaSeparatedStrings(clusterConfig.Subnets.PrivateSubnetIds)
						configuredPublicSubnets := helper.ParseCommaSeparatedStrings(clusterConfig.Subnets.PublicSubnetIds)
						Expect(len(clusterSubnets)).To(BeNumerically("==",
							len(configuredPrivateSubnets)+
								len(configuredPublicSubnets)))
						Expect(clusterSubnets).Should(ContainElements(configuredPrivateSubnets))
						Expect(clusterSubnets).Should(ContainElements(configuredPublicSubnets))
					}

					By("Check the default worker pool")
					if isHosted {
						npList, err := machinePoolService.ListAndReflectNodePools(clusterID)
						Expect(err).ToNot(HaveOccurred())
						for _, np := range npList.NodePools {
							// Only check the machinepool has workers which is default worker pool
							if regexp.MustCompile("workers-?[0-9]?").MatchString(np.ID) {
								Expect(np.Subnet).To(BeElementOf(configuredPrivateSubnets))
							}
						}
					} else {
						mp, err := machinePoolService.DescribeAndReflectMachinePool(
							clusterID, constants.DefaultClassicWorkerPool)
						Expect(err).ToNot(HaveOccurred())
						workerPoolSubnets := helper.ParseCommaSeparatedStrings(mp.Subnets)
						Expect(len(workerPoolSubnets)).To(Equal(len(configuredPrivateSubnets)))
						Expect(workerPoolSubnets).Should(ContainElements(configuredPrivateSubnets))
					}

				})

			It("with proxy set will work - [id:45502]", labels.Runtime.Day1Post, labels.Critical,
				func() {
					By("Check the help message of proxy")
					output, err, _ := clusterService.Create("cl-45502", "-h")
					Expect(err).ToNot(HaveOccurred())
					Expect(output.String()).Should(
						MatchRegexp(`-{2}http-proxy string[\s\t]+A proxy URL to use for creating HTTP connections`))
					Expect(output.String()).Should(
						MatchRegexp(`-{2}https-proxy string[\s\t]+A proxy URL to use for creating HTTPS connections`))
					Expect(output.String()).Should(
						MatchRegexp(`-{2}no-proxy strings[\s\t]+A comma-separated list of destination domain names`))
					Expect(output.String()).Should(
						MatchRegexp(`-{2}additional-trust-bundle-file string[\s\t]+A file`))

					By("Check the proxy is set correctly")
					clusterDescription, err := clusterService.DescribeClusterAndReflect(clusterID)
					Expect(err).ToNot(HaveOccurred())
					if profile.ClusterConfig.ProxyEnabled {
						clusterHTTPProxy, clusterHTTPSProxy, clusterNoProxy :=
							clusterService.DetectProxy(clusterDescription)
						Expect(clusterConfig.Proxy.Http).To(Equal(clusterHTTPProxy))
						Expect(clusterConfig.Proxy.Https).To(Equal(clusterHTTPSProxy))
						Expect(clusterConfig.Proxy.NoProxy).To(Equal(clusterNoProxy))
						Expect(clusterDescription.AdditionalTrustBundle).To(Equal("REDACTED"))
					} else {
						Expect(clusterDescription.Proxy).To(BeEmpty())
					}
				})
		})

		Context("without using cluster configuration file", func() {

			It("the windows certificates expiration - [id:64040]",
				labels.Medium, labels.Runtime.Day1Post, labels.Exclude,
				func() {
					//If the case fails,please open a card to ask dev update windows certificates.
					//Example card: https://issues.redhat.com/browse/SDA-8990
					By("Get ROSA windows certificates on ocm-sdk repo")
					sdkCAFileURL := "https://raw.githubusercontent.com/openshift-online/ocm-sdk-go/main/internal/system_cas_windows.go"
					resp, err := nets.Get(sdkCAFileURL)
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close()
					content, err := io.ReadAll(resp.Body)
					Expect(err).ToNot(HaveOccurred())
					sdkContent := string(content)

					By("Check the domains certificates if it is updated")
					domains := []string{"api.openshift.com", "sso.redhat.com"}
					for _, url := range domains {
						cmd := fmt.Sprintf(
							"openssl s_client -connect %s:443 -showcerts 2>&1  | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p'",
							url)
						stdout, err := rosaClient.Runner.RunCMD([]string{"bash", "-c", cmd})
						Expect(err).ToNot(HaveOccurred())
						result := strings.Trim(stdout.String(), "\n")
						ca := strings.Split(result, "-----END CERTIFICATE-----")
						Expect(sdkContent).To(ContainSubstring(ca[0]))
						Expect(sdkContent).To(ContainSubstring(ca[1]))
					}
				})

			It("rosa hcp cluster creation support imdsv2 - [id:75114]",
				labels.Critical, labels.Runtime.Day1Post,
				func() {
					if !isHosted {
						SkipNotHosted()
					}

					By("Check the help message of 'rosa create cluster -h'")
					res, err := clusterService.CreateDryRun(clusterID, "-h")
					Expect(err).ToNot(HaveOccurred())
					Expect(res.String()).To(ContainSubstring("--ec2-metadata-http-tokens"))

					By("Get the ec2_metadata_http_tokens value from cluster level spec attribute")
					output, err := clusterService.GetJSONClusterDescription(clusterID)
					Expect(err).ToNot(HaveOccurred())
					clusterIMDSv2Value := output.DigString("aws", "ec2_metadata_http_tokens")

					By("Check the cluster description value to match cluster profile configuration")
					if profile.ClusterConfig.Ec2MetadataHttpTokens == "" {
						Expect(clusterIMDSv2Value).To(Equal(constants.DefaultEc2MetadataHttpTokens))
					} else {
						Expect(clusterIMDSv2Value).To(Equal(profile.ClusterConfig.Ec2MetadataHttpTokens))
					}

					By("Check the default workers machinepool value to match cluster level spec attribute")
					npList, err := machinePoolService.ListAndReflectNodePools(clusterID)
					Expect(err).ToNot(HaveOccurred())
					for _, np := range npList.NodePools {
						Expect(np.ID).ToNot(BeNil())
						if strings.HasPrefix(np.ID, constants.DefaultHostedWorkerPool) {
							npDesc, err := machinePoolService.DescribeAndReflectNodePool(clusterID, np.ID)
							Expect(err).ToNot(HaveOccurred())
							Expect(npDesc.EC2MetadataHttpTokens).To(Equal(clusterIMDSv2Value))
						}
					}
				})

			It("rosa cluster with fips enabled can be created successfully - [id:46312]",
				labels.Critical, labels.Runtime.Day1Post,
				func() {
					output, err := clusterService.DescribeCluster(clusterID)
					Expect(err).ToNot(HaveOccurred())
					des, err := clusterService.ReflectClusterDescription(output)
					Expect(err).ToNot(HaveOccurred())

					By("Check if fips is enabled")
					if !profile.ClusterConfig.FIPS {
						Expect(des.FIPSMod).To(Equal(""))
					} else {
						Expect(des.FIPSMod).To(Equal("Enabled"))
					}
				})

			It("cluster is multiarch - [id:75108]", labels.Runtime.Day1Post, labels.High,
				func() {
					By("Check cluster is multiarch")
					jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
					Expect(err).To(BeNil())
					if isHosted {
						clusterVersion := jsonData.DigString("openshift_version")
						if clusterVersion >= "4.14" {
							Expect(jsonData.DigBool("multi_arch_enabled")).To(BeTrue())
						} else {
							Expect(jsonData.DigBool("multi_arch_enabled")).To(BeFalse())
						}
					} else {
						Expect(jsonData.DigBool("multi_arch_enabled")).To(BeFalse())
					}
				})

			It("with policy path will work - [id:75525]", labels.Runtime.Day1Post, labels.High,
				func() {
					if profile.AccountRoleConfig == nil || profile.AccountRoleConfig.Path == "" {
						Skip("No account-role path defined. Skipping ...")
					}
					path := profile.AccountRoleConfig.Path

					By("Retrieve cluster description")
					clusterDesc, err := clusterService.DescribeClusterAndReflect(clusterID)
					Expect(err).To(BeNil())

					By("Check account roles path")
					Expect(clusterDesc.STSRoleArn).To(ContainSubstring(path))
					Expect(clusterDesc.SupportRoleARN).To(ContainSubstring(path))
					for _, roles := range clusterDesc.InstanceIAMRoles {
						for _, role := range roles {
							Expect(role).To(ContainSubstring(path))
						}
					}

					By("Check operator roles path")
					for _, role := range clusterDesc.OperatorIAMRoles {
						Expect(role).To(ContainSubstring(path))
					}
				})
		})
	})

var _ = Describe("Create cluster with the version in some channel group testing",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()
		var (
			clusterID      string
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
		)

		BeforeEach(func() {
			By("Get the cluster")
			var clusterDetail *profilehandler.ClusterDetail
			var err error
			clusterDetail, err = profilehandler.ParserClusterDetail()
			Expect(err).ToNot(HaveOccurred())
			clusterID = clusterDetail.ClusterID
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
		})

		AfterEach(func() {
			By("Clean remaining resources")
			rosaClient.CleanResources(clusterID)
		})

		It("User can create cluster with channel group - [id:35420]",
			labels.Critical, labels.Runtime.Day1Post,
			func() {
				profile := profilehandler.LoadProfileYamlFileByENV()

				By("Check if the cluster using right channel group")
				versionOutput, err := clusterService.GetClusterVersion(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(versionOutput.ChannelGroup).To(Equal(profile.ChannelGroup))
			})
	})

var _ = Describe("Post-Check testing for cluster deletion",
	labels.Feature.Cluster, func() {
		defer GinkgoRecover()
		var (
			rosaClient         *rosacli.Client
			ocmResourceService rosacli.OCMResourceService
			clusterService     rosacli.ClusterService
			clusterID          string
			err                error
			awsClient          *aws_client.AWSClient
			oidcEndpointUrlC   string
			clusterConfig      *config.ClusterConfig
			oidcConfigC        string
			oidcProviderArn    string
			profile            *profilehandler.Profile
		)

		BeforeEach(func() {
			By("Init the client and get profile and config")
			awsClient, err = aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource

			clusterConfig, err = config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())

			profile = profilehandler.LoadProfileYamlFileByENV()
			Expect(err).ToNot(HaveOccurred())
		})

		It("to verifiy the byo oidc cluster is deleted successfully - [id:75210]",
			labels.Critical, labels.Runtime.DestroyPost,
			func() {

				By("Check if it is using oidc config")
				if profile.ClusterConfig.OIDCConfig == "" {
					Skip("Skip this case as it is only for byo oidc cluster")
				}
				By("Get aws account id")
				rosaClient.Runner.JsonFormat()
				whoamiOutput, err := ocmResourceService.Whoami()
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
				AWSAccountID := whoamiData.AWSAccountID

				By("Get the oidc config and cluster id from cluster config file")
				clusterID = config.GetClusterID()
				oidcConfigC = clusterConfig.Aws.Sts.OidcConfigID

				By("Get oidc endpoint URL from cluster detail json file")
				clusterDetail, err := profilehandler.ParserClusterDetail()
				Expect(err).To(BeNil())
				oidcEndpointUrlC = clusterDetail.OIDCEndpointURL
				parts := strings.Split(oidcEndpointUrlC, " ")
				oidcEndpointUrlC = parts[0]
				parsedUrl, err := url.Parse(oidcEndpointUrlC)
				Expect(err).To(BeNil())
				oidcEndpointUrl := parsedUrl.String()
				oidcEndpointUrl, err = helper.ExtractOIDCProviderFromOidcUrl(oidcEndpointUrl)
				Expect(err).To(BeNil())

				By("Check the cluster is deleted")
				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				Expect(err).To(BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				Expect(err).To(BeNil())
				Expect(clusterList.IsExist(clusterID)).To(BeFalse())

				By("Check the oidc config is deleted")
				out, err := ocmResourceService.GetOIDCConfigFromList(oidcConfigC)
				Expect(err).To(BeNil())
				Expect(out).To(Equal(rosacli.OIDCConfig{}))

				By("Check oidc provider is deleted")
				oidcProviderArn = fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s", AWSAccountID, oidcEndpointUrl)
				_, err = awsClient.IamClient.GetOpenIDConnectProvider(
					context.TODO(),
					&iam.GetOpenIDConnectProviderInput{
						OpenIDConnectProviderArn: &oidcProviderArn,
					},
				)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("NoSuchEntity"))

			})
		It("to verify the sts cluster is deleted successfully - [id:75256]",
			labels.High, labels.Runtime.DestroyPost,
			func() {
				By("Check the operator-roles is deleted")
				clusterDetail, err := profilehandler.ParserClusterDetail()
				Expect(err).To(BeNil())

				operatorRolesArn := clusterDetail.OperatorRoleArns
				for _, v := range operatorRolesArn {
					_, roleName, err := helper.ParseRoleARN(v)
					Expect(err).To(BeNil())
					opRole, err := awsClient.IamClient.GetRole(
						context.TODO(),
						&iam.GetRoleInput{
							RoleName: &roleName,
						})
					Expect(opRole).To(BeNil())
					Expect(err.Error()).To(ContainSubstring("NoSuchEntity"))
				}

				By("Check the cluster is deleted")
				clusterID = config.GetClusterID()
				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				Expect(err).To(BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				Expect(err).To(BeNil())
				Expect(clusterList.IsExist(clusterID)).To(BeFalse())

				By("Check account roles are deleted")
				accountRoleArns := []string{
					clusterConfig.Aws.Sts.RoleArn,
					clusterConfig.Aws.Sts.SupportRoleArn,
					clusterConfig.Aws.Sts.WorkerRoleArn,
				}

				for _, v := range accountRoleArns {
					_, roleName, err := helper.ParseRoleARN(v)
					Expect(err).To(BeNil())
					opRole, err := awsClient.IamClient.GetRole(
						context.TODO(),
						&iam.GetRoleInput{
							RoleName: &roleName,
						})
					Expect(opRole).To(BeNil())
					Expect(err.Error()).To(ContainSubstring("NoSuchEntity"))
				}
			})
	})
var _ = Describe("Post-Check testing for cluster creation",
	labels.Feature.Cluster, func() {
		defer GinkgoRecover()
		var (
			rosaClient     *rosacli.Client
			err            error
			clusterID      string
			clusterService rosacli.ClusterService
			oidcConfigC    string
			awsClient      *aws_client.AWSClient
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			awsClient, err = aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())
		})

		It("to verify byo oidc cluster is created successfully - [id:59530]",
			labels.Critical, labels.Runtime.Day1Post,
			func() {
				clusterConfig, err := config.ParseClusterProfile()
				Expect(err).ToNot(HaveOccurred())
				profile := profilehandler.LoadProfileYamlFileByENV()
				Expect(err).ToNot(HaveOccurred())

				By("Check if it is using oidc config")
				if profile.ClusterConfig.OIDCConfig == "" {
					Skip("Skip this case as it is only for byo oidc cluster")
				}

				By("Retrieve oidc config from cluster config")
				clusterID = config.GetClusterID()
				oidcConfigC = clusterConfig.Aws.Sts.OidcConfigID

				By("Get the operator roles")
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())
				oidcConfigID := jsonData.DigString("aws", "sts", "oidc_config", "id")
				oidcConfigIssuerURL := jsonData.DigString("aws", "sts", "oidc_config", "issuer_url")
				Expect(oidcConfigC).To(Equal(oidcConfigID))

				By("Check oidc provider using the oidc config created in day1")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				OidcUrl := CD.OIDCEndpointURL
				if profile.ClusterConfig.OIDCConfig == "unmanaged" {
					Expect(OidcUrl).To(Equal(oidcConfigIssuerURL + " (Unmanaged)"))
				} else {
					Expect(OidcUrl).To(ContainSubstring(oidcConfigC))
				}

				By("Get operator roles from cluster")
				operatorRolesArns := CD.OperatorIAMRoles
				for _, operatorRoleARN := range operatorRolesArns {
					_, roleName, err := helper.ParseRoleARN(operatorRoleARN)
					Expect(err).To(BeNil())
					opRole, err := awsClient.GetRole(roleName)
					Expect(err).To(BeNil())
					if profile.ClusterConfig.OIDCConfig == "unmanaged" {
						Expect(*opRole.AssumeRolePolicyDocument).To(
							ContainSubstring(strings.Replace(oidcConfigIssuerURL, "https://", "", 1)))
					} else {
						Expect(*opRole.AssumeRolePolicyDocument).To(ContainSubstring(oidcConfigC))
					}
				}
			})
		It("to verify sts cluster is created successfully - [id:41822]",
			labels.High, labels.Runtime.Day1Post,
			func() {
				By("Check the cluster is STS cluster")
				profile := profilehandler.LoadProfileYamlFileByENV()
				if !profile.ClusterConfig.STS {
					Skip("This case is only for STS cluster post check")
				}
				By("Check the cluster is in ready status")
				clusterDetail, err := profilehandler.ParserClusterDetail()
				Expect(err).ToNot(HaveOccurred())
				clusterID = clusterDetail.ClusterID
				Expect(clusterID).ToNot(Equal(""), "ClusterID is required.Please debug why it is empty")

				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				clusterDescription, err := clusterService.ReflectClusterDescription(output)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterDescription.State).To(Equal("ready"))
				Expect(clusterDescription.STSRoleArn).ToNot(BeEmpty())

				By("Check the sts cluster created with IAM roles")
				if profile.ClusterConfig.HCP {
					Expect(len(clusterDescription.OperatorIAMRoles)).To(Equal(8))
				} else {
					Expect(len(clusterDescription.OperatorIAMRoles)).To(Equal(6))
				}

			})
		It("to verify cluster with the operator-roles attaching managed policy is created successfully - [id:57410]",
			labels.High, labels.Runtime.Day1Post,
			func() {
				// Till now, only HCP clusters operator roles are attaching managed policies
				By("Skip is the cluster is not HCP cluster")
				profile := profilehandler.LoadProfileYamlFileByENV()
				if !profile.ClusterConfig.HCP {
					Skip("This case is only for STS cluster post check")
				}
				By("Check the attaching policies are managed")
				clusterID = config.GetClusterID()
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				operatorRolesArns := CD.OperatorIAMRoles
				for _, policyArn := range operatorRolesArns {
					By("Check role tag")
					_, operatorRoleName, err := helper.ParseRoleARN(policyArn)
					Expect(err).To(BeNil())
					roleTags, err := awsClient.IamClient.ListRoleTags(context.TODO(), &iam.ListRoleTagsInput{
						RoleName: aws.String(operatorRoleName),
					})
					Expect(err).To(BeNil())
					tagCheckPass := false
					for _, tag := range roleTags.Tags {
						if *tag.Key == "rosa_managed_policies" && *tag.Value == constants.TrueString {
							tagCheckPass = true
							break
						}
					}
					Expect(tagCheckPass).To(BeTrue())
					By("Check policy is aws managed policy")
					attachedPolicies, err := awsClient.ListAttachedRolePolicies(operatorRoleName)
					Expect(err).To(BeNil())
					Expect(*attachedPolicies[0].PolicyArn).To(ContainSubstring("arn:aws:iam::aws"), operatorRoleName)
				}
			})
	})

var _ = Describe("Post-Check testing for cluster clusters with the --disable-scp-checks flag",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()
		var (
			clusterID      string
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
		})

		AfterEach(func() {
			By("Clean remaining resources")
			rosaClient.CleanResources(clusterID)
		})

		It(" Create MOA clusters with the --disable-scp-check flag - [id:35894]",
			labels.Medium, labels.Runtime.Day1Post,
			func() {
				profile := profilehandler.LoadProfileYamlFileByENV()
				By("Skip testing if the cluster is not a y-1 STS classic cluster")
				if profile.ClusterConfig.HCP {
					Skip("Skip this case as it is for classic cluster")
				}

				By("Describe cluster in text format")
				output, err := clusterService.DescribeCluster(clusterID, "-o", "yaml")
				Expect(err).To(BeNil())
				if profile.ClusterConfig.DisableSCPChecks {
					Expect(output.String()).To(ContainSubstring("disable_scp_checks: true"))
				} else {
					Expect(output.String()).To(ContainSubstring("disable_scp_checks: false"))
				}
			})
	})
