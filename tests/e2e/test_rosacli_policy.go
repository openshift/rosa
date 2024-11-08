package e2e

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	"github.com/openshift/rosa/tests/utils/helper"
)

var _ = Describe("Attach and Detach arbitrary policies",
	labels.Feature.Policy,
	func() {
		defer GinkgoRecover()

		var (
			clusterID                string
			rosaClient               *rosacli.Client
			arbitraryPolicyService   rosacli.PolicyService
			clusterService           rosacli.ClusterService
			arbitraryPoliciesToClean []string
			awsClient                *aws_client.AWSClient
			err                      error
			profile                  *handler.Profile
			roleUrlPrefix            = "https://console.aws.amazon.com/iam/home?#/roles/"
		)

		BeforeEach(func() {
			By("Load profile")
			profile = handler.LoadProfileYamlFileByENV()
			if !profile.ClusterConfig.STS {
				Skip("This feature only works for STS cluster")
			}

			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			arbitraryPolicyService = rosaClient.Policy
			clusterService = rosaClient.Cluster

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
					fmt.Sprintf("ocmqe-arpolicy-%s-%d", helper.GenerateRandomString(3), i),
					statement,
				)
				Expect(err).To(BeNil())
				arbitraryPoliciesToClean = append(arbitraryPoliciesToClean, arn)
			}

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

		It("can attach and detach arbitrary policies on existing roles in auto mode - [id:73449]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				By("Get operator-roles arns")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				operatorRolesArns := CD.OperatorIAMRoles
				_, operatorRoleName1, err := helper.ParseRoleARN(operatorRolesArns[2])
				Expect(err).To(BeNil())
				operatorRolePoliciesMap1 := make(map[string][]string)
				operatorRolePoliciesMap1[operatorRoleName1] = arbitraryPoliciesToClean[0:2]

				By("Attach policies to operator-roles")
				for roleName, policyArns := range operatorRolePoliciesMap1 {
					out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
					Expect(err).To(BeNil())
					for _, policyArn := range policyArns {
						Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
							policyArn, roleName, roleUrlPrefix+roleName))
					}

				}

				_, operatorRoleName2, err := helper.ParseRoleARN(operatorRolesArns[4])
				Expect(err).To(BeNil())
				operatorRolePoliciesMap2 := make(map[string][]string)
				operatorRolePoliciesMap2[operatorRoleName2] = append(
					operatorRolePoliciesMap2[operatorRolesArns[4]],
					arbitraryPoliciesToClean[1],
				)

				for roleName, policyArns := range operatorRolePoliciesMap2 {
					out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
					Expect(err).To(BeNil())
					for _, policyArn := range policyArns {
						Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
							policyArn, roleName, roleUrlPrefix+roleName))
					}

				}

				By("Check the arbitrary is attached to operator roles")
				output, err = clusterService.DescribeCluster(clusterID, "--get-role-policy-bindings")
				Expect(err).To(BeNil())
				arbitraryCD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				operatorRolesArnsWithArbitrary := arbitraryCD.OperatorIAMRoles
				for _, v := range operatorRolesArnsWithArbitrary {
					if strings.Contains(v, operatorRolesArns[2]) {
						for _, arbitraryPolicyArn := range operatorRolePoliciesMap1[operatorRoleName1] {
							Expect(v).To(ContainSubstring(arbitraryPolicyArn))
						}
					}
					if strings.Contains(v, operatorRolesArns[4]) {
						for _, arbitraryPolicyArn := range operatorRolePoliciesMap2[operatorRoleName2] {
							Expect(v).To(ContainSubstring(arbitraryPolicyArn))
						}
					}
				}

				By("Detach policies from operator-roles")
				for roleName, policyArns := range operatorRolePoliciesMap1 {
					out, err := arbitraryPolicyService.DetachPolicy(roleName, policyArns, "--mode", "auto")
					Expect(err).To(BeNil())
					for _, policyArn := range policyArns {
						Expect(out.String()).To(ContainSubstring("Detached policy '%s' from role '%s'", policyArn, roleName))
					}

				}
				for roleName, policyArns := range operatorRolePoliciesMap2 {
					out, err := arbitraryPolicyService.DetachPolicy(roleName, policyArns, "--mode", "auto")
					Expect(err).To(BeNil())
					for _, policyArn := range policyArns {
						Expect(out.String()).To(ContainSubstring("Detached policy '%s' from role '%s'", policyArn, roleName))
					}

				}

				By("Check the arbitrary is detached from operator roles")
				output, err = clusterService.DescribeCluster(clusterID, "--get-role-policy-bindings")
				Expect(err).To(BeNil())
				arbitraryCD, err = clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				operatorRolesArnsWithArbitrary = arbitraryCD.OperatorIAMRoles
				for _, v := range operatorRolesArnsWithArbitrary {
					if strings.Contains(v, operatorRolesArns[2]) {
						for _, arbitraryPolicyArn := range operatorRolePoliciesMap1[operatorRoleName1] {
							Expect(v).ToNot(ContainSubstring(arbitraryPolicyArn))
						}
					}
					if strings.Contains(v, operatorRolesArns[4]) {
						for _, arbitraryPolicyArn := range operatorRolePoliciesMap2[operatorRoleName2] {
							Expect(v).ToNot(ContainSubstring(arbitraryPolicyArn))
						}
					}
				}

				By("Get account-roles arns for testing")
				var workerRoleARN string
				supportRoleARN := CD.SupportRoleARN
				for _, rolePolicyMap := range CD.InstanceIAMRoles {
					for k, v := range rolePolicyMap {
						// nolint:goconst
						if k == "Worker" {
							workerRoleARN = v
						} else {
							break
						}
					}
				}
				_, workerRoleName, err := helper.ParseRoleARN(workerRoleARN)
				Expect(err).To(BeNil())
				_, supportRoleName, err := helper.ParseRoleARN(supportRoleARN)
				Expect(err).To(BeNil())

				accountRolePoliciesMap1 := make(map[string][]string)
				accountRolePoliciesMap1[workerRoleName] = arbitraryPoliciesToClean[0:2]

				accountRolePoliciesMap2 := make(map[string][]string)
				accountRolePoliciesMap2[supportRoleName] = append(
					accountRolePoliciesMap2[operatorRolesArns[1]],
					arbitraryPoliciesToClean[1],
				)

				By("Attach policies to account-roles")
				for roleName, policyArns := range accountRolePoliciesMap1 {
					out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
					Expect(err).To(BeNil())
					for _, policyArn := range policyArns {
						Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
							policyArn, roleName, roleUrlPrefix+roleName))
					}

				}

				for roleName, policyArns := range accountRolePoliciesMap2 {
					out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
					Expect(err).To(BeNil())
					for _, policyArn := range policyArns {
						Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
							policyArn, roleName, roleUrlPrefix+roleName))
					}

				}

				By("Check the arbitrary is attached to account roles")
				output, err = clusterService.DescribeCluster(clusterID, "--get-role-policy-bindings")
				Expect(err).To(BeNil())
				arbitraryCD, err = clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				for _, rolePolicyMap := range arbitraryCD.InstanceIAMRoles {
					for k, v := range rolePolicyMap {
						if k == "Worker" {
							Expect(v).To(ContainSubstring(workerRoleARN))
							for _, arbitraryPolicy := range accountRolePoliciesMap1[workerRoleName] {
								Expect(v).To(ContainSubstring(arbitraryPolicy))
							}
						} else {
							break
						}
					}
				}

				Expect(arbitraryCD.SupportRoleARN).To(ContainSubstring(supportRoleARN))
				Expect(arbitraryCD.SupportRoleARN).To(ContainSubstring(arbitraryPoliciesToClean[1]))

				By("Detach policies from accout-roles")
				for roleName, policyArns := range accountRolePoliciesMap1 {
					out, err := arbitraryPolicyService.DetachPolicy(roleName, policyArns, "--mode", "auto")
					Expect(err).To(BeNil())
					for _, policyArn := range policyArns {
						Expect(out.String()).To(ContainSubstring("Detached policy '%s' from role '%s'", policyArn, roleName))
					}

				}

				for roleName, policyArns := range accountRolePoliciesMap2 {
					out, err := arbitraryPolicyService.DetachPolicy(roleName, policyArns, "--mode", "auto")
					Expect(err).To(BeNil())
					for _, policyArn := range policyArns {
						Expect(out.String()).To(ContainSubstring("Detached policy '%s' from role '%s'", policyArn, roleName))
					}

				}

				By("Check the arbitrary is detached from account roles")
				output, err = clusterService.DescribeCluster(clusterID, "--get-role-policy-bindings")
				Expect(err).To(BeNil())
				arbitraryCD, err = clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				for _, rolePolicyMap := range arbitraryCD.InstanceIAMRoles {
					for k, v := range rolePolicyMap {
						if k == "Worker" {
							Expect(v).To(ContainSubstring(workerRoleARN))
							for _, arbitraryPolicy := range accountRolePoliciesMap1[workerRoleName] {
								Expect(v).ToNot(ContainSubstring(arbitraryPolicy))
							}
						} else {
							break
						}
					}
				}
				Expect(arbitraryCD.SupportRoleARN).To(ContainSubstring(supportRoleARN))
				Expect(arbitraryCD.SupportRoleARN).ToNot(ContainSubstring(arbitraryPoliciesToClean[1]))
			})
	})

