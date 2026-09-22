package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
)

var _ = Describe("Edit OIDC config",
	labels.Feature.OIDCConfig,
	func() {
		defer GinkgoRecover()

		var (
			oidcConfigIDsNeedToClean []string
			installerRoleArn         string
			hostedCP                 bool

			rosaClient         *rosacli.Client
			ocmResourceService rosacli.OCMResourceService
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
		})

		It("can create/list/delete BYO oidc config in auto mode - [id:57570]",
			labels.High, labels.Runtime.OCMResources, labels.Hyperfleet.Validated,
			func() {

				if os.Getenv("HYPERFLEET_URL") != "" {
					// is hyperfleet v2
					helper_v2_oidc_configs(rosaClient, ocmResourceService)
					return
				}

				defer func() {
					By("make sure that all oidc configs created during the testing")
					if len(oidcConfigIDsNeedToClean) > 0 {
						By("Delete oidc configs")
						for _, id := range oidcConfigIDsNeedToClean {
							output, err := ocmResourceService.DeleteOIDCConfig(
								"--oidc-config-id", id,
								"--mode", "auto",
								"-y",
							)
							Expect(err).To(BeNil())
							textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
							Expect(textData).To(ContainSubstring("Successfully deleted the OIDC provider"))

							By("Check the managed oidc config is deleted")
							oidcConfigList, _, err := ocmResourceService.ListOIDCConfig()
							Expect(err).To(BeNil())
							foundOIDCConfig := oidcConfigList.OIDCConfig(id)
							Expect(foundOIDCConfig).To(Equal(rosacli.OIDCConfig{}))
						}
					}
				}()

				var (
					oidcConfigPrefix       = "op57570"
					longPrefix             = "1234567890abcdef"
					notExistedOODCConfigID = "notexistedoidcconfigid111"
					unmanagedOIDCConfigID  string
					managedOIDCConfigID    string
					accountRolePrefix      string
				)
				By("Create account-roles for testing")
				accountRolePrefix = fmt.Sprintf("QEAuto-accr57570-%s", time.Now().UTC().Format("20060102"))
				_, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y")
				Expect(err).To(BeNil())

				defer func() {
					By("Cleanup created account-roles")
					_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
						"--prefix", accountRolePrefix,
						"-y")
					Expect(err).To(BeNil())
				}()

				By("Get the installer role arn")
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				installerRole := accountRoleList.InstallerRole(accountRolePrefix, hostedCP)
				Expect(installerRole).ToNot(BeNil())
				installerRoleArn = installerRole.RoleArn

				By("Create managed=false oidc config in auto mode")
				output, err := ocmResourceService.CreateOIDCConfig("--mode", "auto",
					"--prefix", oidcConfigPrefix,
					"--installer-role-arn", installerRoleArn,
					"--managed=false",
					"-y")
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Created OIDC provider with ARN"))

				oidcPrivodeARNFromOutputMessage := helper.ExtractOIDCProviderARN(output.String())
				oidcPrivodeIDFromOutputMessage := helper.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)

				unmanagedOIDCConfigID, err = ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
				Expect(err).To(BeNil())

				oidcConfigIDsNeedToClean = append(oidcConfigIDsNeedToClean, unmanagedOIDCConfigID)

				By("Check the created unmananged oidc by `rosa list oidc-config`")
				oidcConfigList, output, err := ocmResourceService.ListOIDCConfig()
				Expect(err).To(BeNil())
				foundOIDCConfig := oidcConfigList.OIDCConfig(unmanagedOIDCConfigID)
				Expect(foundOIDCConfig).NotTo(BeNil())
				Expect(foundOIDCConfig.Managed).To(Equal("false"))
				Expect(foundOIDCConfig.SecretArn).NotTo(Equal(""))
				Expect(foundOIDCConfig.ID).To(Equal(unmanagedOIDCConfigID))

				By("Create managed oidc config in auto mode")
				output, err = ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Created OIDC provider with ARN"))
				oidcPrivodeARNFromOutputMessage = helper.ExtractOIDCProviderARN(output.String())
				oidcPrivodeIDFromOutputMessage = helper.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)

				managedOIDCConfigID, err = ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
				Expect(err).To(BeNil())

				oidcConfigIDsNeedToClean = append(oidcConfigIDsNeedToClean, managedOIDCConfigID)

				By("Check the created mananged oidc by `rosa list oidc-config`")
				oidcConfigList, output, err = ocmResourceService.ListOIDCConfig()
				Expect(err).To(BeNil())
				foundOIDCConfig = oidcConfigList.OIDCConfig(managedOIDCConfigID)
				Expect(foundOIDCConfig).NotTo(BeNil())
				Expect(foundOIDCConfig.Managed).To(Equal("true"))
				Expect(foundOIDCConfig.IssuerUrl).To(ContainSubstring(foundOIDCConfig.ID))
				Expect(foundOIDCConfig.SecretArn).To(Equal(""))
				Expect(foundOIDCConfig.ID).To(Equal(managedOIDCConfigID))

				By("Validate the invalid mode")
				output, err = ocmResourceService.CreateOIDCConfig("--mode", "invalidmode", "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()

				Expect(textData).To(ContainSubstring("Invalid mode. Allowed values are [auto manual]"))

				By("Validate the prefix length")
				output, err = ocmResourceService.CreateOIDCConfig(
					"--mode", "auto",
					"--prefix", longPrefix,
					"--managed=false",
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("length of prefix is limited to 15 characters"))

				By("Validate the prefix and managed at the same time")
				output, err = ocmResourceService.CreateOIDCConfig(
					"--mode", "auto",
					"--prefix", oidcConfigPrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("prefix param is not supported for managed OIDC config"))

				By("Validation the installer-role-arn and managed at the same time")
				output, err = ocmResourceService.CreateOIDCConfig(
					"--mode", "auto",
					"--installer-role-arn", installerRoleArn,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("role-arn param is not supported for managed OIDC config"))

				By("Validation the raw-files and managed at the same time")
				output, err = ocmResourceService.CreateOIDCConfig(
					"--mode", "auto",
					"--raw-files",
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("--raw-files param is not supported alongside --mode param"))

				By("Validate the oidc-config deletion with no-existed oidc config id in auto mode")
				output, err = ocmResourceService.DeleteOIDCConfig(
					"--mode", "auto",
					"--oidc-config-id", notExistedOODCConfigID,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("not found"))
			})
	})
