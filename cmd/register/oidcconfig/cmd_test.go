package oidcconfig

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/rosa/cmd/create/oidcprovider"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"

	"github.com/openshift/rosa/pkg/rosa/mocks"
)

func TestRegisterOidcConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa register oidc-config")
}

var _ = Describe("register oidc-config", func() {
	Context("register oidc-config command", func() {
		It("returns command", func() {
			cmd := NewRegisterOidcConfigCommand()
			Expect(cmd).NotTo(BeNil())
		})
	})

	Context("execute create oidc-config", func() {

		var runtime *test.TestingRuntime
		var cmd *cobra.Command
		var mockOidcCreateProvider *mocks.MockCommandInterface
		var awsClient *aws.MockClient

		BeforeEach(func() {
			runtime = test.NewTestRuntime()
			output.SetOutput("")
			cmd = NewRegisterOidcConfigCommand()

			ctrl := gomock.NewController(GinkgoT())
			mockOidcCreateProvider = mocks.NewMockCommandInterface(ctrl)
			SetCreateOidcProviderCommand(mockOidcCreateProvider)

			awsClient = aws.NewMockClient(ctrl)
			runtime.RosaRuntime.AWSClient = awsClient
		})

		AfterEach(func() {
			output.SetOutput("")
		})

		It("success", func() {
			// mockOidcConfig := test.MockOidcConfig(func(c *cmv1.OidcConfigBuilder) {
			// 	c.IssuerUrl("https://thumbprint")
			// })
			// var oidcConfigJson bytes.Buffer
			// err := cmv1.MarshalOidcConfig(mockOidcConfig, &oidcConfigJson)
			// Expect(err).To(BeNil())
			// runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, oidcConfigJson.String()))

			awsClient.EXPECT().FindRoleARNs(gomock.Any(), gomock.Any()).Return([]string{"something"}, nil)

			mockOidcCreateProvider.EXPECT().Runner().Return(func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {
				Expect(argv[0]).To(Equal("--oidc-config-id"))

				return nil
			}).AnyTimes()
			mockOidcCreateProvider.EXPECT().NewCommand().Return(oidcprovider.NewCreateOidcProviderCommand(mockOidcCreateProvider))

			args := []string{"--mode", "auto", "--yes", "--issuer-url", "https://thumbprint", "--role-arn", "arn:aws:iam::765374464689:role/some-Installer-Role"}
			cmd.ParseFlags(args)
			runner := RegisterOidcConfigRunner()
			err := runner(context.Background(), runtime.RosaRuntime, cmd, args)

			Expect(err).To(BeNil())
		})
	})
})
