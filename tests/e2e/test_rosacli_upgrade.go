package e2e

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	con "github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
	"github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("Cluster Upgrade testing",
	labels.Feature.Policy,
	func() {
		defer GinkgoRecover()

		var (
			clusterID                string
			rosaClient               *rosacli.Client
			arbitraryPolicyService   rosacli.PolicyService
			clusterService           rosacli.ClusterService
			upgradeService           rosacli.UpgradeService
			versionService           rosacli.VersionService
			arbitraryPoliciesToClean []string
			awsClient                *aws_client.AWSClient
			profile                  *profilehandler.Profile
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			arbitraryPolicyService = rosaClient.Policy
			clusterService = rosaClient.Cluster
			upgradeService = rosaClient.Upgrade
			versionService = rosaClient.Version

			By("Load the profile")
			profile = profilehandler.LoadProfileYamlFileByENV()
		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())

			By("Delete arbitrary policies")
			if len(arbitraryPoliciesToClean) > 0 {
				for _, policyArn := range arbitraryPoliciesToClean {
					err = awsClient.DeletePolicy(policyArn)
					Expect(err).To(BeNil())
				}
			}
		})

		It("to upgrade roles/operator-roles and cluster - [id:73731]", labels.Critical, labels.Runtime.Upgrade, func() {
			By("Skip if the cluster is non-sts")
			isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).To(BeNil())
			IsSTS, err := clusterService.IsSTSCluster(clusterID)
			Expect(err).To(BeNil())
			if !(isHostedCP || IsSTS) {
				Skip("Skip this case as it doesn't supports on not-sts clusters")
			}
			By("Check the cluster version and compare with the profile to decide if skip this case")
			jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
			Expect(err).To(BeNil())
			clusterVersion := jsonData.DigString("version", "raw_id")

			if profile.Version != "y-1" {
				Skip("Skip this case as the version defined in profile is not y-1 for upgrading testing")
			}

			By("Prepare arbitrary policies for testing")
			awsClient, err = aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())
			statement := map[string]interface{}{
				"Effect":   "Allow",
				"Action":   "*",
				"Resource": "*",
			}
			for i := 0; i < 2; i++ {
				arn, err := awsClient.CreatePolicy(
					fmt.Sprintf("ocmqe-arpolicy-%s-%d", common.GenerateRandomString(3), i),
					statement,
				)
				Expect(err).To(BeNil())
				arbitraryPoliciesToClean = append(arbitraryPoliciesToClean, arn)
			}

			By("Get operator-roles policies arns")
			var operatorRolePolicies []string
			output, err := clusterService.DescribeCluster(clusterID)
			Expect(err).To(BeNil())
			clusterDetail, err := clusterService.ReflectClusterDescription(output)
			Expect(err).To(BeNil())
			operatorRolesArns := clusterDetail.OperatorIAMRoles

			for _, rolearn := range operatorRolesArns {
				_, roleName, err := common.ParseRoleARN(rolearn)
				Expect(err).To(BeNil())
				policies, err := awsClient.ListAttachedRolePolicies(roleName)
				Expect(err).To(BeNil())
				operatorRolePolicies = append(operatorRolePolicies, *policies[0].PolicyArn)
			}
			_, operatorRoleName1, err := common.ParseRoleARN(operatorRolesArns[2])
			Expect(err).To(BeNil())
			operatorRolePoliciesMap1 := make(map[string][]string)
			operatorRolePoliciesMap1[operatorRoleName1] = arbitraryPoliciesToClean[0:2]

			By("Attach policies to operator-roles")
			for roleName, policyArns := range operatorRolePoliciesMap1 {
				out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
				Expect(err).To(BeNil())
				defer func() {
					By("Detach policies")
					for roleName, policyArns := range operatorRolePoliciesMap1 {
						out, err := arbitraryPolicyService.DetachPolicy(roleName, policyArns, "--mode", "auto")
						Expect(err).To(BeNil())
						for _, policyArn := range policyArns {
							Expect(out.String()).To(ContainSubstring("Detached policy '%s' from role '%s'", policyArn, roleName))
						}
					}
				}()
				for _, policyArn := range policyArns {
					Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s'", policyArn, roleName))
				}
			}

			_, operatorRoleName2, err := common.ParseRoleARN(operatorRolesArns[4])
			Expect(err).To(BeNil())
			operatorRolePoliciesMap2 := make(map[string][]string)
			operatorRolePoliciesMap2[operatorRoleName2] = append(
				operatorRolePoliciesMap2[operatorRolesArns[4]],
				arbitraryPoliciesToClean[1],
			)

			for roleName, policyArns := range operatorRolePoliciesMap2 {
				out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
				Expect(err).To(BeNil())
				defer func() {
					By("Detach policies")
					for roleName, policyArns := range operatorRolePoliciesMap2 {
						out, err := arbitraryPolicyService.DetachPolicy(roleName, policyArns, "--mode", "auto")
						Expect(err).To(BeNil())
						for _, policyArn := range policyArns {
							Expect(out.String()).To(ContainSubstring("Detached policy '%s' from role '%s'", policyArn, roleName))
						}
					}
				}()
				for _, policyArn := range policyArns {
					Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s'", policyArn, roleName))
				}

			}

			By("Get account-roles arns for testing")
			supportRoleARN := clusterDetail.SupportRoleARN
			_, supportRoleName, err := common.ParseRoleARN(supportRoleARN)
			Expect(err).To(BeNil())

			accountRolePoliciesMap := make(map[string][]string)
			accountRolePoliciesMap[supportRoleName] = arbitraryPoliciesToClean[0:2]

			By("Attach policies to account-roles")
			for roleName, policyArns := range accountRolePoliciesMap {
				out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
				Expect(err).To(BeNil())
				defer func() {
					By("Detach policies")
					for roleName, policyArns := range accountRolePoliciesMap {
						out, err := arbitraryPolicyService.DetachPolicy(roleName, policyArns, "--mode", "auto")
						Expect(err).To(BeNil())
						for _, policyArn := range policyArns {
							Expect(out.String()).To(ContainSubstring("Detached policy '%s' from role '%s'", policyArn, roleName))
						}
					}
				}()
				for _, policyArn := range policyArns {
					Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s'", policyArn, roleName))
				}

			}

			By("Find updating version")
			versionService := rosaClient.Version
			clusterVersionList, err := versionService.ListAndReflectVersions("stable", false)
			Expect(err).ToNot(HaveOccurred())

			versions, err := clusterVersionList.FindYStreamUpgradeVersions(clusterVersion)
			Expect(err).To(BeNil())
			Expect(len(versions)).
				To(
					BeNumerically(">", 0),
					fmt.Sprintf("No available upgrade version is found for the cluster version %s", clusterVersion))
			upgradingVersion := versions[0]

			By("Upgrade roles in auto mode")
			ocmResourceService := rosaClient.OCMResource
			_, err = ocmResourceService.UpgradeRoles(
				"-c", clusterID,
				"--mode", "auto",
				"--cluster-version", upgradingVersion,
				"-y",
			)
			Expect(err).To(BeNil())

			By("Check th arbitrary policies has no change")
			for _, policyArn := range arbitraryPoliciesToClean {
				policy, err := awsClient.GetIAMPolicy(policyArn)
				Expect(err).To(BeNil())
				Expect(len(policy.Tags)).To(Equal(0))
			}
			isHosted, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if !isHosted {
				By("Update the all operator policies tags to low version")
				tagName := "rosa_openshift_version"
				clusterMajorVersion := common.SplitMajorVersion(clusterVersion)
				keysToUntag := []string{tagName}

				for _, operatorRoleArn := range operatorRolePolicies {
					err = awsClient.UntagPolicy(operatorRoleArn, keysToUntag)
					Expect(err).To(BeNil())
				}
				tags := map[string]string{tagName: clusterMajorVersion}
				for _, operatorRoleArn := range operatorRolePolicies {
					err = awsClient.TagPolicy(operatorRoleArn, tags)
					Expect(err).To(BeNil())
				}
				By("Upgrade operator-roles in auto mode")
				_, err = ocmResourceService.UpgradeOperatorRoles(
					"-c", clusterID,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())

				By("Check th arbitrary policies has no change")
				for _, policyArn := range arbitraryPoliciesToClean {
					policy, err := awsClient.GetIAMPolicy(policyArn)
					Expect(err).To(BeNil())
					Expect(len(policy.Tags)).To(Equal(0))
				}
			}
			By("Update cluster")
			// TODO: Wait the upgrade ready. As upgrade profile can only be used for upgrade cluster one time,
			// so both this case and another test case for upgrading cluster
			// without arbitrary policies attach share this profile.
			// It needs to add step to wait the cluster upgrade done
			// and to check the `rosa describe/list upgrade` in both of these two case.
			if !isHosted {
				output, err = upgradeService.Upgrade(
					"-c", clusterID,
					"--version", upgradingVersion,
					"--mode", "auto",
					"-y",
				)
			} else {
				output, err = upgradeService.Upgrade(
					"-c", clusterID,
					"--version", upgradingVersion,
					"--mode", "auto", "--control-plane",
					"-y",
				)
			}
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("are compatible with upgrade"))
			Expect(output.String()).To(ContainSubstring("Upgrade successfully scheduled for cluster"))
			if isHosted {
				Expect(output.String()).To(ContainSubstring("have attached managed policies. An upgrade isn't needed"))
			} else {
				Expect(output.String()).To(ContainSubstring("are already up-to-date"))
			}
		})

		It("to upgrade NON-STS rosa cluster across Y stream - [id:37499]", labels.Critical, labels.Runtime.Upgrade, func() {

			By("Check the cluster version and compare with the profile to decide if skip this case")
			if profile.Version != con.YStreamPreviousVersion || profile.ClusterConfig.STS {
				Skip("Skip this case as the version defined in profile is not y-1 for non-sts cluster upgrading testing")
			}

			By("Check the cluster upgrade version to decide if skip this case")
			jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
			Expect(err).To(BeNil())
			clusterVersion := jsonData.DigString("version", "raw_id")

			upgradingVersion, _, err := FindUpperYStreamVersion(versionService, profile.ChannelGroup, clusterVersion)
			Expect(err).To(BeNil())
			if upgradingVersion == "" {
				Skip("Skip this case as the cluster is being upgraded.")
			}

			By("Upgrade cluster")
			scheduledDate := time.Now().Format("2006-01-02")
			scheduledTime := time.Now().Add(10 * time.Minute).UTC().Format("15:04")
			output, err := upgradeService.Upgrade(
				"-c", clusterID,
				"--version", upgradingVersion,
				"--schedule-date", scheduledDate,
				"--schedule-time", scheduledTime,
				"-y",
			)
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("Upgrade successfully scheduled for cluster"))

			By("Check upgrade state")
			err = WaitForUpgradeToState(upgradeService, clusterID, con.Scheduled, 4)
			Expect(err).To(BeNil())
			err = WaitForUpgradeToState(upgradeService, clusterID, con.Started, 70)
			Expect(err).To(BeNil())
		})

		It("to upgrade STS rosa cluster across Y stream - [id:55883]", labels.Critical, labels.Runtime.Upgrade, func() {

			By("Check the cluster version and compare with the profile to decide if skip this case")
			if profile.Version != con.YStreamPreviousVersion || !profile.ClusterConfig.STS {
				Skip("Skip this case as the version defined in profile is not y-1 for sts cluster upgrading testing")
			}

			By("Check the cluster upgrade version to decide if skip this case")
			jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
			Expect(err).To(BeNil())
			clusterVersion := jsonData.DigString("version", "raw_id")

			upgradingVersion, _, err := FindUpperYStreamVersion(versionService, profile.ChannelGroup, clusterVersion)
			Expect(err).To(BeNil())
			if upgradingVersion == "" {
				Skip("Skip this case as the cluster is being upgraded.")
			}

			By("Upgrade cluster")
			scheduledDate := time.Now().Format("2006-01-02")
			scheduledTime := time.Now().Add(10 * time.Minute).UTC().Format("15:04")
			if profile.ClusterConfig.HCP {
				By("Upgrade HCP cluster")
				output, err := upgradeService.Upgrade(
					"-c", clusterID,
					"--version", upgradingVersion,
					"--schedule-date", scheduledDate,
					"--schedule-time", scheduledTime,
					"--control-plane",
					"-m", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Upgrade successfully scheduled for cluster"))
			} else {
				By("Upgrade STS classic cluster")
				output, err := upgradeService.Upgrade(
					"-c", clusterID,
					"--version", upgradingVersion,
					"--schedule-date", scheduledDate,
					"--schedule-time", scheduledTime,
					"-m", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Upgrade successfully scheduled for cluster"))
			}

			By("Check upgrade state")
			err = WaitForUpgradeToState(upgradeService, clusterID, con.Scheduled, 4)
			Expect(err).To(BeNil())
			err = WaitForUpgradeToState(upgradeService, clusterID, con.Started, 70)
			Expect(err).To(BeNil())
		})
	})