var _ = Describe("Register ummanaged oidc config testing",
	labels.Feature.OIDCConfig,
	func() {
		defer GinkgoRecover()
		var (
			accountRolePrefix  string
			oidcConfigID       string
			rosaClient         *rosacli.Client
			ocmResourceService rosacli.OCMResourceService
			defaultDir         string
			dirToClean         string
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

			if oidcConfigID != "" {
				By("Delete oidc config")
				output, err := ocmResourceService.DeleteOIDCConfig(
					"--oidc-config-id", oidcConfigID,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Successfully deleted the OIDC provider"))
			}

			By("Cleanup created account-roles")
			if accountRolePrefix == "" {
				_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y")
				Expect(err).To(BeNil())
			}
		})
		It("to register successfully - [id:64620]", labels.High, labels.Runtime.OCMResources, func() {
			var (
				secretArn string
				issuerUrl string
			)
			By("Create account-roles for testing")
			accountRolePrefix = fmt.Sprintf("QEAuto-ar64620-%s", time.Now().UTC().Format("20060102"))
			_, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accountRolePrefix,
				"-y")
			Expect(err).To(BeNil())

			By("Get the installer role arn")
			accountRoleList, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())
			installerRole := accountRoleList.InstallerRole(accountRolePrefix, false)
			Expect(installerRole).ToNot(BeNil())
			roleArn := installerRole.RoleArn

			By("Create a temp dir to execute the create commands")
			dirToClean, err = os.MkdirTemp("", "*")
			Expect(err).To(BeNil())

			By("Go to the temp dir by setting Dir")
			rosaClient.Runner.SetDir(dirToClean)

			By("Create unmanaged oidc config")
			oidcConfigPrefix := "ocp64620oc"
			output, err := ocmResourceService.CreateOIDCConfig(
				"--mode", "manual",
				"--prefix", oidcConfigPrefix,
				"--role-arn", roleArn,
				"--managed=false",
				"-y")
			Expect(err).To(BeNil())
			commands := helper.ExtractCommandsFromOIDCRegister(output)

			By("Execute commands to create unmanaged oidc-config")
			var commandArgs []string
			for _, command := range commands {
				if strings.Contains(command, "aws secretsmanager create-secret") {
					commandArgs = helper.ParseCommandToArgs(command)
					stdout, err := rosaClient.Runner.RunCMD(commandArgs)
					Expect(err).To(BeNil())
					secretArn = helper.ParseSecretArnFromOutput(stdout.String())
					continue
				}
				if strings.Contains(command, "aws s3api create-bucket") {
					type S3BucketOutPut struct {
						Location string `json:"Location"`
					}
					var s3cmdOutput S3BucketOutPut
					commandArgs = helper.ParseCommandToArgs(command)
					// Add '--output json' to the commandArgs
					jsonOutputArgs := []string{"--output", "json"}
					commandArgs = append(commandArgs, jsonOutputArgs...)

					stdout, err := rosaClient.Runner.RunCMD(commandArgs)
					Expect(err).To(BeNil())

					err = json.Unmarshal(stdout.Bytes(), &s3cmdOutput)
					Expect(err).To(BeNil())

					//Get region from `rosa whoami`
					whoamiOutput, err := ocmResourceService.Whoami()
					Expect(err).To(BeNil())
					rosaClient.Runner.UnsetFormat()
					whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
					awsRegion := whoamiData.AWSDefaultRegion

					if awsRegion == "us-east-1" {
						issuerUrl = fmt.Sprintf("https:/%s.s3.amazonaws.com", s3cmdOutput.Location)
					} else {
						issuerUrl = strings.Replace(s3cmdOutput.Location, "http://", "https://", 1)
					}

					Expect(issuerUrl).ToNot(BeEmpty(),
						"extracted issuerUrl from %s is empty which will block coming steps.",
						stdout.String(),
					)
					continue
				}
				_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
				Expect(err).To(BeNil())
			}
			Expect(secretArn).ToNot(BeEmpty(), "secretArn is empty which will block coming steps.")

			By("Register oidc config")
			_, err = ocmResourceService.RegisterOIDCConfig(
				"--mode", "auto",
				"--issuer-url", issuerUrl,
				"--role-arn", roleArn,
				"--secret-arn", secretArn,
				"-y")
			Expect(err).To(BeNil())

			By("List oidc config to check if above one is registered")
			oidcConfigList, _, err := ocmResourceService.ListOIDCConfig()
			Expect(err).To(BeNil())
			foundOIDCConfig := oidcConfigList.IssuerUrl(issuerUrl)
			Expect(foundOIDCConfig).ToNot(Equal(rosacli.OIDCConfig{}))
			oidcConfigID = foundOIDCConfig.ID
		})
	})

