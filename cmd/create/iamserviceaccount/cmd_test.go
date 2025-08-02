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

				providers := []aws.OidcProviderOutput{
					{
						Arn: "arn:aws:iam::123456789012:oidc-provider/test.example.com",
					},
				}

				mockAWS.EXPECT().
					ListOidcProviders("", cluster.AWS().STS().OidcConfig()).
					Return(providers, nil)

				mockAWS.EXPECT().
					EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), "", "", gomock.Any(), gomock.Any(), false).
					Return("arn:aws:iam::123456789012:role/test-cluster-default-test-sa", nil)

				options := &iamServiceAccountOpts.CreateIamServiceAccountUserOptions{
					ServiceAccountNames: []string{"test-sa"},
					Namespace:           "default",
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
				}
				testRunner := CreateIamServiceAccountRunner(options)

				err := testRunner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not an STS cluster"))
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
				ListOidcProviders("", cluster.AWS().STS().OidcConfig()).
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
				ListOidcProviders("", cluster.AWS().STS().OidcConfig()).
				Return([]aws.OidcProviderOutput{}, nil)

			_, err := getOIDCProviderARN(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no OIDC provider found"))
		})
	})
})
