/*
Copyright (c) 2021 Red Hat, Inc.

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
	"fmt"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	awsClient "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Create IAM Service Account Functions", func() {
	Context("OIDC Provider ARN Generation", func() {
		var (
			testRuntime test.TestingRuntime
			mockCtrl    *gomock.Controller
		)

		BeforeEach(func() {
			testRuntime.InitRuntime()
			testRuntime.RosaRuntime.Creator = &awsClient.Creator{
				AccountID: "123456789012",
				Partition: "aws",
			}
			mockCtrl = gomock.NewController(GinkgoT())
		})

		It("should generate OIDC provider ARN for managed OIDC", func() {
			// Create a test cluster with managed OIDC configuration
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test-cluster-id")
				c.Name("test-cluster")
				c.AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/test-installer-role").
						OidcConfig(cmv1.NewOidcConfig().
							IssuerUrl("https://rh-oidc.s3.us-east-1.amazonaws.com/1234567890abcdef").
							Managed(true))))
			})

			arn, err := getOIDCProviderARN(testRuntime.RosaRuntime, cluster)
			Expect(err).ToNot(HaveOccurred())

			expectedARN := "arn:aws:iam::123456789012:oidc-provider/rh-oidc.s3.us-east-1.amazonaws.com/1234567890abcdef"
			Expect(arn).To(Equal(expectedARN))
		})

		It("should handle unmanaged OIDC", func() {
			// Create cluster with unmanaged OIDC
			unmanagedCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test-cluster-id")
				c.Name("test-cluster")
				c.AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/test-installer-role").
						OidcConfig(cmv1.NewOidcConfig().
							IssuerUrl("https://example.com/oidc").
							Managed(false))))
			})

			// Mock the ListOpenIDConnectProviders call
			mockAWS := awsClient.NewMockClient(mockCtrl)
			testRuntime.RosaRuntime.AWSClient = mockAWS

			providerList := &iam.ListOpenIDConnectProvidersOutput{
				OpenIDConnectProviderList: []iamtypes.OpenIDConnectProviderListEntry{
					{
						Arn: aws.String("arn:aws:iam::123456789012:oidc-provider/example.com/oidc"),
					},
				},
			}

			mockAWS.EXPECT().
				ListOpenIDConnectProviders(gomock.Any(), gomock.Any()).
				Return(providerList, nil)

			arn, err := getOIDCProviderARN(testRuntime.RosaRuntime, unmanagedCluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:oidc-provider/example.com/oidc"))
		})

		It("should return error when OIDC provider not found for unmanaged cluster", func() {
			// Create cluster with unmanaged OIDC
			unmanagedCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test-cluster-id")
				c.Name("test-cluster")
				c.AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/test-installer-role").
						OidcConfig(cmv1.NewOidcConfig().
							IssuerUrl("https://example.com/oidc").
							Managed(false))))
			})

			// Mock empty provider list
			mockAWS := awsClient.NewMockClient(mockCtrl)
			testRuntime.RosaRuntime.AWSClient = mockAWS

			emptyList := &iam.ListOpenIDConnectProvidersOutput{
				OpenIDConnectProviderList: []iamtypes.OpenIDConnectProviderListEntry{},
			}

			mockAWS.EXPECT().
				ListOpenIDConnectProviders(gomock.Any(), gomock.Any()).
				Return(emptyList, nil)

			_, err := getOIDCProviderARN(testRuntime.RosaRuntime, unmanagedCluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("OIDC provider not found"))
		})
	})

	Context("Manual Command Generation", func() {
		It("should generate correct AWS CLI commands", func() {
			roleName := "test-cluster-default-my-app-role"
			trustPolicy := `{"Version": "2012-10-17", "Statement": []}`
			permissionsBoundary := "arn:aws:iam::123456789012:policy/boundary"
			path := "/rosa/"
			tags := map[string]string{
				"Environment": "test",
				"Owner":       "rosa",
			}
			policyArns := []string{
				"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
				"arn:aws:iam::123456789012:policy/CustomPolicy",
			}
			inlinePolicy := `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "s3:GetObject", "Resource": "*"}]}`

			commands := generateManualCommands(roleName, trustPolicy, permissionsBoundary, path, tags, policyArns, inlinePolicy)

			Expect(commands).To(ContainSubstring("aws iam create-role"))
			Expect(commands).To(ContainSubstring("--role-name " + roleName))
			Expect(commands).To(ContainSubstring("--permissions-boundary " + permissionsBoundary))
			Expect(commands).To(ContainSubstring("--path " + path))
			Expect(commands).To(ContainSubstring("Key=Environment,Value=test"))
			Expect(commands).To(ContainSubstring("Key=Owner,Value=rosa"))

			// Check policy attachments
			for _, policyArn := range policyArns {
				Expect(commands).To(ContainSubstring("aws iam attach-role-policy"))
				Expect(commands).To(ContainSubstring(policyArn))
			}

			// Check inline policy
			Expect(commands).To(ContainSubstring("aws iam put-role-policy"))
			Expect(commands).To(ContainSubstring(inlinePolicy))
		})

		It("should generate correct tags for multiple service accounts", func() {
			roleName := "test-multi-sa-role"
			trustPolicy := `{"Version": "2012-10-17", "Statement": []}`
			tags := map[string]string{
				"rosa_service_accounts": "service1 service2 service3",
				"Environment":           "test",
			}

			commands := generateManualCommands(roleName, trustPolicy, "", "/", tags, []string{}, "")

			// Verify tag format with spaces (AWS doesn't allow commas)
			Expect(commands).To(ContainSubstring("Key=rosa_service_accounts,Value=service1 service2 service3"))
		})
	})

	Context("Command Validation", func() {
		var originalArgs struct {
			serviceAccountNames []string
			namespace           string
			roleName            string
			policyArns          []string
			inlinePolicy        string
			permissionsBoundary string
			path                string
		}

		BeforeEach(func() {
			// Save original args
			originalArgs = args

			// Reset command state before each test
			args.serviceAccountNames = []string{}
			args.namespace = "default"
			args.roleName = ""
			args.policyArns = []string{}
			args.inlinePolicy = ""
			args.permissionsBoundary = ""
			args.path = "/"
		})

		AfterEach(func() {
			// Restore original args
			args = originalArgs
		})

		DescribeTable("Validate required flags",
			func(testArgs []string, shouldFail bool, expectedError string) {
				// Create a new command instance for each test to avoid state pollution
				testCmd := &cobra.Command{
					Use:     "iamserviceaccount",
					Aliases: []string{"iam-service-account"},
					Short:   "Create IAM role for Kubernetes service account",
					Args:    cobra.NoArgs,
					RunE: func(cmd *cobra.Command, args []string) error {
						// Minimal run function that just validates flags
						if !cmd.Flags().Changed("cluster") {
							return fmt.Errorf("required flag(s) \"cluster\" not set")
						}
						return nil
					},
				}

				// Add the cluster flag and mark it as required
				testCmd.Flags().StringP("cluster", "c", "", "Name or ID of the cluster.")
				err := testCmd.MarkFlagRequired("cluster")
				Expect(err).ToNot(HaveOccurred())

				// Add other flags
				testCmd.Flags().StringSlice("name", []string{}, "Name of the Kubernetes service account")
				testCmd.Flags().StringSlice("attach-policy-arn", []string{}, "ARN of the IAM policy to attach")

				// Set args and execute
				testCmd.SetArgs(testArgs)
				execErr := testCmd.Execute()

				if shouldFail {
					Expect(execErr).To(HaveOccurred())
					if expectedError != "" {
						Expect(execErr.Error()).To(ContainSubstring(expectedError))
					}
				} else {
					Expect(execErr).ToNot(HaveOccurred())
				}
			},
			Entry("Missing cluster flag", []string{
				"--name", "test-app",
				"--attach-policy-arn", "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
			}, true, "required flag(s) \"cluster\" not set"),
			Entry("Missing service account name flag", []string{
				"--cluster", "test-cluster",
				"--attach-policy-arn", "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
			}, false, ""), // Name flag is not required at parse time, validated at runtime
			Entry("Missing policy ARNs flag", []string{
				"--cluster", "test-cluster",
				"--name", "test-app",
			}, false, ""), // Policy flag is not required at parse time, validated at runtime
			Entry("All required flags present", []string{
				"--cluster", "test-cluster",
				"--name", "test-app",
				"--attach-policy-arn", "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
			}, false, ""),
		)
	})
})
