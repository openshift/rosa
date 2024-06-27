package e2e

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ciConfig "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("Edit cluster",
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
			labels.Medium, labels.Runtime.Day2,
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
				Expect(CD.DNS).To(Equal(jsonData.DigString("domain_prefix") + "." + jsonData.DigString("dns", "base_domain")))
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
				Expect(CD.Network[4]["Host Prefix"]).
					Should(ContainSubstring(strconv.FormatFloat(jsonData.DigFloat("network", "host_prefix"), 'f', -1, 64)))
				Expect(CD.InfraID).To(Equal(jsonData.DigString("infra_id")))
			})

		It("can restrict master API endpoint to direct, private connectivity or not - [id:38850]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check the cluster is not private cluster")
				private, err := clusterService.IsPrivateCluster(clusterID)
				Expect(err).To(BeNil())
				if private {
					SkipTestOnFeature("private")
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
					Expect(textData).
						Should(ContainSubstring(
							"You are choosing to make your cluster API private. You will not be able to access your cluster"))
					Expect(textData).Should(ContainSubstring("Updated cluster '%s'", clusterID))
				} else {
					Expect(err).ToNot(BeNil())
					Expect(rosaClient.Parser.TextData.Input(out).Parse().Tip()).
						Should(ContainSubstring(
							"Failed to update cluster: Cannot update listening mode of cluster's API on an AWS STS cluster"))
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
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check the cluster UWM is in expected status")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).ToNot(HaveOccurred())
				expectedUWMValue := "Enabled"
				recoverUWMStatus := false
				if clusterConfig.DisableWorkloadMonitoring {
					expectedUWMValue = "Disabled"
					recoverUWMStatus = true
				}
				Expect(clusterDetail.UserWorkloadMonitoring).To(Equal(expectedUWMValue))
				defer clusterService.EditCluster(clusterID,
					fmt.Sprintf("--disable-workload-monitoring=%v", recoverUWMStatus),
					"-y")

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
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Validate that deletion of upgrade policy for rosa cluster will work via rosacli")
				output, err := clusterService.DeleteUpgrade()
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring(`required flag(s) "cluster" not set`))

				By("Delete an non-existent upgrade when cluster has no scheduled policy")
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

		It("validation for create/delete upgrade policies for hypershift clusters via rosacli should work well - [id:73814]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				defer func() {
					_, err := clusterService.DeleteUpgrade("-c", clusterID, "-y")
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Skip testing if the cluster is not a HCP cluster")
				hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if !hostedCluster {
					SkipNotHosted()
				}

				By("Upgrade cluster without --control-plane flag")
				output, err := clusterService.Upgrade("-c", clusterID)
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: The '--control-plane' option is currently mandatory for Hosted Control Planes"))

				By("Upgrade cluster with invalid cluster id")
				invalidClusterID := common.GenerateRandomString(30)
				output, err = clusterService.Upgrade("-c", invalidClusterID)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: Failed to get cluster '%s': There is no cluster with identifier or name '%s'",
						invalidClusterID,
						invalidClusterID))

				By("Upgrade cluster with incorrect format of the date and time")
				output, err = clusterService.Upgrade(
					"-c", clusterID,
					"--control-plane",
					"--mode=auto",
					"--schedule-date=\"2024-06\"",
					"--schedule-time=\"09:00:12\"",
					"-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: schedule date should use the format 'yyyy-mm-dd'"))

				By("Upgrade cluster using --schedule, --schedule-date and --schedule-time flags at the same time")
				output, err = clusterService.Upgrade(
					"-c", clusterID,
					"--control-plane",
					"--mode=auto",
					"--schedule-date=\"2024-06-24\"",
					"--schedule-time=\"09:00\"",
					"--schedule=\"5 5 * * *\"",
					"-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: The '--schedule-date' and '--schedule-time' options are mutually exclusive with '--schedule'"))

				By("Upgrade cluster using --schedule and --version flags at the same time")
				output, err = clusterService.Upgrade(
					"-c", clusterID,
					"--control-plane",
					"--mode=auto",
					"--schedule=\"5 5 * * *\"",
					"--version=4.15.10",
					"-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: The '--schedule' option is mutually exclusive with '--version'"))

				By("Upgrade cluster with value not match the cron epression")
				output, err = clusterService.Upgrade(
					"-c", clusterID,
					"--control-plane",
					"--mode=auto",
					"--schedule=\"5 5\"",
					"-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: Schedule '\"5 5\"' is not a valid cron expression"))
			})

		// Excluded until bug OCM-8408 is resolved
		It("can verify delete protection on a rosa cluster - [id:73161]",
			labels.High, labels.Runtime.Day2, labels.Exclude,
			func() {
				By("Enable delete protection on the cluster")
				deleteProtection := "Enabled"
				_, err := clusterService.EditCluster(clusterID,
					"--enable-delete-protection=true",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Check the enable result from cluster description")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterDetail.EnableDeleteProtection).To(Equal(deleteProtection))

				By("Enable delete protection with invalid values")
				_, err = clusterService.EditCluster(clusterID,
					"--enable-delete-protection=aaa",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring(
					`Error: invalid argument "aaa" for "--enable-delete-protection" flag: strconv.ParseBool: parsing "aaa": invalid syntax`))

				_, err = clusterService.EditCluster(clusterID,
					"--enable-delete-protection=",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring(
					`Error: invalid argument "" for "--enable-delete-protection" flag: strconv.ParseBool: parsing "": invalid syntax`))

				By("Attempt to delete cluster with delete protection enabled")
				_, err = clusterService.DeleteCluster(clusterID, "-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring(
					`Delete-protection has been activated on this cluster and it cannot be deleted until delete-protection is disabled`))

				By("Disable delete protection on the cluster")
				deleteProtection = "Disabled"
				_, err = clusterService.EditCluster(clusterID,
					"--enable-delete-protection=false",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Check the disable result from cluster description")
				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				clusterDetail, err = clusterService.ReflectClusterDescription(output)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterDetail.EnableDeleteProtection).To(Equal(deleteProtection))
			})
	})

