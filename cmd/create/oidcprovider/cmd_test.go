package oidcprovider

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	golangmock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/rosa/oidcconfigs"
	"github.com/openshift-online/ocm-common/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
	"github.com/spf13/cobra"
)

func TestCreateOidcProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa create oidc-config")
}

var _ = Describe("create oidc-provider", func() {
	Context("create oidc-provider command", func() {
		It("returns command", func() {
			createOidcProvider := CreateOidcProviderStruct{}
			cmd := NewCreateOidcProviderCommand(&createOidcProvider)
			Expect(cmd).NotTo(BeNil())
		})
	})

	Context("execute create oidc-provider", func() {

		var runtime *test.TestingRuntime
		var cmd *cobra.Command
		var runner rosa.CommandRunner
		var netCtrl *golangmock.Controller
		var awsClient *aws.MockClient

		BeforeEach(func() {
			runtime = test.NewTestRuntime()
			output.SetOutput("")
			createOidcProvider := CreateOidcProviderStruct{}
			cmd = NewCreateOidcProviderCommand(&createOidcProvider)
			runner = createOidcProvider.Runner()
			netCtrl = golangmock.NewController(GinkgoT())

			ctrl := gomock.NewController(GinkgoT())
			awsClient = aws.NewMockClient(ctrl)
			runtime.RosaRuntime.AWSClient = awsClient
		})

		AfterEach(func() {
			output.SetOutput("")
		})

		It("error on both --cluster and --oidc-config-id", func() {
			args := []string{"--cluster", "1", "--oidc-config-id", "1"}
			cmd.ParseFlags(args)

			err := runner(context.Background(), runtime.RosaRuntime, cmd, args)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("A cluster key for STS cluster and an OIDC" +
				" Config ID cannot be specified alongside each other."))
		})

		It("error running programatically with positional args unable to reach url", func() {
			args := []string{"cluster", "auto", "https://some/thumbprint/url"}
			cmd.ParseFlags(args)

			err := runner(context.Background(), runtime.RosaRuntime, cmd, args)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("Unable to get OIDC thumbprint: Get \"https://some:443\":" +
				" dial tcp: lookup some: no such host"))
		})

		It("success running programatically with positional args", func() {

			mockClient := utils.NewMockHTTPClient(netCtrl)
			oidcconfigs.Client = mockClient

			mockClient.EXPECT().Get("https://thumbprint:443").Return(
				&http.Response{
					StatusCode: 200,
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{
							{
								Raw: []byte("abcdef"),
							},
						},
					},
				},
				nil,
			)

			awsClient.EXPECT().HasOpenIDConnectProvider("https://thumbprint/url",
				gomock.Any(), gomock.Any()).Return(true, nil)

			args := []string{"cluster", "auto", "https://thumbprint/url"}
			err := cmd.ParseFlags(args)
			Expect(err).To(BeNil())

			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)
			Expect(err).To(BeNil())
		})

		It("error on non-sts cluster: --cluster cluster --yes --mode auto", func() {
			mockCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.Hypershift(cmv1.NewHypershift().Enabled(true))
			})
			cluster := test.FormatClusterList([]*cmv1.Cluster{mockCluster})

			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, cluster))

			args := []string{"--cluster", "cluster", "--yes", "--mode", "auto"}
			err := cmd.ParseFlags(args)
			Expect(err).To(BeNil())

			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("Cluster 'cluster' is not an STS clusteruntime."))
		})

		It("error with no oidc endpoint url: --cluster cluster --yes --mode auto", func() {
			mockCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
					OIDCEndpointURL("").
					RoleARN("test-arn")))
				c.State(cmv1.ClusterStateReady)
			})
			cluster := test.FormatClusterList([]*cmv1.Cluster{mockCluster})

			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, cluster))

			args := []string{"--cluster", "cluster", "--yes", "--mode", "auto"}
			err := cmd.ParseFlags(args)
			Expect(err).To(BeNil())

			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Cluster 'cluster' does not have an OIDC endpoint URL; provider cannot be created."))
		})

		It("error reaching CS: --cluster cluster --yes --mode auto", func() {
			mockCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
					OIDCEndpointURL("https://thumbprint").
					RoleARN("test-arn")))
				c.State(cmv1.ClusterStateReady)
			})
			cluster := test.FormatClusterList([]*cmv1.Cluster{mockCluster})

			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, cluster))

			args := []string{"--cluster", "cluster", "--yes", "--mode", "auto"}
			err := cmd.ParseFlags(args)
			Expect(err).To(BeNil())

			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Unable to get OIDC thumbprint"))
		})

		It("success for existing oidc provider", func() {
			thumbprint, _ := cmv1.NewAwsOidcThumbprint().
				Thumbprint("123456").
				IssuerUrl("https://thumbprint").
				Build()

			mockCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
					OIDCEndpointURL("https://thumbprint").
					RoleARN("test-arn")))
				c.State(cmv1.ClusterStateReady)
			})
			cluster := test.FormatClusterList([]*cmv1.Cluster{mockCluster})
			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, cluster))

			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatResource(thumbprint)))
			awsClient.EXPECT().HasOpenIDConnectProvider(gomock.Any(),
				gomock.Any(), gomock.Any()).Return(true, nil)

			args := []string{"--cluster", "cluster", "--yes", "--mode", "auto"}
			err := cmd.ParseFlags(args)
			Expect(err).To(BeNil())

			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)
			Expect(err).To(BeNil())
		})

		It("success creating oidc provider", func() {
			thumbprint, _ := cmv1.NewAwsOidcThumbprint().
				Thumbprint("123456").
				IssuerUrl("https://thumbprint").
				Build()

			mockCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
					OIDCEndpointURL("https://thumbprint").
					RoleARN("test-arn")))
				c.State(cmv1.ClusterStateReady)
			})
			cluster := test.FormatClusterList([]*cmv1.Cluster{mockCluster})
			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, cluster))

			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatResource(thumbprint)))
			awsClient.EXPECT().HasOpenIDConnectProvider(gomock.Any(),
				gomock.Any(), gomock.Any()).Return(false, nil)
			awsClient.EXPECT().CreateOpenIDConnectProvider(gomock.Any(),
				gomock.Any(), gomock.Any()).Return("test-arn", nil)

			args := []string{"--cluster", "cluster", "--yes", "--mode", "auto"}
			err := cmd.ParseFlags(args)
			Expect(err).To(BeNil())

			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)
			Expect(err).To(BeNil())
		})

		It("error with no oidc endpoint url: --oidc-config-id id --yes --mode auto", func() {
			mockOidcConfig := test.MockOidcConfig(func(c *cmv1.OidcConfigBuilder) {
				c.IssuerUrl("")
			})
			var oidcConfigJson bytes.Buffer
			err := cmv1.MarshalOidcConfig(mockOidcConfig, &oidcConfigJson)
			Expect(err).To(BeNil())
			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, oidcConfigJson.String()))

			args := []string{"--oidc-config-id", "id", "--yes", "--mode", "auto"}
			err = cmd.ParseFlags(args)
			Expect(err).To(BeNil())

			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("OIDC config 'id' does not have an OIDC endpoint URL; provider cannot be created."))
		})

		It("success already exists: --oidc-config-id id --yes --mode auto", func() {
			mockOidcConfig := test.MockOidcConfig(func(c *cmv1.OidcConfigBuilder) {
				c.IssuerUrl("https://thumbprint")
			})
			var oidcConfigJson bytes.Buffer
			err := cmv1.MarshalOidcConfig(mockOidcConfig, &oidcConfigJson)
			Expect(err).To(BeNil())
			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, oidcConfigJson.String()))

			thumbprint, _ := cmv1.NewAwsOidcThumbprint().
				Thumbprint("123456").
				IssuerUrl("https://thumbprint").
				Build()
			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatResource(thumbprint)))

			awsClient.EXPECT().HasOpenIDConnectProvider("https://thumbprint",
				gomock.Any(), gomock.Any()).Return(true, nil)

			args := []string{"--oidc-config-id", "id", "--yes", "--mode", "auto"}
			err = cmd.ParseFlags(args)
			Expect(err).To(BeNil())

			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)
			Expect(err).To(BeNil())
		})

		It("success create: --oidc-config-id id --yes --mode auto", func() {
			mockOidcConfig := test.MockOidcConfig(func(c *cmv1.OidcConfigBuilder) {
				c.IssuerUrl("https://thumbprint")
			})
			var oidcConfigJson bytes.Buffer
			err := cmv1.MarshalOidcConfig(mockOidcConfig, &oidcConfigJson)
			Expect(err).To(BeNil())
			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, oidcConfigJson.String()))

			thumbprint, _ := cmv1.NewAwsOidcThumbprint().
				Thumbprint("123456").
				IssuerUrl("https://thumbprint").
				Build()
			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatResource(thumbprint)))

			awsClient.EXPECT().HasOpenIDConnectProvider("https://thumbprint",
				gomock.Any(), gomock.Any()).Return(false, nil)
			awsClient.EXPECT().CreateOpenIDConnectProvider(gomock.Any(),
				gomock.Any(), gomock.Any()).Return("test-arn", nil)

			args := []string{"--oidc-config-id", "id", "--yes", "--mode", "auto"}
			err = cmd.ParseFlags(args)
			Expect(err).To(BeNil())

			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)
			Expect(err).To(BeNil())
		})
	})
})
