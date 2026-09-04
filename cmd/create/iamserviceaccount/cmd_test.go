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
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	rosaerrors "github.com/openshift/rosa/pkg/errors"
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
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
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
					GetOpenIDConnectProviderByOidcEndpointUrl("https://test.example.com").
					Return(providers[0].Arn, nil)

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

				// No GetCreator mock: the missing policies must be caught by
				// the CLI's fail-fast check before that (AWS API) call ever
				// happens.

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					// No policies provided
				}
				testRunner := CreateIamServiceAccountRunner(options)

				var out bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&out)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at least one policy ARN or inline policy must be specified"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())
				Expect(out.String()).To(ContainSubstring("Usage:"))
			})

			It("should fail when no service account names are provided", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role")))
				})

				t.SetCluster(cluster.ID(), cluster)

				// No GetCreator mock: the missing service account names must
				// be caught by the CLI's fail-fast check before that (AWS
				// API) call ever happens.

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					Namespace:  "default",
					PolicyArns: []string{"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"},
					// No service account names provided
				}
				testRunner := CreateIamServiceAccountRunner(options)

				var out bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&out)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at least one service account name is required"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())
				Expect(out.String()).To(ContainSubstring("Usage:"))
			})

			It("should classify a malformed policy ARN as a validation error without showing usage", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
				})

				t.SetCluster(cluster.ID(), cluster)

				// No GetCreator or GetOpenIDConnectProviderByOidcEndpointUrl
				// mock: the malformed ARN must be caught by the CLI's
				// fail-fast check before either (AWS API) call ever happens.

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					PolicyArns:          []string{"not-a-valid-arn"},
				}
				testRunner := CreateIamServiceAccountRunner(options)

				var out bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&out)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("policy ARN at index 0 is invalid"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())
			})

			It("should classify a non-IAM-policy ARN as a validation error", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
				})

				t.SetCluster(cluster.ID(), cluster)

				// No GetCreator or GetOpenIDConnectProviderByOidcEndpointUrl
				// mock: the non-IAM-policy ARN must be caught by the CLI's
				// fail-fast check before either (AWS API) call ever happens.

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					PolicyArns:          []string{"arn:aws:s3:::my-bucket"},
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("policy ARN at index 0 is not an IAM policy ARN"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())
			})

			It("should classify a policy ARN with no policy name as a validation error", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
				})

				t.SetCluster(cluster.ID(), cluster)

				// No GetCreator or GetOpenIDConnectProviderByOidcEndpointUrl
				// mock: the malformed ARN must be caught by the CLI's
				// fail-fast check before either (AWS API) call ever happens.

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					PolicyArns:          []string{"arn:aws:iam::123456789012:policy/"},
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("policy ARN at index 0 is not an IAM policy ARN"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())
			})

			It("should classify a policy ARN with a trailing slash as a validation error", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
				})

				t.SetCluster(cluster.ID(), cluster)

				// No GetCreator or GetOpenIDConnectProviderByOidcEndpointUrl
				// mock: the malformed ARN must be caught by the CLI's
				// fail-fast check before either (AWS API) call ever happens.

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					PolicyArns:          []string{"arn:aws:iam::123456789012:policy/MyPolicy/"},
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("policy ARN at index 0 is not an IAM policy ARN"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())
			})

			It("should classify malformed inline policy JSON as a validation error without making AWS calls", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
				})

				t.SetCluster(cluster.ID(), cluster)

				// No GetCreator or GetOpenIDConnectProviderByOidcEndpointUrl
				// mock: the malformed inline policy must be caught by the
				// CLI's fail-fast check before either (AWS API) call ever
				// happens.

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					InlinePolicy:        `{"Version": "2012-10-17", "Statement": [`,
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("inline policy must be valid JSON"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())
			})

			It("should fail fast on multiple service accounts without a role name, before the OIDC lookup", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
				})

				t.SetCluster(cluster.ID(), cluster)

				// No GetCreator or GetOpenIDConnectProviderByOidcEndpointUrl
				// mock: the missing role name must be caught by the CLI's
				// fail-fast check before either (AWS API) call ever happens.

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"app-one", "app-two"},
					Namespace:           "default",
					PolicyArns:          []string{"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"},
				}
				testRunner := CreateIamServiceAccountRunner(options)

				var out bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&out)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("role name is required when specifying multiple service accounts"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())
				Expect(out.String()).To(ContainSubstring("Usage:"))
			})

			It("should not classify an EnsureRole failure as a validation error or show usage", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
				})

				t.SetCluster(cluster.ID(), cluster)

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
					GetOpenIDConnectProviderByOidcEndpointUrl("https://test.example.com").
					Return(providers[0].Arn, nil)

				mockAWS.EXPECT().
					EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), "", "", gomock.Any(), gomock.Any(), false).
					Return("", errors.New("access denied"))

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					PolicyArns:          []string{"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"},
				}
				testRunner := CreateIamServiceAccountRunner(options)

				var out bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&out)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create role"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeFalse())
				Expect(out.String()).To(BeEmpty())
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
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
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
					GetOpenIDConnectProviderByOidcEndpointUrl("https://test.example.com").
					Return(providers[0].Arn, nil)

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

			It("should reject a file:// inline policy that resolves to empty content instead of silently dropping it", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.ID("test-cluster-id")
					c.Name("test-cluster")
					c.AWS(cmv1.NewAWS().
						STS(cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-role").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								IssuerUrl("https://test.example.com")).
							OIDCEndpointURL("https://test.example.com")))
				})

				t.SetCluster(cluster.ID(), cluster)

				// No GetCreator or GetOpenIDConnectProviderByOidcEndpointUrl
				// mock: the empty resolved inline policy must be caught by
				// the CLI's fail-fast check before either (AWS API) call
				// ever happens.

				emptyPolicyPath := filepath.Join(GinkgoT().TempDir(), "empty-policy.json")
				Expect(os.WriteFile(emptyPolicyPath, []byte{}, 0o600)).To(Succeed())

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
					InlinePolicy:        "file://" + emptyPolicyPath,
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("inline policy must not be empty when provided"))

				var validationErr *rosaerrors.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())

				// No EnsureRole/PutRolePolicy mock was set up above; gomock
				// fails the test if either was called with an invalid request.
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
								IssuerUrl("https://test.gov.example.com")).
							OIDCEndpointURL("https://test.gov.example.com")))
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
					GetOpenIDConnectProviderByOidcEndpointUrl("https://test.gov.example.com").
					Return(providers[0].Arn, nil)

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
							IssuerUrl("https://test.example.com")).
						OIDCEndpointURL("https://test.example.com")))
			})

			providers := []aws.OidcProviderOutput{
				{
					Arn: "arn:aws:iam::123456789012:oidc-provider/test.example.com",
				},
			}

			mockAWS.EXPECT().
				GetOpenIDConnectProviderByOidcEndpointUrl("https://test.example.com").
				Return(providers[0].Arn, nil)

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
							IssuerUrl("https://test.example.com")).
						OIDCEndpointURL("https://test123.example.com")))
			})

			mockAWS.EXPECT().
				GetOpenIDConnectProviderByOidcEndpointUrl("https://test123.example.com").
				Return("", nil)

			_, err := getOIDCProviderARN(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no OIDC provider found for cluster with ID " +
				"'test-cluster-id'"))
		})

		It("should preserve the underlying AWS error when the provider lookup call fails", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test-cluster-id")
				c.Name("test-cluster")
				c.AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						OidcConfig(cmv1.NewOidcConfig().
							ID("test-oidc-id").
							IssuerUrl("https://test.example.com")).
						OIDCEndpointURL("https://test123.example.com")))
			})

			mockAWS.EXPECT().
				GetOpenIDConnectProviderByOidcEndpointUrl("https://test123.example.com").
				Return("", errors.New("access denied"))

			_, err := getOIDCProviderARN(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get OIDC provider for cluster with ID " +
				"'test-cluster-id'"))
			Expect(errors.Unwrap(err)).To(MatchError("access denied"))
		})
	})
})
