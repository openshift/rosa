package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit account roles", labels.Feature.AccountRoles, func() {
	defer GinkgoRecover()

	var (
		accountRolePrefixesNeedCleanup = make([]string, 0)
		rosaClient                     *rosacli.Client
		ocmResourceService             rosacli.OCMResourceService
		permissionsBoundaryArn         string = "arn:aws:iam::aws:policy/AdministratorAccess"
		defaultDir                     string
		dirToClean                     string
	)
	BeforeEach(func() {
		By("Init the client")
		rosaClient = rosacli.NewClient()
		ocmResourceService = rosaClient.OCMResource

		By("Get the default dir")
		defaultDir = rosaClient.Runner.GetDir()
	})
	AfterEach(func() {
		By("Go back original by setting runner dir")
		rosaClient.Runner.SetDir(defaultDir)
	})

	It("can create/list/delete account-roles - [id:43070]",
		labels.High, labels.Runtime.OCMResources,
		func() {
			defer func() {
				By("Cleanup created account-roles in high level of the test case")
				if len(accountRolePrefixesNeedCleanup) > 0 {
					for _, v := range accountRolePrefixesNeedCleanup {
						_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
							"--prefix", v,
							"-y")

						Expect(err).To(BeNil())
					}
				}
			}()

			var (
				userRolePrefixB = "prefixB"
				userRolePrefixH = "prefixH"
				userRolePrefixC = "prefixC"
				path            = "/fd/sd/"
				versionH        string
				versionC        string
			)

			By("Get the testing version")
			versionService := rosaClient.Version
			versionListC, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, false)
			Expect(err).To(BeNil())
			defaultVersionC := versionListC.DefaultVersion()
			Expect(defaultVersionC).ToNot(BeNil())
			_, _, versionC, err = defaultVersionC.MajorMinor()
			Expect(err).To(BeNil())

			versionListH, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, true)
			Expect(err).To(BeNil())
			defaultVersionH := versionListH.DefaultVersion()
			Expect(defaultVersionH).ToNot(BeNil())
			_, _, versionH, err = defaultVersionH.MajorMinor()
			Expect(err).To(BeNil())

			By("Create boundary policy")
			rosaClient.Runner.JsonFormat()

			whoamiOutput, err := ocmResourceService.Whoami()
			Expect(err).To(BeNil())
			rosaClient.Runner.UnsetFormat()
			whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
			AWSAccountID := whoamiData.AWSAccountID

			By("Create advanced account-roles of both hosted-cp and classic")
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", userRolePrefixB,
				"--path", path,
				"--permissions-boundary", permissionsBoundaryArn,
				"-y")
			Expect(err).To(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, userRolePrefixB)
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Creating classic account roles"))
			Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
			Expect(textData).To(ContainSubstring("Created role"))
			Expect(textData).To(ContainSubstring("with ARN 'arn:aws:iam::"))
			Expect(textData).To(ContainSubstring("Attached policy 'arn:aws:iam::"))
			Expect(textData).To(ContainSubstring("Attached trust policy to role"))

			By("Create advance account-roles of only hosted-cp")
			output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", userRolePrefixH,
				"--path", path,
				"--permissions-boundary", permissionsBoundaryArn,
				"--version", versionH,
				"--hosted-cp",
				"-y")
			Expect(err).To(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, userRolePrefixH)
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).ToNot(ContainSubstring("Creating classic account roles"))
			Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
			Expect(textData).To(ContainSubstring("Created role"))
			Expect(textData).To(ContainSubstring("Attached trust policy to role"))

			By("Create advance account-roles of only classic")
			output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", userRolePrefixC,
				"--path", path,
				"--permissions-boundary", permissionsBoundaryArn,
				"--version", versionC,
				"--classic",
				"-y")
			Expect(err).To(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, userRolePrefixC)
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Creating classic account roles"))
			Expect(textData).ToNot(ContainSubstring("Creating hosted CP account roles"))
			Expect(textData).To(ContainSubstring("Created role"))
			Expect(textData).ToNot(ContainSubstring("Attached trust policy to role"))

			By("List account-roles and check the result are expected")
			accountRoleList, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())

			accountRoleSetB := accountRoleList.AccountRoles(userRolePrefixB)
			accountRoleSetH := accountRoleList.AccountRoles(userRolePrefixH)
			accountRoleSetC := accountRoleList.AccountRoles(userRolePrefixC)

			selectedRoleH := accountRoleSetH[common.RandomInt(len(accountRoleSetH))]
			selectedRoleC := accountRoleSetC[common.RandomInt(len(accountRoleSetC))]

			Expect(len(accountRoleSetB)).To(Equal(7))
			Expect(len(accountRoleSetH)).To(Equal(3))
			Expect(len(accountRoleSetC)).To(Equal(4))

			Expect(selectedRoleH.RoleArn).
				To(Equal(
					fmt.Sprintf("arn:aws:iam::%s:role%s%s-HCP-ROSA-%s",
						AWSAccountID,
						path,
						userRolePrefixH,
						rosacli.RoleTypeSuffixMap[selectedRoleH.RoleType])))
			Expect(selectedRoleH.OpenshiftVersion).To(Equal(versionH))
			Expect(selectedRoleH.AWSManaged).To(Equal("Yes"))
			Expect(selectedRoleC.RoleArn).
				To(Equal(
					fmt.Sprintf("arn:aws:iam::%s:role%s%s-%s",
						AWSAccountID,
						path,
						userRolePrefixC,
						rosacli.RoleTypeSuffixMap[selectedRoleC.RoleType])))
			Expect(selectedRoleC.OpenshiftVersion).To(Equal(versionC))
			Expect(selectedRoleC.AWSManaged).To(Equal("No"))

			By("Delete account-roles")
			output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", userRolePrefixB,
				"-y")

			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Successfully deleted the classic account roles"))
			Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

			output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", userRolePrefixH,
				"--hosted-cp",
				"-y",
			)

			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

			output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", userRolePrefixC,
				"--classic",
				"-y",
			)

			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Successfully deleted the classic account roles"))

			By("List account-roles to check they are deleted")
			accountRoleList, _, err = ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())

			accountRoleSetB = accountRoleList.AccountRoles(userRolePrefixB)
			accountRoleSetH = accountRoleList.AccountRoles(userRolePrefixH)
			accountRoleSetC = accountRoleList.AccountRoles(userRolePrefixC)

			Expect(len(accountRoleSetB)).To(Equal(0))
			Expect(len(accountRoleSetH)).To(Equal(0))
			Expect(len(accountRoleSetC)).To(Equal(0))
		})

	It("can validate that upgrade account-roles with the managed policies should be forbidden - [id:57441]",
		labels.High, labels.Runtime.OCMResources,
		func() {
			defer func() {
				By("Cleanup created account-roles in high level of the test case")
				if len(accountRolePrefixesNeedCleanup) > 0 {
					for _, v := range accountRolePrefixesNeedCleanup {
						_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
							"--prefix", v,
							"-y")
						Expect(err).To(BeNil())
					}
				}
			}()
			var (
				accrolePrefix = common.GenerateRandomName("arp57441", 2)
				path          = "/aa/vv/"
				modes         = []string{"auto", "manual"}
			)

			By("Create hosted-cp account-roles")
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"--path", path,
				"--hosted-cp",
				"-y")
			Expect(err).To(BeNil())
			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, accrolePrefix)
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
			Expect(textData).To(ContainSubstring("Created role"))

			By("Upgrade managed account-roles")
			for _, mode := range modes {
				output, err := ocmResourceService.UpgradeAccountRole(
					"--prefix", accrolePrefix,
					"--hosted-cp",
					"--mode", mode,
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("have attached managed policies. An upgrade isn't needed"))
			}

			By("Delete account-roles")
			output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"--hosted-cp",
				"-y")

			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

			By("List account-roles to check they are deleted")
			accountRoleList, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())
			Expect(len(accountRoleList.AccountRoles(accrolePrefix))).To(Equal(0))
		})

	It("can delete account-roles with --hosted-cp and --classic - [id:62083]",
		labels.High, labels.Runtime.OCMResources,
		func() {
			defer func() {
				By("Cleanup created account-roles in high level of the test case")
				if len(accountRolePrefixesNeedCleanup) > 0 {
					for _, v := range accountRolePrefixesNeedCleanup {
						_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
							"--prefix", v,
							"-y")

						Expect(err).To(BeNil())
					}
				}
			}()

			var accrolePrefix = common.GenerateRandomName("arp62083", 2)

			By("Create advanced account-roles of both hosted-cp and classic")
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"-y")
			Expect(err).To(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, accrolePrefix)
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Creating classic account roles"))
			Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
			Expect(textData).To(ContainSubstring("Created role"))

			By("Delete account-roles with --classic flag")
			output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"--classic",
				"-y")
			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Successfully deleted the classic account roles"))

			By("Delete account-roles with --hosted-cp flag")
			output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"--hosted-cp",
				"-y",
			)
			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

			By("List account-roles to check they are deleted")
			accountRoleList, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())
			Expect(len(accountRoleList.AccountRoles(accrolePrefix))).To(Equal(0))
		})

	It("create/delete hypershift account roles with managed policies - [id:61322]",
		labels.Critical, labels.Runtime.OCMResources,
		func() {
			defer func() {
				By("Cleanup created account-roles in the test case")
				if len(accountRolePrefixesNeedCleanup) > 0 {
					for _, v := range accountRolePrefixesNeedCleanup {
						_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
							"--prefix", v,
							"-y")

						Expect(err).To(BeNil())
					}
				}
			}()

			var (
				rolePrefixStable    = "prefixS"
				rolePrefixCandidate = "prefixC"
				rolePrefixClassic   = "prefixClassic"
				versionStable       string
				versionCandidate    string
				path                = "/fd/sd/"
			)
			By("Prepare verson for testing")
			versionService := rosaClient.Version
			versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, true)
			Expect(err).To(BeNil())
			defaultVersion := versionList.DefaultVersion()
			Expect(defaultVersion).ToNot(BeNil())
			version, err := versionList.FindNearestBackwardMinorVersion(defaultVersion.Version, 1, true)
			Expect(err).To(BeNil())
			Expect(version).NotTo(BeNil())
			_, _, versionStable, err = version.MajorMinor()
			Expect(err).To(BeNil())

			versionList, err = versionService.ListAndReflectVersions(rosacli.VersionChannelGroupCandidate, true)
			Expect(err).To(BeNil())
			defaultVersion = versionList.DefaultVersion()
			Expect(defaultVersion).ToNot(BeNil())
			version, err = versionList.FindNearestBackwardMinorVersion(defaultVersion.Version, 1, true)
			Expect(err).To(BeNil())
			Expect(version).NotTo(BeNil())
			_, _, versionCandidate, err = version.MajorMinor()
			Expect(err).To(BeNil())

			By("Get the AWS Account Id")
			rosaClient.Runner.JsonFormat()
			whoamiOutput, err := ocmResourceService.Whoami()
			Expect(err).To(BeNil())
			rosaClient.Runner.UnsetFormat()
			whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
			AWSAccountID := whoamiData.AWSAccountID

			By("Create account-roles of hosted-cp in stable channel")
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", rolePrefixStable,
				"--path", path,
				"--permissions-boundary", permissionsBoundaryArn,
				"--force-policy-creation", "--version", versionStable,
				"--channel-group", "stable",
				"--hosted-cp",
				"-y")
			Expect(err).To(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, rolePrefixStable)
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).ToNot(ContainSubstring("Creating classic account roles"))
			Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
			Expect(textData).
				To(ContainSubstring("WARN: Setting `version` flag for hosted CP managed policies has no effect, " +
					"any supported ROSA version can be installed with managed policies"))
			Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Installer-Role'", rolePrefixStable)))
			Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Support-Role'", rolePrefixStable)))
			Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Worker-Role'", rolePrefixStable)))

			By("Create account-roles of hosted-cp in candidate channel")
			output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", rolePrefixCandidate,
				"--path", path,
				"--permissions-boundary", permissionsBoundaryArn,
				"--version", versionCandidate,
				"--channel-group", "candidate",
				"--hosted-cp",
				"-y")
			Expect(err).To(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, rolePrefixCandidate)
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).ToNot(ContainSubstring("Creating classic account roles"))
			Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
			Expect(textData).To(ContainSubstring("WARN: Setting `version` flag for hosted CP managed policies has no effect, " +
				"any supported ROSA version can be installed with managed policies"))
			Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Installer-Role'", rolePrefixCandidate)))
			Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Support-Role'", rolePrefixCandidate)))
			Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Worker-Role'", rolePrefixCandidate)))

			By("List the account roles ")
			accountRoleList, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())

			By("Get the stable/candidate account roles that are created above")
			accountRoleSetStable := accountRoleList.AccountRoles(rolePrefixStable)
			accountRoleSetCandidate := accountRoleList.AccountRoles(rolePrefixCandidate)

			selectedRoleStable := accountRoleSetStable[common.RandomInt(len(accountRoleSetStable))]
			selectedRoleCandidate := accountRoleSetCandidate[common.RandomInt(len(accountRoleSetCandidate))]

			By("Check 3 roles are created for hosted CP account roles")
			Expect(len(accountRoleSetStable)).To(Equal(3))
			Expect(len(accountRoleSetCandidate)).To(Equal(3))

			By("Check the roles are AWS managed, and path and version flag works correctly")
			Expect(selectedRoleStable.AWSManaged).To(Equal("Yes"))
			Expect(selectedRoleStable.RoleArn).
				To(Equal(
					fmt.Sprintf("arn:aws:iam::%s:role%s%s-HCP-ROSA-%s",
						AWSAccountID,
						path,
						rolePrefixStable,
						rosacli.RoleTypeSuffixMap[selectedRoleStable.RoleType])))
			Expect(selectedRoleStable.OpenshiftVersion).To(Equal(versionStable))
			Expect(selectedRoleCandidate.AWSManaged).To(Equal("Yes"))
			Expect(selectedRoleCandidate.RoleArn).
				To(Equal(
					fmt.Sprintf("arn:aws:iam::%s:role%s%s-HCP-ROSA-%s",
						AWSAccountID,
						path,
						rolePrefixCandidate,
						rosacli.RoleTypeSuffixMap[selectedRoleCandidate.RoleType])))
			Expect(selectedRoleCandidate.OpenshiftVersion).To(Equal(versionCandidate))

			By("Delete the hypershift account roles in auto mode")
			output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", rolePrefixStable,
				"--hosted-cp",
				"-y",
			)

			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

			By("Create a classic account role")
			_, err = ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", rolePrefixClassic,
				"--classic",
				"-y")
			Expect(err).To(BeNil())
			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, rolePrefixClassic)

			By("Try to delete classic account-role with --hosted-cp flag")
			output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", rolePrefixClassic,
				"--hosted-cp",
				"-y")
			Expect(err).ToNot(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("WARN: There are no hosted CP account roles to be deleted"))
		})
	It("create/delete classic account roles with managed policies - [id:57408]",
		labels.Critical, labels.Runtime.OCMResources,
		func() {

			var (
				rolePrefixAuto      = common.GenerateRandomName("ar57408a", 2)
				rolePrefixManual    = common.GenerateRandomName("ar57408m", 2)
				roleVersion         string
				path                = "/fd/sd/"
				policiesArn         []string
				managedPoliciesName = []string{
					"ROSAInstallerCorePolicy",
					"ROSAInstallerVPCPolicy",
					"ROSAInstallerPrivateLinkPolicy",
					"ROSAControlPlanePolicy",
					"ROSAWorkerPolicy",
					"ROSASRESupportPolicy",
				}
			)
			awsClient, err := aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())
			defer func() {
				By("Cleanup created account-roles in the test case")
				_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", rolePrefixManual,
					"-y")

				Expect(err).To(BeNil())
				_, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", rolePrefixAuto,
					"-y")

				Expect(err).To(BeNil())

				By("Check managed policies not deleted by rosa command")
				for _, policyArn := range policiesArn {
					policy, err := awsClient.GetIAMPolicy(policyArn)
					Expect(err).To(BeNil())
					Expect(policy).ToNot(BeNil())
				}

				By("Delete fake managed policies")
				for _, policyArn := range policiesArn {
					err := awsClient.DeletePolicy(policyArn)
					Expect(err).To(BeNil())
				}
			}()

			By("Prepare fake managed policies")
			statement := map[string]interface{}{
				"Effect":   "Allow",
				"Action":   "*",
				"Resource": "*",
			}
			for _, pName := range managedPoliciesName {
				pArn, err := awsClient.CreatePolicy(pName, statement)
				Expect(err).To(BeNil())
				policiesArn = append(policiesArn, pArn)
			}
			err = wait.PollUntilContextTimeout(
				context.Background(),
				20*time.Second,
				200*time.Second,
				false,
				func(context.Context) (bool, error) {
					hasPoliciWaitSync := false
					for _, arn := range policiesArn {
						policy, err := awsClient.GetIAMPolicy(arn)
						if err != nil || policy == nil {
							hasPoliciWaitSync = true
							break
						}
					}
					return !hasPoliciWaitSync, err
				})
			common.AssertWaitPollNoErr(err, "Network verification result are not ready after 200")

			By("Prepare verson for testing")
			versionService := rosaClient.Version
			versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, true)
			Expect(err).To(BeNil())
			defaultVersion := versionList.DefaultVersion()
			Expect(defaultVersion).ToNot(BeNil())
			version, err := versionList.FindNearestBackwardMinorVersion(defaultVersion.Version, 1, true)
			Expect(err).To(BeNil())
			Expect(version).NotTo(BeNil())
			_, _, roleVersion, err = version.MajorMinor()
			Expect(err).To(BeNil())

			By("Create a temp dir to execute the create commands")
			dirToClean, err = os.MkdirTemp("", "*")
			Expect(err).To(BeNil())

			By("Create classic account-roles with managed policies in manual mode")
			rosaClient.Runner.SetDir(dirToClean)
			output, err := ocmResourceService.CreateAccountRole("--mode", "manual",
				"--prefix", rolePrefixManual,
				"--path", path,
				"--permissions-boundary", permissionsBoundaryArn,
				"--version", roleVersion,
				"--managed-policies",
				"-y")
			Expect(err).To(BeNil())
			commands := common.ExtractCommandsToCreateAWSResoueces(output)

			for _, command := range commands {
				_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
				Expect(err).To(BeNil())
			}

			By("List the account roles created in manual mode")
			accountRoleList, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())
			accountRoles := accountRoleList.AccountRoles(rolePrefixManual)
			Expect(len(accountRoles)).To(Equal(4))
			for _, ar := range accountRoles {
				Expect(ar.AWSManaged).To(Equal("Yes"))
			}

			By("Delete the account-roles in manual mode")
			output, err = ocmResourceService.DeleteAccountRole("--mode", "manual",
				"--prefix", rolePrefixManual,
				"-y")

			Expect(err).To(BeNil())
			commands = common.ExtractCommandsToDeleteAccountRoles(output)

			for _, command := range commands {
				_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
				Expect(err).To(BeNil())
			}

			By("Create classic account-roles with managed policies in auto mode")
			output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", rolePrefixAuto,
				"--path", path,
				"--permissions-boundary", permissionsBoundaryArn,
				"--version", roleVersion,
				"--managed-policies",
				"-y")
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("Created role"))

			By("List the account roles created in auto mode")
			accountRoleList, _, err = ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())
			accountRoles = accountRoleList.AccountRoles(rolePrefixAuto)
			Expect(len(accountRoles)).To(Equal(4))
			for _, ar := range accountRoles {
				Expect(ar.AWSManaged).To(Equal("Yes"))
			}

			By("Delete the account-roles in auto mode")
			output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", rolePrefixAuto,
				"-y")
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("Successfully deleted"))
		})

	It("Validation for account-role creation by user - [id:43067]",
		labels.Medium, labels.Runtime.OCMResources,
		func() {
			var (
				validRolePrefix                          = "valid"
				invalidRolePrefix                        = "^^^^"
				longRolePrefix                           = "accountroleprefixlongerthan32characters"
				validModeAuto                            = "auto"
				validModeManual                          = "manual"
				invalidMode                              = "invalid"
				invalidPermissionsBoundaryArn     string = "invalid"
				nonExistingPermissionsBoundaryArn string = "arn:aws:iam::aws:policy/non-existing"
			)

			By("Try to create account-roles with invalid prefix")
			output, err := ocmResourceService.CreateAccountRole("--mode", validModeAuto,
				"--prefix", invalidRolePrefix,
				"--permissions-boundary", permissionsBoundaryArn,
				"-y")
			Expect(err).NotTo(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, invalidRolePrefix)
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Expected a valid role prefix matching ^[\\w+=,.@-]+$"))

			By("Try to create account-roles with longer than 32 chars prefix")
			output, err = ocmResourceService.CreateAccountRole("--mode", validModeAuto,
				"--prefix", longRolePrefix,
				"--permissions-boundary", permissionsBoundaryArn,
				"--hosted-cp",
				"-y")
			Expect(err).NotTo(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, longRolePrefix)
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Expected a prefix with no more than 32 characters"))

			By("Try to create account-roles with invalid mode")
			output, err = ocmResourceService.CreateAccountRole("--mode", invalidMode,
				"--prefix", validRolePrefix,
				"--permissions-boundary", permissionsBoundaryArn,
				"-y")
			Expect(err).NotTo(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, validRolePrefix)
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Invalid mode. Allowed values are [auto manual]"))

			By("Try to create account-roles with force-policy-creation and manual mode")
			output, err = ocmResourceService.CreateAccountRole("--mode", validModeManual,
				"--prefix", validRolePrefix,
				"-f",
				"--hosted-cp",
				"--permissions-boundary", permissionsBoundaryArn,
			)
			Expect(err).NotTo(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, validRolePrefix)
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Forcing creation of policies only works in auto mode"))

			By("Try to create account-roles with invalid permission boundary")
			output, err = ocmResourceService.CreateAccountRole("--mode", validModeAuto,
				"--prefix", validRolePrefix,
				"--permissions-boundary", invalidPermissionsBoundaryArn,
				"-y",
			)
			Expect(err).NotTo(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, validRolePrefix)
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).
				To(ContainSubstring(
					"Expected a valid policy ARN for permissions boundary: Invalid ARN: arn: invalid prefix"))

			By("Try to create account-roles with non-existing permission boundary")
			output, err = ocmResourceService.CreateAccountRole("--mode", validModeAuto,
				"--prefix", validRolePrefix,
				"--hosted-cp",
				"--permissions-boundary", nonExistingPermissionsBoundaryArn,
				"-y",
			)
			Expect(err).NotTo(BeNil())

			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, validRolePrefix)
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("There was an error creating the account roles"))
			Expect(textData).To(ContainSubstring("policy/non-existing does not exist or is not attachable"))
		})
})