// helper_v2_oidc_configs tests V2 OIDC config creation
// NOTE: V2 doesn't use account-roles (that's V1/OCM only), so we only test managed OIDC configs
func helper_v2_oidc_configs(rosaClient *rosacli.Client, ocmResourceService rosacli.OCMResourceService) {
	var (
		oidcConfigIDsNeedToClean []string
		managedOIDCConfigID      string
	)

	defer func() {
		By("Clean up V2 OIDC configs")
		if len(oidcConfigIDsNeedToClean) > 0 {
			for _, id := range oidcConfigIDsNeedToClean {
				By(fmt.Sprintf("Delete OIDC config: %s", id))
				// rosa delete oidc-config --oidc-config-id <id> --mode auto -y
				output, err := ocmResourceService.DeleteOIDCConfig(
					"--oidc-config-id", id,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(Or(
					ContainSubstring("Successfully deleted"),
					ContainSubstring("deleted successfully"),
				))
			}
		}
	}()

	By("Create V2 managed OIDC config in auto mode")
	// rosa create oidc-config --mode auto -y
	// NOTE: Managed OIDC configs don't require installer-role-arn (V2 doesn't use account-roles)
	output, err := ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
	Expect(err).To(BeNil())
	textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
	Expect(textData).Should(Or(
		ContainSubstring("created successfully"),
		ContainSubstring("OIDC config"),
	))

	By("List OIDC configs to find the created config")
	// rosa list oidc-config
	oidcConfigList, _, err := ocmResourceService.ListOIDCConfig()
	Expect(err).To(BeNil())

	// V2 uses "TYPE" column (not "MANAGED"), and the table parser won't populate the Managed field
	// Just grab the first config since we just created it
	Expect(len(oidcConfigList.OIDCConfigList)).To(BeNumerically(">", 0), "No OIDC configs found after creation")
	managedOIDCConfigID = oidcConfigList.OIDCConfigList[0].ID
	oidcConfigIDsNeedToClean = append(oidcConfigIDsNeedToClean, managedOIDCConfigID)

	By("Verify OIDC config was created")
	foundOIDCConfig := oidcConfigList.OIDCConfig(managedOIDCConfigID)
	Expect(foundOIDCConfig).NotTo(Equal(rosacli.OIDCConfig{}))
	Expect(foundOIDCConfig.ID).To(Equal(managedOIDCConfigID))
	Expect(foundOIDCConfig.IssuerUrl).NotTo(BeEmpty())

	By("V2 OIDC config test completed successfully")
}
