package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("Edit operator roles", labels.Feature.OperatorRoles, func() {
	defer GinkgoRecover()

	var (
		operatorRolePrefixedNeedCleanup = make([]string, 0)

		rosaClient             *rosacli.Client
		ocmResourceService     rosacli.OCMResourceService
		permissionsBoundaryArn string = "arn:aws:iam::aws:policy/AdministratorAccess"
		clusterConfig          *config.ClusterConfig
		err                    error
	)
	BeforeEach(func() {
		By("Init the client")
		rosaClient = rosacli.NewClient()
		ocmResourceService = rosaClient.OCMResource
	})

	Describe("on cluster", func() {
		var (
			clusterID  string
			defaultDir string
			dirToClean string
		)
		BeforeEach(func() {
			By("Get the cluster id")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Get the default dir")
			defaultDir = rosaClient.Runner.GetDir()

			By("Get cluster config")
			clusterConfig, err = config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			By("Go back original by setting runner dir")
			rosaClient.Runner.SetDir(defaultDir)

			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})
		It("to delete in-used operator-roles and byo oidc-config  [id:74761]",
			labels.Critical, labels.Runtime.Day2, func() {
				By("Check if the cluster is using BYO oidc config")
				profile := profilehandler.LoadProfileYamlFileByENV()
				if profile.ClusterConfig.OIDCConfig == "" {
					SkipTestOnFeature("This testing only work for byo oidc cluster")
				}

				By("Describe cluster")
				clusterService := rosaClient.Cluster
				rosaClient.Runner.JsonFormat()
				jsonOutput, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()
				oidcConfigID := jsonData.DigString("aws", "sts", "oidc_config", "id")
				operatorRolePrefix := jsonData.DigString("aws", "sts", "operator_role_prefix")

				By("Delete in-used operator roles by prefix in auto mode")
				output, err := ocmResourceService.DeleteOperatorRoles(
					"--prefix", operatorRolePrefix,
					"-y",
					"--mode", "auto",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("There are clusters using Operator Roles Prefix"))

				By("Create a temp dir to execute the create commands")
				dirToClean, err = os.MkdirTemp("", "*")
				Expect(err).To(BeNil())

				By("Delete in-used operator roles by prefix in manual mode")
				rosaClient.Runner.SetDir(dirToClean)
				output, err = ocmResourceService.DeleteOperatorRoles(
					"--prefix", operatorRolePrefix,
					"-y",
					"--mode", "manual",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("There are clusters using Operator Roles Prefix"))

				By("Delete in-used oidc config in auto mode")
				output, err = ocmResourceService.DeleteOIDCConfig(
					"--oidc-config-id", oidcConfigID,
					"--region", clusterConfig.Region,
					"--mode", "auto",
					"--region", clusterConfig.Region,
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("There are clusters using OIDC config"))

				By("Delete in-used oidc config in manual mode")
				output, err = ocmResourceService.DeleteOIDCConfig(
					"--oidc-config-id", oidcConfigID,
					"--mode", "manual",
					"--region", clusterConfig.Region,
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("There are clusters using OIDC config"))
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
			oidcPrivodeARNFromOutputMessage = helper.ExtractOIDCProviderARN(output.String())
			oidcPrivodeIDFromOutputMessage = helper.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)

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
			Expect(textData).To(ContainSubstring("Attached trust policy"))
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
			Expect(textData).Should(ContainSubstring("to be a valid IAM role ARN"))

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

var _ = Describe("create operator-roles forcely testing",
	func() {
		defer GinkgoRecover()
		var (
			rosaClient         *rosacli.Client
			ocmResourceService rosacli.OCMResourceService
			awsClient          *aws_client.AWSClient
			err                error
			clusterService     rosacli.ClusterService
			clusterID          string
		)
		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
			clusterService = rosaClient.Cluster

			awsClient, err = aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())

			By("Get cluster id")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")
		})

		It("to create operator-roles which were created with cluster forcely - [id:74661]",
			labels.Critical, labels.Runtime.Destructive, func() {
				By("Check cluster type")
				isSTS, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).To(BeNil())
				isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).To(BeNil())

				if !isSTS || isHostedCP {
					Skip("Skip this case as this case is only for classic STS cluster")
				}
				operatorRoleNamePermissionMap := map[string]string{
					"openshift-image-registry-installer": "GetObject",
					"openshift-ingress-operator-cloud":   "ListHostedZones",
					"openshift-cluster-csi-drivers-ebs":  "CreateTags",
					"openshift-cloud-network-config":     "DescribeInstanceTypes",
					"openshift-machine-api-aws-cloud":    "DescribeImages",
					"openshift-cloud-credential":         "GetUserPolicy",
				}
				By("Get operator-roles arn and name")
				rolePolicyMap := make(map[string]string)
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				operatorRolesArns := CD.OperatorIAMRoles
				for _, policyArn := range operatorRolesArns {
					_, operatorRoleName, err := helper.ParseRoleARN(policyArn)
					Expect(err).To(BeNil())
					attachedPolicy, err := awsClient.ListAttachedRolePolicies(operatorRoleName)
					Expect(err).To(BeNil())
					rolePolicyMap[operatorRoleName] = *attachedPolicy[0].PolicyArn
				}

				By("Update opertor-roles policies permission")
				updatedPolicyDocument := `{
					"Version": "2012-10-17",
					"Statement": [
						{
							"Action": [
								"autoscaling:DescribeAutoScalingGroups"
							],
							"Effect": "Allow",
							"Resource": "*"
						}
					]
				}`
				for _, policyArn := range rolePolicyMap {
					_, err = awsClient.IamClient.CreatePolicyVersion(context.Background(), &iam.CreatePolicyVersionInput{
						PolicyArn:      aws.String(policyArn),
						PolicyDocument: aws.String(updatedPolicyDocument),
						SetAsDefault:   true,
					})
					Expect(err).To(BeNil())
				}

				By("Detach and Delete one operator-role policy,openshift-cluster-csi-drivers-ebs")
				var deletingPolicyRoleName string
				for roleName := range rolePolicyMap {
					if strings.Contains(roleName, "openshift-cluster-csi-driver") {
						deletingPolicyRoleName = roleName
						break
					}
				}
				err = awsClient.DetachRolePolicies(deletingPolicyRoleName)
				Expect(err).To(BeNil())
				err = awsClient.DeleteIAMPolicy(rolePolicyMap[deletingPolicyRoleName])
				Expect(err).To(BeNil())

				By("Create operator-role forcefully")
				output, err = ocmResourceService.CreateOperatorRoles(
					"-c", clusterID,
					"--mode", "auto",
					"--force-policy-creation",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("Created role"))
				Expect(output.String()).Should(ContainSubstring("Attached trust policy to role"))
				Expect(output.String()).Should(ContainSubstring("Attached policy 'arn:aws:iam::"))

				By("Check the operator role policies are regenerated")

				for roleName, policy := range rolePolicyMap {
					policy, err := awsClient.GetIAMPolicy(policy)
					Expect(err).To(BeNil())
					policyVersion, err := awsClient.IamClient.GetPolicyVersion(context.TODO(), &iam.GetPolicyVersionInput{
						PolicyArn: aws.String(*policy.Arn),
						VersionId: policy.DefaultVersionId,
					})
					Expect(err).To(BeNil())
					for rolSubName, permission := range operatorRoleNamePermissionMap {
						if strings.Contains(roleName, rolSubName) {
							Expect(*policyVersion.PolicyVersion.Document).To(ContainSubstring(permission))
						}
					}
				}
			})
	})
