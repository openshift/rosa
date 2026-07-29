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

var _ = Describe("Ingresses", Ordered, func() {
	const clusterID = "cluster-1"

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

	Describe("GetIngress", func() {
		const ingressesBody = `{
			"kind": "IngressList",
			"page": 1,
			"size": 2,
			"total": 2,
			"items": [
				{"id": "default-ingress", "default": true},
				{"id": "custom-ingress", "default": false}
			]
		}`

		It("returns the default ingress for apps", func() {
			apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ingressesBody))

			ingress, err := ocmClient.GetIngress(clusterID, "apps")
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress).NotTo(BeNil())
			Expect(ingress.ID()).To(Equal("default-ingress"))
		})

		It("returns the non-default ingress for apps2", func() {
			apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ingressesBody))

			ingress, err := ocmClient.GetIngress(clusterID, "apps2")
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress).NotTo(BeNil())
			Expect(ingress.ID()).To(Equal("custom-ingress"))
		})

		It("returns ingress by explicit id lookup", func() {
			apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ingressesBody))

			ingress, err := ocmClient.GetIngress(clusterID, "custom-ingress")
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress).NotTo(BeNil())
			Expect(ingress.ID()).To(Equal("custom-ingress"))
		})

		It("returns not found error when ingress key has no match", func() {
			apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ingressesBody))

			ingress, err := ocmClient.GetIngress(clusterID, "missing")
			Expect(err).To(HaveOccurred())
			Expect(ingress).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("Failed to get ingress 'missing' for cluster 'cluster-1'"))
		})
	})

	Describe("GetIngresses", func() {
		It("returns ingress list", func() {
			apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, `{
				"kind": "IngressList",
				"page": 1,
				"size": 1,
				"total": 1,
				"items": [{"id": "ingress-1", "default": true}]
			}`))

			ingresses, err := ocmClient.GetIngresses(clusterID)
			Expect(err).NotTo(HaveOccurred())
			Expect(ingresses).To(HaveLen(1))
			Expect(ingresses[0].ID()).To(Equal("ingress-1"))
		})
	})

	Describe("UpdateIngress", func() {
		It("updates and returns ingress", func() {
			apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, `{
				"id": "ingress-1",
				"default": false
			}`))

			ingress, err := cmv1.NewIngress().
				ID("ingress-1").
				Default(false).
				Build()
			Expect(err).NotTo(HaveOccurred())

			updated, err := ocmClient.UpdateIngress(clusterID, ingress)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).NotTo(BeNil())
			Expect(updated.ID()).To(Equal("ingress-1"))
		})
	})

	Describe("DeleteIngress", func() {
		It("deletes ingress", func() {
			apiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ""))

			err := ocmClient.DeleteIngress(clusterID, "ingress-1")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