var _ = Describe("Classic cluster creation validation",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient  *rosacli.Client
			profilesMap map[string]*profilehandler.Profile
			profile     *profilehandler.Profile
		)

		BeforeEach(func() {
			// Init the client
			rosaClient = rosacli.NewClient()
			// Get a random profile
			profilesMap = profilehandler.ParseProfilesByFile(path.Join(ciConfig.Test.YAMLProfilesDir, "rosa-classic.yaml"))
			profilesNames := make([]string, 0, len(profilesMap))
			for k := range profilesMap {
				profilesNames = append(profilesNames, k)
			}
			profile = profilesMap[profilesNames[common.RandomInt(len(profilesNames))]]
			profile.NamePrefix = constants.DefaultNamePrefix

		})

		AfterEach(func() {
			errs := profilehandler.DestroyResourceByProfile(profile, rosaClient)
			Expect(len(errs)).To(Equal(0))
		})

		It("to check the basic validation for the classic rosa cluster creation by the rosa cli - [id:38770]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				By("Prepare creation command")
				var command string
				var rosalCommand config.Command
				flags, err := profilehandler.GenerateClusterCreateFlags(profile, rosaClient)
				Expect(err).To(BeNil())

				// nolint
				command = "rosa create cluster --cluster-name " + profile.ClusterConfig.Name + " " + strings.Join(flags, " ")
				rosalCommand = config.GenerateCommand(command)

				rosalCommand.AddFlags("--dry-run")
				originalClusterName := rosalCommand.GetFlagValue("--cluster-name", true)
				originalMachineType := rosalCommand.GetFlagValue("--compute-machine-type", true)
				originalRegion := rosalCommand.GetFlagValue("--region", true)
				if !rosalCommand.CheckFlagExist("--replicas") {
					rosalCommand.AddFlags("--replicas", "3")
				}
				originalReplicas := rosalCommand.GetFlagValue("--replicas", true)

				invalidClusterNames := []string{
					"1-test-1",
					"-test-cluster",
					"test-cluster-",
				}
				for _, cn := range invalidClusterNames {
					By("Check the validation for cluster-name")
					rosalCommand.ReplaceFlagValue(map[string]string{
						"--cluster-name": cn,
					})
					stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
					Expect(err).NotTo(BeNil())
					Expect(stdout.String()).
						To(ContainSubstring(
							"Cluster name must consist of no more than 54 lowercase alphanumeric characters or '-', " +
								"start with a letter, and end with an alphanumeric character"))
				}

				By("Check the validation for compute-machine-type")
				invalidMachineType := "not-exist-machine-type"
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--compute-machine-type": invalidMachineType,
					"--cluster-name":         originalClusterName,
				})
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).NotTo(BeNil())
				Expect(stdout.String()).To(ContainSubstring("Expected a valid machine type"))

				By("Check the validation for replicas")
				invalidReplicasTypeErrorMap := map[string]string{
					"4.5":  "invalid argument \"4.5\" for \"--replicas\" flag",
					"five": "invalid argument \"five\" for \"--replicas\" flag",
				}
				for k, v := range invalidReplicasTypeErrorMap {
					rosalCommand.ReplaceFlagValue(map[string]string{
						"--compute-machine-type": originalMachineType,
						"--replicas":             k,
					})
					stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
					Expect(err).NotTo(BeNil())
					Expect(stdout.String()).To(ContainSubstring(v))
				}
				if rosalCommand.CheckFlagExist("--multi-az") {
					invalidReplicasErrorMapMultiAZ := map[string]string{
						"2":  "Multi AZ cluster requires at least 3 compute nodes",
						"0":  "Multi AZ cluster requires at least 3 compute nodes",
						"-3": "must be non-negative",
						"5":  "Multi AZ clusters require that the number of compute nodes be a multiple of 3",
					}
					for k, v := range invalidReplicasErrorMapMultiAZ {
						rosalCommand.ReplaceFlagValue(map[string]string{
							"--replicas": k,
						})
						stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
						Expect(err).NotTo(BeNil())
						Expect(stdout.String()).To(ContainSubstring(v))
					}
				} else {
					invalidReplicasErrorMapSingeAZ := map[string]string{
						"1":  "requires at least 2 compute nodes",
						"0":  "requires at least 2 compute nodes",
						"-1": "must be non-negative",
					}
					for k, v := range invalidReplicasErrorMapSingeAZ {
						rosalCommand.ReplaceFlagValue(map[string]string{
							"--replicas": k,
						})
						stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
						Expect(err).NotTo(BeNil())
						Expect(stdout.String()).To(ContainSubstring(v))
					}
				}

				By("Check the validation for region")
				invalidRegion := "not-exist-region"
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--region":   invalidRegion,
					"--replicas": originalReplicas,
				})
				stdout, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).NotTo(BeNil())
				Expect(stdout.String()).To(ContainSubstring("Unsupported region"))

				By("Check the validation for billing-account for classic sts cluster")
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--region": originalRegion,
				})
				rosalCommand.AddFlags("--billing-account", "123456789")
				stdout, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).NotTo(BeNil())
				Expect(stdout.String()).
					To(ContainSubstring(
						"Billing accounts are only supported for Hosted Control Plane clusters"))
			})

		It("can allow sts cluster installation with compatible policies - [id:45161]",
			labels.High, labels.Runtime.Day1Supplemental,
			func() {
				By("Prepare creation command")
				var command string
				var rosalCommand config.Command
				flags, err := profilehandler.GenerateClusterCreateFlags(profile, rosaClient)
				Expect(err).To(BeNil())

				command = "rosa create cluster --cluster-name " + profile.ClusterConfig.Name + " " + strings.Join(flags, " ")
				rosalCommand = config.GenerateCommand(command)

				if !profile.ClusterConfig.STS {
					SkipTestOnFeature("policy")
				}

				clusterName := "cluster-45161"
				operatorPrefix := "cluster-45161-asdf"

				By("Create cluster with one Y-1 version")
				ocmResourceService := rosaClient.OCMResource
				versionService := rosaClient.Version
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())

				installerRole := rosalCommand.GetFlagValue("--role-arn", true)
				ar := accountRoleList.AccountRole(installerRole)
				Expect(ar).ToNot(BeNil())

				cg := rosalCommand.GetFlagValue("--channel-group", true)
				if cg == "" {
					cg = rosacli.VersionChannelGroupStable
				}

				versionList, err := versionService.ListAndReflectVersions(cg, rosalCommand.CheckFlagExist("--hosted-cp"))
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
					"-c":                      clusterName,
					"--operator-roles-prefix": operatorPrefix,
					"--domain-prefix":         clusterName,
				}

				if rosalCommand.GetFlagValue("--https-proxy", true) != "" {
					err = rosalCommand.DeleteFlag("--https-proxy", true)
					Expect(err).To(BeNil())
				}
				if rosalCommand.GetFlagValue("--no-proxy", true) != "" {
					err = rosalCommand.DeleteFlag("--no-proxy", true)
					Expect(err).To(BeNil())
				}
				if rosalCommand.GetFlagValue("--http-proxy", true) != "" {
					err = rosalCommand.DeleteFlag("--http-proxy", true)
					Expect(err).To(BeNil())
				}
				if rosalCommand.CheckFlagExist("--base-domain") {
					rosalCommand.DeleteFlag("--base-domain", true)
				}

				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run")
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(BeNil())
				Expect(stdout.String()).To(ContainSubstring(fmt.Sprintf("Creating cluster '%s' should succeed", clusterName)))
			})

		It("to validate to create the sts cluster with invalid tag - [id:56440]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterService := rosaClient.Cluster
				clusterName := "ocp-56440"

				By("Create cluster with invalid tag key")
				out, err := clusterService.CreateDryRun(
					clusterName, "--tags=~~~:cluster",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"expected a valid user tag key '~~~' matching ^[\\pL\\pZ\\pN_.:/=+\\-@]{1,128}$"))

				By("Create cluster with invalid tag value")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags=name:****",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"expected a valid user tag value '****' matching ^[\\pL\\pZ\\pN_.:/=+\\-@]{0,256}$"))

				By("Create cluster with duplicate tag key")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags=name:test1,op:clound,name:test2",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"invalid tags, user tag keys must be unique, duplicate key 'name' found"))

				By("Create cluster with invalid tag format")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags=test1,test2,test4",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"invalid tag format for tag '[test1]'. Expected tag format: 'key value'"))

				By("Create cluster with empty tag value")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags", "foo:",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"invalid tag format, tag key or tag value can not be empty"))

				By("Create cluster with invalid tag format")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags=name:gender:age",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"invalid tag format for tag '[name gender age]'. Expected tag format: 'key value'"))

			})

		It("Create cluster with invalid volume size [id:66372]",
			labels.Medium,
			labels.Runtime.Day1Negative,
			func() {
				minSize := 128
				maxSize := 16384
				clusterService := rosaClient.Cluster
				clusterName := "ocp-66372"
				client := rosacli.NewClient()

				By("Try a worker disk size that's too small")
				out, err := clusterService.CreateDryRun(
					clusterName, "--worker-disk-size", fmt.Sprintf("%dGiB", minSize-1),
				)
				Expect(err).To(HaveOccurred())
				stdout := client.Parser.TextData.Input(out).Parse().Tip()
				Expect(stdout).
					To(
						ContainSubstring(
							"Invalid root disk size: %d GiB. Must be between %d GiB and %d GiB.", minSize-1, minSize, maxSize))

				By("Try a worker disk size that's too big")
				out, err = clusterService.CreateDryRun(
					clusterName, "--worker-disk-size", fmt.Sprintf("%dGiB", maxSize+1),
				)
				Expect(err).To(HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				Expect(stdout).
					To(
						ContainSubstring(
							"Invalid root disk size: %d GiB. Must be between %d GiB and %d GiB.",
							maxSize+1,
							minSize,
							maxSize))

				By("Try a worker disk size that's negative")
				out, err = clusterService.CreateDryRun(
					clusterName, "--worker-disk-size", "-1GiB",
				)
				Expect(err).To(HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				Expect(stdout).
					To(
						ContainSubstring(
							"Expected a valid machine pool root disk size value '-1GiB': invalid disk size: " +
								"'-1Gi'. positive size required"))

				By("Try a worker disk size that's a string")
				out, err = clusterService.CreateDryRun(
					clusterName, "--worker-disk-size", "invalid",
				)
				Expect(err).To(HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				Expect(stdout).
					To(
						ContainSubstring(
							"Expected a valid machine pool root disk size value 'invalid': invalid disk size " +
								"format: 'invalid'. accepted units are Giga or Tera in the form of " +
								"g, G, GB, GiB, Gi, t, T, TB, TiB, Ti"))
			})

		It("to validate to create cluster with availability zones - [id:52692]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterService := rosaClient.Cluster
				clusterName := "ocp-52692"

				By("Create cluster with the zone not available in the region")
				out, err := clusterService.CreateDryRun(
					clusterName, "--availability-zones", "us-east-2e", "--region", "us-east-2",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"Expected a valid availability zone, 'us-east-2e' doesn't belong to region 'us-east-2' availability zones"))

				By("Create cluster with zones not match region")
				out, err = clusterService.CreateDryRun(
					clusterName, "--availability-zones", "us-west-2b", "--region", "us-east-2",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"Expected a valid availability zone, 'us-west-2b' doesn't belong to region 'us-east-2' availability zones"))

				By("Create cluster with dup zones set")
				out, err = clusterService.CreateDryRun(
					clusterName,
					"--availability-zones", "us-west-2b,us-west-2b,us-west-2b",
					"--region", "us-west-2",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"Found duplicate Availability Zone: us-west-2b"))

				By("Create cluster with both zone and subnet set")
				out, err = clusterService.CreateDryRun(
					clusterName,
					"--availability-zones", "us-west-2b",
					"--subnet-ids", "subnet-039f2a2a2d2d83e7f",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(ContainSubstring(
						"Setting availability zones is not supported for BYO VPC. " +
							"ROSA autodetects availability zones from subnet IDs provided"))
			})

		It("Validate --worker-mp-labels option for ROSA cluster creation - [id:71329]",
			labels.Medium, labels.Runtime.Day1Supplemental,
			func() {
				var (
					clusterName        = "cluster-71329"
					operatorPrefix     = "cluster-op-prefix"
					invalidKey         = "p*=test"
					emptyKey           = "=test"
					emptyWorkerMpLabel = ""
					longKey            = strings.Repeat("abcd1234", 16) + "=test"
					longValue          = "test=" + strings.Repeat("abcd1234", 16)
					duplicateKey       = "test=test1,test=test2"
					replacingFlags     = map[string]string{
						"-c":                     clusterName,
						"--cluster-name":         clusterName,
						"--domain-prefix":        clusterName,
						"--operator-role-prefix": operatorPrefix,
					}
				)

				By("Prepare creation command")
				var command string
				var rosalCommand config.Command
				flags, err := profilehandler.GenerateClusterCreateFlags(profile, rosaClient)
				Expect(err).To(BeNil())

				command = "rosa create cluster --cluster-name " + profile.ClusterConfig.Name + " " + strings.Join(flags, " ")
				rosalCommand = config.GenerateCommand(command)

				By("Create ROSA cluster with the --worker-mp-labels flag and invalid key")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", invalidKey, "-y")
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				index := strings.Index(invalidKey, "=")
				key := invalidKey[:index]
				Expect(output.String()).
					To(
						ContainSubstring(
							"Invalid label key '%s': name part must consist of alphanumeric characters, '-', '_' "+
								"or '.', and must start and end with an alphanumeric character",
							key))

				By("Create ROSA cluster with the --worker-mp-labels flag and empty key")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", emptyKey, "-y")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).
					To(
						ContainSubstring(
							"Invalid label key '': name part must be non-empty; name part must consist of alphanumeric " +
								"characters, '-', '_' or '.', and must start and end with an alphanumeric character"))

				By("Create ROSA cluster with the --worker-mp-labels flag without any value")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", emptyWorkerMpLabel, "-y")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Expected key=value format for labels"))

				By("Create ROSA cluster with the --worker-mp-labels flag and >63 character label key")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", longKey, "-y")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				index = strings.Index(longKey, "=")
				longLabelKey := longKey[:index]
				Expect(output.String()).
					To(
						ContainSubstring(
							"Invalid label key '%s': name part must be no more than 63 characters", longLabelKey))

				By("Create ROSA cluster with the --worker-mp-labels flag and >63 character label value")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", longValue, "-y")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				index = strings.Index(longValue, "=")
				longLabelValue := longValue[index+1:]
				key = longValue[:index]
				Expect(output.String()).
					To(
						ContainSubstring("Invalid label value '%s': at key: '%s': must be no more than 63 characters",
							longLabelValue,
							key))

				By("Create ROSA cluster with the --worker-mp-labels flag and duplicated key")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", duplicateKey, "-y")
				index = strings.Index(duplicateKey, "=")
				key = duplicateKey[:index]
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Duplicated label key '%s' used", key))
			})

		It("to validate to create the cluster with version not in the channel group - [id:74399]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterService := rosaClient.Cluster
				clusterName := "ocp-74399"

				By("Create cluster with version not in channel group")
				errorOutput, err := clusterService.CreateDryRun(
					clusterName, "--version=4.15.100",
				)
				Expect(err).NotTo(BeNil())
				Expect(errorOutput.String()).
					To(
						ContainSubstring("Expected a valid OpenShift version: A valid version number must be specified"))
			})

		It("to validate to create the cluster with setting 'fips' flag but '--etcd-encryption=false' - [id:74436]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterService := rosaClient.Cluster
				clusterName := "ocp-74436"

				By("Create cluster with fips flag but '--etcd-encryption=false")
				errorOutput, err := clusterService.CreateDryRun(
					clusterName, "--fips", "--etcd-encryption=false",
				)
				Expect(err).NotTo(BeNil())
				Expect(errorOutput.String()).To(ContainSubstring("etcd encryption cannot be disabled on clusters with FIPS mode"))
			})
	})