var _ = Describe("create IAM roles forcely testing",
	func() {
		defer GinkgoRecover()
		var (
			rosaClient          *rosacli.Client
			ocmResourceService  rosacli.OCMResourceService
			awsClient           *aws_client.AWSClient
			accountRolePrefix   string
			operatorRolePrefix  string
			managedOIDCConfigID string
			installerRoleArn    string
		)
		BeforeEach(func() {
			var err error

			By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource

			awsClient, err = aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())

			By("Create oidconfig for testing")
			var output bytes.Buffer
			output, err = ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("Created OIDC provider with ARN"))
			oidcPrivodeARNFromOutputMessage := helper.ExtractOIDCProviderARN(output.String())
			oidcPrivodeIDFromOutputMessage := helper.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)

			managedOIDCConfigID, err = ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			By("Delete the testing account-roles")
			_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", accountRolePrefix,
				"-y",
			)
			Expect(err).To(BeNil())

			By("Delete testing operator-roles")
			output, err := ocmResourceService.DeleteOperatorRoles(
				"--prefix", operatorRolePrefix,
				"--mode", "auto",
				"-y",
			)
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("Successfully deleted the operator roles"))

			By("Detete testing oidc condig")
			if managedOIDCConfigID != "" {
				output, err := ocmResourceService.DeleteOIDCConfig(
					"--oidc-config-id", managedOIDCConfigID,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Successfully deleted the OIDC provider"))
			}
		})
		It("to create account-roles and prior-to-cluster operator-roles forcely - [id:59551]",
			labels.Critical, labels.Runtime.OCMResources, func() {
				accountRolePrefix = helper.GenerateRandomName("ar59551", 2)
				operatorRolePrefix = helper.GenerateRandomName("op59551", 2)
				accountRoleNamePermissionMap := map[string]string{
					fmt.Sprintf("%s-Installer-Role", accountRolePrefix):    "AssumeRole",
					fmt.Sprintf("%s-Support-Role", accountRolePrefix):      "DescribeInstances",
					fmt.Sprintf("%s-Worker-Role", accountRolePrefix):       "DescribeRegions",
					fmt.Sprintf("%s-ControlPlane-Role", accountRolePrefix): "DescribeAvailabilityZones",
				}

				operatorRoleNamePermissionMap := map[string]string{
					fmt.Sprintf("%s-openshift-image-registry-installer-cloud-credentials", operatorRolePrefix):  "GetObject",
					fmt.Sprintf("%s-openshift-ingress-operator-cloud-credentials", operatorRolePrefix):          "ListHostedZones",
					fmt.Sprintf("%s-openshift-cluster-csi-drivers-ebs-cloud-credentials", operatorRolePrefix):   "CreateTags",
					fmt.Sprintf("%s-openshift-cloud-network-config-controller-cloud-crede", operatorRolePrefix): "DescribeSubnets",
					fmt.Sprintf("%s-openshift-machine-api-aws-cloud-credentials", operatorRolePrefix):           "DescribeImages",
					fmt.Sprintf("%s-openshift-cloud-credential-operator-cloud-credential-", operatorRolePrefix): "GetUserPolicy",
				}

				accountRolePolicyMap := map[string]string{}
				operatorRolePolicyMap := map[string]string{}

				By("Create account-role")
				output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Created role"))

				By("Create prior-to-cluster operator roles for testing")
				installerRole, err := awsClient.GetRole(fmt.Sprintf("%s-Installer-Role", accountRolePrefix))
				Expect(err).To(BeNil())
				installerRoleArn = *installerRole.Arn

				output, err = ocmResourceService.CreateOperatorRoles(
					"--oidc-config-id", managedOIDCConfigID,
					"--installer-role-arn", installerRoleArn,
					"--mode", "auto",
					"--prefix", operatorRolePrefix,
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("Created role"))
				Expect(output.String()).Should(ContainSubstring("Attached trust policy to role"))
				Expect(output.String()).Should(ContainSubstring("Attached policy 'arn:aws:iam::"))

				By("Get test policies")
				for k := range accountRoleNamePermissionMap {
					attachedPolicy, err := awsClient.ListAttachedRolePolicies(k)
					Expect(err).To(BeNil())
					accountRolePolicyMap[k] = *attachedPolicy[0].PolicyArn
				}

				for k := range operatorRoleNamePermissionMap {
					attachedPolicy, err := awsClient.ListAttachedRolePolicies(k)
					Expect(err).To(BeNil())
					operatorRolePolicyMap[k] = *attachedPolicy[0].PolicyArn
				}

				By("Update account-role policy permission")
				updatedPolicyDocument := `{
					"Version": "2012-10-17",
					"Statement": [
						{
							"Action": [
								"autoscaling:DescribeAutoScalingGroups"
							],
							"Effect": "Allow",
							"Resource": "*"
						}
					]
				}`
				for _, policyArn := range accountRolePolicyMap {
					_, err = awsClient.IamClient.CreatePolicyVersion(context.Background(), &iam.CreatePolicyVersionInput{
						PolicyArn:      aws.String(policyArn),
						PolicyDocument: aws.String(updatedPolicyDocument),
						SetAsDefault:   true,
					})
					Expect(err).To(BeNil())
				}

				By("Detach and Delete Support-Role policy")
				supportRoleName := fmt.Sprintf("%s-Support-Role", accountRolePrefix)
				err = awsClient.DetachRolePolicies(supportRoleName)
				Expect(err).To(BeNil())
				err = awsClient.DeleteIAMPolicy(accountRolePolicyMap[supportRoleName])
				Expect(err).To(BeNil())

				By("Create account-role forcely")
				output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y",
					"--force-policy-creation",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Created role"))

				By("Check the account role policies are regenerated")
				for roleName := range accountRoleNamePermissionMap {
					policy, err := awsClient.GetIAMPolicy(accountRolePolicyMap[roleName])
					Expect(err).To(BeNil())
					policyVersion, err := awsClient.IamClient.GetPolicyVersion(context.TODO(), &iam.GetPolicyVersionInput{
						PolicyArn: aws.String(*policy.Arn),
						VersionId: policy.DefaultVersionId,
					})
					Expect(err).To(BeNil())
					Expect(*policyVersion.PolicyVersion.Document).To(ContainSubstring(accountRoleNamePermissionMap[roleName]))
				}
				By("Update operator-role policy permission")
				for _, policyArn := range operatorRolePolicyMap {
					_, err = awsClient.IamClient.CreatePolicyVersion(context.Background(), &iam.CreatePolicyVersionInput{
						PolicyArn:      aws.String(policyArn),
						PolicyDocument: aws.String(updatedPolicyDocument),
						SetAsDefault:   true,
					})
					Expect(err).To(BeNil())
				}

				By("Detach and Delete one operator-role policy")
				policyToDel := fmt.Sprintf("%s-openshift-cloud-network-config-controller-cloud-crede", operatorRolePrefix)
				err = awsClient.DetachRolePolicies(policyToDel)
				Expect(err).To(BeNil())
				err = awsClient.DeleteIAMPolicy(operatorRolePolicyMap[policyToDel])
				Expect(err).To(BeNil())

				By("Create operator-role forcely")
				output, err = ocmResourceService.CreateOperatorRoles(
					"--oidc-config-id", managedOIDCConfigID,
					"--installer-role-arn", installerRoleArn,
					"--mode", "auto",
					"--prefix", operatorRolePrefix,
					"--force-policy-creation",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("Created role"))

				By("Check the operator role policies are regenerated")
				for roleName := range operatorRoleNamePermissionMap {
					policy, err := awsClient.GetIAMPolicy(operatorRolePolicyMap[roleName])
					Expect(err).To(BeNil())
					policyVersion, err := awsClient.IamClient.GetPolicyVersion(context.TODO(), &iam.GetPolicyVersionInput{
						PolicyArn: aws.String(*policy.Arn),
						VersionId: policy.DefaultVersionId,
					})
					Expect(err).To(BeNil())
					Expect(*policyVersion.PolicyVersion.Document).To(ContainSubstring(operatorRoleNamePermissionMap[roleName]))
				}
			})
	})
