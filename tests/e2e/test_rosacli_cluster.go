package e2e

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

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
			upgradeService rosacli.UpgradeService
			clusterConfig  *config.ClusterConfig
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			upgradeService = rosaClient.Upgrade

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
						Expect(CD.MultiAZ).To(Equal(jsonData.DigBool("multi_az")))
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
				// nolint
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
				output, err := upgradeService.DeleteUpgrade()
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring(`required flag(s) "cluster" not set`))

				By("Delete an non-existent upgrade when cluster has no scheduled policy")
				output, err = upgradeService.DeleteUpgrade("-c", clusterID)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring(`There are no scheduled upgrades on cluster '%s'`, clusterID))

				By("Delete with unknown flag --interactive")
				output, err = upgradeService.DeleteUpgrade("-c", clusterID, "--interactive")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Error: unknown flag: --interactive"))
			})

		It("validation for create/delete upgrade policies for hypershift clusters via rosacli should work well - [id:73814]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				defer func() {
					_, err := upgradeService.DeleteUpgrade("-c", clusterID, "-y")
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Skip testing if the cluster is not a HCP cluster")
				hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if !hostedCluster {
					SkipNotHosted()
				}

				By("Upgrade cluster without --control-plane flag")
				output, err := upgradeService.Upgrade("-c", clusterID)
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: The '--control-plane' option is currently mandatory for Hosted Control Planes"))

				By("Upgrade cluster with invalid cluster id")
				invalidClusterID := common.GenerateRandomString(30)
				output, err = upgradeService.Upgrade("-c", invalidClusterID)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: Failed to get cluster '%s': There is no cluster with identifier or name '%s'",
						invalidClusterID,
						invalidClusterID))

				By("Upgrade cluster with incorrect format of the date and time")
				output, err = upgradeService.Upgrade(
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
				output, err = upgradeService.Upgrade(
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
				output, err = upgradeService.Upgrade(
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
				output, err = upgradeService.Upgrade(
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

		It("can via rosa-cli - [id:60275]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				By("Check the cluster is private cluster")
				private, err := clusterService.IsPrivateCluster(clusterID)
				Expect(err).To(BeNil())
				if !private {
					SkipTestOnFeature("private")
				}
				isSTS, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).To(BeNil())
				isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).To(BeNil())

				By("Run command to check help message of edit cluster")
				out, editErr := clusterService.EditCluster(clusterID, "-h")
				Expect(editErr).ToNot(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("rosa edit cluster [flags]"))
				Expect(out.String()).Should(ContainSubstring("rosa edit cluster -c mycluster --private"))
				Expect(out.String()).Should(ContainSubstring("rosa edit cluster -c mycluster --interactive"))

				By("Edit the cluster with '--private=false' flag")
				out, editErr = clusterService.EditCluster(
					clusterID,
					"--private=false",
					"-y",
				)

				defer func() {
					By("Edit cluster to private back to false")
					out, err := clusterService.EditCluster(
						clusterID,
						"--private",
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
					Expect(CD.Private).To(Equal("Yes"))
				}()

				textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()

				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				if isSTS && !isHostedCP {
					Expect(editErr).ToNot(BeNil())
					Expect(textData).Should(ContainSubstring("Failed to update cluster: Cannot update listening " +
						"mode of cluster's API on an AWS STS cluster"))

					Expect(CD.Private).To(Equal("Yes"))
				} else {
					Expect(editErr).To(BeNil())
					Expect(textData).Should(ContainSubstring("Updated cluster '%s'", clusterID))

					Expect(CD.Private).To(Equal("No"))
				}

				By("Edit the cluster with '--private' flag")
				out, editErr = clusterService.EditCluster(
					clusterID,
					"--private",
					"-y",
				)
				Expect(editErr).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
				Expect(textData).Should(ContainSubstring("You are choosing to make your cluster API private. " +
					"You will not be able to access your cluster until you edit network settings in your cloud provider. " +
					"To also change the privacy setting of the application router endpoints, use the 'rosa edit ingress' command."))
				Expect(textData).Should(ContainSubstring("Updated cluster '%s'", clusterID))

				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err = clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				Expect(CD.Private).To(Equal("Yes"))
			})

		// Excluded until bug on OCP-73161 is resolved
		It("can verify delete protection on a rosa cluster - [id:73161]",
			labels.High, labels.Runtime.Day2, labels.Exclude,
			func() {
				By("Get original delete protection value")
				output, err := clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				originalDeleteProtection := output.EnableDeleteProtection

				By("Enable delete protection on the cluster")
				deleteProtection := constants.DeleteProtectionEnabled
				_, err = clusterService.EditCluster(clusterID, "--enable-delete-protection=true", "-y")
				Expect(err).ToNot(HaveOccurred())
				defer clusterService.EditCluster(clusterID,
					fmt.Sprintf("--enable-delete-protection=%s", originalDeleteProtection), "-y")

				By("Check the enable result from cluster description")
				output, err = clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.EnableDeleteProtection).To(Equal(deleteProtection))

				By("Attempt to delete cluster with delete protection enabled")
				out, err := clusterService.DeleteCluster(clusterID, "-y")
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
				Expect(textData).Should(ContainSubstring(
					`Delete-protection has been activated on this cluster and 
				it cannot be deleted until delete-protection is disabled`))

				By("Disable delete protection on the cluster")
				deleteProtection = constants.DeleteProtectionDisabled
				_, err = clusterService.EditCluster(clusterID, "--enable-delete-protection=false", "-y")
				Expect(err).ToNot(HaveOccurred())

				By("Check the disable result from cluster description")
				output, err = clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.EnableDeleteProtection).To(Equal(deleteProtection))
			})

		// Excluded until bug on OCP-74656 is resolved
		It("can verify delete protection on a rosa cluster negative - [id:74656]",
			labels.Medium, labels.Runtime.Day2, labels.Exclude,
			func() {
				By("Enable delete protection with invalid values")
				resp, err := clusterService.EditCluster(clusterID,
					"--enable-delete-protection=aaa",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).Should(ContainSubstring(`Error: invalid argument "aaa" for "--enable-delete-protection"`))

				resp, err = clusterService.EditCluster(clusterID, "--enable-delete-protection=", "-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).Should(ContainSubstring(`Error: invalid argument "" for "--enable-delete-protection"`))
			})
	})

