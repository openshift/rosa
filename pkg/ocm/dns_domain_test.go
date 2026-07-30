package ocm

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

var _ = Describe("DNS Domains", Ordered, func() {
	var apiServer *ghttp.Server
	var ocmClient *Client

	BeforeEach(func() {
		apiServer = MakeTCPServer()
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)
		ocmClient = buildTestOCMClient(apiServer.URL())
	})

	AfterEach(func() {
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("lists DNS domains with search and order parameters", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/dns_domains"),
				ghttp.VerifyFormKV("search", "name like 'example'"),
				ghttp.VerifyFormKV("order", "organization.id asc"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "DNSDomainList",
					"page": 1,
					"size": 1,
					"total": 1,
					"items": [{"id":"dns-1","user_defined":true}]
				}`),
			),
		)

		domains, err := ocmClient.ListDNSDomains("name like 'example'")
		Expect(err).NotTo(HaveOccurred())
		Expect(domains).To(HaveLen(1))
		Expect(domains[0].ID()).To(Equal("dns-1"))
	})

	It("creates and deletes DNS domains", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/dns_domains"),
				func(_ http.ResponseWriter, request *http.Request) {
					payload := map[string]interface{}{}
					Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
					Expect(payload).To(HaveKeyWithValue("id", "dns-2"))
					Expect(payload).To(HaveKeyWithValue("user_defined", true))
				},
				RespondWithJSON(http.StatusCreated, `{"id":"dns-2","user_defined":true}`),
			),
		)
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodDelete, "/api/clusters_mgmt/v1/dns_domains/dns-2"),
				RespondWithJSON(http.StatusNoContent, ``),
			),
		)

		domainInput, err := cmv1.NewDNSDomain().ID("dns-2").UserDefined(true).Build()
		Expect(err).NotTo(HaveOccurred())

		created, err := ocmClient.CreateDNSDomain(domainInput)
		Expect(err).NotTo(HaveOccurred())
		Expect(created).NotTo(BeNil())
		Expect(created.ID()).To(Equal("dns-2"))

		err = ocmClient.DeleteDNSDomain("dns-2")
		Expect(err).NotTo(HaveOccurred())
	})
})
