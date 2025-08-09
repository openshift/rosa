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

package iamserviceaccount

import (
	"net/http"
	"os"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	awsClient "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/iamserviceaccount"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Delete IAM Service Account", Ordered, func() {
	var testRuntime test.TestingRuntime

	BeforeAll(func() {
		// Set AWS credentials to mock values to prevent AWS client initialization errors
		os.Setenv("AWS_ACCESS_KEY_ID", "mock-access-key")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "mock-secret-key")
		os.Setenv("AWS_REGION", "us-east-1")
	})

	AfterAll(func() {
		// Clean up environment variables
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_REGION")
	})

	BeforeEach(func() {
		testRuntime.InitRuntime()
		testRuntime.RosaRuntime.Creator = &awsClient.Creator{
			AccountID: "123456789012",
			Partition: "aws",
		}
	})

	Context("Command Validation", func() {
		It("should validate service account name format", func() {
			// Test valid service account name
			err := iamserviceaccount.ValidateServiceAccountName("my-app")
			Expect(err).ToNot(HaveOccurred())

			// Test invalid service account name
			err = iamserviceaccount.ValidateServiceAccountName("my_app")
			Expect(err).To(HaveOccurred())
		})

		It("should validate namespace name format", func() {
			// Test valid namespace name
			err := iamserviceaccount.ValidateNamespaceName("default")
			Expect(err).ToNot(HaveOccurred())

			// Test invalid namespace name
			err = iamserviceaccount.ValidateNamespaceName("_invalid")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Role Discovery", func() {
		var mockOCMServer *ghttp.Server

		BeforeEach(func() {
			// Create a mock OCM server
			mockOCMServer = ghttp.NewServer()
			os.Setenv("OCM_URL", mockOCMServer.URL())

			// Mock cluster response
			cluster, err := cmv1.NewCluster().
				ID("test-cluster-id").
				Name("test-cluster").
				AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/test-installer-role"))).
				Build()
			Expect(err).ToNot(HaveOccurred())

			testRuntime.ApiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/test-cluster"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, cluster),
				),
			)
		})

		AfterEach(func() {
			mockOCMServer.Close()
		})

		PIt("should generate role name from service account details (requires AWS setup)", func() {
			mockCtrl := gomock.NewController(GinkgoT())
			defer mockCtrl.Finish()

			mockAWS := testRuntime.RosaRuntime.AWSClient.(*awsClient.MockClient)

			expectedRoleName := "test-cluster-default-test-app-role"

			// Mock CheckRoleExists call
			mockAWS.EXPECT().
				CheckRoleExists(expectedRoleName).
				Return(true, "arn:aws:iam::123456789012:role/"+expectedRoleName, nil)

			// Mock GetServiceAccountRoleDetails call
			role := &iamtypes.Role{
				RoleName: aws.String(expectedRoleName),
				Arn:      aws.String("arn:aws:iam::123456789012:role/" + expectedRoleName),
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String(iamserviceaccount.RoleTypeTagKey),
						Value: aws.String(iamserviceaccount.ServiceAccountRoleType),
					},
					{
						Key:   aws.String(iamserviceaccount.ClusterTagKey),
						Value: aws.String("test-cluster"),
					},
					{
						Key:   aws.String(iamserviceaccount.NamespaceTagKey),
						Value: aws.String("default"),
					},
					{
						Key:   aws.String(iamserviceaccount.ServiceAccountTagKey),
						Value: aws.String("test-app"),
					},
				},
			}

			attachedPolicies := []iamtypes.AttachedPolicy{
				{
					PolicyArn:  aws.String("arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"),
					PolicyName: aws.String("AmazonS3ReadOnlyAccess"),
				},
			}

			inlinePolicies := []string{}

			mockAWS.EXPECT().
				GetServiceAccountRoleDetails(expectedRoleName).
				Return(role, attachedPolicies, inlinePolicies, nil)

			// Mock DeleteServiceAccountRole call
			mockAWS.EXPECT().
				DeleteServiceAccountRole(expectedRoleName).
				Return(nil)

			Cmd.SetArgs([]string{
				"--cluster", "test-cluster",
				"--name", "test-app",
				"--namespace", "default",
				"--mode", "auto",
				"--yes",
			})

			err := Cmd.Execute()
			Expect(err).ToNot(HaveOccurred())
		})

		PIt("should use explicit role name when provided (requires AWS setup)", func() {
			mockCtrl := gomock.NewController(GinkgoT())
			defer mockCtrl.Finish()

			mockAWS := testRuntime.RosaRuntime.AWSClient.(*awsClient.MockClient)

			explicitRoleName := "my-custom-role"

			// Mock CheckRoleExists call
			mockAWS.EXPECT().
				CheckRoleExists(explicitRoleName).
				Return(true, "arn:aws:iam::123456789012:role/"+explicitRoleName, nil)

			// Mock GetServiceAccountRoleDetails call
			role := &iamtypes.Role{
				RoleName: aws.String(explicitRoleName),
				Arn:      aws.String("arn:aws:iam::123456789012:role/" + explicitRoleName),
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String(iamserviceaccount.RoleTypeTagKey),
						Value: aws.String(iamserviceaccount.ServiceAccountRoleType),
					},
				},
			}

			mockAWS.EXPECT().
				GetServiceAccountRoleDetails(explicitRoleName).
				Return(role, []iamtypes.AttachedPolicy{}, []string{}, nil)

			// Mock DeleteServiceAccountRole call
			mockAWS.EXPECT().
				DeleteServiceAccountRole(explicitRoleName).
				Return(nil)

			Cmd.SetArgs([]string{
				"--cluster", "test-cluster",
				"--role-name", explicitRoleName,
				"--mode", "auto",
				"--yes",
			})

			err := Cmd.Execute()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Manual Command Generation", func() {
		It("should generate correct delete commands", func() {
			roleName := "test-role"
			attachedPolicies := []iamtypes.AttachedPolicy{
				{
					PolicyArn:  aws.String("arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"),
					PolicyName: aws.String("AmazonS3ReadOnlyAccess"),
				},
				{
					PolicyArn:  aws.String("arn:aws:iam::123456789012:policy/CustomPolicy"),
					PolicyName: aws.String("CustomPolicy"),
				},
			}
			inlinePolicies := []string{"InlinePolicy1", "InlinePolicy2"}

			commands := generateManualDeleteCommands(roleName, attachedPolicies, inlinePolicies)

			// Check detach policy commands
			Expect(commands).To(ContainSubstring("aws iam detach-role-policy"))
			Expect(commands).To(ContainSubstring("arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"))
			Expect(commands).To(ContainSubstring("arn:aws:iam::123456789012:policy/CustomPolicy"))

			// Check delete inline policy commands
			Expect(commands).To(ContainSubstring("aws iam delete-role-policy"))
			Expect(commands).To(ContainSubstring("InlinePolicy1"))
			Expect(commands).To(ContainSubstring("InlinePolicy2"))

			// Check delete role command
			Expect(commands).To(ContainSubstring("aws iam delete-role --role-name " + roleName))
		})

		It("should handle role with no policies", func() {
			roleName := "test-role"
			commands := generateManualDeleteCommands(roleName, []iamtypes.AttachedPolicy{}, []string{})

			// Should only contain the delete role command
			Expect(commands).To(ContainSubstring("aws iam delete-role --role-name " + roleName))
			Expect(commands).ToNot(ContainSubstring("detach-role-policy"))
			Expect(commands).ToNot(ContainSubstring("delete-role-policy"))
		})
	})

	PContext("Role Validation (requires AWS setup)", func() {
		BeforeEach(func() {
			// Mock cluster response
			cluster, err := cmv1.NewCluster().
				ID("test-cluster-id").
				Name("test-cluster").
				AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/test-installer-role"))).
				Build()
			Expect(err).ToNot(HaveOccurred())

			testRuntime.ApiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/test-cluster"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, cluster),
				),
			)
		})

		It("should handle non-existent role gracefully", func() {
			mockCtrl := gomock.NewController(GinkgoT())
			defer mockCtrl.Finish()

			mockAWS := testRuntime.RosaRuntime.AWSClient.(*awsClient.MockClient)

			roleName := "non-existent-role"

			// Mock CheckRoleExists call returning false
			mockAWS.EXPECT().
				CheckRoleExists(roleName).
				Return(false, "", nil)

			Cmd.SetArgs([]string{
				"--cluster", "test-cluster",
				"--role-name", roleName,
				"--mode", "auto",
			})

			err := Cmd.Execute()
			Expect(err).ToNot(HaveOccurred()) // Should exit gracefully, not error
		})

		It("should warn about non-service-account roles", func() {
			mockCtrl := gomock.NewController(GinkgoT())
			defer mockCtrl.Finish()

			mockAWS := testRuntime.RosaRuntime.AWSClient.(*awsClient.MockClient)

			roleName := "regular-role"

			// Mock CheckRoleExists call
			mockAWS.EXPECT().
				CheckRoleExists(roleName).
				Return(true, "arn:aws:iam::123456789012:role/"+roleName, nil)

			// Mock GetServiceAccountRoleDetails call - role without service account tags
			role := &iamtypes.Role{
				RoleName: aws.String(roleName),
				Arn:      aws.String("arn:aws:iam::123456789012:role/" + roleName),
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String("Environment"),
						Value: aws.String("test"),
					},
				}, // Missing service account role type tag
			}

			mockAWS.EXPECT().
				GetServiceAccountRoleDetails(roleName).
				Return(role, []iamtypes.AttachedPolicy{}, []string{}, nil)

			Cmd.SetArgs([]string{
				"--cluster", "test-cluster",
				"--role-name", roleName,
				"--mode", "manual", // Use manual mode to avoid confirmation prompts
			})

			err := Cmd.Execute()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