var _ = Describe("List account roles", labels.Feature.AccountRoles, func() {
	defer GinkgoRecover()

	var (
		rosaClient         *rosacli.Client
		ocmResourceService rosacli.OCMResourceService
	)
	BeforeEach(func() {
		By("Init the client")
		rosaClient = rosacli.NewClient()
		ocmResourceService = rosaClient.OCMResource
	})

	It("to list account-roles by rosa-cli - [id:44511]",
		labels.High, labels.Runtime.OCMResources,
		func() {

			accrolePrefix := "arPrefix44511"
			path := "/a/b/"

			By("Prepare a version for testing")
			var version string
			versionService := rosaClient.Version
			versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, false)
			Expect(err).To(BeNil())

			defaultVersion := versionList.DefaultVersion()
			Expect(defaultVersion).ToNot(BeNil())

			_, _, version, err = defaultVersion.MajorMinor()
			Expect(err).To(BeNil())

			By("Create account-roles")
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"--path", path,
				"--version", version,
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

			By("List account-roles")
			arl, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())

			ars := arl.AccountRoles(accrolePrefix)
			fmt.Println(ars)

			Expect(len(ars)).To(Equal(7))

			for _, v := range ars {
				Expect(v.OpenshiftVersion).To(Equal(version))
				Expect(v.RoleArn).NotTo(BeEmpty())
				if strings.Contains(v.RoleName, "HCP-ROSA") {
					Expect(v.AWSManaged).To(Equal("Yes"))
				} else {
					Expect(v.AWSManaged).To(Equal("No"))
				}
			}
		})
})

