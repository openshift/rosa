package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit ocm role", labels.Feature.OCMRole,
	func() {
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
		It("make sure no ocm-role and user role linked - [id:52106]", labels.High, labels.Runtime.Day2, func() {
			By("Check if there is the ocm-role linked, and delete it")
			ocmRoleList, _, err := ocmResourceService.ListOCMRole()
			ocmRole := ocmRoleList.FindLinkedOCMRole()
			Expect(err).To(BeNil())
			if (ocmRole != rosacli.OCMRole{}) {
				output, err := ocmResourceService.DeleteOCMRole(
					"--role-arn", ocmRole.RoleArn,
					"--mode", "auto",
					"-y")
				Expect(err).To(BeNil())
				Expect(output.String()).Should(ContainSubstring("Successfully deleted the OCM role"))
			}
			By("Check if there is the user-role linked, and delete it")
			userRoleList, _, err := ocmResourceService.ListUserRole()
			Expect(err).To(BeNil())
			userRole := userRoleList.FindLinkedUserRole()
			Expect(err).To(BeNil())
			if (userRole != rosacli.UserRole{}) {
				output, err := ocmResourceService.DeleteUserRole(
					"--role-arn", userRole.RoleArn,
					"--mode", "auto",
					"-y")
				Expect(err).To(BeNil())
				Expect(output.String()).Should(ContainSubstring("Successfully deleted the user role"))
			}
		})

		It("can create/delete/unlink/link ocm-roles in auto mode - [id:46187]",
			labels.High, labels.Runtime.OCMResources,
			func() {
				var (
					ocmrolePrefix                                 string
					invalidPermisionBoundary                      string
					notExistedPermissionBoundaryUnderDifferentAWS string
					ocmOrganizationExternalID                     string
					notExistedOcmroleocmRoleArn                   string
					ocmroleArnInWrongFormat                       string
					foundOcmrole                                  rosacli.OCMRole
					path                                          = "/aa/bb/"
					ocmRoleList                                   rosacli.OCMRoleList
					ocmRoleNeedRecoved                            rosacli.OCMRole
				)
				rosaClient.Runner.JsonFormat()
				whoamiOutput, err := ocmResourceService.Whoami()
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
				ocmOrganizationExternalID = whoamiData.OCMOrganizationExternalID
				ocmrolePrefix = fmt.Sprintf("QEAuto-ocmr-%s-46187", time.Now().UTC().Format("20060102"))

				By("Check linked ocm role then unlink it")
				ocmRoleList, _, err = ocmResourceService.ListOCMRole()
				ocmRoleNeedRecoved = ocmRoleList.FindLinkedOCMRole()
				Expect(err).To(BeNil())
				if (ocmRoleNeedRecoved != rosacli.OCMRole{}) {
					output, err := ocmResourceService.UnlinkOCMRole("--role-arn", ocmRoleNeedRecoved.RoleArn, "-y")
					Expect(err).To(BeNil())
					Expect(output.String()).Should(ContainSubstring("Successfully unlinked role"))
					defer func() {
						By("Link the ocm-role to recover the original status")
						if (ocmRoleNeedRecoved != rosacli.OCMRole{}) {
							output, err := ocmResourceService.LinkOCMRole("--role-arn", ocmRoleNeedRecoved.RoleArn, "-y")
							Expect(err).To(BeNil())
							Expect(output.String()).Should(ContainSubstring("Successfully linked role"))
						}
					}()
				}
				defer func() {
					By("Delete ocm-role")
					ocmRoleList, _, err := ocmResourceService.ListOCMRole()
					Expect(err).To(BeNil())
					foundOcmrole := ocmRoleList.OCMRole(ocmrolePrefix, ocmOrganizationExternalID)
					output, err := ocmResourceService.DeleteOCMRole("--mode", "auto",
						"--role-arn", foundOcmrole.RoleArn,
						"-y")

					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).Should(ContainSubstring("Successfully deleted the OCM role"))
				}()

				By("Create an ocm-role with invalid mode")
				output, err := ocmResourceService.CreateOCMRole("--mode", "invalidamode",
					"--prefix", ocmrolePrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Invalid mode. Allowed values are [auto manual]"))

				By("Create an ocm-role with invalid permision boundady")
				invalidPermisionBoundary = "arn-permission-boundary"
				output, err = ocmResourceService.CreateOCMRole("--mode", "auto",
					"--permissions-boundary", invalidPermisionBoundary,
					"--prefix", ocmrolePrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Expected a valid policy ARN for permissions boundary"))

				By("Create ocm-role with the permision boundady under another aws account")
				notExistedPermissionBoundaryUnderDifferentAWS = "arn:aws:iam::aws:policy/notexisted"
				output, err = ocmResourceService.CreateOCMRole("--mode", "auto",
					"--permissions-boundary", notExistedPermissionBoundaryUnderDifferentAWS,
					"--prefix", ocmrolePrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("There was an error creating the ocm role"))
				Expect(textData).Should(ContainSubstring("NoSuchEntity"))

				By("Create an ocm-role")
				output, err = ocmResourceService.CreateOCMRole("--mode", "auto",
					"--prefix", ocmrolePrefix,
					"--path", path,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Created role"))
				Expect(textData).Should(ContainSubstring("Successfully linked role"))
				Expect(textData).Should(ContainSubstring("Attached trust policy to role"))

				By("Get the ocm-role info")
				ocmRoleList, output, err = ocmResourceService.ListOCMRole()
				Expect(output).ToNot(BeNil())
				Expect(err).To(BeNil())
				foundOcmrole = ocmRoleList.OCMRole(ocmrolePrefix, ocmOrganizationExternalID)
				Expect(foundOcmrole).ToNot(BeNil())

				By("Unlink ocm-role with not-exist role")
				notExistedOcmroleocmRoleArn = "arn:aws:iam::301721915996:role/notexistuserrolearn"
				output, err = ocmResourceService.UnlinkOCMRole("--role-arn", notExistedOcmroleocmRoleArn, "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("is not linked with the organization account"))

				By("Unlink ocm-role with the role arn in incorrect format")
				ocmroleArnInWrongFormat = "arn301721915996:rolenotexistuserrolearn"
				output, err = ocmResourceService.UnlinkOCMRole("--role-arn", ocmroleArnInWrongFormat, "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Expected a valid ocm role ARN to unlink from the current organization"))

				By("Unlink ocm-role")
				output, err = ocmResourceService.UnlinkOCMRole("--role-arn", foundOcmrole.RoleArn, "-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Successfully unlinked role"))

				By("Get the ocm-role info")
				ocmRoleList, output, err = ocmResourceService.ListOCMRole()
				Expect(output).ToNot(BeNil())
				Expect(err).To(BeNil())
				foundOcmrole = ocmRoleList.OCMRole(ocmrolePrefix, ocmOrganizationExternalID)
				Expect(foundOcmrole.Linded).To(Equal("No"))

				By("Link ocm-role with the role arn in incorrect format")
				output, err = ocmResourceService.LinkOCMRole("--role-arn", ocmroleArnInWrongFormat, "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Expected a valid ocm role ARN to link to a current organization"))

				By("Delete the role and keep the attached policy then create again")
				awsClient, err := aws_client.CreateAWSClient("", "")
				Expect(err).To(BeNil())
				err = awsClient.DetachRolePolicies(foundOcmrole.RoleName)
				Expect(err).To(BeNil())
				err = awsClient.DeleteRole(foundOcmrole.RoleName)
				Expect(err).To(BeNil())

				output, err = ocmResourceService.CreateOCMRole("--mode", "auto",
					"--prefix", ocmrolePrefix,
					"--path", path,
					"--admin",
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Created role"))
				Expect(textData).Should(ContainSubstring("Successfully linked role"))

				attachedPolicies, err := awsClient.ListAttachedRolePolicies(foundOcmrole.RoleName)
				Expect(err).To(BeNil())
				Expect(len(attachedPolicies)).To(Equal(2))
			})

	})