var _ = Describe("Detele operator roles with byo oidc", labels.Feature.OperatorRoles, func() {
	defer GinkgoRecover()
	var (
		rosaClient          *rosacli.Client
		ocmResourceService  rosacli.OCMResourceService
		awsClient           *aws_client.AWSClient
		err                 error
		accountRolePrefix   string
		operatorRolePrefixC string
		operatorRolePrefixH string
		managedOIDCConfigID string

		installerRoleArnC string
		installerRoleArnH string

		defaultDir string
		dirToClean string
	)
	BeforeEach(func() {
		By("Init the client")
		rosaClient = rosacli.NewClient()
		ocmResourceService = rosaClient.OCMResource

		awsClient, err = aws_client.CreateAWSClient("", "")
		Expect(err).To(BeNil())

		By("Get the default dir")
		defaultDir = rosaClient.Runner.GetDir()

	})
	AfterEach(func() {

		By("Delete testing operator-roles")
		_, err = ocmResourceService.DeleteOperatorRoles(
			"--prefix", operatorRolePrefixC,
			"--mode", "auto",
			"-y",
		)
		Expect(err).To(BeNil())

		_, err = ocmResourceService.DeleteOperatorRoles(
			"--prefix", operatorRolePrefixH,
			"--mode", "auto",
			"-y",
		)
		Expect(err).To(BeNil())

		By("Detete testing oidc condig")
		output, err := ocmResourceService.DeleteOIDCConfig(
			"--oidc-config-id", managedOIDCConfigID,
			"--mode", "manual",
			"-y",
		)
		Expect(err).To(BeNil())
		commands := helper.ExtractCommandsToDeleteAWSResoueces(output)
		for k, v := range commands {
			fmt.Printf("the %d command is %s\n", k, v)
		}
		By("Delete the testing account-roles")
		_, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
			"--prefix", accountRolePrefix,
			"-y",
		)
		Expect(err).To(BeNil())

		By("Go back original by setting runner dir")
		rosaClient.Runner.SetDir(defaultDir)
	})
	It("to delete operator-roles and byo oidc-config in manual mode - [id:60956]",
		labels.Critical, labels.Runtime.OCMResources, func() {
			By("Create account-roles")
			accountRolePrefix = helper.GenerateRandomName("arp60956", 2)
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accountRolePrefix,
				"-y",
			)
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("Created role"))
			installerRoleC, err := awsClient.GetRole(fmt.Sprintf("%s-Installer-Role", accountRolePrefix))
			Expect(err).To(BeNil())
			installerRoleArnC = *installerRoleC.Arn

			installerRoleH, err := awsClient.GetRole(fmt.Sprintf("%s-HCP-ROSA-Installer-Role", accountRolePrefix))
			Expect(err).To(BeNil())
			installerRoleArnH = *installerRoleH.Arn

			By("Create oidc-config")
			output, err = ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
			Expect(err).To(BeNil())
			Expect(output.String()).To(ContainSubstring("Created OIDC provider with ARN"))
			oidcPrivodeARNFromOutputMessage := helper.ExtractOIDCProviderARN(output.String())
			oidcPrivodeIDFromOutputMessage := helper.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)

			managedOIDCConfigID, err = ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
			Expect(err).To(BeNil())

			By("Create hosted-cp operator-roles")
			operatorRolePrefixH = helper.GenerateRandomName("opp60956h", 2)
			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", managedOIDCConfigID,
				"--installer-role-arn", installerRoleArnH,
				"--mode", "auto",
				"--prefix", operatorRolePrefixH,
				"--hosted-cp",
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).Should(ContainSubstring("Created role"))

			By("Create a temp dir to execute the create commands")
			dirToClean, err = os.MkdirTemp("", "*")
			Expect(err).To(BeNil())

			By("Delete the hosted-cp operator-roles by prefix in manual mode")
			rosaClient.Runner.SetDir(dirToClean)
			output, err = ocmResourceService.DeleteOperatorRoles("--prefix", operatorRolePrefixH, "-y", "--mode", "manual")
			Expect(err).NotTo(HaveOccurred())
			commands := helper.ExtractCommandsToDeleteAWSResoueces(output)
			for _, command := range commands {
				_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
				Expect(err).To(BeNil())
			}

			By("Create classic operator-roles")
			operatorRolePrefixC = helper.GenerateRandomName("opp60956c", 2)
			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", managedOIDCConfigID,
				"--installer-role-arn", installerRoleArnC,
				"--mode", "auto",
				"--prefix", operatorRolePrefixC,
				"-y",
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).Should(ContainSubstring("Created role"))

			By("Delete the classic operator-roles by prefix in manual mode")
			output, err = ocmResourceService.DeleteOperatorRoles("--prefix", operatorRolePrefixC, "-y", "--mode", "manual")
			Expect(err).NotTo(HaveOccurred())
			commands = helper.ExtractCommandsToDeleteAWSResoueces(output)
			for _, command := range commands {
				_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
				Expect(err).To(BeNil())
			}
		})

})

