/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/pkg/iamserviceaccount"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
)

var _ = Describe("IAM Service Account", labels.Feature.IAMServiceAccount, func() {
	defer GinkgoRecover()

	var (
		rosaClient *rosacli.Client

		// Track resources for cleanup
		// Generate unique namespace for tests
		firstAttachPolicy      string
		secondAttachPolicy     string
		permissionsBoundaryArn = "arn:aws:iam::aws:policy/AdministratorAccess"
		awsClient              *aws_client.AWSClient
		err                    error
		iamSAaccountService    rosacli.IAMServiceAccountService
		clusterService         rosacli.ClusterService
		clusterID              string
		inlinePolicyFilePath   string
		tempDir                string
	)

	BeforeEach(func() {
		By("Get the cluster")
		clusterID = config.GetClusterID()
		Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

		By("Init the clients")
		rosaClient = rosacli.NewClient()
		iamSAaccountService = rosaClient.IAMServiceAccount
		clusterService = rosaClient.Cluster

		By("Skip testing if the cluster is not a STS(Hosted-cp) cluster")
		stsCluster, err := clusterService.IsSTSCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())
		if !stsCluster {
			SkipNotSTS()
		}

		By("Prepare policies for testing")
		awsClient, err = aws_client.CreateAWSClient("", "")
		Expect(err).To(BeNil())
		statement := map[string]interface{}{
			"Effect":   "Allow",
			"Action":   "ec2:DescribeAccountAttributes",
			"Resource": "*",
		}
		statement2 := map[string]interface{}{
			"Effect":   "Allow",
			"Action":   "iam:GetAccountSummary",
			"Resource": "*",
		}
		firstAttachPolicy, err = awsClient.CreatePolicy(
			"rosacli-iamserviceaccount-policy1",
			statement,
		)
		Expect(err).To(BeNil())
		secondAttachPolicy, err = awsClient.CreatePolicy(
			"rosacli-iamserviceaccount-policy2",
			statement2,
		)
		Expect(err).To(BeNil())

		By("Prepare inline policy file")
		tempDir, err = os.MkdirTemp("", "*")
		Expect(err).To(BeNil())
		inlinePolicyContent := `{
  			"Version": "2012-10-17",
  			"Statement": [
    			{
      				"Effect": "Allow",
      				"Action": [
        				"s3:GetObject",
        				"s3:PutObject"
      				],
      				"Resource": "arn:aws:s3:::my-app-bucket/*"
    			}
  			]
		}`
		inlinePolicyFilePath = fmt.Sprintf("%s/inline-policy.json", tempDir)

		err = os.WriteFile(inlinePolicyFilePath, []byte(inlinePolicyContent), 0644)
		Expect(err).To(BeNil())

	})

	AfterEach(func() {
		By("Delete the testing policies")
		err = awsClient.DeletePolicy(firstAttachPolicy)
		Expect(err).To(BeNil())
		err = awsClient.DeletePolicy(secondAttachPolicy)
		Expect(err).To(BeNil())

	})

	Context("IAM Service Account Management", func() {
		It("can create, list, describe, and delete IAM service account role in auto mode - [id:84391]",
			labels.High, labels.Runtime.Day2,
			func() {
				testName := "iamsaname"
				testNameSpace := "iamsanamespace"
				iamServiceAccountCustomRoleName := helper.GenerateRandomName("rosa-e2e-iamserviceaccount", 3)
				testRolePath := "/test/path/"
				By("With multiple policies and custom name and all required flag")
				output, err := iamSAaccountService.CreateIAMServiceAccountRole(
					"-c", clusterID,
					"--name", testName,
					"--namespace", testNameSpace,
					"--role-name", iamServiceAccountCustomRoleName,
					"--attach-policy-arn", fmt.Sprintf("%s,%s", firstAttachPolicy, secondAttachPolicy),
					"--path", "/test/path/",
					"--permissions-boundary", permissionsBoundaryArn,
				)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					By("Delete the IAM service account role with custom name")
					output, err = iamSAaccountService.DeleteIAMServiceAccountRole(
						"-c", clusterID,
						"--role-name", iamServiceAccountCustomRoleName,
						"--mode", "auto",
						"-y",
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(output.String()).To(ContainSubstring("Role details"))
					Expect(output.String()).To(ContainSubstring("Successfully deleted IAM service account role"))
				}()
				Expect(output.String()).To(ContainSubstring("Created IAM role"))
				Expect(output.String()).To(ContainSubstring("Attached policy"))

				By("With inline policy and auto-generated name and all required flag")
				output, err = iamSAaccountService.CreateIAMServiceAccountRole(
					"-c", clusterID,
					"--name", testName,
					"--namespace", testNameSpace,
					"--attach-policy-arn", fmt.Sprintf("%s,%s", firstAttachPolicy, secondAttachPolicy),
					"--path", testRolePath,
					"--permissions-boundary", permissionsBoundaryArn,
					"--inline-policy", fmt.Sprintf("file://%s", inlinePolicyFilePath),
				)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					By("Delete the IAM service account role with auto-generated name")
					output, err = iamSAaccountService.DeleteIAMServiceAccountRole(
						"-c", clusterID,
						"--name", testName,
						"--namespace", testNameSpace,
						"--mode", "auto",
						"-y",
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(output.String()).To(ContainSubstring("Role details"))
					Expect(output.String()).To(ContainSubstring("Successfully deleted IAM service account role"))

				}()
				Expect(output.String()).To(ContainSubstring("Created IAM role"))
				Expect(output.String()).To(ContainSubstring("Attached policy"))
				Expect(output.String()).To(ContainSubstring("Attached inline policy"))

				By("List IAM service account roles")
				output, err = iamSAaccountService.ListIAMServiceAccountRoles(
					"-c", clusterID,
				)
				Expect(err).ToNot(HaveOccurred())
				iamSARolesList, err := iamSAaccountService.ReflectIamServiceAccountList(output)
				Expect(err).ToNot(HaveOccurred())

				clusterName, err := clusterService.GetClusterName(clusterID)
				Expect(err).ToNot(HaveOccurred())
				generatedIamSaRoleName := iamserviceaccount.GenerateRoleName(clusterName, testNameSpace, testName)

				iamSaRoleWithGeneratedName := iamSARolesList.GetIAMServiceAccountRoleByName(generatedIamSaRoleName)
				Expect(iamSaRoleWithGeneratedName.Arn).ToNot(BeEmpty())
				Expect(iamSaRoleWithGeneratedName.Cluster).To(Equal(clusterName))
				Expect(iamSaRoleWithGeneratedName.Name).To(Equal(generatedIamSaRoleName))
				Expect(iamSaRoleWithGeneratedName.ServiceAccount).To(Equal(testName))
				Expect(iamSaRoleWithGeneratedName.Namespace).To(Equal(testNameSpace))

				iamSaRoleWithCustomName := iamSARolesList.GetIAMServiceAccountRoleByName(iamServiceAccountCustomRoleName)
				Expect(iamSaRoleWithCustomName.Arn).ToNot(BeEmpty())
				Expect(iamSaRoleWithCustomName.Cluster).To(Equal(clusterName))
				Expect(iamSaRoleWithCustomName.Name).To(Equal(iamServiceAccountCustomRoleName))
				Expect(iamSaRoleWithCustomName.ServiceAccount).To(Equal(testName))
				Expect(iamSaRoleWithCustomName.Namespace).To(Equal(testNameSpace))

				By("Descrine IAM service account role")
				rosaClient.Runner.JsonFormat()
				jsonOut, err := iamSAaccountService.DescribeIAMServiceAccountRole(
					"-c", clusterID,
					"--role-name", generatedIamSaRoleName,
				)
				Expect(err).ToNot(HaveOccurred())
				rosaClient.Runner.UnsetFormat()
				jsonData := rosaClient.Parser.JsonData.Input(jsonOut).Parse()
				Expect(jsonData.DigString("roleName")).To(Equal(generatedIamSaRoleName))
				Expect(jsonData.DigString("arn")).ToNot(BeEmpty())
				Expect(jsonData.DigString("path")).To(Equal(testRolePath))
				Expect(jsonData.DigString("permissionsBoundary")).To(Equal(permissionsBoundaryArn))
				Expect(jsonData.DigObject("attachedPolicies")).ToNot(BeEmpty())
				Expect(jsonData.DigObject("inlinePolicies")).ToNot(BeEmpty())
				Expect(jsonData.DigString("trustPolicy")).ToNot(BeEmpty())
				Expect(jsonData.DigString("oidcProvider")).ToNot(BeEmpty())
				Expect(jsonData.DigObject("tags")).ToNot(BeEmpty())

				By("Validation of createing IAM service account roles")

				output, err = iamSAaccountService.CreateIAMServiceAccountRole(
					"-c", clusterID,
					"--namespace", testNameSpace,
					"--role-name", iamServiceAccountCustomRoleName,
					"--attach-policy-arn", fmt.Sprintf("%s,%s", firstAttachPolicy, secondAttachPolicy),
					"--path", "/test/path/",
					"--permissions-boundary", permissionsBoundaryArn,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("at least one service account name is required"))

				output, err = iamSAaccountService.CreateIAMServiceAccountRole(
					"-c", clusterID,
					"--name", testName,
					"--namespace", testNameSpace,
					"--path", testRolePath,
					"--permissions-boundary", permissionsBoundaryArn,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("at least one policy ARN or inline policy must be specified"))

				output, err = iamSAaccountService.CreateIAMServiceAccountRole(
					"-c", clusterID,
					"--name", testName,
					"--namespace", testNameSpace,
					"--role-name", iamServiceAccountCustomRoleName,
					"--attach-policy-arn", fmt.Sprintf("%s,%s", firstAttachPolicy, secondAttachPolicy),
					"--path", "/test/path2/",
					"--permissions-boundary", permissionsBoundaryArn,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Role with same name but different path exists"))

				By("Validation of deleting IAM service account roles")
				output, err = iamSAaccountService.DeleteIAMServiceAccountRole(
					"-c", clusterID,
					"--name", "notexist",
					"--namespace", testNameSpace,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("does not exist"))

				output, err = iamSAaccountService.DeleteIAMServiceAccountRole(
					"-c", clusterID,
					"--role-name", "notexistrole",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("does not exist"))
			})

		It("can delete IAM service account role in manual mode - [id:84626]",
			labels.High, labels.Runtime.Day2,
			func() {
				testName := "iamsaname"
				testNameSpace := "iamsanamespace"
				iamServiceAccountCustomRoleName := helper.GenerateRandomName("rosa-e2e-iamserviceaccount", 3)
				testRolePath := "/test/path/"
				By("With multiple policies and custom name and all required flag")
				output, err := iamSAaccountService.CreateIAMServiceAccountRole(
					"-c", clusterID,
					"--name", testName,
					"--namespace", testNameSpace,
					"--role-name", iamServiceAccountCustomRoleName,
					"--attach-policy-arn", fmt.Sprintf("%s,%s", firstAttachPolicy, secondAttachPolicy),
					"--path", "/test/path/",
					"--permissions-boundary", permissionsBoundaryArn,
				)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					By("Delete the IAM service account role with custom name")
					output, err = iamSAaccountService.DeleteIAMServiceAccountRole(
						"-c", clusterID,
						"--role-name", iamServiceAccountCustomRoleName,
						"--mode", "auto",
						"-y",
					)
					Expect(err).ToNot(HaveOccurred())
				}()
				Expect(output.String()).To(ContainSubstring("Created IAM role"))
				Expect(output.String()).To(ContainSubstring("Attached policy"))

				By("With inline policy and auto-generated name and all required flag")
				output, err = iamSAaccountService.CreateIAMServiceAccountRole(
					"-c", clusterID,
					"--name", testName,
					"--namespace", testNameSpace,
					"--attach-policy-arn", fmt.Sprintf("%s,%s", firstAttachPolicy, secondAttachPolicy),
					"--path", testRolePath,
					"--permissions-boundary", permissionsBoundaryArn,
					"--inline-policy", fmt.Sprintf("file://%s", inlinePolicyFilePath),
				)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					By("Delete the IAM service account role with auto-generated name")
					output, err = iamSAaccountService.DeleteIAMServiceAccountRole(
						"-c", clusterID,
						"--name", testName,
						"--namespace", testNameSpace,
						"--mode", "auto",
						"-y",
					)
					Expect(err).ToNot(HaveOccurred())

				}()
				Expect(output.String()).To(ContainSubstring("Created IAM role"))
				Expect(output.String()).To(ContainSubstring("Attached policy"))
				Expect(output.String()).To(ContainSubstring("Attached inline policy"))

				By("With inline policy and custom name and all required flag")
				output, err = iamSAaccountService.DeleteIAMServiceAccountRole(
					"-c", clusterID,
					"--role-name", iamServiceAccountCustomRoleName,
					"--mode", "manual",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Run the following AWS CLI commands to delete the IAM role manually"))
				commands := helper.ExtractCommandsToDeleteIAMServiceAccount(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					Expect(err).To(BeNil())
				}

				By("Check the role with custom name deleted")
				output, err = iamSAaccountService.ListIAMServiceAccountRoles(
					"-c", clusterID,
				)
				Expect(err).ToNot(HaveOccurred())
				iamSARolesList, err := iamSAaccountService.ReflectIamServiceAccountList(output)
				Expect(err).ToNot(HaveOccurred())
				iamSaRoleWithGeneratedName := iamSARolesList.GetIAMServiceAccountRoleByName(iamServiceAccountCustomRoleName)
				Expect(iamSaRoleWithGeneratedName).To(Equal(rosacli.IamServiceAccountRole{}))

				By("Delete the IAM service account role with auto-generated name")
				output, err = iamSAaccountService.DeleteIAMServiceAccountRole(
					"-c", clusterID,
					"--name", testName,
					"--namespace", testNameSpace,
					"--mode", "manual",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Run the following AWS CLI commands to delete the IAM role manually"))
				commands = helper.ExtractCommandsToDeleteIAMServiceAccount(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					Expect(err).To(BeNil())
				}

				By("Check the role with auto-generated deleted")
				output, err = iamSAaccountService.ListIAMServiceAccountRoles(
					"-c", clusterID,
				)
				Expect(err).ToNot(HaveOccurred())
				iamSARolesList, err = iamSAaccountService.ReflectIamServiceAccountList(output)
				Expect(err).ToNot(HaveOccurred())
				clusterName, err := clusterService.GetClusterName(clusterID)
				Expect(err).ToNot(HaveOccurred())
				generatedIamSaRoleName := iamserviceaccount.GenerateRoleName(clusterName, testNameSpace, testName)
				iamSaRoleWithGeneratedName = iamSARolesList.GetIAMServiceAccountRoleByName(generatedIamSaRoleName)
				Expect(iamSaRoleWithGeneratedName).To(Equal(rosacli.IamServiceAccountRole{}))
			})

	})
})
