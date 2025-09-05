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
	"context"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	iamServiceAccountOpts "github.com/openshift/rosa/pkg/options/iamserviceaccount"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Create IAM Service Account", func() {
	var (
		t       *test.TestingRuntime
		ctrl    *gomock.Controller
		mockAWS *aws.MockClient
		cmd     *cobra.Command
	)

	BeforeEach(func() {
		t = test.NewTestRuntime()
		ctrl = gomock.NewController(GinkgoT())
		mockAWS = aws.NewMockClient(ctrl)
		t.RosaRuntime.AWSClient = mockAWS

		cmd = NewCreateIamServiceAccountCommand()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("CreateIamServiceAccountRunner", func() {
		Context("with valid cluster", func() {
			It("should create a service account role successfully", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com"))))
				})

				t.SetCluster(cluster.ID(), cluster)

				// Mock GetCreator to return standard AWS creator
				mockAWS.EXPECT().
					GetCreator().
					Return(&aws.Creator{
						ARN:        "arn:aws:iam::123456789012:user/test-user",
						AccountID:  "123456789012",
						IsSTS:      false,
						IsGovcloud: false,
						Partition:  "aws",
					}, nil)

				providers := []aws.OidcProviderOutput{
					{
						Arn: "arn:aws:iam::123456789012:oidc-provider/test.example.com",
					},
				}

				mockAWS.EXPECT().
					ListOidcProviders(cluster.ID(), cluster.AWS().STS().OidcConfig()).
					Return(providers, nil)

				mockAWS.EXPECT().
					EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), "", "", gomock.Any(), gomock.Any(), false).
					Return("arn:aws:iam::123456789012:role/test-cluster-default-test-sa", nil)

				mockAWS.EXPECT().
					AttachRolePolicy(gomock.Any(), gomock.Any(), "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess").
					Return(nil)

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					PolicyArns:          []string{"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"},
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should fail with non-STS cluster", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS()) // No STS configuration
				})

				t.SetCluster(cluster.ID(), cluster)

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					PolicyArns:          []string{"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"},
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not an STS cluster"))
			})

			It("should fail when no policies are provided", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role")))
				})

				t.SetCluster(cluster.ID(), cluster)

				// Mock GetCreator to return standard AWS creator
				mockAWS.EXPECT().
					GetCreator().
					Return(&aws.Creator{
						ARN:        "arn:aws:iam::123456789012:user/test-user",
						AccountID:  "123456789012",
						IsSTS:      false,
						IsGovcloud: false,
						Partition:  "aws",
					}, nil)

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					// No policies provided
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at least one policy ARN or inline policy must be specified"))
			})

			It("should create a service account role with inline policy", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com"))))
				})

				t.SetCluster(cluster.ID(), cluster)

				// Mock GetCreator to return standard AWS creator
				mockAWS.EXPECT().
					GetCreator().
					Return(&aws.Creator{
						ARN:        "arn:aws:iam::123456789012:user/test-user",
						AccountID:  "123456789012",
						IsSTS:      false,
						IsGovcloud: false,
						Partition:  "aws",
					}, nil)

				providers := []aws.OidcProviderOutput{
					{
						Arn: "arn:aws:iam::123456789012:oidc-provider/test.example.com",
					},
				}

				mockAWS.EXPECT().
					ListOidcProviders(cluster.ID(), cluster.AWS().STS().OidcConfig()).
					Return(providers, nil)

				mockAWS.EXPECT().
					EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), "", "", gomock.Any(), gomock.Any(), false).
					Return("arn:aws:iam::123456789012:role/test-cluster-default-test-sa", nil)

				mockAWS.EXPECT().
					PutRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					InlinePolicy:        `{"Version": "2012-10-17", "Statement": []}`,
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle FedRAMP environment correctly", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws-us-gov:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.gov.example.com"))))
				})

				t.SetCluster(cluster.ID(), cluster)

				// Mock GetCreator to return GovCloud creator
				mockAWS.EXPECT().
					GetCreator().
					Return(&aws.Creator{
						ARN:        "arn:aws-us-gov:iam::123456789012:user/test-user",
						AccountID:  "123456789012",
						IsSTS:      false,
						IsGovcloud: true,
						Partition:  "aws-us-gov",
					}, nil)

				providers := []aws.OidcProviderOutput{
					{
						Arn: "arn:aws-us-gov:iam::123456789012:oidc-provider/test.gov.example.com",
					},
				}

				mockAWS.EXPECT().
					ListOidcProviders(cluster.ID(), cluster.AWS().STS().OidcConfig()).
					Return(providers, nil)

				mockAWS.EXPECT().
					EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), "", "", gomock.Any(), gomock.Any(), false).
					Return("arn:aws-us-gov:iam::123456789012:role/test-cluster-default-test-sa", nil)

				mockAWS.EXPECT().
					AttachRolePolicy(gomock.Any(), gomock.Any(), "arn:aws-us-gov:iam::aws:policy/AmazonS3ReadOnlyAccess").
					Return(nil)

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					PolicyArns:          []string{"arn:aws-us-gov:iam::aws:policy/AmazonS3ReadOnlyAccess"},
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("getOIDCProviderARN", func() {
		It("should return provider ARN for managed cluster", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test-cluster-id")
				c.Name("test-cluster")
				c.AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						OidcConfig(cmv1.NewOidcConfig().
							ID("test-oidc-id").
							IssuerUrl("https://test.example.com"))))
			})

			providers := []aws.OidcProviderOutput{
				{
					Arn: "arn:aws:iam::123456789012:oidc-provider/test.example.com",
				},
			}

			mockAWS.EXPECT().
				ListOidcProviders(cluster.ID(), cluster.AWS().STS().OidcConfig()).
				Return(providers, nil)

			arn, err := getOIDCProviderARN(t.RosaRuntime, cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:oidc-provider/test.example.com"))
		})

		It("should return error when no OIDC provider found", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test-cluster-id")
				c.Name("test-cluster")
				c.AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						OidcConfig(cmv1.NewOidcConfig().
							ID("test-oidc-id").
							IssuerUrl("https://test.example.com"))))
			})

			mockAWS.EXPECT().
				ListOidcProviders(cluster.ID(), cluster.AWS().STS().OidcConfig()).
				Return([]aws.OidcProviderOutput{}, nil)

			_, err := getOIDCProviderARN(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no OIDC provider found"))
		})

		It("should handle classic OIDC scenario (no oidcConfig but has OIDCEndpointURL)", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test-cluster-id")
				c.Name("test-cluster")
				c.AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						OIDCEndpointURL("https://test.example.com")))
			})

			mockAWS.EXPECT().
				GetCreator().
				Return(&aws.Creator{
					ARN:        "arn:aws:iam::123456789012:user/test-user",
					AccountID:  "123456789012",
					IsSTS:      false,
					IsGovcloud: false,
					Partition:  "aws",
				}, nil)

			mockAWS.EXPECT().
				HasOpenIDConnectProvider("https://test.example.com", "aws", "123456789012").
				Return(true, nil)

			arn, err := getOIDCProviderARN(t.RosaRuntime, cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:oidc-provider/test.example.com"))
		})

		It("should handle BYO OIDC scenario (cluster ID search fails but endpoint URL works)", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test-cluster-id")
				c.Name("test-cluster")
				c.AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						OIDCEndpointURL("https://byo.example.com").
						OidcConfig(cmv1.NewOidcConfig().
							ID("test-oidc-id").
							IssuerUrl("https://byo.example.com"))))
			})

			// First try cluster ID-based search which fails for BYO OIDC
			mockAWS.EXPECT().
				ListOidcProviders(cluster.ID(), cluster.AWS().STS().OidcConfig()).
				Return([]aws.OidcProviderOutput{}, nil)

			// Then fallback to endpoint URL-based search
			mockAWS.EXPECT().
				GetCreator().
				Return(&aws.Creator{
					ARN:        "arn:aws:iam::123456789012:user/test-user",
					AccountID:  "123456789012",
					IsSTS:      false,
					IsGovcloud: false,
					Partition:  "aws",
				}, nil)

			mockAWS.EXPECT().
				HasOpenIDConnectProvider("https://byo.example.com", "aws", "123456789012").
				Return(true, nil)

			arn, err := getOIDCProviderARN(t.RosaRuntime, cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:oidc-provider/byo.example.com"))
		})

		It("should return error when cluster has no OIDC configuration at all", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test-cluster-id")
				c.Name("test-cluster")
				c.AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS()))
			})

			_, err := getOIDCProviderARN(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not have OIDC configuration"))
		})
	})
})