var _ = Describe("Classic cluster creation validation",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			profilesMap    map[string]*profilehandler.Profile
			profile        *profilehandler.Profile
			clusterService rosacli.ClusterService
		)

		BeforeEach(func() {
			// Init the client
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			// Get a random profile
			profilesMap = profilehandler.ParseProfilesByFile(path.Join(ciConfig.Test.YAMLProfilesDir, "rosa-classic.yaml"))
			profilesNames := make([]string, 0, len(profilesMap))
			for k, v := range profilesMap {
				if !v.ClusterConfig.SharedVPC {
					profilesNames = append(profilesNames, k)
				}
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
					if !profile.ClusterConfig.AutoscalerEnabled {
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
					}
				} else {
					if !profile.ClusterConfig.AutoscalerEnabled {
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
				clusterName := "ocp-74436"

				By("Create cluster with fips flag but '--etcd-encryption=false")
				errorOutput, err := clusterService.CreateDryRun(
					clusterName, "--fips", "--etcd-encryption=false",
				)
				Expect(err).NotTo(BeNil())
				Expect(errorOutput.String()).To(ContainSubstring("etcd encryption cannot be disabled on clusters with FIPS mode"))
			})

		It("Create rosa cluster with additional security groups will validate well via rosacli - [id:68971]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				var (
					ocmResourceService  = rosaClient.OCMResource
					ocpVersionBelow4_14 = "4.13.44"
					ocpVersion4_14      = "4.14.0"
					index               int
					flagName            string
					hostedCP            = true
					installerRoleArn    string
					region              = "us-west-2"
					SGIdsMoreThanTen    = 11
					caseNumber          = "68971"
					clusterName         = "ocp-68971"
					securityGroups      = map[string]string{
						"--additional-infra-security-group-ids":         "sg-aisgi",
						"--additional-control-plane-security-group-ids": "sg-acpsgi",
						"--additional-compute-security-group-ids":       "sg-acsgi",
					}
					invalidSecurityGroups = map[string]string{
						"--additional-infra-security-group-ids":         "invalid",
						"--additional-control-plane-security-group-ids": "invalid",
						"--additional-compute-security-group-ids":       "invalid",
					}
				)

				By("Prepare a vpc for the testing")
				vpc, err := vpc_client.PrepareVPC(caseNumber, region, "", false, "")
				Expect(err).ToNot(HaveOccurred())
				defer vpc.DeleteVPCChain()

				subnetMap, err := profilehandler.PrepareSubnets(vpc, region, []string{}, false)
				Expect(err).ToNot(HaveOccurred())

				By("Prepare additional security group ids for testing")
				sgIDs, err := profilehandler.PrepareAdditionalSecurityGroups(vpc, SGIdsMoreThanTen, caseNumber)
				Expect(err).ToNot(HaveOccurred())

				subnetsFlagValue := strings.Join(append(subnetMap["private"], subnetMap["public"]...), ",")
				rosaclient := rosacli.NewClient()

				By("Try creating cluster with additional security groups but no subnet-ids")
				for additionalSecurityGroupFlag := range securityGroups {
					output, err, _ := rosaclient.Cluster.Create(
						clusterName,
						"--region", region,
						"--replicas", "3",
						additionalSecurityGroupFlag, strings.Join(sgIDs, ","),
					)
					Expect(err).To(HaveOccurred())
					index = strings.Index(additionalSecurityGroupFlag, "a")
					flagName = additionalSecurityGroupFlag[index:]
					Expect(output.String()).To(ContainSubstring(
						"Setting the `%s` flag is only allowed for BYO VPC clusters",
						flagName))
				}

				By("Try creating cluster with additional security groups and ocp version lower than 4.14")
				for additionalSecurityGroupFlag := range securityGroups {
					output, err, _ := rosaclient.Cluster.Create(
						clusterName,
						"--region", region,
						"--replicas", "3",
						"--subnet-ids", subnetsFlagValue,
						additionalSecurityGroupFlag, strings.Join(sgIDs, ","),
						"--version", ocpVersionBelow4_14,
						"-y",
					)
					Expect(err).To(HaveOccurred())
					index = strings.Index(additionalSecurityGroupFlag, "a")
					flagName = additionalSecurityGroupFlag[index:]
					Expect(output.String()).To(ContainSubstring(
						"Parameter '%s' is not supported prior to version '4.14.0'",
						flagName))
				}

				By("Try creating cluster with invalid additional security groups")
				for additionalSecurityGroupFlag, value := range invalidSecurityGroups {
					output, err, _ := rosaclient.Cluster.Create(
						clusterName,
						"--region", region,
						"--replicas", "3",
						"--subnet-ids", subnetsFlagValue,
						additionalSecurityGroupFlag, value,
						"--version", ocpVersion4_14,
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).To(ContainSubstring("Security Group ID '%s' doesn't have 'sg-' prefix", value))
				}

				By("Try creating cluster with additional security groups with invalid and more than 10 SG ids")
				for additionalSecurityGroupFlag := range securityGroups {

					output, err, _ := rosaclient.Cluster.Create(
						clusterName,
						"--region", region,
						"--replicas", "3",
						"--subnet-ids", subnetsFlagValue,
						additionalSecurityGroupFlag, strings.Join(sgIDs, ","),
						"--version", ocpVersion4_14,
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).To(ContainSubstring(
						"Failed to create cluster: The limit for Additional Security Groups is '10', but '11' have been supplied"),
					)
				}

				By("Try creating HCP cluster with additional security groups flag")
				for additionalSecurityGroupFlag := range securityGroups {
					By("Create account-roles of hosted-cp")
					_, err := ocmResourceService.CreateAccountRole("--mode", "auto",
						"--prefix", "akanni",
						"--hosted-cp",
						"-y")
					Expect(err).To(BeNil())

					By("Get the installer role arn")
					accountRoleList, _, err := ocmResourceService.ListAccountRole()
					Expect(err).To(BeNil())
					installerRole := accountRoleList.InstallerRole("akanni", hostedCP)
					Expect(installerRole).ToNot(BeNil())
					installerRoleArn = installerRole.RoleArn

					By("Create managed=false oidc config in auto mode")
					output, err := ocmResourceService.CreateOIDCConfig("--mode", "auto",
						"--prefix", "akanni",
						"--managed=false",
						"--installer-role-arn", installerRoleArn,
						"-y")
					Expect(err).To(BeNil())
					oidcPrivodeARNFromOutputMessage := common.ExtractOIDCProviderARN(output.String())
					oidcPrivodeIDFromOutputMessage := common.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)
					unmanagedOIDCConfigID, err := ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("Created OIDC provider with ARN"))
					output, err, _ = rosaclient.Cluster.Create(
						clusterName,
						"--region", region,
						"--replicas", "3",
						"--subnet-ids", subnetsFlagValue,
						additionalSecurityGroupFlag, strings.Join(sgIDs, ","),
						"--hosted-cp",
						"--oidc-config-id", unmanagedOIDCConfigID,
						"--billing-account", profile.ClusterConfig.BillingAccount,
						"-y",
					)
					Expect(err).To(HaveOccurred())
					index = strings.Index(additionalSecurityGroupFlag, "a")
					flagName = additionalSecurityGroupFlag[index:]
					Expect(output.String()).To(ContainSubstring(
						"Parameter '%s' is not supported for Hosted Control Plane clusters",
						flagName))
				}
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
			rosaClient               *rosacli.Client
			clusterService           rosacli.ClusterService
			accountRolePrefixToClean string
			ocmResourceService       rosacli.OCMResourceService
		)
		BeforeEach(func() {

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource
		})
		AfterEach(func() {
			By("Delete the resources for testing")
			if accountRolePrefixToClean != "" {
				By("Delete the account-roles")
				rosaClient.Runner.UnsetArgs()
				_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", accountRolePrefixToClean,
					"-y")
				Expect(err).To(BeNil())
			}
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
		It("to validate to create sts cluster with invalid role arn and operator IAM roles prefix - [id:41824]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				By("Create account-roles for testing")
				accountRolePrefixToClean = "testAr41824"
				output, err := ocmResourceService.CreateAccountRole(
					"--mode", "auto",
					"--prefix", accountRolePrefixToClean,
					"-y")
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				ar := arl.DigAccountRoles(accountRolePrefixToClean, false)

				By("Create cluster with operator roles prefix longer than 32 characters")
				clusterName := "test41824"
				oprPrefixExceed32Chars := "opPrefix45742opPrefix45742opPrefix45742"
				output, err, _ = clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", oprPrefixExceed32Chars,
					"-y",
				)
				Expect(err).ToNot(BeNil())
				Expect(output.String()).To(ContainSubstring("Expected a prefix with no more than 32 characters"))

				By("Create cluster with operator roles prefix with invalid format")
				oprPrefixInvaliad := "%%%###@@@"
				output, err, _ = clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", oprPrefixInvaliad,
					"-y",
				)
				Expect(err).ToNot(BeNil())
				Expect(output.String()).To(ContainSubstring("Expected valid operator roles prefix matching"))

				By("Create cluster with account roles with invalid format")
				invalidArn := "invalidaArnFormat"
				output, err, _ = clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", invalidArn,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", clusterName,
					"-y",
				)
				Expect(err).ToNot(BeNil())
				Expect(output.String()).To(ContainSubstring("Expected a valid Role ARN"))
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
		It("to validate to create sts cluster with dulicated role arns- [id:74620]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				By("Create account-roles for testing")
				accountRolePrefixToClean = "testAr74620"
				output, err := ocmResourceService.CreateAccountRole(
					"--mode", "auto",
					"--prefix", accountRolePrefixToClean,
					"-y")
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				ar := arl.DigAccountRoles(accountRolePrefixToClean, false)

				By("Create cluster with operator roles prefix longer than 32 characters")
				clusterName := "test41824"
				output, err, _ = clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.SupportRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", clusterName,
					"-y",
				)
				Expect(err).ToNot(BeNil())
				Expect(output.String()).To(ContainSubstring("ROSA IAM roles must have unique ARNs"))
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

		It("to validate creating a hosted cluster with invalid subnets - [id:75916]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-75916"
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

		It("to validate '--ec2-metadata-http-tokens' flag during creating cluster - [id:64078]",
			labels.Medium,
			labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-64078"

				By("Create classic cluster with invalid httpTokens")
				errorOutput, err := clusterService.CreateDryRun(
					clusterName, "--ec2-metadata-http-tokens=invalid",
				)
				Expect(err).NotTo(BeNil())
				Expect(errorOutput.String()).
					To(
						ContainSubstring(
							"ERR: Expected a valid http tokens value : " +
								"ec2-metadata-http-tokens value should be one of 'required', 'optional'"))

				By("Create HCP cluster  with invalid httpTokens")
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}

				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--ec2-metadata-http-tokens=invalid", "-y")
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).NotTo(BeNil())
				Expect(out.String()).
					To(
						ContainSubstring(
							"ERR: Expected a valid http tokens value : " +
								"ec2-metadata-http-tokens value should be one of 'required', 'optional'"))
			})

		It("expose additional allowed principals for HCP negative - [id:74433]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				By("Create hcp cluster using --additional-allowed-principals and invalid formatted arn")
				clusterName := "ocp-74408"
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}

				By("Create cluster with invalid additional allowed principals")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				if rosalCommand.CheckFlagExist("--additional-allowed-principals") {
					rosalCommand.DeleteFlag("--additional-allowed-principals", true)
				}
				rosalCommand.AddFlags("--dry-run", "--additional-allowed-principals", "zzzz", "-y")
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					To(
						ContainSubstring(
							"ERR: Expected valid ARNs for additional allowed principals list: Invalid ARN: arn: invalid prefix"))

				By("Create classic cluster with additional allowed principals")
				output, err := clusterService.CreateDryRun(clusterName,
					"--additional-allowed-principals", "zzzz",
					"-y", "--debug")
				Expect(err).To(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(
						ContainSubstring(
							"ERR: Additional Allowed Principals is supported only for Hosted Control Planes"))
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
var _ = Describe("Create sts and hcp cluster with the IAM roles with path setting", labels.Feature.Cluster, func() {
	defer GinkgoRecover()
	var (
		clusterID      string
		rosaClient     *rosacli.Client
		profile        *profilehandler.Profile
		err            error
		clusterService rosacli.ClusterService
		path           string
		awsClient      *aws_client.AWSClient
	)

	BeforeEach(func() {
		By("Get the cluster")
		profile = profilehandler.LoadProfileYamlFileByENV()
		Expect(err).ToNot(HaveOccurred())

		By("Init the client")
		rosaClient = rosacli.NewClient()
		clusterService = rosaClient.Cluster
		clusterID = config.GetClusterID()
	})

	AfterEach(func() {
		By("Clean remaining resources")
		rosaClient.CleanResources(clusterID)

	})

	It("to check the IAM roles can be used to create clsuters - [id:53570]",
		labels.Critical, labels.Runtime.Day1Post,
		func() {
			By("Skip testing if the cluster is a Classic NON-STS cluster")
			isSTS, err := clusterService.IsSTSCluster(clusterID)
			Expect(err).To(BeNil())
			if !isSTS {
				Skip("Skip this case as it only supports on STS clusters")
			}

			By("Check the account-roles using on the cluster has path setting")
			if profile.AccountRoleConfig.Path == "" {
				Skip("Skip this case as it only checks the cluster which has the account-roles with path setting")
			} else {
				path = profile.AccountRoleConfig.Path
			}

			By("Get operator-roles arns and installer role arn")
			output, err := clusterService.DescribeCluster(clusterID)
			Expect(err).To(BeNil())
			CD, err := clusterService.ReflectClusterDescription(output)
			Expect(err).To(BeNil())
			operatorRolesArns := CD.OperatorIAMRoles

			installerRole := CD.STSRoleArn
			Expect(installerRole).To(ContainSubstring(path))

			By("Check the operator-roles has the path setting")
			for _, pArn := range operatorRolesArns {
				Expect(pArn).To(ContainSubstring(path))
			}
			if profile.ClusterConfig.STS && !profile.ClusterConfig.HCP {
				By("Check the operator role policies has the path setting")
				awsClient, err = aws_client.CreateAWSClient("", "")
				Expect(err).To(BeNil())
				for _, pArn := range operatorRolesArns {
					_, roleName, err := common.ParseRoleARN(pArn)
					Expect(err).To(BeNil())
					attachedPolicy, err := awsClient.ListRoleAttachedPolicies(roleName)
					Expect(err).To(BeNil())
					Expect(*(attachedPolicy[0].PolicyArn)).To(ContainSubstring(path))
				}
			}
		})
})

var _ = Describe("Create cluster with existing operator-roles prefix which roles are not using byo oidc",
	labels.Feature.Cluster, func() {
		defer GinkgoRecover()
		var (
			// clusterID  string
			rosaClient         *rosacli.Client
			err                error
			accountRolePrefix  string
			ocmResourceService rosacli.OCMResourceService
			clusterNameToClean string
			clusterService     rosacli.ClusterService
			clusterID          string
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
			clusterService = rosaClient.Cluster
		})

		AfterEach(func() {
			By("Delete the cluster")
			if clusterNameToClean != "" {
				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				Expect(err).To(BeNil())

				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				Expect(err).To(BeNil())
				clusterID = clusterList.ClusterByName(clusterNameToClean).ID

				rosaClient.Runner.UnsetArgs()
				_, err = clusterService.DeleteCluster(clusterID, "-y")
				Expect(err).To(BeNil())

				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterDeleted(clusterID, 3, 30)
				Expect(err).To(BeNil())
			}
			By("Delete operator-roles")
			_, err = ocmResourceService.DeleteOperatorRoles(
				"-c", clusterID,
				"--mode", "auto",
				"-y",
			)
			Expect(err).To(BeNil())

			By("Delete oidc-provider")
			_, err = ocmResourceService.DeleteOIDCProvider(
				"-c", clusterID,
				"--mode", "auto",
				"-y")
			Expect(err).To(BeNil())

			By("Delete account-roles")
			if accountRolePrefix != "" {
				By("Delete the account-roles")
				rosaClient.Runner.UnsetArgs()
				_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y")
				Expect(err).To(BeNil())
			}

		})

		It("to validate to create cluster with existing operator roles prefix - [id:45742]",
			labels.Critical, labels.Runtime.Day1Supplemental,
			func() {
				By("Create acount-roles")
				accountRolePrefix = "testAr45742"
				output, err := ocmResourceService.CreateAccountRole(
					"--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y")
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				ar := arl.DigAccountRoles(accountRolePrefix, false)

				By("Create one sts cluster")
				clusterNameToClean = "test-45742"
				operatorRolePreifx := "opPrefix45742"
				_, err, _ = clusterService.Create(
					clusterNameToClean, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", operatorRolePreifx,
					"-y",
				)
				Expect(err).To(BeNil())

				By("Create another cluster with the same operator-roless-prefix")
				clusterName := "test-45742b"
				out, err, _ := clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", operatorRolePreifx,
					"-y",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("already exists"))
				Expect(out.String()).To(ContainSubstring("provide a different prefix"))
			})
	})