var _ = Describe("Classic cluster deletion validation",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient *rosacli.Client
		)

		BeforeEach(func() {
			// Init the client
			rosaClient = rosacli.NewClient()
		})

		It("to validate the ROSA cluster deletion will work via rosacli	- [id:38778]",
			labels.Medium, labels.Runtime.Day1Supplemental,
			func() {
				clusterService := rosaClient.Cluster
				notExistID := "no-exist-cluster-id"
				By("Delete the cluster without indicated cluster Name or ID")
				cmd := []string{"rosa", "delete", "cluster"}
				out, err := rosaClient.Runner.RunCMD(cmd)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("\"cluster\" not set"))

				By("Delete a non-existed cluster")
				out, err = clusterService.DeleteCluster(notExistID, "-y")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("There is no cluster with identifier or name"))

				By("Delete with unknown flag --interactive")
				out, err = clusterService.DeleteCluster(notExistID, "-y", "--interactive")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("unknown flag: --interactive"))
			})
	})

var _ = Describe("Classic cluster creation negative testing",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
		)
		BeforeEach(func() {

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
		})

		It("to validate to create the sts cluster with the version not compatible with the role version	- [id:45176]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterService = rosaClient.Cluster
				ocmResourceService := rosaClient.OCMResource

				By("Porepare version for testing")
				var accRoleversion string
				versionService := rosaClient.Version
				versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, false)
				Expect(err).To(BeNil())
				defaultVersion := versionList.DefaultVersion()
				Expect(defaultVersion).ToNot(BeNil())
				lowerVersion, err := versionList.FindNearestBackwardMinorVersion(defaultVersion.Version, 1, true)
				Expect(err).To(BeNil())
				Expect(lowerVersion).NotTo(BeNil())

				_, _, accRoleversion, err = lowerVersion.MajorMinor()
				Expect(err).To(BeNil())

				By("Create account-roles in low version 4.14")
				accrolePrefix := "testAr45176"
				path := "/a/b/"
				output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"--path", path,
					"--version", accRoleversion,
					"-y")
				Expect(err).To(BeNil())
				defer func() {
					By("Delete the account-roles")
					output, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
						"--prefix", accrolePrefix,
						"-y")

					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("Successfully deleted"))
				}()
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				ar := arl.DigAccountRoles(accrolePrefix, false)
				fmt.Println(ar)

				By("Create cluster with latest version and use the low version account-roles")
				clusterName := "cluster45176"
				operatorRolePrefix := "cluster45176-xvfa"
				out, err, _ := clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", operatorRolePrefix,
					"-y", "--dry-run",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("is not compatible with version"))
				Expect(out.String()).To(ContainSubstring("to create compatible roles and try again"))
			})

		It("to validate creating a cluster with invalid subnets - [id:72657]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				clusterService := rosaClient.Cluster
				clusterName := "ocp-72657"

				By("Create cluster with invalid subnets")
				out, err := clusterService.CreateDryRun(
					clusterName, "--subnet-ids", "subnet-xxx",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("The subnet ID 'subnet-xxx' does not exist"))

			})
	})