var _ = Describe("Describe/List rosa upgrade",
	labels.Feature.Cluster, func() {
		defer GinkgoRecover()
		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			upgradeService rosacli.UpgradeService
			versionService rosacli.VersionService
			clusterID      string
			profile        *profilehandler.Profile
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			upgradeService = rosaClient.Upgrade
			versionService = rosaClient.Version

			By("Load the profile")
			profile = profilehandler.LoadProfileYamlFileByENV()
		})

		AfterEach(func() {
			if profile.Version == con.YStreamPreviousVersion {
				By("Delete cluster upgrade")
				output, err := upgradeService.DeleteUpgrade("-c", clusterID, "-y")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Successfully canceled scheduled upgrade on cluster "+
					"'%s'", clusterID))
			}

			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("to list/describe rosa upgrade via ROSA CLI - [id:57094]",
			labels.High, labels.Runtime.Day2, labels.Runtime.Upgrade,
			func() {

				By("Check the help message of 'rosa describe upgrade -h'")
				output, err := upgradeService.DescribeUpgrade(clusterID, "-h")
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("rosa describe upgrade [flags]"))
				Expect(output.String()).To(ContainSubstring("-c, --cluster"))
				Expect(output.String()).To(ContainSubstring("--machinepool"))
				Expect(output.String()).To(ContainSubstring("-y, --yes"))

				if profile.Version == "latest" {
					By("Check list upgrade for the cluster with latest version")
					output, err = upgradeService.ListUpgrades(clusterID)
					Expect(err).To(BeNil())
					Expect(output.String()).To(ContainSubstring("There are no available upgrades for cluster "+
						"'%s'", clusterID))
				}

				if profile.Version == con.YStreamPreviousVersion {
					By("Upgrade cluster and check list/describe upgrade")
					scheduledDate := time.Now().Format("2006-01-02")
					scheduledTime := time.Now().Add(20 * time.Minute).UTC().Format("15:04")

					jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
					Expect(err).To(BeNil())
					clusterVersion := jsonData.DigString("version", "raw_id")

					By("Find upper Y stream version")
					upgradingVersion, _, err := FindUpperYStreamVersion(versionService, profile.ChannelGroup, clusterVersion)
					Expect(err).To(BeNil())
					Expect(upgradingVersion).NotTo(BeEmpty())

					By("Upgrade cluster")
					if profile.ClusterConfig.STS {
						hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
						Expect(err).ToNot(HaveOccurred())
						if !hostedCluster {
							_, errSTSUpgrade := upgradeService.Upgrade(
								"-c", clusterID,
								"--version", upgradingVersion,
								"--schedule-date", scheduledDate,
								"--schedule-time", scheduledTime,
								"-m", "auto",
								"-y",
							)
							Expect(errSTSUpgrade).To(BeNil())
						} else {
							_, errHCPUpgrade := upgradeService.Upgrade(
								"-c", clusterID,
								"--version", upgradingVersion,
								"--schedule-date", scheduledDate,
								"--schedule-time", scheduledTime,
								"-m", "auto",
								"--control-plane",
								"-y",
							)
							Expect(errHCPUpgrade).To(BeNil())
						}
					} else {
						_, errUpgrade := upgradeService.Upgrade(
							"-c", clusterID,
							"--version", upgradingVersion,
							"--schedule-date", scheduledDate,
							"--schedule-time", scheduledTime,
							"-y",
						)
						Expect(errUpgrade).To(BeNil())
					}

					time.Sleep(2 * time.Minute)
					By("Check list upgrade")
					out, err := upgradeService.ListUpgrades(clusterID)
					Expect(err).To(BeNil())
					Expect(out.String()).To(MatchRegexp(`%s\s+recommended\s+-\s+scheduled for %s %s UTC`,
						upgradingVersion, scheduledDate, scheduledTime))

					By("Check describe upgrade")
					UD, err := upgradeService.DescribeUpgradeAndReflect(clusterID)
					Expect(err).To(BeNil())
					Expect(UD.ClusterID).To(Equal(clusterID))
					Expect(UD.NextRun).To(Equal(fmt.Sprintf("%s %s UTC", scheduledDate, scheduledTime)))
					Expect(UD.UpgradeState).To(Equal("scheduled"))
				}
			})
	})

