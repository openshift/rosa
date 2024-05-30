package e2e

import (
	"fmt"
	"math/rand"
	"path"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ciConfig "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	con "github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/profilehandler"
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
			hostedCluster  bool
		)

		BeforeEach(func() {
			var err error

			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster

			By("Get cluster hosted information")
			hostedCluster, err = clusterService.IsHostedCPCluster(clusterID)
			Expect(err).To(BeNil())

			By("Load the original cluster config")
			clusterConfig, err = config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			By("Clean the cluster")
			rosaClient.CleanResources(clusterID)
		})

		It("can check the description of the cluster - [id:34102]",
			labels.Medium,
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
				Expect(CD.Network[4]["Host Prefix"]).Should(ContainSubstring(strconv.FormatFloat(jsonData.DigFloat("network", "host_prefix"), 'f', -1, 64)))
				Expect(CD.InfraID).To(Equal(jsonData.DigString("infra_id")))
			})

		It("can restrict master API endpoint to direct, private connectivity or not - [id:38850]",
			labels.High,
			func() {
				By("Check the cluster is not private cluster")
				private, err := clusterService.IsPrivateCluster(clusterID)
				Expect(err).To(BeNil())
				if private {
					Skip("This case needs to test on private cluster as the prerequirement,it was not fullfilled, skip the case!!")
				}
				isSTS, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).To(BeNil())
				By("Edit cluster to private to true")
				out, err := clusterService.EditCluster(
					clusterID,
					"--private",
					"-y",
				)
				if !isSTS || hostedCluster {
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
				if !isSTS || hostedCluster {
					Expect(CD.Private).To(Equal("Yes"))
				} else {
					Expect(CD.Private).To(Equal("No"))
				}
			})

		// OCM-5231 caused the description parser issue
		It("can disable workload monitoring on/off - [id:45159]",
			labels.High,
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
			labels.Medium,
			func() {
				By("Validate that deletion of upgrade policy for rosa cluster will work via rosacli")
				output, err := clusterService.DeleteUpgrade()
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

		It("validation for create/delete upgrade policies for hypershift clusters via rosacli should work well - [id:73814]",
			labels.Medium,
			func() {
				if !hostedCluster {
					Skip("This case applies only on hosted cluster")
				}

				defer func() {
					_, err := clusterService.DeleteUpgrade("-c", clusterID, "-y")
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Upgrade cluster without --control-plane flag")
				output, err := clusterService.Upgrade("-c", clusterID)
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: The '--control-plane' option is currently mandatory for Hosted Control Planes"))

				By("Upgrade cluster with invalid cluster id")
				invalidClusterID := common.GenerateRandomString(30)
				output, err = clusterService.Upgrade("-c", invalidClusterID)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: Failed to get cluster '%s': There is no cluster with identifier or name '%s'", invalidClusterID, invalidClusterID))

				By("Upgrade cluster with incorrect format of the date and time")
				output, err = clusterService.Upgrade("-c", clusterID, "--control-plane", "--mode=auto", "--schedule-date=\"2024-06\"", "--schedule-time=\"09:00:12\"", "-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: schedule date should use the format 'yyyy-mm-dd'"))

				By("Upgrade cluster using --schedule, --schedule-date and --schedule-time flags at the same time")
				output, err = clusterService.Upgrade("-c", clusterID, "--control-plane", "--mode=auto", "--schedule-date=\"2024-06-24\"", "--schedule-time=\"09:00\"", "--schedule=\"5 5 * * *\"", "-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: The '--schedule-date' and '--schedule-time' options are mutually exclusive with '--schedule'"))

				By("Upgrade cluster using --schedule and --version flags at the same time")
				output, err = clusterService.Upgrade("-c", clusterID, "--control-plane", "--mode=auto", "--schedule=\"5 5 * * *\"", "--version=4.15.10", "-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: The '--schedule' option is mutually exclusive with '--version'"))

				By("Upgrade cluster with value not match the cron epression")
				output, err = clusterService.Upgrade("-c", clusterID, "--control-plane", "--mode=auto", "--schedule=\"5 5\"", "-y")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: Schedule '\"5 5\"' is not a valid cron expression"))
			})

		It("can allow sts cluster installation with compatible policies - [id:45161]",
			labels.High,
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
				rosalCommand, err := config.RetrieveClusterCreationCommand(ciConfig.Test.CreateCommandFile)
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
					"-c":                      clusterName,
					"--operator-roles-prefix": operatorPrefix,
					"--domain-prefix":         clusterName,
				}

				if rosalCommand.GetFlagValue("--https-proxy", true) != "" {
					err = rosalCommand.DeleteFlag("--https-proxy", true)
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
	})