var _ = Describe("Create account roles", labels.Feature.AccountRoles, func() {
	defer GinkgoRecover()

	var (
		rosaClient                     *rosacli.Client
		ocmResourceService             rosacli.OCMResourceService
		accountRolePrefixesNeedCleanup []string
		path                           = "/aa/bb/"
	)
	BeforeEach(func() {
		By("Init the client")
		rosaClient = rosacli.NewClient()
		ocmResourceService = rosaClient.OCMResource
	})

	AfterEach(func() {
		By("Cleanup account-roles")
		if len(accountRolePrefixesNeedCleanup) > 0 {
			for _, v := range accountRolePrefixesNeedCleanup {
				_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", v,
					"-y")

				Expect(err).To(BeNil())
			}
		}
	})
	It("to check the trust policy attach information when create account-roles multiple times - [id:75904]",
		labels.High, labels.Runtime.OCMResources,
		func() {
			accountRoleNames := []string{
				"ControlPlane-Role",
				"Worker-Role",
				"Support-Role",
				"Installer-Role",
				"HCP-ROSA-Installer-Role",
				"HCP-ROSA-Support-Role",
				"HCP-ROSA-Worker-Role",
			}

			By("Create account roles")
			accrolePrefix := common.GenerateRandomString(5)
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"--path", path,
				"-y")
			Expect(err).To(BeNil())
			accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, accrolePrefix)
			for _, roleName := range accountRoleNames {
				Expect(output.String()).To(ContainSubstring(fmt.Sprintf(
					"Attached trust policy to role '%s-%s", accrolePrefix, roleName)))
			}

			By("Create account roles with the same configuration again")
			output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"--path", path,
				"-y")
			Expect(err).To(BeNil())
			for _, roleName := range accountRoleNames {
				Expect(output.String()).NotTo(ContainSubstring(fmt.Sprintf(
					"Attached trust policy to role '%s-%s", accrolePrefix, roleName)))
			}

			By("Update the trust relationship")
			awsClient, err := aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())
			roleName := fmt.Sprintf("%s-Support-Role", accrolePrefix)
			opRole, err := awsClient.IamClient.GetRole(
				context.TODO(),
				&iam.GetRoleInput{
					RoleName: &roleName,
				})
			Expect(err).To(BeNil())

			decodedPolicyDocument, err := url.QueryUnescape(*opRole.Role.AssumeRolePolicyDocument)
			Expect(err).To(BeNil())

			By("update the trust relationship")
			var policyDocument map[string]interface{}

			err = json.Unmarshal([]byte(decodedPolicyDocument), &policyDocument)
			Expect(err).To(BeNil())

			newPrincipal := fmt.Sprintf(
				"arn:aws:iam::%s:role/RH-Technical-Support-13849960", constants.JumpAccounts["production"])

			statements := policyDocument["Statement"].([]interface{})
			for _, statement := range statements {
				stmt := statement.(map[string]interface{})
				principal := stmt["Principal"].(map[string]interface{})
				principal["AWS"] = newPrincipal
			}
			updatedPolicyDocument, err := json.Marshal(policyDocument)
			Expect(err).To(BeNil())

			_, err = awsClient.IamClient.UpdateAssumeRolePolicy(context.TODO(), &iam.UpdateAssumeRolePolicyInput{
				RoleName:       aws.String(roleName),
				PolicyDocument: aws.String(string(updatedPolicyDocument)),
			})
			Expect(err).To(BeNil())

			By("Create account roles with the same configuration again after the trust relationship update")
			output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"--path", path,
				"-y")
			Expect(err).To(BeNil())
			for _, roleName := range accountRoleNames {
				if roleName == "Support-Role" {
					Expect(output.String()).To(ContainSubstring(fmt.Sprintf(
						"Attached trust policy to role '%s-%s", accrolePrefix, roleName)))
				} else {
					Expect(output.String()).NotTo(ContainSubstring(fmt.Sprintf(
						"Attached trust policy to role '%s-%s", accrolePrefix, roleName)))
				}
			}
			By("Check the role trust relationship again")
			opRole, err = awsClient.IamClient.GetRole(
				context.TODO(),
				&iam.GetRoleInput{
					RoleName: &roleName,
				})
			Expect(err).To(BeNil())

			for _, accountID := range constants.JumpAccounts {
				Expect(*opRole.Role.AssumeRolePolicyDocument).To(ContainSubstring(accountID))
			}

		})

	It("to create account-roles with invalid version/channel group - [id:75246]",
		labels.High, labels.Runtime.OCMResources,
		func() {
			By("Create account roles with invalid version")
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", "ocp75246",
				"--version", "4.8",
				"-y")
			Expect(err).ToNot(BeNil())
			Expect(output.String()).To(ContainSubstring("Error getting version: A valid policy version number " +
				"must be specified"))

			By("Create account roles with invalid version format")
			output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", "ocp75246",
				"--version", "4.8.10",
				"-y")
			Expect(err).ToNot(BeNil())
			Expect(output.String()).To(ContainSubstring("A valid policy version number must be specified"))

			By("Create account roles with invalid channel group")
			versionService := rosaClient.Version
			// get stable channel version
			versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, true)
			Expect(err).To(BeNil())
			defaultVersion := versionList.DefaultVersion()
			Expect(defaultVersion).ToNot(BeNil())

			output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", "ocp75246",
				"--version", defaultVersion.Version,
				"--channel-group", "fakecg",
				"-y")
			Expect(err).ToNot(BeNil())
			Expect(output.String()).To(ContainSubstring("Error getting version: could not find versions for the " +
				"provided channel-group: 'fakecg'"))

		})

	It("to create/Upgrade account-roles by setting version and channel-group via rosacli - [id:54469]",
		labels.High, labels.Runtime.OCMResources,
		func() {
			By("Prepare y-1 version for testing")
			versionService := rosaClient.Version
			// get stable channel version
			versionListS, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, true)
			Expect(err).To(BeNil())
			defaultVersionS := versionListS.DefaultVersion()
			Expect(defaultVersionS).ToNot(BeNil())
			_, _, upgradeVersionS, err := defaultVersionS.MajorMinor()
			Expect(err).To(BeNil())
			versionS, err := versionListS.FindNearestBackwardMinorVersion(defaultVersionS.Version, 1, true)
			Expect(err).To(BeNil())
			Expect(versionS).NotTo(BeNil())
			_, _, versionStable, err := versionS.MajorMinor()
			Expect(err).To(BeNil())

			// get candidate channel version
			versionListC, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupCandidate, true)
			Expect(err).To(BeNil())
			defaultVersionC := versionListC.DefaultVersion()
			Expect(defaultVersionC).ToNot(BeNil())
			_, _, upgradeVersionC, err := defaultVersionC.MajorMinor()
			Expect(err).To(BeNil())
			versionC, err := versionListC.FindNearestBackwardMinorVersion(defaultVersionC.Version, 1, true)
			Expect(err).To(BeNil())
			Expect(versionC).NotTo(BeNil())
			_, _, versionCandidate, err := versionC.MajorMinor()
			Expect(err).To(BeNil())

			channelVersion := map[string]string{"stable": versionStable, "candidate": versionCandidate}

			By("Create/upgrade account role with version and channel group in auto mode")
			for c, v := range channelVersion {
				By("Create account roles in auto mode with channel group and version")
				accrolePrefix := common.GenerateRandomString(5)
				output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"--version", v,
					"--channel-group", c,
					"-y")
				Expect(err).To(BeNil())
				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, accrolePrefix)

				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Creating classic account roles"))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-ControlPlane-Role'", accrolePrefix)))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-Worker-Role'", accrolePrefix)))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-Support-Role'", accrolePrefix)))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-Installer-Role'", accrolePrefix)))
				Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Installer-Role'", accrolePrefix)))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Support-Role'", accrolePrefix)))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Worker-Role'", accrolePrefix)))

				By("Upgrade account roles in auto mode")
				upgradeVersion := upgradeVersionC
				if c == "stable" {
					upgradeVersion = upgradeVersionS
				}
				output, err = ocmResourceService.UpgradeAccountRole(
					"--prefix", accrolePrefix,
					"--mode", "auto",
					"--version", upgradeVersion,
					"--channel-group", c,
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Ensuring account role policies compatibility for upgrade"))
				Expect(output.String()).To(ContainSubstring("Starting to upgrade the policies"))
				Expect(output.String()).To(ContainSubstring("policy/%s-ControlPlane-Role-Policy' to version '%s'",
					accrolePrefix, upgradeVersion))
				Expect(output.String()).To(ContainSubstring("policy/%s-Worker-Role-Policy' to version '%s'",
					accrolePrefix, upgradeVersion))
				Expect(output.String()).To(ContainSubstring("policy/%s-Support-Role-Policy' to version '%s'",
					accrolePrefix, upgradeVersion))
				Expect(output.String()).To(ContainSubstring("policy/%s-Installer-Role-Policy' to version '%s'",
					accrolePrefix, upgradeVersion))
			}

			By("Create/upgrade account role with version and channel group in manual mode")
			for c, v := range channelVersion {
				By("Create account roles in manual mode with channel group and version")
				accrolePrefix := common.GenerateRandomString(5)
				output, err := ocmResourceService.CreateAccountRole("--mode", "manual",
					"--prefix", accrolePrefix,
					"--path", path,
					"--version", v,
					"--channel-group", c,
					"-y")
				Expect(err).To(BeNil())
				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, accrolePrefix)

				commands := common.ExtractCommandsToCreateAWSResoueces(output)

				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					Expect(err).To(BeNil())
				}

				By("List the account roles created in manual mode")
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				accountRoles := accountRoleList.AccountRoles(accrolePrefix)
				Expect(len(accountRoles)).To(Equal(7))
				for _, ar := range accountRoles {
					Expect(ar.OpenshiftVersion).To(Equal(v))
				}

				By("Upgrade account roles in manual mode")
				upgradeVersion := upgradeVersionC
				if c == "stable" {
					upgradeVersion = upgradeVersionS
				}
				output, err = ocmResourceService.UpgradeAccountRole(
					"--prefix", accrolePrefix,
					"--mode", "manual",
					"--version", upgradeVersion,
					"--channel-group", c,
					"-y",
				)
				Expect(err).To(BeNil())

				commands = common.ExtractCommandsToCreateAWSResoueces(output)

				for _, command := range commands {
					cmd := strings.Split(command, " ")
					if len(cmd) > 0 && cmd[len(cmd)-1] == "" {
						cmd = cmd[:len(cmd)-1]
					}
					_, err := rosaClient.Runner.RunCMD(cmd)
					Expect(err).To(BeNil())
				}

				By("List the account roles upgraded in manual mode")
				accountRoleList, _, err = ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				accountRoles = accountRoleList.AccountRoles(accrolePrefix)
				Expect(len(accountRoles)).To(Equal(7))
				for _, ar := range accountRoles {
					if ar.AWSManaged == "Yes" {
						Expect(ar.OpenshiftVersion).To(Equal(v))
					} else {
						Expect(ar.OpenshiftVersion).To(Equal(upgradeVersion))
					}
				}
			}
		})
})