var _ = Describe("ROSA HCP cluster upgrade",
	labels.Feature.Upgrade, func() {
		defer GinkgoRecover()
		var (
			clusterID      string
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			upgradeService rosacli.UpgradeService
			yStreamVersion string
			zStreamVersion string
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the service")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			upgradeService = rosaClient.Upgrade

			By("Skip testing if the cluster is not a HCP cluster")
			hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if !hostedCluster {
				SkipNotHosted()
			}

			By("Get cluster version")
			clusterVersionInfo, err := clusterService.GetClusterVersion(clusterID)
			Expect(err).ToNot(HaveOccurred())
			clusterVersion := clusterVersionInfo.RawID

			jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
			Expect(err).To(BeNil())

			var availableUpgrades []string
			if jsonData.DigObject("version", "available_upgrades") != nil {
				for _, value := range jsonData.DigObject("version", "available_upgrades").([]interface{}) {
					availableUpgrades = append(availableUpgrades, fmt.Sprint(value))
				}
			}

			By("Get cluster z stream available version")
			yStreamVersions, zStreamVersions, err := FindUpgradeVersions(availableUpgrades, clusterVersion)
			Expect(err).To(BeNil())
			if len(yStreamVersions) != 0 {
				yStreamVersion = yStreamVersions[len(yStreamVersions)-1]
			}
			if len(zStreamVersions) != 0 {
				zStreamVersion = zStreamVersions[len(zStreamVersions)-1]
			}
			log.Logger.Infof("The available y stream latest %s version:", yStreamVersions)
			log.Logger.Infof("The available z stream latest %s version:", zStreamVersions)
		})

		It("automatic upgrade for HCP cluster  - [id:64187", labels.Critical, labels.Runtime.Upgrade, func() {
			By("Check the help message for 'upgrade cluster'")
			output, err := upgradeService.Upgrade(
				"-c", clusterID,
				"-h",
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).To(ContainSubstring("--control-plane"))

			By("List the available ugrades for cluster")
			output, err = upgradeService.ListUpgrades(clusterID)
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("%s", zStreamVersion))
			Expect(output.String()).To(ContainSubstring("%s", yStreamVersion))

			By("Describe the upgrades of the cluster which has no upgrade")
			output, err = upgradeService.DescribeUpgrade(clusterID)
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("INFO: No scheduled upgrades for cluster '%s", clusterID))

			By("Try to upgrade the cluster with available version")
			scheduled := "20 20 * * *"
			output, err = upgradeService.Upgrade(
				"-c", clusterID,
				"--mode", "auto",
				"--schedule", scheduled,
				"--control-plane",
				"-y",
			)
			defer upgradeService.DeleteUpgrade("-c", clusterID, "-y")
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("Upgrade successfully scheduled for cluster"))

			By("Describe the upgrades of the cluster")
			upgDesResp, err := upgradeService.DescribeUpgradeAndReflect(clusterID)
			Expect(err).To(BeNil())
			Expect(upgDesResp.ClusterID).To(Equal(clusterID))
			Expect(upgDesResp.ScheduleAt).To(Equal(scheduled))
			Expect(upgDesResp.ScheduleType).To(Equal("automatic"))
			Expect(upgDesResp.EnableMinorVersionUpgrades).To(Equal("false"))
			if zStreamVersion != "" {
				By("Check upgrade state")
				err = WaitForUpgradeToState(upgradeService, clusterID, con.Scheduled, 4)
				Expect(err).To(BeNil())
				Expect(upgDesResp.Version).To(Equal(zStreamVersion))
				Expect(upgDesResp.UpgradeState).To(Equal("scheduled"))
				Expect(upgDesResp.StateMesage).To(ContainSubstring("Upgrade scheduled"))
			} else {
				Expect(upgDesResp.Version).To(Equal(""))
				Expect(upgDesResp.UpgradeState).To(Equal("pending"))
				Expect(upgDesResp.StateMesage).To(ContainSubstring("pending scheduling"))
			}

			By("List the upgrades of the cluster")
			output, err = upgradeService.ListUpgrades(clusterID)
			Expect(err).To(BeNil())
			if zStreamVersion != "" {
				Expect(output.String()).To(ContainSubstring("scheduled for %s", upgDesResp.NextRun))
			}

			By("Delete the upgrade policies")
			output, err = upgradeService.DeleteUpgrade("-c", clusterID, "-y")
			Expect(err).To(BeNil())
			Expect(output.String()).To(
				ContainSubstring("INFO: Successfully canceled scheduled upgrade on cluster '%s'",
					clusterID))
		})
	})

