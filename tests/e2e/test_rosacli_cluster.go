package e2e

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit cluster",
	labels.Day2,
	labels.FeatureCluster,
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

			By("Load the original cluster config")
			var err error
			clusterConfig, err = config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			By("Clean the cluster")
			rosaClient.CleanResources(clusterID)
		})

		It("can check the description of the cluster - [id:34102]",
			labels.Medium,
			labels.MigrationToVerify,
			labels.Exclude,
			func() {
				By("Describe cluster in text format")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				By("Describe cluster in json format")
				rosaClient.Runner.JsonFormat()
				jsonOutput, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()

				By("Compare the text result with the json result")
				Expect(CD.ID).To(Equal(jsonData.DigString("id")))
				Expect(CD.ExternalID).To(Equal(jsonData.DigString("external_id")))
				Expect(CD.ChannelGroup).To(Equal(jsonData.DigString("version", "channel_group")))
				Expect(CD.DNS).To(Equal(jsonData.DigString("name") + "." + jsonData.DigString("dns", "base_domain")))
				Expect(CD.AWSAccount).NotTo(BeEmpty())
				Expect(CD.APIURL).To(Equal(jsonData.DigString("api", "url")))
				Expect(CD.ConsoleURL).To(Equal(jsonData.DigString("console", "url")))
				Expect(CD.Region).To(Equal(jsonData.DigString("region", "id")))

				Expect(CD.State).To(Equal(jsonData.DigString("status", "state")))
				Expect(CD.Created).NotTo(BeEmpty())
				Expect(CD.DetailsPage).NotTo(BeEmpty())

				if jsonData.DigBool("aws", "private_link") {
					Expect(CD.Private).To(Equal("Yes"))
				} else {
					Expect(CD.Private).To(Equal("No"))
				}

				if jsonData.DigBool("hypershift", "enabled") {
					//todo
				} else {
					if jsonData.DigBool("multi_az") {
						Expect(CD.MultiAZ).To(Equal(strconv.FormatBool(jsonData.DigBool("multi_az"))))
					} else {
						Expect(CD.Nodes[0]["Control plane"]).To(Equal(int(jsonData.DigFloat("nodes", "master"))))
						Expect(CD.Nodes[1]["Infra"]).To(Equal(int(jsonData.DigFloat("nodes", "infra"))))
						Expect(CD.Nodes[2]["Compute"]).To(Equal(int(jsonData.DigFloat("nodes", "compute"))))
					}
				}

				Expect(CD.Network[1]["Service CIDR"]).To(Equal(jsonData.DigString("network", "service_cidr")))
				Expect(CD.Network[2]["Machine CIDR"]).To(Equal(jsonData.DigString("network", "machine_cidr")))
				Expect(CD.Network[3]["Pod CIDR"]).To(Equal(jsonData.DigString("network", "pod_cidr")))
				Expect(CD.Network[4]["Host Prefix"]).Should(ContainSubstring(strconv.FormatFloat(jsonData.DigFloat("network", "host_prefix"), 'f', -1, 64)))
				Expect(CD.InfraID).To(Equal(jsonData.DigString("infra_id")))
			})

		It("can restrict master API endpoint to direct, private connectivity or not - [id:38850]",
			labels.High,
			labels.MigrationToVerify,
			labels.Exclude,
			func() {
				By("Check the cluster is not private cluster")
				private, err := clusterService.IsPrivateCluster(clusterID)
				Expect(err).To(BeNil())
				if private {
					Skip("This case needs to test on private cluster as the prerequirement,it was not fullfilled, skip the case!!")
				}
				isSTS, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).To(BeNil())
				isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).To(BeNil())
				By("Edit cluster to private to true")
				out, err := clusterService.EditCluster(
					clusterID,
					"--private",
					"-y",
				)
				if !isSTS || isHostedCP {
					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
					Expect(textData).Should(ContainSubstring("You are choosing to make your cluster API private. You will not be able to access your cluster"))
					Expect(textData).Should(ContainSubstring("Updated cluster '%s'", clusterID))
				} else {
					Expect(err).ToNot(BeNil())
					Expect(rosaClient.Parser.TextData.Input(out).Parse().Tip()).Should(ContainSubstring("Failed to update cluster: Cannot update listening mode of cluster's API on an AWS STS cluster"))
				}
				defer func() {
					By("Edit cluster to private back to false")
					out, err = clusterService.EditCluster(
						clusterID,
						"--private=false",
						"-y",
					)
					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
					Expect(textData).Should(ContainSubstring("Updated cluster '%s'", clusterID))

					By("Describe cluster to check Private is true")
					output, err := clusterService.DescribeCluster(clusterID)
					Expect(err).To(BeNil())
					CD, err := clusterService.ReflectClusterDescription(output)
					Expect(err).To(BeNil())
					Expect(CD.Private).To(Equal("No"))
				}()

				By("Describe cluster to check Private is true")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				if !isSTS || isHostedCP {
					Expect(CD.Private).To(Equal("Yes"))
				} else {
					Expect(CD.Private).To(Equal("No"))
				}
			})

		// OCM-5231 caused the description parser issue
		It("can disable workload monitoring on/off - [id:45159]",
			labels.High,
			labels.MigrationToVerify,
			labels.Exclude,
			func() {
				By("Check the cluster UWM is in expected status")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).ToNot(HaveOccurred())
				expectedUWMValue := "Enabled"
				if clusterConfig.DisableWorkloadMonitoring {
					expectedUWMValue = "Disabled"
				}
				Expect(clusterDetail.UserWorkloadMonitoring).To(Equal(expectedUWMValue))

				By("Disable the UWM")
				expectedUWMValue = "Disabled"
				_, err = clusterService.EditCluster(clusterID,
					"--disable-workload-monitoring",
					"-y")
				Expect(err).ToNot(HaveOccurred())

				By("Check the disable result for cluster description")
				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				clusterDetail, err = clusterService.ReflectClusterDescription(output)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterDetail.UserWorkloadMonitoring).To(Equal(expectedUWMValue))

				By("Enable the UWM again")
				expectedUWMValue = "Enabled"
				_, err = clusterService.EditCluster(clusterID,
					"--disable-workload-monitoring=false",
					"-y")
				Expect(err).ToNot(HaveOccurred())

				By("Check the disable result for cluster description")
				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				clusterDetail, err = clusterService.ReflectClusterDescription(output)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterDetail.UserWorkloadMonitoring).To(Equal(expectedUWMValue))
			})

		It("can validate for deletion of upgrade policy of rosa cluster - [id:38787]",
			labels.Medium,
			labels.MigrationToVerify,
			labels.Exclude,
			func() {
				By("Validate that deletion of upgrade policy for rosa cluster will work via rosacli")
				output, err := clusterService.DeleteUpgrade("")
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring(`required flag(s) "cluster" not set`))

				By("Delete an non-existant upgrade when cluster has no scheduled policy")
				output, err = clusterService.DeleteUpgrade("-c", clusterID)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring(`There are no scheduled upgrades on cluster '%s'`, clusterID))

				By("Delete with unknown flag --interactive")
				output, err = clusterService.DeleteUpgrade("-c", clusterID, "--interactive")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Error: unknown flag: --interactive"))

			})

		It("can allow sts cluster installation with compatible policies - [id:45161]",
			labels.High,
			labels.MigrationToVerify,
			labels.Exclude,
			func() {
				By("Check the cluster is STS cluster or skip")
				isSTSCluster, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if !isSTSCluster {
					Skip("This case 45161 is only supported on STS cluster")
				}

				clusterName := "cluster-45161"
				operatorPrefix := "cluster-45161-asdf"
				isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).To(BeNil())

				By("Create cluster with one Y-1 version")
				ocmResourceService := rosaClient.OCMResource
				versionService := rosaClient.Version
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				rosalCommand, err := config.RetrieveClusterCreationCommand()
				Expect(err).To(BeNil())

				installerRole := rosalCommand.GetFlagValue("--role-arn", true)
				ar := accountRoleList.AccountRole(installerRole)
				Expect(ar).ToNot(BeNil())

				cg := rosalCommand.GetFlagValue("--channel-group", true)
				if cg == "" {
					cg = rosacli.VersionChannelGroupStable
				}

				versionList, err := versionService.ListAndReflectVersions(cg, isHostedCP)
				Expect(err).To(BeNil())
				Expect(versionList).ToNot(BeNil())
				foundVersion, err := versionList.FindNearestBackwardMinorVersion(ar.OpenshiftVersion, 1, false)
				Expect(err).To(BeNil())
				var clusterVersion string
				if foundVersion == nil {
					Skip("No cluster version < y-1 found for compatibility testing")
				}
				clusterVersion = foundVersion.Version

				replacingFlags := map[string]string{
					"--version":               clusterVersion,
					"--cluster-name":          clusterName,
					"--operator-roles-prefix": operatorPrefix,
				}

				if rosalCommand.GetFlagValue("--https-proxy", true) != "" {
					err = rosalCommand.DeleteFlag("--https-proxy", true)
					Expect(err).To(BeNil())
				}
				if rosalCommand.GetFlagValue("--http-proxy", true) != "" {
					err = rosalCommand.DeleteFlag("--http-proxy", true)
					Expect(err).To(BeNil())
				}

				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run")
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(BeNil())
				Expect(stdout.String()).To(ContainSubstring(fmt.Sprintf("Creating cluster '%s' should succeed", clusterName)))
			})
	})
