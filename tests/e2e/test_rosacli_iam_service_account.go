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
	"math/rand"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	. "github.com/openshift/rosa/tests/utils/helper"
)

// getPolicyARN returns the correct policy ARN based on FedRAMP status
func getPolicyARN(profile *handler.Profile, policyName string) string {
	if profile != nil && profile.ClusterConfig.FedRAMP {
		return fmt.Sprintf("arn:aws-us-gov:iam::aws:policy/%s", policyName)
	}
	return fmt.Sprintf("arn:aws:iam::aws:policy/%s", policyName)
}

var _ = Describe("IAM Service Account", labels.Feature.IAMServiceAccount, func() {
	defer GinkgoRecover()

	var (
		rosaClient *rosacli.Client
		profile    *handler.Profile
		// Track resources for cleanup
		serviceAccountRolesToClean []string
		testClusterID              string
		// Generate unique namespace for tests
		testNamespace string
	)

	BeforeEach(func() {
		By("Get the cluster")
		testClusterID = config.GetClusterID()
		Expect(testClusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

		By("Init the clients")
		rosaClient = rosacli.NewClient()

		By("Load the profile")
		profile = handler.LoadProfileYamlFileByENV()

		// Generate unique namespace for this test run
		rand.Seed(time.Now().UnixNano())
		testNamespace = fmt.Sprintf("test-ns-%d", rand.Intn(10000))
	})

	AfterEach(func() {
		By("Clean up service account roles")
		for _, roleArn := range serviceAccountRolesToClean {
			// Extract role name from ARN for deletion
			roleNameParts := strings.Split(roleArn, "/")
			if len(roleNameParts) > 1 {
				roleName := roleNameParts[len(roleNameParts)-1]
				_, err := rosaClient.IAMServiceAccount.DeleteIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--role-name", roleName,
					"--mode", "auto",
					"--approve",
				)
				Expect(err).ToNot(HaveOccurred())
			}
		}
		serviceAccountRolesToClean = []string{}
	})

	Context("IAM Service Account Management", func() {
		It("can create, list, describe, and delete IAM service account role",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Create an IAM service account role")
				serviceAccountName := "test-service-account"
				namespace := testNamespace
				policyArn := getPolicyARN(profile, "AmazonS3ReadOnlyAccess")

				createOutput, err := rosaClient.IAMServiceAccount.CreateIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--name", serviceAccountName,
					"--namespace", namespace,
					"--attach-policy-arn", policyArn,
					"--mode", "auto",
					"--approve",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(createOutput.String()).To(ContainSubstring("Created IAM role"))

				// Extract role ARN for cleanup
				createOutputStr := createOutput.String()
				lines := strings.Split(createOutputStr, "\n")
				var roleArn string
				for _, line := range lines {
					if (strings.Contains(line, "arn:aws:iam::") || strings.Contains(line, "arn:aws-us-gov:iam::")) && strings.Contains(line, ":role/") {
						// Extract ARN from the line - find start of ARN
						var arnStart int = -1
						if strings.Contains(line, "arn:aws:iam::") {
							arnStart = strings.Index(line, "arn:aws:iam::")
						} else if strings.Contains(line, "arn:aws-us-gov:iam::") {
							arnStart = strings.Index(line, "arn:aws-us-gov:iam::")
						}

						if arnStart != -1 {
							arnEnd := strings.Index(line[arnStart:], " ")
							if arnEnd == -1 {
								arnEnd = len(line) - arnStart
							}
							roleArn = strings.Trim(line[arnStart:arnStart+arnEnd], "'\"")
							break
						}
					}
				}
				Expect(roleArn).ToNot(BeEmpty(), "Failed to extract role ARN from create output")
				serviceAccountRolesToClean = append(serviceAccountRolesToClean, roleArn)

				By("List IAM service account roles")
				listOutput, err := rosaClient.IAMServiceAccount.ListIAMServiceAccountRoles(
					"--cluster", testClusterID,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(listOutput.String()).To(ContainSubstring(serviceAccountName))
				Expect(listOutput.String()).To(ContainSubstring(namespace))

				By("List with namespace filter")
				listNamespaceOutput, err := rosaClient.IAMServiceAccount.ListIAMServiceAccountRoles(
					"--cluster", testClusterID,
					"--namespace", namespace,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(listNamespaceOutput.String()).To(ContainSubstring(serviceAccountName))

				By("Describe the IAM service account role")
				describeOutput, err := rosaClient.IAMServiceAccount.DescribeIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--name", serviceAccountName,
					"--namespace", namespace,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(describeOutput.String()).To(ContainSubstring(serviceAccountName))
				Expect(describeOutput.String()).To(ContainSubstring(namespace))
				Expect(describeOutput.String()).To(ContainSubstring(policyArn))
				Expect(describeOutput.String()).To(ContainSubstring("Trust Policy"))

				By("Delete the IAM service account role")
				deleteOutput, err := rosaClient.IAMServiceAccount.DeleteIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--name", serviceAccountName,
					"--namespace", namespace,
					"--mode", "auto",
					"--approve",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(deleteOutput.String()).To(ContainSubstring("Successfully deleted"))

				// Remove from cleanup list since we deleted it
				serviceAccountRolesToClean = []string{}

				By("Verify role is deleted")
				listAfterDeleteOutput, err := rosaClient.IAMServiceAccount.ListIAMServiceAccountRoles(
					"--cluster", testClusterID,
					"--namespace", namespace,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(listAfterDeleteOutput.String()).ToNot(ContainSubstring(serviceAccountName))
			})

		It("can create role with custom name and multiple policies",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Create an IAM service account role with custom name and multiple policies")
				serviceAccountName := "multi-policy-app"
				namespace := testNamespace + "-multi"
				customRoleName := "custom-test-role-" + GenerateRandomStringWithSymbols(5)
				policyArns := []string{
					getPolicyARN(profile, "AmazonS3ReadOnlyAccess"),
					getPolicyARN(profile, "AmazonEC2ReadOnlyAccess"),
				}

				createArgs := []string{
					"--cluster", testClusterID,
					"--name", serviceAccountName,
					"--namespace", namespace,
					"--role-name", customRoleName,
					"--mode", "auto",
					"--approve",
				}
				// Add multiple policy ARNs
				for _, policyArn := range policyArns {
					createArgs = append(createArgs, "--attach-policy-arn", policyArn)
				}
				createOutput, err := rosaClient.IAMServiceAccount.CreateIAMServiceAccountRole(createArgs...)
				Expect(err).ToNot(HaveOccurred())
				Expect(createOutput.String()).To(ContainSubstring(customRoleName))

				// Track for cleanup using the custom role name - construct ARN based on profile
				roleArnPrefix := "arn:aws:iam::"
				if profile != nil && profile.ClusterConfig.FedRAMP {
					roleArnPrefix = "arn:aws-us-gov:iam::"
				}
				serviceAccountRolesToClean = append(serviceAccountRolesToClean, roleArnPrefix+"123456789012:role/"+customRoleName)

				By("Describe role and verify multiple policies")
				describeOutput, err := rosaClient.IAMServiceAccount.DescribeIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--role-name", customRoleName,
				)
				Expect(err).ToNot(HaveOccurred())

				describeStr := describeOutput.String()
				for _, policyArn := range policyArns {
					Expect(describeStr).To(ContainSubstring(policyArn))
				}

				By("List with output format JSON")
				listJSONOutput, err := rosaClient.IAMServiceAccount.ListIAMServiceAccountRoles(
					"--cluster", testClusterID,
					"--output", "json",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(listJSONOutput.String()).To(ContainSubstring(customRoleName))
				Expect(listJSONOutput.String()).To(ContainSubstring("\"roleName\""))

				By("Clean up - Delete using custom role name")
				deleteOutput, err := rosaClient.IAMServiceAccount.DeleteIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--role-name", customRoleName,
					"--mode", "auto",
					"--approve",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(deleteOutput.String()).To(ContainSubstring("Successfully deleted"))

				serviceAccountRolesToClean = []string{}
			})

		It("can handle manual mode operations",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Create IAM service account role in manual mode")
				serviceAccountName := "manual-test-app"
				namespace := testNamespace + "-manual"
				policyArn := getPolicyARN(profile, "AmazonS3ReadOnlyAccess")

				createOutput, err := rosaClient.IAMServiceAccount.CreateIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--name", serviceAccountName,
					"--namespace", namespace,
					"--attach-policy-arn", policyArn,
					"--mode", "manual",
				)
				Expect(err).ToNot(HaveOccurred())

				// Manual mode should output AWS CLI commands
				createStr := createOutput.String()
				Expect(createStr).To(ContainSubstring("aws iam create-role"))
				Expect(createStr).To(ContainSubstring("aws iam attach-role-policy"))
				Expect(createStr).To(ContainSubstring(policyArn))

				By("Test manual delete mode")
				expectedRoleName := fmt.Sprintf("%s-%s-%s-role", testClusterID, namespace, serviceAccountName)
				deleteOutput, err := rosaClient.IAMServiceAccount.DeleteIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--name", serviceAccountName,
					"--namespace", namespace,
					"--mode", "manual",
				)
				Expect(err).ToNot(HaveOccurred())

				// Manual mode should output AWS CLI commands
				deleteStr := deleteOutput.String()
				Expect(deleteStr).To(ContainSubstring("aws iam detach-role-policy"))
				Expect(deleteStr).To(ContainSubstring("aws iam delete-role"))
				Expect(deleteStr).To(ContainSubstring(expectedRoleName))
			})

		It("can validate input parameters",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Test invalid service account name")
				_, err := rosaClient.IAMServiceAccount.CreateIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--name", "Invalid-Name-With-Capitals",
					"--namespace", "test",
					"--attach-policy-arn", getPolicyARN(profile, "AmazonS3ReadOnlyAccess"),
					"--mode", "auto",
				)
				Expect(err).To(HaveOccurred())

				By("Test invalid namespace name")
				_, err = rosaClient.IAMServiceAccount.CreateIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--name", "test-app",
					"--namespace", "kube-system", // Reserved namespace
					"--attach-policy-arn", getPolicyARN(profile, "AmazonS3ReadOnlyAccess"),
					"--mode", "auto",
				)
				Expect(err).To(HaveOccurred())

				By("Test missing required parameters")
				_, err = rosaClient.IAMServiceAccount.CreateIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--name", "test-app",
					// Missing namespace and policies
					"--mode", "auto",
				)
				Expect(err).To(HaveOccurred())
			})

		It("can handle non-existent resources gracefully",
			labels.Low, labels.Runtime.Day2,
			func() {
				By("Try to describe non-existent role")
				_, err := rosaClient.IAMServiceAccount.DescribeIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--name", "non-existent-app",
					"--namespace", "non-existent-namespace",
				)
				Expect(err).To(HaveOccurred())

				By("Try to delete non-existent role")
				deleteOutput, err := rosaClient.IAMServiceAccount.DeleteIAMServiceAccountRole(
					"--cluster", testClusterID,
					"--role-name", "non-existent-role",
					"--mode", "auto",
				)
				// Should not error, but should indicate role doesn't exist
				Expect(err).ToNot(HaveOccurred())
				Expect(deleteOutput.String()).To(ContainSubstring("does not exist"))

				By("List roles for cluster with no service account roles")
				// This should work and return empty results
				listOutput, err := rosaClient.IAMServiceAccount.ListIAMServiceAccountRoles(
					"--cluster", testClusterID,
					"--namespace", "empty-namespace",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(listOutput.String()).To(ContainSubstring("No IAM service account roles found"))
			})
	})
})
