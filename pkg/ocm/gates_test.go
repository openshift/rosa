package ocm

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

var _ = Describe("Version gates API client behavior", func() {
	var apiServer *ghttp.Server
	var ocmClient *Client

	BeforeEach(func() {
		apiServer = MakeTCPServer()
		apiServer.SetAllowUnhandledRequests(true)
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)
		ocmClient = buildTestOCMClient(apiServer.URL())
	})

	AfterEach(func() {
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("paginates all version gate pages and applies the version prefix filter", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/version_gates"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("page")).To(Equal("1"))
					Expect(request.URL.Query().Get("size")).To(Equal("100"))
					Expect(request.URL.Query().Get("search")).To(Equal("version_raw_id_prefix = '4.15'"))
				},
				RespondWithJSON(http.StatusOK, `{
					"kind":"VersionGateList",
					"page":1,
					"size":100,
					"total":101,
					"items":[{"id":"gate-1","sts_only":true}]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/version_gates"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("page")).To(Equal("2"))
					Expect(request.URL.Query().Get("size")).To(Equal("100"))
				},
				RespondWithJSON(http.StatusOK, `{
					"kind":"VersionGateList",
					"page":2,
					"size":1,
					"total":101,
					"items":[{"id":"gate-2","sts_only":false}]
				}`),
			),
		)

		gates, err := ocmClient.ListAllOcpGates("4.15")
		Expect(err).NotTo(HaveOccurred())
		Expect(gates).To(HaveLen(2))
		Expect(gates[0].ID()).To(Equal("gate-1"))
		Expect(gates[1].ID()).To(Equal("gate-2"))
	})

	It("returns only STS-only gates", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/version_gates"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"VersionGateList",
					"page":1,
					"size":2,
					"total":2,
					"items":[
						{"id":"gate-sts","sts_only":true},
						{"id":"gate-shared","sts_only":false}
					]
				}`),
			),
		)

		gates, err := ocmClient.ListStsGates("4.15")
		Expect(err).NotTo(HaveOccurred())
		Expect(gates).To(HaveLen(1))
		Expect(gates[0].ID()).To(Equal("gate-sts"))
		Expect(gates[0].STSOnly()).To(BeTrue())
	})

	It("returns only non-STS gates", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/version_gates"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"VersionGateList",
					"page":1,
					"size":2,
					"total":2,
					"items":[
						{"id":"gate-sts","sts_only":true},
						{"id":"gate-shared","sts_only":false}
					]
				}`),
			),
		)

		gates, err := ocmClient.ListOcpGates("4.15")
		Expect(err).NotTo(HaveOccurred())
		Expect(gates).To(HaveLen(1))
		Expect(gates[0].ID()).To(Equal("gate-shared"))
		Expect(gates[0].STSOnly()).To(BeFalse())
	})

	It("returns an error when listing all gates fails", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/version_gates"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"CLUSTERS-MGMT-500",
					"reason":"unable to list version gates"
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/version_gates"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"CLUSTERS-MGMT-500",
					"reason":"unable to list version gates"
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/version_gates"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"CLUSTERS-MGMT-500",
					"reason":"unable to list version gates"
				}`),
			),
		)

		gates, err := ocmClient.ListAllOcpGates("4.15")
		Expect(err).To(HaveOccurred())
		Expect(gates).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("unable to list version gates"))
	})
})
