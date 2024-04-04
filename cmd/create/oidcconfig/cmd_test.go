package oidcconfig

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/openshift/rosa/cmd/create/oidcprovider"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"

	"github.com/openshift/rosa/pkg/rosa/mocks"
)

func TestCreateOidcConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa create oidc-config")
}

var _ = Describe("create oidc-config", func() {
	Context("create oidc-config command", func() {
		It("returns command", func() {
			cmd := NewCreateOidcConfigCommand()
			Expect(cmd).NotTo(BeNil())
		})
	})

	Context("execute create oidc-config", func() {

		var runtime *test.TestingRuntime
		var cmd *cobra.Command
		var mockOidcCreateProvider *mocks.MockCommandInterface

		BeforeEach(func() {
			runtime = test.NewTestRuntime()
			output.SetOutput("")
			cmd = NewCreateOidcConfigCommand()

			ctrl := gomock.NewController(GinkgoT())
			mockOidcCreateProvider = mocks.NewMockCommandInterface(ctrl)
			SetCreateOidcProviderCommand(mockOidcCreateProvider)
		})

		AfterEach(func() {
			output.SetOutput("")
		})

		It("success", func() {
			mockOidcConfig := test.MockOidcConfig(func(c *cmv1.OidcConfigBuilder) {
				c.IssuerUrl("https://thumbprint")
			})
			var oidcConfigJson bytes.Buffer
			err := cmv1.MarshalOidcConfig(mockOidcConfig, &oidcConfigJson)

			Expect(err).To(BeNil())
			runtime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, oidcConfigJson.String()))

			mockOidcCreateProvider.EXPECT().Runner().Return(func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {
				Expect(argv[0]).To(Equal("--oidc-config-id"))

				return nil
			}).AnyTimes()
			mockOidcCreateProvider.EXPECT().NewCommand().Return(oidcprovider.NewCreateOidcProviderCommand(mockOidcCreateProvider))

			args := []string{"--mode", "auto", "--yes", "--managed"}
			cmd.ParseFlags(args)
			runner := CreateOidcConfigRunner()
			err = runner(context.Background(), runtime.RosaRuntime, cmd, args)

			Expect(err).To(BeNil())
		})
	})
})