var _ = Describe("create/delete operator-roles and oidc-provider to cluster",
	labels.Feature.Cluster, func() {
		defer GinkgoRecover()
		var (
			rosaClient *rosacli.Client

			accountRolePrefix  string
			ocmResourceService rosacli.OCMResourceService
			clusterNameToClean string
			clusterService     rosacli.ClusterService
			clusterID          string
			defaultDir         string
			dirToClean         string
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
			clusterService = rosaClient.Cluster

			By("Get the default dir")
			defaultDir = rosaClient.Runner.GetDir()
		})

		AfterEach(func() {
			By("Go back original by setting runner dir")
			rosaClient.Runner.SetDir(defaultDir)

			By("Delete cluster")
			rosaClient.Runner.UnsetArgs()
			clusterListout, err := clusterService.List()
			Expect(err).To(BeNil())
			clusterList, err := clusterService.ReflectClusterList(clusterListout)
			Expect(err).To(BeNil())

			if clusterList.IsExist(clusterID) {
				rosaClient.Runner.UnsetArgs()
				_, err = clusterService.DeleteCluster(clusterID, "-y")
				Expect(err).To(BeNil())
			}
			By("Delete operator-roles")
			_, err = ocmResourceService.DeleteOperatorRoles(
				"-c", clusterID,
				"--mode", "auto",
				"-y",
			)
			Expect(err).To(BeNil())

			By("Delete oidc-provider")
			_, err = ocmResourceService.DeleteOIDCProvider(
				"-c", clusterID,
				"--mode", "auto",
				"-y")
			Expect(err).To(BeNil())

			By("Delete the account-roles")
			rosaClient.Runner.UnsetArgs()
			_, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", accountRolePrefix,
				"-y")
			Expect(err).To(BeNil())
		})

		It("to create/delete operator-roles and oidc-provider to cluster in manual mode - [id:43053]",
			labels.Critical, labels.Runtime.Day1Supplemental,
			func() {
				By("Create acount-roles")
				accountRolePrefix = "testAr43053"
				output, err := ocmResourceService.CreateAccountRole(
					"--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y")
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				ar := arl.DigAccountRoles(accountRolePrefix, false)

				By("Create a temp dir to execute the create commands")
				dirToClean, err = os.MkdirTemp("", "*")
				Expect(err).To(BeNil())

				By("Create one sts cluster in manual mode")
				rosaClient.Runner.SetDir(dirToClean)
				clusterNameToClean = "test-43053"
				// Configure with a random str, which can solve the rerun failure
				operatorRolePreifx := common.GenerateRandomName("opPrefix43053", 2)
				_, err, _ = clusterService.Create(
					clusterNameToClean, "--sts",
					"--mode", "manual",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", operatorRolePreifx,
					"-y",
				)
				Expect(err).To(BeNil())

				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				Expect(err).To(BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				Expect(err).To(BeNil())
				clusterID = clusterList.ClusterByName(clusterNameToClean).ID

				By("Create operator-roles in manual mode")
				output, err = ocmResourceService.CreateOperatorRoles(
					"-c", clusterID,
					"--mode", "manual",
					"-y",
				)
				Expect(err).To(BeNil())
				commands := common.ExtractCommandsToCreateAWSResoueces(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					Expect(err).To(BeNil())
				}

				By("Create oidc provider in manual mode")
				output, err = ocmResourceService.CreateOIDCProvider(
					"-c", clusterID,
					"--mode", "manual",
					"-y",
				)
				Expect(err).To(BeNil())
				commands = common.ExtractCommandsToCreateAWSResoueces(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					Expect(err).To(BeNil())
				}

				By("Check cluster status to installing")
				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterStatus(clusterID, "installing", 3, 24)
				Expect(err).To(BeNil(), "It met error or timeout when waiting cluster to installing status")

				By("Delete cluster and wait it deleted")
				rosaClient.Runner.UnsetArgs()
				_, err = clusterService.DeleteCluster(clusterID, "-y")
				Expect(err).To(BeNil())

				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterDeleted(clusterID, 3, 24)
				Expect(err).To(BeNil())

				By("Delete operator-roles in manual mode")
				output, err = ocmResourceService.DeleteOperatorRoles(
					"-c", clusterID,
					"--mode", "manual",
					"-y",
				)
				Expect(err).To(BeNil())
				commands = common.ExtractCommandsToDeleteAWSResoueces(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					Expect(err).To(BeNil())
				}

				By("Delete oidc provider in manual mode")
				output, err = ocmResourceService.DeleteOIDCProvider(
					"-c", clusterID,
					"--mode", "manual",
					"-y",
				)
				Expect(err).To(BeNil())
				commands = common.ExtractCommandsToDeleteAWSResoueces(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					Expect(err).To(BeNil())
				}
			})
	})