var _ = Describe("HCP cluster creation negative testing",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			profilesMap    map[string]*profilehandler.Profile
			profile        *profilehandler.Profile
			command        string
			rosalCommand   config.Command
		)
		BeforeEach(func() {

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster

			// Get a random profile
			profilesMap = profilehandler.ParseProfilesByFile(path.Join(ciConfig.Test.YAMLProfilesDir, "rosa-hcp.yaml"))
			profilesNames := make([]string, 0, len(profilesMap))
			for k := range profilesMap {
				profilesNames = append(profilesNames, k)
			}
			profile = profilesMap[profilesNames[common.RandomInt(len(profilesNames))]]
			profile.NamePrefix = constants.DefaultNamePrefix

			By("Prepare creation command")
			flags, err := profilehandler.GenerateClusterCreateFlags(profile, rosaClient)
			Expect(err).To(BeNil())

			command = "rosa create cluster --cluster-name " + profile.ClusterConfig.Name + " " + strings.Join(flags, " ")
			rosalCommand = config.GenerateCommand(command)
		})

		AfterEach(func() {
			errs := profilehandler.DestroyResourceByProfile(profile, rosaClient)
			Expect(len(errs)).To(Equal(0))
		})

		It("create HCP cluster with network type validation can work well via rosa cli - [id:73725]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterName := common.GenerateRandomName("cluster-73725", 2)
				By("Create HCP cluster with --no-cni and \"--network-type={OVNKubernetes, OpenshiftSDN}\" at the same time")
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--no-cni", "--network-type='{OVNKubernetes,OpenshiftSDN}'", "-y")
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).
					To(
						ContainSubstring(
							"ERR: Expected a valid network type. Valid values: [OpenShiftSDN OVNKubernetes]"))

				By("Create HCP cluster with invalid --no-cni value")
				rosalCommand.DeleteFlag("--network-type", true)
				rosalCommand.DeleteFlag("--no-cni", true)
				rosalCommand.AddFlags("--no-cni=ui")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).
					To(
						ContainSubstring(
							`Failed to execute root command: invalid argument "ui" for "--no-cni" flag: ` +
								`strconv.ParseBool: parsing "ui": invalid syntax`))

				By("Create HCP cluster with --no-cni and --network-type=OVNKubernetes at the same time")
				rosalCommand.DeleteFlag("--no-cni=ui", false)
				rosalCommand.AddFlags("--no-cni", "--network-type=OVNKubernetes")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("ERR: --no-cni and --network-type are mutually exclusive parameters"))

				By("Create non-HCP cluster with --no-cni flag")
				output, err = clusterService.CreateDryRun("ocp-73725", "--no-cni")
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("ERR: Disabling CNI is supported only for Hosted Control Planes"))
			})

		It("to validate creating a hosted cluster with invalid subnets - [id:72657]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-72657"
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}

				By("Create cluster with invalid subnets")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--subnet-ids", "subnet-xxx", "-y")
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("The subnet ID 'subnet-xxx' does not exist"))
			})

		It("to validate creating a hosted cluster with invalid ingress - [id:71174]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				rosalCommand, err := config.RetrieveClusterCreationCommand(ciConfig.Test.CreateCommandFile)
				Expect(err).To(BeNil())

				clusterName := "ocp-71174"
				operatorPrefix := common.GenerateRandomName("cluster-oper", 2)

				replacingFlags := map[string]string{
					"-c":                     clusterName,
					"--cluster-name":         clusterName,
					"--domain-prefix":        clusterName,
					"--operator-role-prefix": operatorPrefix,
				}

				By("Create cluster with invalid ingress")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--default-ingress-route-selector", "10.0.0.1", "-y")
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))

				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(
					ContainSubstring("Updating default ingress settings is not supported for Hosted Control Plane clusters"))
			})

		It("to validate creating a hosted cluster with CIDR that doesn't exist - [id:70970]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-70970"
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}

				By("Create cluster with a CIDR that doesn't exist")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--machine-cidr", "192.168.1.0/23", "-y")
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(
						ContainSubstring(
							"ERR: All Hosted Control Plane clusters need a pre-configured VPC. " +
								"Please check: " +
								"https://docs.openshift.com/rosa/rosa_hcp/rosa-hcp-sts-creating-a-cluster-quickly.html#rosa-hcp-creating-vpc"))
			})

		It("to validate create cluster with external_auth_config can work well - [id:73755]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				By("Create non-HCP cluster with --external-auth-providers-enabled")
				clusterName := common.GenerateRandomName("ocp-73755", 2)
				output, err := clusterService.CreateDryRun(clusterName, "--external-auth-providers-enabled")
				Expect(err).To(HaveOccurred())
				Expect(output.String()).
					To(
						ContainSubstring(
							"ERR: External authentication configuration is only supported for a Hosted Control Plane cluster."))

				By("Create HCP cluster with --external-auth-providers-enabled and cluster version lower than 4.15")
				cg := rosalCommand.GetFlagValue("--channel-group", true)
				if cg == "" {
					cg = rosacli.VersionChannelGroupStable
				}
				versionList, err := rosaClient.Version.ListAndReflectVersions(cg, rosalCommand.CheckFlagExist("--hosted-cp"))
				Expect(err).To(BeNil())
				Expect(versionList).ToNot(BeNil())
				previousVersionsList, err := versionList.FindNearestBackwardMinorVersion("4.14", 0, true)
				Expect(err).ToNot(HaveOccurred())
				foundVersion := previousVersionsList.Version
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
					"--version":       foundVersion,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				if !rosalCommand.CheckFlagExist("--external-auth-providers-enabled") {
					rosalCommand.AddFlags("--dry-run", "--external-auth-providers-enabled", "-y")
				} else {
					rosalCommand.AddFlags("--dry-run", "-y")
				}
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).
					To(
						ContainSubstring(
							"External authentication is only supported in version '4.15.9' or greater, current cluster version is '%s'",
							foundVersion))
			})
	})