var _ = Describe("Validation testing",
	labels.Feature.Policy,
	func() {
		defer GinkgoRecover()

		var (
			clusterID                string
			rosaClient               *rosacli.Client
			arbitraryPolicyService   rosacli.PolicyService
			clusterService           rosacli.ClusterService
			arbitraryPoliciesToClean []string
			testingRolesToClean      []string
			awsClient                *aws_client.AWSClient
			err                      error
			profile                  *handler.Profile
		)

		BeforeEach(func() {
			By("Load profile")
			profile = handler.LoadProfileYamlFileByENV()

			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			arbitraryPolicyService = rosaClient.Policy
			clusterService = rosaClient.Cluster
			By("Prepare arbitrary policies for testing")

			awsClient, err = aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())
			statement := map[string]interface{}{
				"Effect":   "Allow",
				"Action":   "*",
				"Resource": "*",
			}
			for i := 0; i < 10; i++ {
				arn, err := awsClient.CreatePolicy(
					fmt.Sprintf("ocmqe-arpolicy-%s-%d", helper.GenerateRandomString(3), i),
					statement,
				)
				Expect(err).To(BeNil())
				arbitraryPoliciesToClean = append(arbitraryPoliciesToClean, arn)
			}

		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())

			By("Delete the testing role")
			if len(testingRolesToClean) > 0 {
				for _, roleName := range testingRolesToClean {
					attachedPolicy, err := awsClient.ListRoleAttachedPolicies(roleName)
					Expect(err).To(BeNil())
					if len(attachedPolicy) > 0 {
						err = awsClient.DetachRolePolicies(roleName)
						Expect(err).To(BeNil())
					}
					err = awsClient.DeleteRole(roleName)
					Expect(err).To(BeNil())
				}
			}

			By("Delete arbitrary policies")
			if len(arbitraryPoliciesToClean) > 0 {
				for _, policyArn := range arbitraryPoliciesToClean {
					err = awsClient.DeletePolicy(policyArn)
					Expect(err).To(BeNil())
				}
			}
		})

		It("to check the validations for attaching and detaching arbitrary policies - [id:74225]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				if !profile.ClusterConfig.STS {
					Skip("This feature only works for STS cluster")
				}
				By("Prepare a role wihtout red-hat-managed=true label for testing")
				notRHManagedRoleName := fmt.Sprintf("ocmqe-role-%s", helper.GenerateRandomString(3))

				statement := map[string]interface{}{
					"Effect":   "Allow",
					"Action":   "*",
					"Resource": "*",
				}
				notRHManagedRolePolicy, err := awsClient.CreatePolicy(
					fmt.Sprintf("%s-policy", notRHManagedRoleName),
					statement,
				)
				Expect(err).To(BeNil())
				_, err = awsClient.CreateRegularRole(notRHManagedRoleName, notRHManagedRolePolicy)
				Expect(err).To(BeNil())
				defer func() {
					By("Detach the attached policy " + notRHManagedRolePolicy + " from role" + notRHManagedRoleName)
					err := awsClient.DetachIAMPolicy(notRHManagedRoleName, notRHManagedRolePolicy)
					Expect(err).To(BeNil())
				}()
				testingRolesToClean = append(testingRolesToClean, notRHManagedRoleName)
				arbitraryPoliciesToClean = append(arbitraryPoliciesToClean, notRHManagedRolePolicy)

				By("Get one managed role for testing,using support role in this case")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				supportRoleARN := CD.SupportRoleARN
				_, supportRoleName, err := helper.ParseRoleARN(supportRoleARN)
				Expect(err).To(BeNil())

				By("policy arn with invalid format when attach")
				policyArnsWithOneInValidFormat := []string{
					"arn:aws:polict:invalidformat",
					arbitraryPoliciesToClean[0],
					arbitraryPoliciesToClean[1],
				}
				out, err := arbitraryPolicyService.AttachPolicy(
					supportRoleName,
					policyArnsWithOneInValidFormat,
					"--mode", "auto",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("Invalid policy arn"))

				By("not-existed policies arn when attach")
				policyArnsWithNotExistedOne := []string{
					"arn:aws:iam::123456789012:policy/ocmqe-arpolicy-rta-0",
					arbitraryPoliciesToClean[0],
					arbitraryPoliciesToClean[1],
				}
				out, err = arbitraryPolicyService.AttachPolicy(
					supportRoleName,
					policyArnsWithNotExistedOne,
					"--mode", "auto",
				)
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("not found"))

				By("not-existed role name when attach")
				notExistedRoleName := "notExistedRoleName"
				policyArns := []string{
					arbitraryPoliciesToClean[0],
					arbitraryPoliciesToClean[1],
				}
				out, err = arbitraryPolicyService.AttachPolicy(notExistedRoleName, policyArns, "--mode", "auto")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("role with name %s cannot be found", notExistedRoleName))

				By("number of the attaching policies exceed the quote (L-0DA4ABF3) when attach")
				policyArnsWithNine := arbitraryPoliciesToClean[0:9]
				out, err = arbitraryPolicyService.AttachPolicy(supportRoleName, policyArnsWithNine, "--mode", "auto")
				Expect(err).To(BeNil())
				defer func() {
					By("Detach the attached policies from role " + supportRoleName)
					out, err = arbitraryPolicyService.DetachPolicy(supportRoleName, policyArnsWithNine, "--mode", "auto")
					Expect(err).To(BeNil())
				}()

				policyArnWithTen := []string{arbitraryPoliciesToClean[9]}
				out, err = arbitraryPolicyService.AttachPolicy(supportRoleName, policyArnWithTen, "--mode", "auto")
				Expect(err).ToNot(BeNil())
				Expect(out.String()).To(ContainSubstring("Failed to attach policies due to quota limitations (total limit: 10"))

				By("role has no red-hat-managed=true tag when attach")
				out, err = arbitraryPolicyService.AttachPolicy(notRHManagedRoleName, policyArns, "--mode", "auto")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("Cannot attach/detach policies to non-ROSA roles"))

				By("empry string in the policy-arn when attach")
				policyArnsWithEmptyString := []string{""}
				out, err = arbitraryPolicyService.AttachPolicy(supportRoleName, policyArnsWithEmptyString, "--mode", "auto")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("expected a valid policy"))

				By("policy arn with invalid format when detach")

				out, err = arbitraryPolicyService.DetachPolicy(supportRoleName, policyArnsWithOneInValidFormat, "--mode", "auto")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("Invalid policy arn"))

				By("not-existed policies arn when detach")
				out, err = arbitraryPolicyService.DetachPolicy(supportRoleName, policyArnsWithNotExistedOne, "--mode", "auto")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("not found"))

				By("not-existed role name when detach")
				out, err = arbitraryPolicyService.DetachPolicy(notExistedRoleName, policyArns, "--mode", "auto")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("role with name %s cannot be found", notExistedRoleName))

				By("role has no red-hat-managed=true tag when detach")
				out, err = arbitraryPolicyService.DetachPolicy(notRHManagedRoleName, policyArns, "--mode", "auto")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("Cannot attach/detach policies to non-ROSA roles"))

				By("empry string in the policy-arn when detach")
				out, err = arbitraryPolicyService.DetachPolicy(supportRoleName, policyArnsWithEmptyString, "--mode", "auto")
				Expect(err).NotTo(BeNil())
				Expect(out.String()).To(ContainSubstring("expected a valid policy"))
			})
	})

