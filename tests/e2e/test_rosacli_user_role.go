package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit user role", labels.Feature.UserRole, func() {
	defer GinkgoRecover()

	var (
		rosaClient *rosacli.Client

		ocmResourceService     rosacli.OCMResourceService
		permissionsBoundaryArn string = "arn:aws:iam::aws:policy/AdministratorAccess"
	)
	BeforeEach(func() {
		By("Init the client")
		rosaClient = rosacli.NewClient()
		ocmResourceService = rosaClient.OCMResource
	})

	It("can validate create/link/unlink user-role - [id:52580]",
		labels.High, labels.Runtime.OCMResources,
		func() {
			var (
				userRolePrefix                                string
				invalidPermisionBoundary                      string
				notExistedPermissionBoundaryUnderDifferentAWS string
				ocmAccountUsername                            string
				notExistedUserRoleArn                         string
				userRoleArnInWrongFormat                      string
				foundUserRole                                 rosacli.UserRole
			)
			rosaClient.Runner.JsonFormat()
			whoamiOutput, err := ocmResourceService.Whoami()
			Expect(err).To(BeNil())
			rosaClient.Runner.UnsetFormat()
			whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
			ocmAccountUsername = whoamiData.OCMAccountUsername
			userRolePrefix = fmt.Sprintf("QEAuto-user-%s-OCP-52580", time.Now().UTC().Format("20060102"))

			By("Create an user-role with invalid mode")
			output, err := ocmResourceService.CreateUserRole("--mode", "invalidamode",
				"--prefix", userRolePrefix,
				"-y")
			Expect(err).NotTo(BeNil())
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Invalid mode. Allowed values are [auto manual]"))

			By("Create an user-role with invalid permision boundady")
			invalidPermisionBoundary = "arn-permission-boundary"
			output, err = ocmResourceService.CreateUserRole("--mode", "auto",
				"--permissions-boundary", invalidPermisionBoundary,
				"--prefix", userRolePrefix,
				"-y")
			Expect(err).NotTo(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Expected a valid policy ARN for permissions boundary"))

			By("Create an user-role with the permision boundady under another aws account")
			notExistedPermissionBoundaryUnderDifferentAWS = "arn:aws:iam::aws:policy/notexisted"
			output, err = ocmResourceService.CreateUserRole("--mode", "auto",
				"--permissions-boundary", notExistedPermissionBoundaryUnderDifferentAWS,
				"--prefix", userRolePrefix,
				"-y")
			Expect(err).NotTo(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).
				Should(ContainSubstring(
					"There was an error creating the ocm user role: operation error IAM: CreateRole"))
			Expect(textData).Should(ContainSubstring("api error NoSuchEntity"))

			By("Create an user-role")
			output, err = ocmResourceService.CreateUserRole("--mode", "auto",
				"--prefix", userRolePrefix,
				"-y")
			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Created role"))
			Expect(textData).Should(ContainSubstring("Successfully linked role"))

			By("Get the user-role info")
			userRoleList, output, err := ocmResourceService.ListUserRole()
			Expect(err).To(BeNil())
			foundUserRole = userRoleList.UserRole(userRolePrefix, ocmAccountUsername)
			Expect(foundUserRole).ToNot(BeNil())

			defer func() {
				By("Delete user-role")
				output, err = ocmResourceService.DeleteUserRole("--mode", "auto",
					"--role-arn", foundUserRole.RoleArn,
					"-y")

				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Successfully deleted the user role"))
			}()

			By("Unlink user-role with not-exist role")
			notExistedUserRoleArn = "arn:aws:iam::301721915996:role/notexistuserrolearn"
			output, err = ocmResourceService.UnlinkUserRole("--role-arn", notExistedUserRoleArn, "-y")
			Expect(err).NotTo(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("is not linked with the current account"))

			By("Unlink user-role with the role arn in incorrect format")
			userRoleArnInWrongFormat = "arn301721915996:rolenotexistuserrolearn"
			output, err = ocmResourceService.UnlinkUserRole("--role-arn", userRoleArnInWrongFormat, "-y")
			Expect(err).NotTo(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).
				Should(ContainSubstring(
					"Expected a valid user role ARN to unlink from the current account"))

			By("Unlink user-role")
			output, err = ocmResourceService.UnlinkUserRole("--role-arn", foundUserRole.RoleArn, "-y")
			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Successfully unlinked role"))

			By("Get the user-role info")
			userRoleList, output, err = ocmResourceService.ListUserRole()
			Expect(err).To(BeNil())
			foundUserRole = userRoleList.UserRole(userRolePrefix, ocmAccountUsername)
			Expect(foundUserRole.Linded).To(Equal("No"))

			By("Link user-role with the role arn in incorrect format")
			output, err = ocmResourceService.LinkUserRole("--role-arn", userRoleArnInWrongFormat, "-y")
			Expect(err).NotTo(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Expected a valid user role ARN to link to a current account"))
		})

	It("can create/link/unlink/delete user-role in auto mode - [id:52419]",
		labels.High, labels.Runtime.OCMResources,
		func() {
			var (
				userrolePrefix     string
				ocmAccountUsername string
				foundUserRole      rosacli.UserRole

				path = "/aa/bb/"
			)

			rosaClient.Runner.JsonFormat()
			whoamiOutput, err := ocmResourceService.Whoami()
			Expect(err).To(BeNil())
			rosaClient.Runner.UnsetFormat()
			whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
			ocmAccountUsername = whoamiData.OCMAccountUsername
			userrolePrefix = fmt.Sprintf("QEAuto-userr-%s-52419", time.Now().UTC().Format("20060102"))

			By("Create an user-role")
			output, err := ocmResourceService.CreateUserRole(
				"--mode", "auto",
				"--prefix", userrolePrefix,
				"--path", path,
				"--permissions-boundary", permissionsBoundaryArn,
				"-y")
			Expect(err).To(BeNil())
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Created role"))
			Expect(textData).Should(ContainSubstring("Successfully linked role"))
			defer func() {
				By("Delete user-role")
				output, err = ocmResourceService.DeleteUserRole("--mode", "auto",
					"--role-arn", foundUserRole.RoleArn,
					"-y")

				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Successfully deleted the user role"))
			}()

			By("Get the ocm-role info")
			userRoleList, output, err := ocmResourceService.ListUserRole()
			Expect(err).To(BeNil())
			foundUserRole = userRoleList.UserRole(userrolePrefix, ocmAccountUsername)
			Expect(foundUserRole).ToNot(BeNil())

			By("Get the user-role info")
			userRoleList, output, err = ocmResourceService.ListUserRole()
			Expect(err).To(BeNil())
			foundUserRole = userRoleList.UserRole(userrolePrefix, ocmAccountUsername)
			Expect(foundUserRole.Linded).To(Equal("Yes"))

			By("Unlink user-role")
			output, err = ocmResourceService.UnlinkUserRole("--role-arn", foundUserRole.RoleArn, "-y")
			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Successfully unlinked role"))

			By("Get the user-role info")
			userRoleList, output, err = ocmResourceService.ListUserRole()
			Expect(err).To(BeNil())
			foundUserRole = userRoleList.UserRole(userrolePrefix, ocmAccountUsername)
			Expect(foundUserRole.Linded).To(Equal("No"))
		})
})