var _ = Describe("Reusing opeartor prefix and oidc config to create clsuter", labels.Feature.Cluster, func() {
	defer GinkgoRecover()
	var (
		rosaClient               *rosacli.Client
		profile                  *profilehandler.Profile
		err                      error
		oidcConfigToClean        string
		ocmResourceService       rosacli.OCMResourceService
		originalMajorMinorVerson string
		clusterService           rosacli.ClusterService
		awsClient                *aws_client.AWSClient
		operatorPolicyArn        string
		clusterID                string
	)
	const versionTagName = "rosa_openshift_version"

	BeforeEach(func() {
		By("Init the client")
		rosaClient = rosacli.NewClient()
		ocmResourceService = rosaClient.OCMResource
		clusterService = rosaClient.Cluster
		profile = profilehandler.LoadProfileYamlFileByENV()
		Expect(err).ToNot(HaveOccurred())

		awsClient, err = aws_client.CreateAWSClient("", "")
		Expect(err).To(BeNil())

		By("Get the cluster")
		clusterID = config.GetClusterID()
		Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")
	})

	AfterEach(func() {
		hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())
		if !hostedCluster {
			By("Recover the operator role policy version")
			keysToUntag := []string{versionTagName}
			err = awsClient.UntagPolicy(operatorPolicyArn, keysToUntag)
			Expect(err).To(BeNil())
			tags := map[string]string{versionTagName: originalMajorMinorVerson}
			err = awsClient.TagPolicy(operatorPolicyArn, tags)
			Expect(err).To(BeNil())
		}

		By("Delete resources for testing")
		output, err := ocmResourceService.DeleteOIDCConfig(
			"--oidc-config-id", oidcConfigToClean,
			"--mode", "auto",
			"-y",
		)
		Expect(err).To(BeNil())
		textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
		Expect(textData).To(ContainSubstring("Successfully deleted the OIDC provider"))

	})

	It("to reuse operator-roles prefix and oidc config - [id:60688]",
		labels.Critical, labels.Runtime.Day2,
		func() {
			By("Check if it is using oidc config")
			if profile.ClusterConfig.OIDCConfig == "" {
				Skip("Skip this case as it is only for byo oidc cluster")
			}

			By("Skip if the cluster is shared vpc cluster")
			if profile.ClusterConfig.SharedVPC {
				Skip("Skip this case as it is not supported for byo oidc cluster")
			}
			By("Prepare creation command")
			var originalOidcConfigID string
			var rosalCommand config.Command

			sharedDIR := os.Getenv("SHARED_DIR")
			filePath := sharedDIR + "/create_cluster.sh"
			rosalCommand, err = config.RetrieveClusterCreationCommand(filePath)
			Expect(err).To(BeNil())

			originalOidcConfigID = rosalCommand.GetFlagValue("--oidc-config-id", true)
			rosalCommand.AddFlags("--dry-run")
			testClusterName := "cluster60688"
			rosalCommand.ReplaceFlagValue(map[string]string{
				"-c": testClusterName,
			})
			if profile.ClusterConfig.DomainPrefixEnabled {
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--domain-prefix": "dp60688",
				})
			}

			By("Reuse the oidc config and operator-roles")
			stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
			Expect(err).To(BeNil())
			Expect(stdout.String()).To(ContainSubstring("Creating cluster '%s' should succeed", testClusterName))

			By("Reuse the operator prefix to create cluster but using different oidc config")
			output, err := ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
			Expect(err).To(BeNil())
			oidcPrivodeARNFromOutputMessage := common.ExtractOIDCProviderARN(output.String())
			oidcPrivodeIDFromOutputMessage := common.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)
			oidcConfigToClean, err = ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
			Expect(err).To(BeNil())

			rosalCommand.ReplaceFlagValue(map[string]string{
				"--oidc-config-id": oidcConfigToClean,
			})
			stdout, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
			Expect(err).NotTo(BeNil())
			Expect(stdout.String()).To(ContainSubstring("does not have trusted relationship to"))

			By("Find the nearest backward minor version")
			output, err = clusterService.DescribeCluster(clusterID)
			Expect(err).To(BeNil())
			clusterDetail, err := clusterService.ReflectClusterDescription(output)
			Expect(err).To(BeNil())
			operatorRolesArns := clusterDetail.OperatorIAMRoles

			versionOutput, err := clusterService.GetClusterVersion(clusterID)
			Expect(err).To(BeNil())
			clusterVersion := versionOutput.RawID
			major, minor, _, err := common.ParseVersion(clusterVersion)
			Expect(err).To(BeNil())
			originalMajorMinorVerson = fmt.Sprintf("%d.%d", major, minor)
			testingRoleVersion := fmt.Sprintf("%d.%d", major, minor-1)

			isHosted, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if !isHosted {
				By("Update the all operator policies tags to low version")
				_, roleName, err := common.ParseRoleARN(operatorRolesArns[1])
				Expect(err).To(BeNil())
				policies, err := awsClient.ListAttachedRolePolicies(roleName)
				Expect(err).To(BeNil())
				operatorPolicyArn = *policies[0].PolicyArn

				keysToUntag := []string{versionTagName}
				err = awsClient.UntagPolicy(operatorPolicyArn, keysToUntag)
				Expect(err).To(BeNil())

				tags := map[string]string{versionTagName: testingRoleVersion}

				err = awsClient.TagPolicy(operatorPolicyArn, tags)
				Expect(err).To(BeNil())

				By("Reuse operatot-role prefix and oidc config to create cluster with not-compatible version")

				rosalCommand.ReplaceFlagValue(map[string]string{
					"--oidc-config-id": originalOidcConfigID,
				})
				stdout, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).NotTo(BeNil())
				Expect(stdout.String()).To(ContainSubstring("is not compatible with cluster version"))
			}
		})
})
