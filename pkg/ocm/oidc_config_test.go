package ocm

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

var _ = Describe("Oidc Config", Ordered, func() {
	const awsAccountID = "123456789012"

	var ssoServer, apiServer *ghttp.Server
	var ocmClient *Client

	BeforeEach(func() {
		ssoServer = MakeTCPServer()
		apiServer = MakeTCPServer()
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)

		accessToken := MakeTokenString("Bearer", 15*time.Minute)
		ssoServer.AppendHandlers(RespondWithAccessToken(accessToken))

		logger, err := logging.NewGoLoggerBuilder().Debug(true).Build()
		Expect(err).NotTo(HaveOccurred())

		connection, err := sdk.NewConnectionBuilder().
			Logger(logger).
			Tokens(accessToken).
			URL(apiServer.URL()).
			Build()
		Expect(err).NotTo(HaveOccurred())

		ocmClient = NewClientWithConnection(connection)
	})

	AfterEach(func() {
		ssoServer.Close()
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("lists OIDC configs with expected search query", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/oidc_configs"),
				ghttp.VerifyFormKV("search", "aws.account_id='123456789012' or aws.account_id=''"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "OidcConfigList",
					"page": 1,
					"size": 1,
					"total": 1,
					"items": [{"id": "oidc-1", "secret": "s"}]
				}`),
			),
		)

		configs, err := ocmClient.ListOidcConfigs(awsAccountID)
		Expect(err).NotTo(HaveOccurred())
		Expect(configs).To(HaveLen(1))
		Expect(configs[0].ID()).To(Equal("oidc-1"))
	})

	It("fetches OIDC thumbprint", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/aws_inquiries/oidc_thumbprint"),
				func(_ http.ResponseWriter, request *http.Request) {
					payload := map[string]interface{}{}
					Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
					Expect(payload).To(HaveKeyWithValue("oidc_config_id", "oidc-1"))
				},
				RespondWithJSON(http.StatusOK, `{
					"id": "thumb-1",
					"thumbprint": "ABCD"
				}`),
			),
		)

		input, err := cmv1.NewOidcThumbprintInput().OidcConfigId("oidc-1").Build()
		Expect(err).NotTo(HaveOccurred())

		thumbprint, err := ocmClient.FetchOidcThumbprint(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(thumbprint).NotTo(BeNil())
		Expect(thumbprint.Thumbprint()).To(Equal("ABCD"))
	})

	It("returns error when fetching OIDC thumbprint fails", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/aws_inquiries/oidc_thumbprint"),
				func(_ http.ResponseWriter, request *http.Request) {
					payload := map[string]interface{}{}
					Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
					Expect(payload).To(HaveKeyWithValue("oidc_config_id", "missing-oidc"))
				},
				RespondWithJSON(http.StatusBadRequest, `{"reason":"invalid issuer"}`),
			),
		)

		input, err := cmv1.NewOidcThumbprintInput().OidcConfigId("missing-oidc").Build()
		Expect(err).NotTo(HaveOccurred())

		thumbprint, fetchErr := ocmClient.FetchOidcThumbprint(input)
		Expect(fetchErr).To(HaveOccurred())
		Expect(thumbprint).To(BeNil())
	})

	It("supports basic OIDC config CRUD wrappers", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/oidc_configs/oidc-1"),
				RespondWithJSON(http.StatusOK, `{"id":"oidc-1","secret_arn":"arn:aws:secretsmanager:us-east-1:123:secret:one"}`),
			),
		)
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/oidc_configs"),
				func(_ http.ResponseWriter, request *http.Request) {
					payload := map[string]interface{}{}
					Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
					Expect(payload).To(HaveKeyWithValue("id", "oidc-2"))
					Expect(payload).To(HaveKeyWithValue(
						"secret_arn",
						"arn:aws:secretsmanager:us-east-1:123:secret:two",
					))
				},
				RespondWithJSON(http.StatusCreated, `{"id":"oidc-2","secret_arn":"arn:aws:secretsmanager:us-east-1:123:secret:two"}`),
			),
		)
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodDelete, "/api/clusters_mgmt/v1/oidc_configs/oidc-2"),
				RespondWithJSON(http.StatusNoContent, ``),
			),
		)

		config, err := ocmClient.GetOidcConfig("oidc-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(config).NotTo(BeNil())
		Expect(config.ID()).To(Equal("oidc-1"))

		createInput, err := cmv1.NewOidcConfig().
			ID("oidc-2").
			SecretArn("arn:aws:secretsmanager:us-east-1:123:secret:two").
			Build()
		Expect(err).NotTo(HaveOccurred())

		created, err := ocmClient.CreateOidcConfig(createInput)
		Expect(err).NotTo(HaveOccurred())
		Expect(created).NotTo(BeNil())
		Expect(created.ID()).To(Equal("oidc-2"))

		err = ocmClient.DeleteOidcConfig("oidc-2")
		Expect(err).NotTo(HaveOccurred())
	})
})