var _ = Describe("Classic cluster creation validation",
	labels.Day1Validation,
	labels.FeatureCluster,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient    *rosacli.Client
			profilesMap   map[string]*profilehandler.Profile
			profile       *profilehandler.Profile
			hostedCluster bool
		)

		BeforeEach(func() {
			// Init the client
			rosaClient = rosacli.NewClient()
			// Get a random profile
			profilesMap = profilehandler.ParseProfilesByFile(path.Join(ciConfig.Test.YAMLProfilesDir, "rosa-classic.yaml"))
			rand.New(rand.NewSource(time.Now().UnixNano()))
			profilesNames := make([]string, 0, len(profilesMap))
			for k := range profilesMap {
				profilesNames = append(profilesNames, k)
			}
			profile = profilesMap[profilesNames[rand.Intn(len(profilesNames))]]
			profile.NamePrefix = con.DefaultNamePrefix

			By("Get the cluster")
			clusterID := config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Get cluster hosted information")
			var err error
			hostedCluster, err = rosaClient.Cluster.IsHostedCPCluster(clusterID)
			Expect(err).To(BeNil())

		})

		AfterEach(func() {
			errs := profilehandler.DestroyResourceByProfile(profile, rosaClient)
			Expect(len(errs)).To(Equal(0))
		})

		It("to check the basic validation for the classic rosa cluster creation by the rosa cli - [id:38770]",
			labels.Medium,
			labels.Day1Validation,
			func() {
				if hostedCluster {
					Skip("This case applies only on classic cluster")
				}

				var command string
				var rosalCommand config.Command
				By("Prepare creation command")
				flags, err := profilehandler.GenerateClusterCreateFlags(profile, rosaClient)
				Expect(err).To(BeNil())

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
					Expect(stdout.String()).To(ContainSubstring("Cluster name must consist of no more than 54 lowercase alphanumeric characters or '-', start with a letter, and end with an alphanumeric character"))
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
				Expect(stdout.String()).To(ContainSubstring("Billing accounts are only supported for Hosted Control Plane clusters"))
			})

		It("to validate to create the sts cluster with invalid tag - [id:56440]", labels.Medium, labels.Day1Validation, func() {
			clusterService := rosaClient.Cluster
			clusterName := "ocp-56440"

			By("Create cluster with invalid tag key")
			out, err := clusterService.CreateDryRun(
				clusterName, "--tags=~~~:cluster",
			)
			Expect(err).NotTo(BeNil())
			Expect(out.String()).To(ContainSubstring("expected a valid user tag key '~~~' matching ^[\\pL\\pZ\\pN_.:/=+\\-@]{1,128}$"))

			By("Create cluster with invalid tag value")
			out, err = clusterService.CreateDryRun(
				clusterName, "--tags=name:****",
			)
			Expect(err).NotTo(BeNil())
			Expect(out.String()).To(ContainSubstring("expected a valid user tag value '****' matching ^[\\pL\\pZ\\pN_.:/=+\\-@]{0,256}$"))

			By("Create cluster with duplicate tag key")
			out, err = clusterService.CreateDryRun(
				clusterName, "--tags=name:test1,op:clound,name:test2",
			)
			Expect(err).NotTo(BeNil())
			Expect(out.String()).To(ContainSubstring("invalid tags, user tag keys must be unique, duplicate key 'name' found"))

			By("Create cluster with invalid tag format")
			out, err = clusterService.CreateDryRun(
				clusterName, "--tags=test1,test2,test4",
			)
			Expect(err).NotTo(BeNil())
			Expect(out.String()).To(ContainSubstring("invalid tag format for tag '[test1]'. Expected tag format: 'key:value'"))

			By("Create cluster with empty tag value")
			out, err = clusterService.CreateDryRun(
				clusterName, "--tags", "foo:",
			)
			Expect(err).NotTo(BeNil())
			Expect(out.String()).To(ContainSubstring("invalid tag format, tag key or tag value can not be empty"))

			By("Create cluster with invalid tag format")
			out, err = clusterService.CreateDryRun(
				clusterName, "--tags=name:gender:age",
			)
			Expect(err).NotTo(BeNil())
			Expect(out.String()).To(ContainSubstring("invalid tag format for tag '[name gender age]'. Expected tag format: 'key:value'"))

		})
	})