var _ = Describe("Account roles with attaching arbitrary policies",
	labels.Feature.Policy,
	func() {
		defer GinkgoRecover()

		var (
			clusterID                string
			rosaClient               *rosacli.Client
			arbitraryPolicyService   rosacli.PolicyService
			arbitraryPoliciesToClean []string
			awsClient                *aws_client.AWSClient
			err                      error
			roleUrlPrefix            = "https://console.aws.amazon.com/iam/home?#/roles/"
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			arbitraryPolicyService = rosaClient.Policy

			awsClient, err = aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())
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

		It("can be upgraded and deleted successfully - [id:74402]", labels.Critical, labels.Runtime.Day2, func() {
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
					fmt.Sprintf("ocmqe-arpolicy-%s-%d", helper.GenerateRandomString(3), i),
					statement,
				)
				Expect(err).To(BeNil())
				arbitraryPoliciesToClean = append(arbitraryPoliciesToClean, arn)
			}
			By("Prepare version for testing")
			var accountRoleLowVersion string
			versionService := rosaClient.Version
			versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, false)
			Expect(err).To(BeNil())
			defaultVersion := versionList.DefaultVersion()
			Expect(defaultVersion).ToNot(BeNil())
			lowerVersion, err := versionList.FindNearestBackwardMinorVersion(defaultVersion.Version, 1, true)
			Expect(err).To(BeNil())
			Expect(lowerVersion).NotTo(BeNil())

			_, _, accountRoleLowVersion, err = lowerVersion.MajorMinor()
			Expect(err).To(BeNil())

			By("Create account-roles in low version")
			ocmResourceService := rosaClient.OCMResource
			aRolePrefix := "aroleprefix131313"
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", aRolePrefix,
				"--version", accountRoleLowVersion,
				"-y")
			Expect(err).To(BeNil())
			defer func() {
				By("Delete the account-roles")
				output, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", aRolePrefix,
					"-y")

				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted"))

				By("Check the arbitrary polcies not deleted by rosa command of deleting account-roles")
				for _, policyArn := range arbitraryPoliciesToClean {
					policy, err := awsClient.GetIAMPolicy(policyArn)
					Expect(err).To(BeNil())
					Expect(policy).ToNot(BeNil())
				}
			}()
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Created role"))

			By("Get account-roles arns for testing")
			arl, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())
			ars := arl.DigAccountRoles(aRolePrefix, false)
			fmt.Println(ars)
			supportRoleArn := ars.SupportRole
			workerRoleArn := ars.WorkerRole

			_, supportRoleName, err := helper.ParseRoleARN(supportRoleArn)
			Expect(err).To(BeNil())
			_, workerRoleName, err := helper.ParseRoleARN(workerRoleArn)
			Expect(err).To(BeNil())

			By("Attach two arbitrary policies to Support roles")
			accountRolePoliciesMap1 := make(map[string][]string)
			accountRolePoliciesMap1[supportRoleName] = arbitraryPoliciesToClean[0:2]
			for roleName, policyArns := range accountRolePoliciesMap1 {
				out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
				Expect(err).To(BeNil())
				Expect(out.String()).To(
					Or(
						ContainSubstring(fmt.Sprintf(
							"Attached policy '%s' to role '%s(%s)'", policyArns[0], roleName, roleUrlPrefix+roleName)),
						ContainSubstring(fmt.Sprintf(
							"Attached policy '%s' to role '%s(%s)'", policyArns[1], roleName, roleUrlPrefix+roleName)),
					),
				)
			}

			By("Detach and delete redhat managed policies from worker role")
			attachWorkerRolePolicies, err := awsClient.ListAttachedRolePolicies(workerRoleName)
			Expect(err).To(BeNil())
			Expect(len(attachWorkerRolePolicies)).To(Equal(1))
			err = awsClient.DetachRolePolicies(workerRoleName)
			Expect(err).To(BeNil())
			err = awsClient.DeleteIAMPolicy(*(attachWorkerRolePolicies[0].PolicyArn))
			Expect(err).To(BeNil())

			By("Attach one arbitrary policy to worker role")
			accountRolePoliciesMap2 := make(map[string][]string)
			accountRolePoliciesMap2[workerRoleName] = append(
				accountRolePoliciesMap2[workerRoleName],
				arbitraryPoliciesToClean[1],
			)
			for roleName, policyArns := range accountRolePoliciesMap2 {
				out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
				Expect(err).To(BeNil())
				for _, policyArn := range policyArns {
					Expect(out.String()).
						To(
							ContainSubstring("Attached policy '%s' to role '%s(%s)'", policyArn, roleName, roleUrlPrefix+roleName))
				}
			}

			By("Upgrade account-roles in auto mode")
			output, err = ocmResourceService.UpgradeAccountRole(
				"--prefix", aRolePrefix,
				"--mode", "auto",
				"-y",
			)
			Expect(err).To(BeNil())
			Expect(output.String()).To(MatchRegexp(`Upgraded policy with ARN .* to latest version`))

			By("Check the support and worker role policy binding")
			attachWorkerRolePolicies, err = awsClient.ListAttachedRolePolicies(workerRoleName)
			Expect(err).To(BeNil())
			Expect(len(attachWorkerRolePolicies)).To(Equal(2))

			attachWorkerRolePolicies, err = awsClient.ListAttachedRolePolicies(supportRoleName)
			Expect(err).To(BeNil())
			Expect(len(attachWorkerRolePolicies)).To(Equal(3))

			By("Check the attached arbitrary policies")
			for _, policyArn := range arbitraryPoliciesToClean {
				policy, err := awsClient.GetIAMPolicy(policyArn)
				Expect(err).To(BeNil())
				Expect(len(policy.Tags)).To(Equal(0))
			}
		})
	})

