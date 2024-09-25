package e2e

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	ciConfig "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/common/constants"
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
			ocmResourceService       rosacli.OCMResourceService
			versionService           rosacli.VersionService
			arbitraryPoliciesToClean []string
			awsClient                *aws_client.AWSClient
			profile                  *profilehandler.Profile
			roleUrlPrefix            = "https://console.aws.amazon.com/iam/home?#/roles/"
			accountRoles             []string
			operatorRoles            []string
		)
		const versionTagName = "rosa_openshift_version"

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			arbitraryPolicyService = rosaClient.Policy
			clusterService = rosaClient.Cluster
			upgradeService = rosaClient.Upgrade
			ocmResourceService = rosaClient.OCMResource
			versionService = rosaClient.Version

			By("Load the profile")
			profile = profilehandler.LoadProfileYamlFileByENV()
		})

		AfterEach(func() {
			if profile.Version == con.YStreamPreviousVersion {
				By("Delete cluster upgrade")
				output, err := upgradeService.DeleteUpgrade("-c", clusterID, "-y")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring(
					"Successfully canceled scheduled upgrade on cluster '%s'", clusterID))
			}

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

			if profile.Version != con.YStreamPreviousVersion {
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
					Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
						policyArn, roleName, roleUrlPrefix+roleName))
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
					Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
						policyArn, roleName, roleUrlPrefix+roleName))
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
					Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
						policyArn, roleName, roleUrlPrefix+roleName))
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

		It("to upgrade wide AMI roles with the managed policies in auto mode - [id:57444]",
			labels.Critical, labels.Runtime.Upgrade, func() {
				By("Check the cluster version and compare with the profile to decide if skip this case")
				if !profile.ClusterConfig.STS || profile.Version != con.YStreamPreviousVersion {
					Skip("Skip this case as the version defined in profile is not y-1 or non-sts cluster for " +
						"upgrading testing")
				}

				By("Upgrade wide AMI roles in auto mode")
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())
				clusterVersion := jsonData.DigString("version", "raw_id")

				if profile.ClusterConfig.HCP {
					By("Find HCP cluster upgrade version")
					hcpUpgradingVersion, _, err := FindUpperYStreamVersion(versionService,
						profile.ChannelGroup, clusterVersion)
					Expect(err).To(BeNil())
					if hcpUpgradingVersion == "" {
						Skip("Skip this case as no version available for upgrade")
					}

					By("upgrade HCP cluster wide AMI roles in auto mode")
					ud, err := profilehandler.ParseUserData()
					Expect(err).To(BeNil())
					Expect(ud).NotTo(BeNil())
					output1, err := ocmResourceService.UpgradeRoles(
						"-c", clusterID,
						"--cluster-version", hcpUpgradingVersion,
						"--mode", "auto",
						"-y",
					)
					Expect(err).To(BeNil())
					Expect(output1.String()).To(ContainSubstring("Account roles with the prefix '%s' have attached "+
						"managed policies.", ud.AccountRolesPrefix))
					Expect(output1.String()).To(ContainSubstring("Cluster '%s' operator roles have attached managed "+
						"policies. An upgrade isn't needed", ud.OperatorRolesPrefix))
				} else {
					By("Find STS Classic cluster upgrade version")
					classicUpgradingVersion, classicUpgradingMajorVersion, err := FindUpperYStreamVersion(versionService,
						profile.ChannelGroup, clusterVersion)
					Expect(err).To(BeNil())
					if classicUpgradingVersion == "" || classicUpgradingMajorVersion == "" {
						Skip("Skip this case as no version available for upgrade")
					}

					By("get account roles and operator roles from cluster description")
					description, err := clusterService.DescribeClusterAndReflect(clusterID)
					Expect(err).ToNot(HaveOccurred())

					_, installerRoleName, err := common.ParseRoleARN(description.STSRoleArn)
					Expect(err).To(BeNil())
					_, supportRoleName, err := common.ParseRoleARN(description.SupportRoleARN)
					Expect(err).To(BeNil())

					accountRoles = append(accountRoles, installerRoleName)
					accountRoles = append(accountRoles, supportRoleName)

					for _, i := range description.InstanceIAMRoles {
						for _, v := range i {
							_, accountRoleName, err := common.ParseRoleARN(v)
							Expect(err).To(BeNil())
							accountRoles = append(accountRoles, accountRoleName)
						}
					}

					for _, v := range description.OperatorIAMRoles {
						_, operatorRoleName, err := common.ParseRoleARN(v)
						Expect(err).To(BeNil())
						operatorRoles = append(operatorRoles, operatorRoleName)
					}

					awsClient, err = aws_client.CreateAWSClient("", "")
					Expect(err).To(BeNil())

					By("upgrade STS Classic cluster wide AMI roles in auto mode")
					output, err := ocmResourceService.UpgradeRoles(
						"-c", clusterID,
						"--cluster-version", classicUpgradingVersion,
						"--mode", "auto",
						"-y",
					)
					Expect(err).To(BeNil())
					Expect(output.String()).To(ContainSubstring("Ensuring account role/policies compatibility for " +
						"upgrade"))
					Expect(output.String()).To(ContainSubstring("Starting to upgrade the policies"))

					for _, accountRoleName := range accountRoles {
						accountRolePolicyArns, err := awsClient.ListRoleAttachedPolicies(accountRoleName)
						Expect(err).To(BeNil())
						Expect(output.String()).To(ContainSubstring("Upgraded policy with ARN '%s' to version '%s'",
							*accountRolePolicyArns[0].PolicyArn, classicUpgradingMajorVersion))
					}
					Expect(output.String()).To(ContainSubstring("Ensuring operator role/policies compatibility for" +
						" upgrade"))

					for _, operatorRoleName := range operatorRoles {
						operatorRolePolicyArns, err := awsClient.ListRoleAttachedPolicies(operatorRoleName)
						Expect(err).To(BeNil())
						Expect(output.String()).To(ContainSubstring("Upgraded policy with ARN '%s' to version '%s'",
							*operatorRolePolicyArns[0].PolicyArn, classicUpgradingMajorVersion))
					}
				}
			})

		It("to upgrade wide AMI roles with the managed policies in manual mode - [id:75445]",
			labels.Critical, labels.Runtime.Upgrade, func() {
				By("Check the cluster version and compare with the profile to decide if skip this case")
				if !profile.ClusterConfig.STS || profile.Version != con.YStreamPreviousVersion {
					Skip("Skip this case as the version defined in profile is not y-1 or non-sts cluster for " +
						"upgrading testing")
				}

				By("Upgrade wide AMI roles in manual mode")
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())
				clusterVersion := jsonData.DigString("version", "raw_id")

				if profile.ClusterConfig.HCP {
					By("Find HCP cluster upgrade version")
					hcpUpgradingVersion, _, err := FindUpperYStreamVersion(versionService, profile.ChannelGroup,
						clusterVersion)
					Expect(err).To(BeNil())
					if hcpUpgradingVersion == "" {
						Skip("Skip this case as no version available for upgrade")
					}

					By("upgrade HCP cluster wide AMI roles in manual mode")
					ud, err := profilehandler.ParseUserData()
					Expect(err).To(BeNil())
					Expect(ud).NotTo(BeNil())

					output1, err := ocmResourceService.UpgradeRoles(
						"-c", clusterID,
						"--cluster-version", hcpUpgradingVersion,
						"--mode", "manual",
						"-y",
					)
					Expect(err).To(BeNil())
					Expect(output1.String()).To(ContainSubstring("Account roles with the prefix '%s' have attached "+
						"managed policies.", ud.AccountRolesPrefix))
					Expect(output1.String()).To(ContainSubstring("Cluster '%s' operator roles have attached managed "+
						"policies. An upgrade isn't needed", ud.OperatorRolesPrefix))
				} else {
					By("Find STS Classic cluster upgrade version")
					classicUpgradingVersion, upgradingMajorVersion, err := FindUpperYStreamVersion(versionService,
						profile.ChannelGroup, clusterVersion)
					Expect(err).To(BeNil())
					if classicUpgradingVersion == "" {
						Skip("Skip this case as no version available for upgrade")
					}

					By("upgrade STS Classic cluster wide AMI roles in manual mode")
					output2, err := ocmResourceService.UpgradeRoles(
						"-c", clusterID,
						"--cluster-version", classicUpgradingVersion,
						"--mode", "manual",
						"-y",
					)
					Expect(err).To(BeNil())
					Expect(output2.String()).To(ContainSubstring("Ensuring account role/policies compatibility " +
						"for upgrade"))

					commands := common.ExtractCommandsToCreateAWSResoueces(output2)
					for _, command := range commands {
						info := "INFO: Ensuring operator role/policies compatibility for upgrade"
						if strings.Contains(command, info) {
							index := strings.Index(command, info)
							cmd1 := strings.Fields(command[:index])
							cmd2 := strings.Fields(command[index+len(info):])

							_, err := rosaClient.Runner.RunCMD(cmd1)
							Expect(err).To(BeNil())

							_, err = rosaClient.Runner.RunCMD(cmd2)
							Expect(err).To(BeNil())
						} else {
							cmd := strings.Split(command, " ")
							if len(cmd) > 0 && cmd[len(cmd)-1] == "" {
								cmd = cmd[:len(cmd)-1]
							}
							_, err := rosaClient.Runner.RunCMD(cmd)
							Expect(err).To(BeNil())
						}
					}

					By("Check final result from aws")
					output, err := clusterService.DescribeCluster(clusterID)
					Expect(err).To(BeNil())
					CD, err := clusterService.ReflectClusterDescription(output)
					Expect(err).To(BeNil())

					var accRoles []string
					accRoles = append(accRoles, CD.STSRoleArn)
					accRoles = append(accRoles, CD.SupportRoleARN)
					accRoles = append(accRoles, CD.InstanceIAMRoles[0]["Control plane"])
					accRoles = append(accRoles, CD.InstanceIAMRoles[1]["Worker"])

					operatorRoles := CD.OperatorIAMRoles

					awsClient, err = aws_client.CreateAWSClient("", "")
					Expect(err).To(BeNil())

					By("Check account role version")
					for _, accArn := range accRoles {
						parse, err := arn.Parse(accArn)
						Expect(err).To(BeNil())
						accRoleName := strings.Split(parse.Resource, "/")[1]
						accRole, err := awsClient.GetRole(accRoleName)
						Expect(err).To(BeNil())
						for _, tag := range accRole.Tags {
							if *tag.Key == versionTagName {
								Expect(*tag.Value).To(Equal(upgradingMajorVersion))
							}
						}
					}

					By("Check operator role version")
					for _, opArn := range operatorRoles {
						parse, err := arn.Parse(opArn)
						Expect(err).To(BeNil())
						opRoleName := strings.Split(parse.Resource, "/")[1]
						opPolicy, err := awsClient.ListAttachedRolePolicies(opRoleName)
						Expect(err).To(BeNil())
						policyArn := *opPolicy[0].PolicyArn
						policy, err := awsClient.GetIAMPolicy(policyArn)
						Expect(err).To(BeNil())
						for _, tag := range policy.Tags {
							if *tag.Key == versionTagName {
								Expect(*tag.Value).To(Equal(upgradingMajorVersion))
							}
						}
					}
				}
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
							output, errSTSUpgrade := upgradeService.Upgrade(
								"-c", clusterID,
								"--version", upgradingVersion,
								"--schedule-date", scheduledDate,
								"--schedule-time", scheduledTime,
								"-m", "auto",
								"-y",
							)
							Expect(errSTSUpgrade).To(BeNil())
							Expect(output.String()).NotTo(ContainSubstring("There is already a scheduled upgrade"))
						} else {
							output, errHCPUpgrade := upgradeService.Upgrade(
								"-c", clusterID,
								"--version", upgradingVersion,
								"--schedule-date", scheduledDate,
								"--schedule-time", scheduledTime,
								"-m", "auto",
								"--control-plane",
								"-y",
							)
							Expect(errHCPUpgrade).To(BeNil())
							Expect(output.String()).NotTo(ContainSubstring("There is already a scheduled upgrade"))
						}
					} else {
						output, errUpgrade := upgradeService.Upgrade(
							"-c", clusterID,
							"--version", upgradingVersion,
							"--schedule-date", scheduledDate,
							"--schedule-time", scheduledTime,
							"-y",
						)
						Expect(errUpgrade).To(BeNil())
						Expect(output.String()).NotTo(ContainSubstring("There is already a scheduled upgrade"))
					}

					time.Sleep(2 * time.Minute)
					By("Check list upgrade")
					out, err := upgradeService.ListUpgrades(clusterID)
					Expect(err).To(BeNil())
					Expect(out.String()).To(MatchRegexp(`%s\s+recommended\s+-\s+(scheduled|started) for %s %s UTC`,
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

		It("automatic upgrade for HCP cluster  - [id:64187]", labels.Critical, labels.Runtime.Upgrade, func() {
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
				upgDesResp, err := upgradeService.DescribeUpgradeAndReflect(clusterID)
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

		It("to validate role's policy when upgrade hcp cluster - [id:62161]",
			labels.Medium, labels.Runtime.Day2, labels.Runtime.Upgrade,
			func() {
				By("update operator-roles for hcp cluster")
				ocmResourceService := rosaClient.OCMResource
				output, err := ocmResourceService.UpgradeOperatorRoles(
					"--cluster", clusterID,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("operator roles have attached managed policies. " +
					"An upgrade isn't needed"))

				var arbitraryPolicyService rosacli.PolicyService
				output, err = clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				awsClient, err := aws_client.CreateAWSClient("", "")
				Expect(err).To(BeNil())
				var rolePolicyMap = make(map[string]string)
				roles := []string{CD.STSRoleArn, CD.SupportRoleARN, CD.InstanceIAMRoles[0]["Worker"]}
				for _, policyArn := range roles {
					_, accountRoleName, err := common.ParseRoleARN(policyArn)
					Expect(err).To(BeNil())
					attachedPolicy, err := awsClient.ListAttachedRolePolicies(accountRoleName)
					Expect(err).To(BeNil())
					rolePolicyMap[accountRoleName] = *attachedPolicy[0].PolicyArn
				}

				for _, policyArn := range CD.OperatorIAMRoles {
					_, operatorRoleName, err := common.ParseRoleARN(policyArn)
					Expect(err).To(BeNil())
					attachedPolicy, err := awsClient.ListAttachedRolePolicies(operatorRoleName)
					Expect(err).To(BeNil())
					rolePolicyMap[operatorRoleName] = *attachedPolicy[0].PolicyArn
				}

				By("detach managed policies from account role and operator role and update cluster")
				arbitraryPolicyService = rosaClient.Policy
				upgradeVersion := zStreamVersion
				if zStreamVersion == "" {
					upgradeVersion = yStreamVersion
				}
				for r, p := range rolePolicyMap {
					_, err := arbitraryPolicyService.DetachPolicy(r, []string{p}, "--mode", "auto")
					Expect(err).To(BeNil())
					defer arbitraryPolicyService.AttachPolicy(r, []string{p}, "--mode", "auto")

					By("upgrade cluster with account roles which is detached managed policies")
					scheduledDate := time.Now().Format("2006-01-02")
					scheduledTime := time.Now().Add(10 * time.Minute).UTC().Format("15:04")
					output, err = upgradeService.Upgrade(
						"-c", clusterID,
						"--version", upgradeVersion,
						"--schedule-date", scheduledDate,
						"--schedule-time", scheduledTime,
						"--control-plane",
						"--mode", "manual",
						"-y",
					)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).
						To(
							ContainSubstring(
								fmt.Sprintf("Failed while validating managed policies: role"+
									" '%s' is missing the attached managed policy '%s'", r, p)))

					By("Attach the deleted managed policies")
					_, err = arbitraryPolicyService.AttachPolicy(r, []string{p}, "--mode", "auto")
					Expect(err).To(BeNil())
				}
			})
	})

var _ = Describe("Sucessfull Upgrade Testing",
	labels.Feature.Policy,
	func() {
		defer GinkgoRecover()

		var (
			clusterID      string
			clusterName    string
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			upgradeService rosacli.UpgradeService
			versionService rosacli.VersionService
			// ocmResourceService rosacli.OCMResourceService
			// rolePrefix         string
			profile *profilehandler.Profile
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			upgradeService = rosaClient.Upgrade
			versionService = rosaClient.Version
			// ocmResourceService = rosaClient.OCMResource
		})

		AfterEach(func() {
			if profile.ClusterConfig.HCP {
				// At least try to delete testing cluster
				By("Delete cluster by name")
				rosaClient.Runner.UnsetArgs()
				_, err := clusterService.DeleteCluster(clusterName, "-y")
				Expect(err).To(BeNil())
			}
			errs := profilehandler.DestroyResourceByProfile(profile, rosaClient)
			Expect(len(errs)).To(Equal(0))
		})

		Context("for Y-stream cluster - [id:55883]", labels.Critical, labels.Runtime.Day1Supplemental, func() {
			It("on STS rosa classic cluster", func() {
				By("Create a Y-stream sts rosa classic cluster")
				profilesMap := profilehandler.ParseProfilesByFile(path.Join(ciConfig.Test.YAMLProfilesDir, "rosa-classic.yaml"))
				profile = profilesMap["rosa-upgrade-y-stream"]
				profile.NamePrefix = constants.DefaultNamePrefix

				flags, err := profilehandler.GenerateClusterCreateFlags(profile, rosaClient)
				Expect(err).ToNot(HaveOccurred())

				clusterName = common.GenerateRandomName("cluster-55883", 2)
				command := "rosa create cluster --cluster-name " + clusterName + " " + strings.Join(flags, " ") + " --mode auto"
				rosalCommand := config.GenerateCommand(command)
				log.Logger.Info(rosalCommand)
				// rolePrefix = rosalCommand.GetFlagValue("--operator-roles-prefix", true)
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring(fmt.Sprintf("Cluster '%s' has been created", clusterName)))

				By("Retrieve cluster ID")
				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				Expect(err).To(BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				Expect(err).To(BeNil())
				clusterID = clusterList.ClusterByName(clusterName).ID

				By("Wait for Cluster to be ready")
				err = clusterService.WaitClusterStatus(clusterID, constants.Ready, 3, 120)
				Expect(err).To(BeNil(), "It met error or timeout when waiting cluster to ready status")

				By("Check the cluster upgrade version to decide if skip this case")
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())
				clusterVersion := jsonData.DigString("version", "raw_id")

				upgradingVersion, _, err := FindUpperYStreamVersion(versionService, profile.ChannelGroup, clusterVersion)
				Expect(err).To(BeNil())
				if upgradingVersion == "" {
					Skip("Skip this case as the cluster is being upgraded.")
				}

				scheduledDate := time.Now().Format("2006-01-02")
				scheduledTime := time.Now().Add(10 * time.Minute).UTC().Format("15:04")
				By("Upgrade STS classic cluster")
				output, err = upgradeService.Upgrade(
					"-c", clusterID,
					"--version", upgradingVersion,
					"--schedule-date", scheduledDate,
					"--schedule-time", scheduledTime,
					"-m", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Upgrade successfully scheduled for cluster"))

				By("Check upgrade state")
				err = WaitForUpgradeToState(upgradeService, clusterID, con.Scheduled, 4)
				Expect(err).To(BeNil())
				err = WaitForUpgradeToState(upgradeService, clusterID, con.Started, 70)
				Expect(err).To(BeNil())

				By("Check the scheduled upgrade is completed successfully")
				Eventually(func() (string, error) {
					output, err := upgradeService.DescribeUpgrade(clusterID)
					return rosaClient.Parser.TextData.Input(output).Parse().Tip(), err
				}, time.Minute*70, time.Minute*1).Should(
					ContainSubstring("INFO: No scheduled upgrades for cluster '%s'", clusterID))
			})

			It("on rosa hcp cluster", func() {
				By("Create a Y-stream rosa hcp cluster")
				profilesMap := profilehandler.ParseProfilesByFile(path.Join(ciConfig.Test.YAMLProfilesDir, "rosa-hcp.yaml"))
				profile = profilesMap["rosa-hcp-upgrade-y-stream"]
				profile.NamePrefix = constants.DefaultNamePrefix

				flags, err := profilehandler.GenerateClusterCreateFlags(profile, rosaClient)
				Expect(err).ToNot(HaveOccurred())

				clusterName = common.GenerateRandomName("cluster-55883", 2)
				command := "rosa create cluster --cluster-name " + clusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				// rolePrefix = rosalCommand.GetFlagValue("--operator-roles-prefix", true)
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring(fmt.Sprintf("Cluster '%s' has been created", clusterName)))

				By("Retrieve cluster ID")
				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				Expect(err).To(BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				Expect(err).To(BeNil())
				clusterID = clusterList.ClusterByName(clusterName).ID

				By("Wait for Cluster to be ready")
				err = clusterService.WaitClusterStatus(clusterID, constants.Ready, 3, 60)
				Expect(err).To(BeNil(), "It met error or timeout when waiting cluster to ready status")

				By("Check the cluster upgrade version to decide if skip this case")
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				Expect(err).To(BeNil())
				clusterVersion := jsonData.DigString("version", "raw_id")

				upgradingVersion, _, err := FindUpperYStreamVersion(versionService, profile.ChannelGroup, clusterVersion)
				Expect(err).To(BeNil())
				if upgradingVersion == "" {
					Skip("Skip this case as the cluster is being upgraded.")
				}

				scheduledDate := time.Now().Format("2006-01-02")
				scheduledTime := time.Now().Add(10 * time.Minute).UTC().Format("15:04")
				By("Upgrade HCP cluster")
				output, err = upgradeService.Upgrade(
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
				By("Check upgrade state")
				err = WaitForUpgradeToState(upgradeService, clusterID, con.Scheduled, 4)
				Expect(err).To(BeNil())
				err = WaitForUpgradeToState(upgradeService, clusterID, con.Started, 70)
				Expect(err).To(BeNil())

				By("Check the scheduled upgrade is completed successfully")
				Eventually(func() (string, error) {
					output, err := upgradeService.DescribeUpgrade(clusterID)
					return rosaClient.Parser.TextData.Input(output).Parse().Tip(), err
				}, time.Minute*30, time.Minute*1).Should(
					ContainSubstring("INFO: No scheduled upgrades for cluster '%s'", clusterID))
			})
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
