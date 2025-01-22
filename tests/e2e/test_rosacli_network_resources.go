package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
)

var _ = Describe("Network Resources",
	labels.Feature.NetworkResources,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient              *rosacli.Client
			networkResourcesService rosacli.NetworkResourcesService
			ocmResourceService      rosacli.OCMResourceService
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			networkResourcesService = rosaClient.NetworkResources
			ocmResourceService = rosaClient.OCMResource
		})

		It("should be created successfully - [id:77140]",
			labels.High, labels.Runtime.OCMResources,
			func() {
				By("Get region value")
				region := "us-west-2"
				if os.Getenv("REGION") != "" {
					region = os.Getenv("REGION")
				} else if os.Getenv("AWS_REGION") != "" {
					region = os.Getenv("AWS_REGION")
				}
				By("Create aws client")
				awsClient, err := aws_client.CreateAWSClient("", region)
				Expect(err).ToNot(HaveOccurred())

				By("Get the organization id")
				accInfo, err := ocmResourceService.UserInfo()
				Expect(err).ToNot(HaveOccurred())
				awsAccountID := accInfo.AWSAccountID

				By("Check the help message")
				_, err = networkResourcesService.CreateNetworkResources(false, "--help")
				Expect(err).ToNot(HaveOccurred())

				By("Create template dir for template file creating single vpc")
				templateContent := helper.TemplateForSingleVPC()
				templatePath, err := helper.CreateTemplateDirForNetworkResources("single-vpc", templateContent)
				defer func() {
					os.Remove(templatePath)
					Eventually(func() (bool, error) {
						_, err := os.Stat(templatePath)
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
					os.Remove("single-vpc")
					Eventually(func() (bool, error) {
						_, err := os.Stat("single-vpc")
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
				}()
				Expect(err).ToNot(HaveOccurred())

				By("Get current working directory as template dir path")
				templateDir := filepath.Dir(templatePath)

				templateDirPath := filepath.Dir(templateDir)
				templateDirName := filepath.Base(templateDir)
				Expect(err).ToNot(HaveOccurred())

				By("Create network resources without passing template name and parameter")
				defaultName := fmt.Sprintf("rosa-network-stack-%s", awsAccountID)
				output, err := networkResourcesService.CreateNetworkResources(false)
				defer func() {
					params := cloudformation.DeleteStackInput{
						StackName: &defaultName,
					}
					_, err = awsClient.StackFormationClient.DeleteStack(context.TODO(), &params)
					Expect(err).ToNot(HaveOccurred())
				}()
				Expect(err).ToNot(HaveOccurred())
				resp := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(resp).To(And(
					ContainSubstring("Name not provided, using default name %s", defaultName),
					ContainSubstring("No template name provided in the command. Defaulting to rosa-quickstart-default-vpc"),
					ContainSubstring("Region not provided, using default region")))

				By("Create network resources by passing template name and all parameters")
				stackName_1 := helper.GenerateRandomName("ocp-77140", 2)
				paramNameFlag := fmt.Sprintf("--param=Name=%s", stackName_1)
				paramRegionFlag := fmt.Sprintf("--param=Region=%s", region)
				output, err = networkResourcesService.CreateNetworkResources(false, templateDirName,
					paramNameFlag,
					paramRegionFlag,
					"--template-dir", templateDirPath,
					"--param=AvailabilityZoneCount=3",
					"--param=Tags=Key1=Value1,Key2=Value2",
					"--param=VpcCidr=10.0.0.0/20")
				defer func() {
					params := cloudformation.DeleteStackInput{
						StackName: &stackName_1,
					}
					_, err = awsClient.StackFormationClient.DeleteStack(context.TODO(), &params)
					Expect(err).ToNot(HaveOccurred())
				}()
				Expect(err).ToNot(HaveOccurred())
				resp_tip := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				resp = rosaClient.Parser.TextData.Input(output).Parse().Output()
				Expect(resp_tip).ToNot(
					ContainSubstring(
						"No template name provided in the command. Defaulting to rosa-quickstart-default-vpc"))
				Expect(resp).To(
					ContainSubstring("msg=\"Stack %s created\"", stackName_1))

				By("Create network using manual mode")
				stackName_2 := helper.GenerateRandomName("ocp-77140", 2)
				paramNameFlag = fmt.Sprintf("--param=Name=%s", stackName_2)
				output, err = networkResourcesService.CreateNetworkResources(false, templateDirName,
					paramNameFlag,
					paramRegionFlag,
					"--template-dir", templateDirPath,
					"--param=AvailabilityZoneCount=3",
					"--param=Tags=Key1=Value1,Key2=Value2",
					"--mode", "manual")
				defer func() {
					params := cloudformation.DeleteStackInput{
						StackName: &stackName_2,
					}
					_, err = awsClient.StackFormationClient.DeleteStack(context.TODO(), &params)
					Expect(err).ToNot(HaveOccurred())
				}()
				Expect(err).ToNot(HaveOccurred())
				resp = rosaClient.Parser.TextData.Input(output).Parse().Output()
				Expect(resp).To(
					ContainSubstring("aws cloudformation create-stack --stack-name"))

				By("Try to create network by setting OCM_TEMPLATE_DIR env variable")
				err = os.Setenv("OCM_TEMPLATE_DIR", templateDirPath)
				Expect(err).ToNot(HaveOccurred())
				stackName_3 := helper.GenerateRandomName("ocp-77140", 2)
				paramNameFlag = fmt.Sprintf("--param=Name=%s", stackName_3)
				output, err = networkResourcesService.CreateNetworkResources(false, templateDirName,
					paramNameFlag,
					paramRegionFlag)

				Expect(err).To(HaveOccurred())

				Expect(output.String()).To(
					ContainSubstring("when using a custom template please use `--template-dir` to specify the template directory"))

				By("Try to override 'OCM_TEMPLATE_DIR' env variable using --template-dir flag")
				err = os.Setenv("OCM_TEMPLATE_DIR", "/fake/dir")
				Expect(err).ToNot(HaveOccurred())
				stackName_4 := helper.GenerateRandomName("ocp-77140", 2)
				paramNameFlag = fmt.Sprintf("--param=Name=%s", stackName_4)
				output, err = networkResourcesService.CreateNetworkResources(false, templateDirName,
					paramNameFlag,
					paramRegionFlag,
					"--template-dir", templateDirPath)
				defer func() {
					params := cloudformation.DeleteStackInput{
						StackName: &stackName_4,
					}
					_, err = awsClient.StackFormationClient.DeleteStack(context.TODO(), &params)
					Expect(err).ToNot(HaveOccurred())
				}()
				Expect(err).ToNot(HaveOccurred())
				resp = rosaClient.Parser.TextData.Input(output).Parse().Output()
				Expect(resp).To(
					ContainSubstring("msg=\"Stack %s created\"", stackName_4))
			})

		It("should be validated successfully - [id:77159]",
			labels.Medium, labels.Runtime.OCMResources,
			func() {
				By("Create aws client")
				awsClient, err := aws_client.CreateAWSClient("", "us-west-2")
				Expect(err).ToNot(HaveOccurred())

				By("Create template dir for template file missing Region Param")
				templateContent := helper.TemplateWithoutRegionParam()
				templatePath_1, err := helper.CreateTemplateDirForNetworkResources("without-region", templateContent)

				templateDir := filepath.Dir(templatePath_1)
				tdpWithoutReion := filepath.Dir(templateDir)
				tdnWithoutReion := filepath.Base(templateDir)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					os.Remove(templatePath_1)
					Eventually(func() (bool, error) {
						_, err := os.Stat(templatePath_1)
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
					os.Remove("without-region")
					Eventually(func() (bool, error) {
						_, err := os.Stat("without-region")
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
				}()
				Expect(err).ToNot(HaveOccurred())

				By("Create template dir for template file missing Name Param")
				templateContent = helper.TemplateWithoutNameParam()
				templatePath_2, err := helper.CreateTemplateDirForNetworkResources("without-name", templateContent)
				templateDir = filepath.Dir(templatePath_2)
				tdpWithoutName := filepath.Dir(templateDir)
				tdnWithoutName := filepath.Base(templateDir)
				defer func() {
					os.Remove(templatePath_2)
					Eventually(func() (bool, error) {
						_, err := os.Stat(templatePath_2)
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
					os.Remove("without-name")
					Eventually(func() (bool, error) {
						_, err := os.Stat("without-name")
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
				}()
				Expect(err).ToNot(HaveOccurred())

				By("Create template dir for template file missing VpcCidr value")
				templateContent = helper.TemplateWithoutCidrValueForVPC()
				templatePath_3, err := helper.CreateTemplateDirForNetworkResources("without-vpccidr", templateContent)
				templateDir = filepath.Dir(templatePath_3)
				tdpWithoutVPCCIDR := filepath.Dir(templateDir)
				tdnWithoutVPCCIDR := filepath.Base(templateDir)
				defer func() {
					os.Remove(templatePath_3)
					Eventually(func() (bool, error) {
						_, err := os.Stat(templatePath_3)
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
					os.Remove("without-vpccidr")
					Eventually(func() (bool, error) {
						_, err := os.Stat("without-vpccidr")
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
				}()
				Expect(err).ToNot(HaveOccurred())

				By("Create template dir for template file creating single vpc")
				templateContent = helper.TemplateForSingleVPC()
				templatePath_4, err := helper.CreateTemplateDirForNetworkResources("single-vpc", templateContent)
				templateDir = filepath.Dir(templatePath_4)
				tdpSingleVPC := filepath.Dir(templateDir)
				tdnSingleVPC := filepath.Base(templateDir)
				defer func() {
					os.Remove(templatePath_4)
					Eventually(func() (bool, error) {
						_, err := os.Stat(templatePath_4)
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
					os.Remove("single-vpc")
					Eventually(func() (bool, error) {
						_, err := os.Stat("single-vpc")
						if err != nil {
							if os.IsNotExist(err) {
								return true, nil
							} else {
								return false, err
							}
						} else {
							return false, err
						}
					}, time.Minute*1, time.Second*5).Should(Equal(true))
				}()
				Expect(err).ToNot(HaveOccurred())

				By("Get current working directory as template dir path")
				invalidTemplateDir := "/ss"
				nonExistentTemplate := "non-existent"
				invalidTemplateDirErrorMessage := "ERR: failed to read template file"
				nonExistentTemplateErrorMessage := "ERR: failed to read template file"
				rollBackInProgress := "ROLLBACK_IN_PROGRESS"
				rollBackRequested := "Rollback requested by user"
				withoutCidrErrorMessage := "Either CIDR Block or IPv4 IPAM Pool and IPv4 Netmask Length must be provided"
				incorrectCidrErrorMessage := "Value (10.0.) for parameter cidrBlock is invalid"
				incorrectStackNameErrorMessage := "Value '$#aaraj' at 'stackName' failed to satisfy constraint: " +
					"Member must satisfy regular expression pattern: [a-zA-Z][-a-zA-Z0-9]*"

				reqAndErrBody := map[string][]string{
					"Error: flag needs an argument: --param": {"--param"},
					"ERR: invalid parameter format":          {"--param="},
					"Parameters: [Namwe] do not exist in the template": {"--param=Namwe=",
						"--param=Name=invalid-param"},
					invalidTemplateDirErrorMessage: {"ss",
						"--template-dir", invalidTemplateDir, "--param=Name=invalid-dir"},
					"Error: unknown flag: --invalid": {"--invalid"},
					"ERR: duplicate tag key Key1": {"--param=Tags=Key1=Value1,Key1=Value2",
						"--param=Name=duplicate-key"},
					nonExistentTemplateErrorMessage: {nonExistentTemplate,
						"--template-dir", invalidTemplateDir, "--param=Name=invalid-dir"},
					"Parameters: [Region] do not exist in the template": {tdnWithoutReion,
						"--template-dir", tdpWithoutReion, fmt.Sprintf("--param=Name=%s", tdnWithoutReion)},
					"Parameters: [Name] do not exist in the template": {tdnWithoutName,
						"--template-dir", tdpWithoutName, fmt.Sprintf("--param=Name=%s", tdnWithoutName)},
					withoutCidrErrorMessage: {tdnWithoutVPCCIDR,
						"--template-dir", tdpWithoutVPCCIDR, fmt.Sprintf(
							"--param=Name=%s", tdnWithoutVPCCIDR), "--param=Region=us-west-2",
					},
					"Parameter 'AvailabilityZoneCount' must be a number not greater than 3": {
						"--param=AvailabilityZoneCount=10", "--param=Name=invalid-az"},
					"Parameter 'AvailabilityZoneCount' must be a number not less than 1": {
						"--param=AvailabilityZoneCount=0", "--param=Name=invalid-az"},
					"request send failed, Post \"https://cloudformation.ind-west-2.amazonaws.com/\"": {
						"--param=Region=ind-west-2", "--param=Name=invalid-region"},
					incorrectStackNameErrorMessage: {"--param=Name=$#aaraj"},
					incorrectCidrErrorMessage: {tdnSingleVPC,
						"--template-dir", tdpSingleVPC,
						"--param=VpcCidr=10.0.", "--param=Name=incorrectcidr", "--param=Region=us-west-2"},
				}

				By("Try to create network by setting invalid arguments and flags")
				for key, value := range reqAndErrBody {
					output, err := networkResourcesService.CreateNetworkResources(false, value...)
					Expect(err).To(HaveOccurred())
					resp := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(resp).To(ContainSubstring(key))
					if key == withoutCidrErrorMessage || key == incorrectCidrErrorMessage {
						Expect(resp).To(ContainSubstring(rollBackInProgress))
						Expect(resp).To(ContainSubstring(rollBackRequested))
						var name string
						if key == withoutCidrErrorMessage {
							name = "withoutcidr"
						}
						if key == incorrectCidrErrorMessage {
							name = "incorrectcidr"
						}
						params := cloudformation.DeleteStackInput{
							StackName: &name,
						}
						_, err := awsClient.StackFormationClient.DeleteStack(context.TODO(), &params)
						Expect(err).ToNot(HaveOccurred())
					}
				}
			})
	})
