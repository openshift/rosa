package ocm

import (
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

var _ = Describe("DNS Domains", Ordered, func() {
	var ssoServer, apiServer *ghttp.Server
	var ocmClient *Client

	BeforeEach(func() {
		ssoServer = MakeTCPServer()
		apiServer = MakeTCPServer()
		apiServer.SetAllowUnhandledRequests(true)
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

	It("lists DNS domains with search and order parameters", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
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
		apiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, `{"id":"dns-2","user_defined":true}`))
		apiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ``))

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
