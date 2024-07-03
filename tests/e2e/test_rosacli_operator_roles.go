package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit operator roles", labels.Feature.OperatorRoles, func() {
	defer GinkgoRecover()

	var (
		operatorRolePrefixedNeedCleanup = make([]string, 0)

		rosaClient             *rosacli.Client
		ocmResourceService     rosacli.OCMResourceService
		permissionsBoundaryArn string = "arn:aws:iam::aws:policy/AdministratorAccess"
	)
	BeforeEach(func() {
		By("Init the client")
		rosaClient = rosacli.NewClient()
		ocmResourceService = rosaClient.OCMResource
	})

	Describe("on cluster", func() {
		var (
			clusterID string
		)
		BeforeEach(func() {
			By("Get the cluster id")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")
		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("can validate when user create operator-roles to cluster - [id:43051]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Check if cluster is sts cluster")
				clusterService := rosaClient.Cluster
				StsCluster, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).To(BeNil())

				By("Check if cluster is using reusable oidc config")
				notExistedClusterID := "notexistedclusterid111"

				switch StsCluster {
				case true:
					By("Create operator-roles on sts cluster which status is not pending")
					output, err := ocmResourceService.CreateOperatorRoles(
						"--mode", "auto",
						"-c", clusterID,
						"-y")
					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("Operator Roles already exists"))
				case false:
					By("Create operator-roles on classic non-sts cluster")
					output, err := ocmResourceService.CreateOIDCProvider(
						"--mode", "auto",
						"-c", clusterID,
						"-y")
					Expect(err).NotTo(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("is not an STS cluster"))
				}
				By("Create operator-roles on not-existed cluster")
				output, err := ocmResourceService.CreateOIDCProvider(
					"--mode", "auto",
					"-c", notExistedClusterID,
					"-y")
				Expect(err).NotTo(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("There is no cluster with identifier or name"))
			})

		It("to validate operator roles and oidc provider will work well - [id:70859]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Check cluster is sts cluster")
				clusterService := rosaClient.Cluster
				isSTS, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				By("Check the cluster is using reusable oIDCConfig")
				IsUsingReusableOIDCConfig, err := clusterService.IsUsingReusableOIDCConfig(clusterID)
				Expect(err).ToNot(HaveOccurred())

				if isSTS && IsUsingReusableOIDCConfig {
					By("Create operator roles to the cluster again")
					output, err := ocmResourceService.CreateOperatorRoles("-c", clusterID, "-y", "--mode", "auto")
					Expect(err).ToNot(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(
							"Operator Roles already exists"))

					By("Create oidc config to the cluster again")
					output, err = ocmResourceService.CreateOIDCProvider("-c", clusterID, "-y", "--mode", "auto")
					Expect(err).ToNot(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(
							"OIDC provider already exists"))

					By("Delete the oidc-provider to the cluster")
					output, err = ocmResourceService.DeleteOIDCProvider("-c", clusterID, "-y", "--mode", "auto")
					Expect(err).To(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(
							"ERR: Cluster '%s' is in 'ready' state. OIDC provider can be deleted only for the uninstalled clusters",
							clusterID))

					By("Delete the operator-roles to the cluster")
					output, err = ocmResourceService.DeleteOperatorRoles("-c", clusterID, "-y", "--mode", "auto")
					Expect(err).To(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(
							"ERR: Cluster '%s' is in 'ready' state. Operator roles can be deleted only for the uninstalled clusters",
							clusterID))

					By("Get the --oidc-config-id from the cluster and it's issuer url")
					rosaClient.Runner.JsonFormat()
					jsonOutput, err := clusterService.DescribeCluster(clusterID)
					Expect(err).To(BeNil())
					rosaClient.Runner.UnsetFormat()
					jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()
					oidcConfigID := jsonData.DigString("aws", "sts", "oidc_config", "id")
					issuerURL := jsonData.DigString("aws", "sts", "oidc_config", "issuer_url")

					By("Try to delete oidc provider with --oidc-config-id")
					output, err = ocmResourceService.DeleteOIDCProvider("--oidc-config-id", oidcConfigID, "-y", "--mode", "auto")
					Expect(err).To(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(
							"ERR: There are clusters using OIDC config '%s', can't delete the provider",
							issuerURL))

					By("Try to create oidc provider with --oidc-config-id")
					output, err = ocmResourceService.CreateOIDCProvider("--oidc-config-id", oidcConfigID, "-y", "--mode", "auto")
					Expect(err).ToNot(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(
							"OIDC provider already exists"))

					By("Try to create operator-roles with --oic-config-id and cluster id at the same time")
					output, err = ocmResourceService.CreateOperatorRoles("-c", clusterID, "--oidc-config-id", oidcConfigID)
					Expect(err).To(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
						To(ContainSubstring(
							"ERR: A cluster key for STS cluster and an OIDC configuration ID" +
								" cannot be specified alongside each other."))
				}
			})
	})

	It("can create operator-roles prior to cluster creation - [id:60971]",
		labels.High, labels.Runtime.OCMResources,
		func() {
			defer func() {
				By("Cleanup created operator-roles in high level of the test case")
				if len(operatorRolePrefixedNeedCleanup) > 0 {
					for _, v := range operatorRolePrefixedNeedCleanup {
						_, err := ocmResourceService.DeleteOperatorRoles(
							"--prefix", v,
							"--mode", "auto",
							"-y",
						)
						Expect(err).To(BeNil())
					}
				}
			}()

			var (
				oidcPrivodeIDFromOutputMessage  string
				oidcPrivodeARNFromOutputMessage string
				notExistedOIDCConfigID          = "asdasdfsdfsdf"
				invalidInstallerRole            = "arn:/qeci-default-accountroles-Installer-Role"
				notExistedInstallerRole         = "arn:aws:iam::301721915996:role/notexisted-accountroles-Installer-Role"
				hostedCPOperatorRolesPrefix     = "hopp60971"
				classicSTSOperatorRolesPrefix   = "sopp60971"
				managedOIDCConfigID             string
				hostedCPInstallerRoleArn        string
				classicInstallerRoleArn         string
				accountRolePrefix               string
			)

			listOperatorRoles := func(prefix string) (rosacli.OperatorRoleList, error) {
				var operatorRoleList rosacli.OperatorRoleList
				output, err := ocmResourceService.ListOperatorRoles(
					"--prefix", prefix,
				)
				if err != nil {
					return operatorRoleList, err
				}
				operatorRoleList, err = ocmResourceService.ReflectOperatorRoleList(output)
				return operatorRoleList, err
			}

			By("Create account-roles for testing")
			accountRolePrefix = fmt.Sprintf("QEAuto-accr60971-%s", time.Now().UTC().Format("20060102"))
			_, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accountRolePrefix,
				"--permissions-boundary", permissionsBoundaryArn,
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
			classicInstallerRoleArn = accountRoleList.InstallerRole(accountRolePrefix, false).RoleArn
			hostedCPInstallerRoleArn = accountRoleList.InstallerRole(accountRolePrefix, true).RoleArn

			By("Create managed oidc-config in auto mode")
			output, err := ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
			Expect(err).To(BeNil())
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Created OIDC provider with ARN"))
			oidcPrivodeARNFromOutputMessage = common.ExtractOIDCProviderARN(output.String())
			oidcPrivodeIDFromOutputMessage = common.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)

			managedOIDCConfigID, err = ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
			Expect(err).To(BeNil())
			defer func() {
				output, err := ocmResourceService.DeleteOIDCConfig(
					"--oidc-config-id", managedOIDCConfigID,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the OIDC provider"))
			}()
			By("Create hosted-cp and classic sts Operator-roles pror to cluster spec")
			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
				"--installer-role-arn", classicInstallerRoleArn,
				"--mode", "auto",
				"--prefix", classicSTSOperatorRolesPrefix,
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Created role"))
			operatorRolePrefixedNeedCleanup = append(operatorRolePrefixedNeedCleanup, classicSTSOperatorRolesPrefix)

			defer func() {
				output, err := ocmResourceService.DeleteOperatorRoles(
					"--prefix", classicSTSOperatorRolesPrefix,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the operator roles"))

			}()

			roles, err := listOperatorRoles(classicSTSOperatorRolesPrefix)
			Expect(err).To(BeNil())
			Expect(len(roles.OperatorRoleList)).To(Equal(6))

			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
				"--installer-role-arn", hostedCPInstallerRoleArn,
				"--mode", "auto",
				"--prefix", hostedCPOperatorRolesPrefix,
				"--hosted-cp",
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Created role"))
			operatorRolePrefixedNeedCleanup = append(operatorRolePrefixedNeedCleanup, hostedCPOperatorRolesPrefix)

			roles, err = listOperatorRoles(hostedCPOperatorRolesPrefix)
			Expect(err).To(BeNil())
			Expect(len(roles.OperatorRoleList)).To(Equal(8))
			defer func() {
				output, err := ocmResourceService.DeleteOperatorRoles(
					"--prefix", hostedCPOperatorRolesPrefix,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the operator roles"))
			}()

			By("Create operator roles with not-existed role")
			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
				"--installer-role-arn", notExistedInstallerRole,
				"--mode", "auto",
				"--prefix", classicSTSOperatorRolesPrefix,
				"-y",
			)
			Expect(err).To(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("cannot be found"))

			By("Create operator roles with role arn in incorrect format")
			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
				"--installer-role-arn", invalidInstallerRole,
				"--mode", "auto",
				"--prefix", classicSTSOperatorRolesPrefix,
				"-y",
			)
			Expect(err).To(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Invalid ARN"))

			By("Create operator roles with not-existed oidc id")
			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", notExistedOIDCConfigID,
				"--installer-role-arn", classicInstallerRoleArn,
				"--mode", "auto",
				"--prefix", classicSTSOperatorRolesPrefix,
				"-y",
			)
			Expect(err).To(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("not found"))

			By("Create operator-role without setting oidc-config-id")
			output, err = ocmResourceService.CreateOperatorRoles(
				"--installer-role-arn", classicInstallerRoleArn,
				"--mode", "auto",
				"--prefix", hostedCPOperatorRolesPrefix,
				"--hosted-cp",
				"-y",
			)
			Expect(err).To(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("oidc-config-id is mandatory for prefix param flow"))

			By("Create operator-role without setting installer-role-arn")
			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
				"--mode", "auto",
				"--prefix", hostedCPOperatorRolesPrefix,
				"--hosted-cp",
				"-y",
			)
			Expect(err).To(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("role-arn is mandatory for prefix param flow"))

			By("Create operator-role without setting id neither prefix")
			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
				"--installer-role-arn", classicInstallerRoleArn,
				"--mode", "auto",
				"--hosted-cp",
				"-y",
			)
			Expect(err).To(HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).
				Should(ContainSubstring(
					"Either a cluster key for STS cluster or an operator roles prefix must be specified"))
		})

})