var _ = Describe("Create cluster with oprator roles which are attaching managed policies in manual mode",
	labels.Feature.Cluster, func() {
		defer GinkgoRecover()
		var (
			rosaClient         *rosacli.Client
			clusterService     rosacli.ClusterService
			customProfile      *profilehandler.Profile
			ocmResourceService rosacli.OCMResourceService
			AWSAccountID       string
			testingClusterName string
			defaultDir         string
			dirForManual       string
			clusterID          string
		)

		BeforeEach(func() {
			// Init the client
			By("Init client and service")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
			clusterService = rosaClient.Cluster

			By("Get AWS account id")
			rosaClient.Runner.JsonFormat()
			whoamiOutput, err := ocmResourceService.Whoami()
			Expect(err).To(BeNil())
			rosaClient.Runner.UnsetFormat()
			whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
			AWSAccountID = whoamiData.AWSAccountID

			By("Prepare custom profile")
			customProfile = &profilehandler.Profile{
				ClusterConfig: &profilehandler.ClusterConfig{
					HCP:           true,
					MultiAZ:       true,
					STS:           true,
					OIDCConfig:    "managed",
					NetworkingSet: true,
					BYOVPC:        true,
					Zones:         "",
				},
				AccountRoleConfig: &profilehandler.AccountRoleConfig{
					Path:               "",
					PermissionBoundary: "",
				},
				Version:      "latest",
				ChannelGroup: "candidate",
				Region:       "us-west-2",
			}
			customProfile.NamePrefix = constants.DefaultNamePrefix

			By("Get the default dir")
			defaultDir = rosaClient.Runner.GetDir()
		})

		AfterEach(func() {

			By("Go back original by setting runner dir")
			rosaClient.Runner.SetDir(defaultDir)

			By("Delete cluster")
			rosaClient.Runner.UnsetArgs()
			_, err := clusterService.DeleteCluster(clusterID, "-y")
			Expect(err).To(BeNil())

			rosaClient.Runner.UnsetArgs()
			err = clusterService.WaitClusterDeleted(clusterID, 3, 30)
			Expect(err).To(BeNil())

			By("Delete operator-roles")
			_, err = ocmResourceService.DeleteOperatorRoles(
				"-c", testingClusterName,
				"--mode", "auto",
				"-y",
			)
			Expect(err).To(BeNil())

			By("Clean resource")
			errs := profilehandler.DestroyResourceByProfile(customProfile, rosaClient)
			Expect(len(errs)).To(Equal(0))
		})

		It("to create and delete operatorroles attaching managed policies in manual mode - [id:75504]",
			labels.Critical, labels.Runtime.Day1Supplemental, func() {
				By("Create hcp cluster in manual mode")
				testingClusterName = helper.GenerateRandomName("c75504", 2)
				testOperatorRolePrefix := helper.GenerateRandomName("opp75504", 2)
				flags, err := profilehandler.GenerateClusterCreateFlags(customProfile, rosaClient)
				Expect(err).ToNot(HaveOccurred())

				command := "rosa create cluster --cluster-name " + testingClusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--operator-roles-prefix": testOperatorRolePrefix,
				})

				rosalCommand.AddFlags("--mode", "manual")
				rosalCommand.AddFlags("--billing-account", AWSAccountID)

				By("Create a temp dir to execute the create commands")
				dirForManual, err = os.MkdirTemp("", "*")
				Expect(err).To(BeNil())

				rosaClient.Runner.SetDir(dirForManual)
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				Expect(err).To(BeNil())

				commands := helper.ExtractAWSCmdsForClusterCreation(stdout)
				hasCreatePolicyFlag := false
				for _, command := range commands {
					if strings.Contains(command, "aws iam create-policy") {
						hasCreatePolicyFlag = true
						break
					}
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					Expect(err).To(BeNil())
				}
				Expect(hasCreatePolicyFlag).To(BeFalse())

				By("Check and wait cluster to installing status")
				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				Expect(err).To(BeNil())

				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				Expect(err).To(BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				Expect(clusterID).ToNot(BeNil())
				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterStatus(clusterID, "installing", 3, 24)
				Expect(err).To(BeNil(), "It met error or timeout when waiting cluster to installing status")
			})
	})