func FindUpperYStreamVersion(v rosacli.VersionService, channelGroup string, clusterVersion string) (string, string,
	error) {
	clusterVersionList, err := v.ListAndReflectVersions(channelGroup, false)
	if err != nil {
		return "", "", err
	}
	// Sorted version from high to low
	sortedVersionList, err := clusterVersionList.Sort(true)
	if err != nil {
		return "", "", err
	}
	versions, err := sortedVersionList.FindYStreamUpgradeVersions(clusterVersion)
	if err != nil {
		return "", "", err
	}
	if len(versions) == 0 {
		return "", "", nil
	} else {
		upgradingVersion := versions[len(versions)-1]
		upgradingMajorVersion := common.SplitMajorVersion(upgradingVersion)
		return upgradingVersion, upgradingMajorVersion, nil
	}
}

func WaitForUpgradeToState(u rosacli.UpgradeService, clusterID string, state string, timeout int) error {
	startTime := time.Now()
	for time.Now().Before(startTime.Add(time.Duration(timeout) * time.Minute)) {
		UD, err := u.DescribeUpgradeAndReflect(clusterID)
		if err != nil {
			return err
		} else {
			if UD.UpgradeState == state {
				return nil
			}
			time.Sleep(1 * time.Minute)
		}
	}
	return fmt.Errorf("ERROR!Timeout after %d minutes to wait for the upgrade into status %s of cluster %s",
		timeout, state, clusterID)
}

func FindUpgradeVersions(versionList []string, clusterVersion string) (
	yStreamVersions []string, zStreamVersions []string, err error) {
	clusterBaseVersionSemVer, err := semver.NewVersion(clusterVersion)
	if err != nil {
		return yStreamVersions, zStreamVersions, err
	}

	for _, version := range versionList {
		baseVersionSemVer, err := semver.NewVersion(version)
		if err != nil {
			return yStreamVersions, zStreamVersions, err
		}
		if baseVersionSemVer.Minor() == clusterBaseVersionSemVer.Minor() {
			zStreamVersions = append(zStreamVersions, version)
		}

		if baseVersionSemVer.Minor() > clusterBaseVersionSemVer.Minor() {
			yStreamVersions = append(yStreamVersions, version)
		}
	}
	return yStreamVersions, zStreamVersions, err
}
