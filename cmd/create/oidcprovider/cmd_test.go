package oidcprovider

import (
	"fmt"
	"net/http"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	awsClient "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Create OIDC Provider", func() {
	Context("CreateOIDCProvider", func() {
		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
		})

		It("creates the provider successfully", func() {
			oidcConfigId := "oidc-config-123"
			issuerUrl := "https://oidc.example.com/abc123"
			thumbprint := "a]b]c]d]e]f]0]1]2]3]4]5]6]7]8]9]a]b]c]d"

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, fmt.Sprintf(`{
				"kind": "OidcConfig",
				"id": "%s",
				"issuer_url": "%s"
			}`, oidcConfigId, issuerUrl)))

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, fmt.Sprintf(`{
				"kind": "OidcThumbprint",
				"thumbprint": "%s",
				"oidc_config_id": "%s"
			}`, thumbprint, oidcConfigId)))

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().CreateOpenIDConnectProvider(
				issuerUrl,
				thumbprint,
				gomock.Any(),
			).Return("arn:aws:iam::123456789012:oidc-provider/oidc.example.com/abc123", nil)

			err := CreateOIDCProvider(t.RosaRuntime, oidcConfigId, "", true)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error when GetOidcConfig fails", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, `{
				"kind": "Error",
				"id": "404",
				"href": "/api/clusters_mgmt/v1/errors/404",
				"code": "CLUSTERS-MGMT-404",
				"reason": "not found"
			}`))

			err := CreateOIDCProvider(t.RosaRuntime, "nonexistent", "", true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("retrieving OIDC Config"))
		})

		It("returns error when FetchOidcThumbprint fails", func() {
			oidcConfigId := "oidc-config-123"
			issuerUrl := "https://oidc.example.com/abc123"

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, fmt.Sprintf(`{
				"kind": "OidcConfig",
				"id": "%s",
				"issuer_url": "%s"
			}`, oidcConfigId, issuerUrl)))

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, `{
				"kind": "Error",
				"id": "500",
				"href": "/api/clusters_mgmt/v1/errors/500",
				"code": "CLUSTERS-MGMT-500",
				"reason": "thumbprint service unavailable"
			}`))

			err := CreateOIDCProvider(t.RosaRuntime, oidcConfigId, "", true)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when CreateOpenIDConnectProvider fails", func() {
			oidcConfigId := "oidc-config-123"
			issuerUrl := "https://oidc.example.com/abc123"
			thumbprint := "a]b]c]d]e]f]0]1]2]3]4]5]6]7]8]9]a]b]c]d"

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, fmt.Sprintf(`{
				"kind": "OidcConfig",
				"id": "%s",
				"issuer_url": "%s"
			}`, oidcConfigId, issuerUrl)))

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, fmt.Sprintf(`{
				"kind": "OidcThumbprint",
				"thumbprint": "%s",
				"oidc_config_id": "%s"
			}`, thumbprint, oidcConfigId)))

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().CreateOpenIDConnectProvider(
				issuerUrl,
				thumbprint,
				gomock.Any(),
			).Return("", fmt.Errorf("access denied"))

			err := CreateOIDCProvider(t.RosaRuntime, oidcConfigId, "", true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
		})
	})
})