var _ = Describe("Operator roles with attaching arbitrary policies",
	labels.Feature.Policy,
	func() {
		defer GinkgoRecover()

		var (
			clusterID                string
			rosaClient               *rosacli.Client
			arbitraryPolicyService   rosacli.PolicyService
			arbitraryPoliciesToClean []string
			awsClient                *aws_client.AWSClient
			err                      error
			managedOIDCConfigID      string
			ocmResourceService       rosacli.OCMResourceService
			roleUrlPrefix            = "https://console.aws.amazon.com/iam/home?#/roles/"
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			arbitraryPolicyService = rosaClient.Policy

			awsClient, err = aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())

			ocmResourceService = rosaClient.OCMResource
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
			By("Delete testing oidc-config")
			output, err := ocmResourceService.DeleteOIDCConfig(
				"--oidc-config-id", managedOIDCConfigID,
				"--mode", "auto",
				"-y",
			)
			Expect(err).To(BeNil())
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Successfully deleted the OIDC provider"))
		})

		It("can be deleted successfully - [id:74403]", labels.Critical, labels.Runtime.OCMResources, func() {
			By("Prepare * arbitrary policies for testing")
			awsClient, err = aws_client.CreateAWSClient("", "")
			Expect(err).To(BeNil())
			statement := map[string]interface{}{
				"Effect":   "Allow",
				"Action":   "*",
				"Resource": "*",
			}
			for i := 0; i < 2; i++ {
				arn, err := awsClient.CreatePolicy(
					fmt.Sprintf("ocmqe-arpolicy-%s-%d", helper.GenerateRandomString(3), i),
					statement,
				)
				Expect(err).To(BeNil())
				arbitraryPoliciesToClean = append(arbitraryPoliciesToClean, arn)
			}
			By("Prepare version for testing")
			var accountRoleLowVersion string
			versionService := rosaClient.Version
			versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, false)
			Expect(err).To(BeNil())
			defaultVersion := versionList.DefaultVersion()
			Expect(defaultVersion).ToNot(BeNil())
			lowerVersion, err := versionList.FindNearestBackwardMinorVersion(defaultVersion.Version, 1, true)
			Expect(err).To(BeNil())
			Expect(lowerVersion).NotTo(BeNil())

			_, _, accountRoleLowVersion, err = lowerVersion.MajorMinor()
			Expect(err).To(BeNil())

			By("Create account-roles in low version")
			aRolePrefix := "aroleprefix242424"
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", aRolePrefix,
				"--version", accountRoleLowVersion,
				"-y")
			Expect(err).To(BeNil())
			defer func() {
				By("Delete the account-roles")
				output, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", aRolePrefix,
					"-y")

				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted"))

				By("Check the arbitrary polcies not deleted by rosa command of deleting account-roles")
				for _, policyArn := range arbitraryPoliciesToClean {
					policy, err := awsClient.GetIAMPolicy(policyArn)
					Expect(err).To(BeNil())
					Expect(policy).ToNot(BeNil())
				}
			}()
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Created role"))

			By("Get installer role arn for testing")
			arl, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())
			ars := arl.DigAccountRoles(aRolePrefix, false)
			fmt.Println(ars)
			installerRoleArn := ars.InstallerRole

			By("Create managed oidc-config in auto mode")
			output, err = ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
			Expect(err).To(BeNil())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Created OIDC provider with ARN"))
			oidcPrivodeARNFromOutputMessage := helper.ExtractOIDCProviderARN(output.String())
			oidcPrivodeIDFromOutputMessage := helper.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)

			managedOIDCConfigID, err = ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
			Expect(err).To(BeNil())

			By("Create operator-roles prior to cluster spec")
			operatorRolesPrefix := "oproleprefix242424"
			output, err = ocmResourceService.CreateOperatorRoles(
				"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
				"--installer-role-arn", installerRoleArn,
				"--mode", "auto",
				"--prefix", operatorRolesPrefix,
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				By("Delete the operator-roles")
				output, err := ocmResourceService.DeleteOperatorRoles(
					"--prefix", operatorRolesPrefix,
					"--mode", "auto",
					"-y",
				)
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the operator roles"))

				By("Check the arbitrary-roles not deleted by rosa command of deleting operator-roles")
				for _, policyArn := range arbitraryPoliciesToClean {
					policy, err := awsClient.GetIAMPolicy(policyArn)
					Expect(err).To(BeNil())
					Expect(policy).ToNot(BeNil())
				}
			}()

			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).Should(ContainSubstring("Created role"))

			output, err = ocmResourceService.ListOperatorRoles(
				"--prefix", operatorRolesPrefix,
			)
			Expect(err).To(BeNil())
			operatorRoleList, err := ocmResourceService.ReflectOperatorRoleList(output)
			Expect(err).To(BeNil())

			By("Attach two arbitrary policies to operator-roles")
			operatorRolePoliciesMap1 := make(map[string][]string)
			operatorRolePoliciesMap1[operatorRoleList.OperatorRoleList[1].RoleName] = arbitraryPoliciesToClean[0:2]
			operatorRolePoliciesMap2 := make(map[string][]string)
			operatorRolePoliciesMap2[operatorRoleList.OperatorRoleList[2].RoleName] = append(
				operatorRolePoliciesMap2[operatorRoleList.OperatorRoleList[2].RoleName],
				arbitraryPoliciesToClean[1],
			)

			for roleName, policyArns := range operatorRolePoliciesMap1 {
				out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
				Expect(err).To(BeNil())
				for _, policyArn := range policyArns {
					Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
						policyArn, roleName, roleUrlPrefix+roleName))
				}
			}
			for roleName, policyArns := range operatorRolePoliciesMap2 {
				out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
				Expect(err).To(BeNil())
				for _, policyArn := range policyArns {
					Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
						policyArn, roleName, roleUrlPrefix+roleName))
				}
			}

			By("Attach two arbitrary policies to one account role")
			supportRoleArn := ars.SupportRole
			_, supportRoleName, err := helper.ParseRoleARN(supportRoleArn)
			Expect(err).To(BeNil())

			accountRolePoliciesMap := make(map[string][]string)
			accountRolePoliciesMap[supportRoleName] = arbitraryPoliciesToClean[0:2]
			for roleName, policyArns := range accountRolePoliciesMap {
				out, err := arbitraryPolicyService.AttachPolicy(roleName, policyArns, "--mode", "auto")
				Expect(err).To(BeNil())
				for _, policyArn := range policyArns {
					Expect(out.String()).To(ContainSubstring("Attached policy '%s' to role '%s(%s)'",
						policyArn, roleName, roleUrlPrefix+roleName))
				}
			}
		})
	})