var _ = Describe("Create cluster with availability zones testing",
	labels.Feature.Machinepool,
	func() {
		defer GinkgoRecover()
		var (
			availabilityZones  string
			clusterID          string
			rosaClient         *rosacli.Client
			machinePoolService rosacli.MachinePoolService
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
			machinePoolService = rosaClient.MachinePool

			By("Skip testing if the cluster is not a Classic cluster")
			isHostedCP, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if isHostedCP {
				SkipNotClassic()
			}
		})

		AfterEach(func() {
			By("Clean remaining resources")
			rosaClient.CleanResources(clusterID)

		})

		It("User can set availability zones - [id:52691]",
			labels.Critical, labels.Runtime.Day1Post,
			func() {
				profile := profilehandler.LoadProfileYamlFileByENV()
				mpID := "mp-52691"
				machineType := "m5.2xlarge" // nolint:goconst

				if profile.ClusterConfig.BYOVPC || profile.ClusterConfig.Zones == "" {
					SkipTestOnFeature("create rosa cluster with availability zones")
				}

				By("List machine pool and check the default one")
				availabilityZones = profile.ClusterConfig.Zones
				output, err := machinePoolService.ListMachinePool(clusterID)
				Expect(err).To(BeNil())
				mpList, err := machinePoolService.ReflectMachinePoolList(output)
				Expect(err).To(BeNil())
				mp := mpList.Machinepool(constants.DefaultClassicWorkerPool)
				Expect(err).To(BeNil())
				Expect(common.ReplaceCommaSpaceWithComma(mp.AvalaiblityZones)).To(Equal(availabilityZones))

				By("Create another machinepool")
				_, err = machinePoolService.CreateMachinePool(clusterID, mpID,
					"--replicas", "3",
					"--instance-type", machineType,
				)
				Expect(err).ToNot(HaveOccurred())

				By("List machine pool and check availability zone")
				output, err = machinePoolService.ListMachinePool(clusterID)
				Expect(err).To(BeNil())
				mpList, err = machinePoolService.ReflectMachinePoolList(output)
				Expect(err).To(BeNil())
				mp = mpList.Machinepool(mpID)
				Expect(err).To(BeNil())
				Expect(common.ReplaceCommaSpaceWithComma(mp.AvalaiblityZones)).To(Equal(availabilityZones))
			})
	})