var _ = Describe("Classic cluster deletion validation",
	labels.Day3,
	labels.FeatureCluster,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient *rosacli.Client
		)

		BeforeEach(func() {
			// Init the client
			rosaClient = rosacli.NewClient()
		})

		It("To validate the ROSA cluster deletion will work via rosacli	- [id:38778]", labels.Medium, labels.Day3, func() {
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
var _ = Describe("Classic cluster creation negavite testing",
	labels.Day1Negative,
	labels.FeatureCluster,
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

		It("to validate to create the sts cluster with the version not compatible with the role version	- [id:45176]", labels.Medium, func() {
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
	})

var _ = Describe("HCP cluster creation negative testing",
	labels.Day1Negative,
	labels.FeatureCluster,
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

		It("create HCP cluster with network type validation can work well via rosa cli - [id:73725]", labels.Medium, func() {
			isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).To(BeNil())

			rosalCommand, err := config.RetrieveClusterCreationCommand(ciConfig.Test.CreateCommandFile)
			Expect(err).To(BeNil())

			if !isHostedCP {
				By("Create non-HCP cluster with --no-cni flag")
				clusterName := common.GenerateRandomName("classic-73725", 2)
				operatorPrefix := common.GenerateRandomName("classic-oper", 2)
				replacingFlags := map[string]string{
					"-c":                     clusterName,
					"--cluster-name":         clusterName,
					"--domain-prefix":        clusterName,
					"--operator-role-prefix": operatorPrefix,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--no-cni", "-y")
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("ERR: Disabling CNI is supported only for Hosted Control Planes"))
			} else {
				By("Create HCP cluster with --no-cni and \"--network-type={OVNKubernetes, OpenshiftSDN}\" at the same time")
				clusterName := common.GenerateRandomName("cluster-71946", 2)
				operatorPrefix := common.GenerateRandomName("cluster-oper", 2)

				replacingFlags := map[string]string{
					"-c":                     clusterName,
					"--cluster-name":         clusterName,
					"--domain-prefix":        clusterName,
					"--operator-role-prefix": operatorPrefix,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--no-cni", "--network-type='{OVNKubernetes,OpenshiftSDN}'", "-y")
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("ERR: Expected a valid network type. Valid values: [OpenShiftSDN OVNKubernetes]"))

				By("Create hcp cluster with invalid --no-cni value")
				rosalCommand.DeleteFlag("--network-type", true)
				rosalCommand.DeleteFlag("--no-cni", true)
				rosalCommand.AddFlags("--no-cni=ui")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring(`Failed to execute root command: invalid argument "ui" for "--no-cni" flag: strconv.ParseBool: parsing "ui": invalid syntax`))

				By("Create hcp cluster with --no-cni and --network-type=OVNKubernetes at the same time")
				rosalCommand.DeleteFlag("--no-cni=ui", false)
				rosalCommand.AddFlags("--no-cni", "--network-type=OVNKubernetes")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("ERR: --no-cni and --network-type are mutually exclusive parameters"))
			}
		})
	})