var _ = Describe("Upgrade operator roles in auto mode",
	labels.Feature.Cluster, func() {
		defer GinkgoRecover()
		var (
			rosaClient         *rosacli.Client
			clusterService     rosacli.ClusterService
			customProfile      *profilehandler.Profile
			ocmResourceService rosacli.OCMResourceService
			upgradeService     rosacli.UpgradeService
			clusterName        string
			clusterID          string
		)

		BeforeEach(func() {
			By("Init client and service")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
			clusterService = rosaClient.Cluster
			upgradeService = rosaClient.Upgrade

			By("Prepare custom profile")
			customProfile = &profilehandler.Profile{
				ClusterConfig: &profilehandler.ClusterConfig{
					STS: true,
				},
				AccountRoleConfig: &profilehandler.AccountRoleConfig{
					Path:               "",
					PermissionBoundary: "",
				},
				Version:      "y-1",
				ChannelGroup: "candidate",
				Region:       "us-west-2",
			}
			customProfile.NamePrefix = constants.DefaultNamePrefix
		})

		AfterEach(func() {
			if clusterID != "" {
				By("Delete cluster")
				rosaClient.Runner.UnsetArgs()
				_, err := clusterService.DeleteCluster(clusterID, "-y")
				Expect(err).To(BeNil())

				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterDeleted(clusterID, 3, 30)
				Expect(err).To(BeNil())
			}

			By("Clean resource")
			errs := profilehandler.DestroyResourceByProfile(customProfile, rosaClient)
			Expect(len(errs)).To(Equal(0))
		})

		It("to create and upgrade operator roles in auto mode - [id:45745]",
			labels.Critical, labels.Runtime.Day1Supplemental, func() {
				By("Create classic STS cluster")
				clusterName = helper.GenerateRandomName("c45745", 2)
				flags, err := profilehandler.GenerateClusterCreateFlags(customProfile, rosaClient)
				Expect(err).ToNot(HaveOccurred())

				command := "rosa create cluster --cluster-name " + clusterName + " " + strings.Join(flags, " ")
				rosaCommand := config.GenerateCommand(command)

				rosaCommand.AddFlags("--mode", "auto")

				By("Execute the create commands to create cluster")
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosaCommand.GetFullCommand(), " "))
				Expect(err).To(BeNil())
				Expect(stdout.String()).To(Not(BeEmpty()))

				By("Get cluster ID")
				rosaClient.Runner.UnsetArgs()
				clusterListOut, err := clusterService.List()
				Expect(err).To(BeNil())

				clusterList, err := clusterService.ReflectClusterList(clusterListOut)
				Expect(err).To(BeNil())
				clusterID = clusterList.ClusterByName(clusterName).ID

				By("wait cluster ready")
				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterStatus(clusterID, constants.Ready, 3, 60)
				Expect(err).To(BeNil())

				By("Get cluster upgrade version")
				versionService := rosaClient.Version
				versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupCandidate, true)
				Expect(err).To(BeNil())
				defaultVersion := versionList.DefaultVersion()
				Expect(defaultVersion).ToNot(BeNil())
				_, _, upgradeVersion, err := defaultVersion.MajorMinor()
				Expect(err).To(BeNil())
				Expect(upgradeVersion).NotTo(BeEmpty())

				By("Upgrade cluster to verify if there are any prompts to upgrade account roles firstly")
				scheduledDate := time.Now().Format("2006-01-02")
				scheduledTime := time.Now().Add(10 * time.Minute).UTC().Format("15:04")
				output, err := upgradeService.Upgrade(
					"-c", clusterID,
					"--version", defaultVersion.Version,
					"--schedule-date", scheduledDate,
					"--schedule-time", scheduledTime,
					"-m", "auto",
				)
				Expect(err).NotTo(BeNil())
				Expect(output.String()).To(ContainSubstring("Account roles need to be upgraded to proceed"))

				By("Upgrade account roles in auto mode")
				accountRolePrefix := customProfile.ClusterConfig.Name
				_, err = ocmResourceService.UpgradeAccountRole(
					"--prefix", accountRolePrefix,
					"--mode", "auto",
					"--version", upgradeVersion,
					"--channel-group", "candidate",
					"-y",
				)
				Expect(err).To(BeNil())

				By("Upgrade operator roles in auto mode")
				output, err = ocmResourceService.UpgradeOperatorRoles(
					"--cluster", clusterName,
					"--version", defaultVersion.Version,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Starting to upgrade the operator IAM roles and " +
					"policies"))
				Expect(output.String()).To(ContainSubstring(
					"policy/%s-openshift-image-registry-installer-cloud-credent' to version '%s'",
					accountRolePrefix, upgradeVersion))
				Expect(output.String()).To(ContainSubstring(
					"policy/%s-openshift-ingress-operator-cloud-credentials' to version '%s'",
					accountRolePrefix, upgradeVersion))
				Expect(output.String()).To(ContainSubstring(
					"policy/%s-openshift-cluster-csi-drivers-ebs-cloud-credenti' to version '%s'",
					accountRolePrefix, upgradeVersion))
				Expect(output.String()).To(ContainSubstring(
					"policy/%s-openshift-cloud-network-config-controller-cloud-' to version '%s'",
					accountRolePrefix, upgradeVersion))
				Expect(output.String()).To(ContainSubstring(
					"policy/%s-openshift-machine-api-aws-cloud-credentials' to version '%s'",
					accountRolePrefix, upgradeVersion))
				Expect(output.String()).To(ContainSubstring(
					"policy/%s-openshift-cloud-credential-operator-cloud-creden' to version '%s'",
					accountRolePrefix, upgradeVersion))

				By("Upgrade cluster")
				scheduledDate = time.Now().Format("2006-01-02")
				scheduledTime = time.Now().Add(10 * time.Minute).UTC().Format("15:04")
				output, err = upgradeService.Upgrade(
					"-c", clusterID,
					"--version", defaultVersion.Version,
					"--schedule-date", scheduledDate,
					"--schedule-time", scheduledTime,
					"-m", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				Expect(output.String()).To(ContainSubstring("Upgrade successfully scheduled for cluster"))

				By("Check upgrade state")
				err = upgradeService.WaitForUpgradeToState(clusterID, constants.Scheduled, 5)
				Expect(err).To(BeNil())
			})
	})
