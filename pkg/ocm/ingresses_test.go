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

var _ = Describe("Ingresses", Ordered, func() {
	const clusterID = "cluster-1"

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
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/ingresses"),
					RespondWithJSON(http.StatusOK, ingressesBody),
				),
			)

			ingress, err := ocmClient.GetIngress(clusterID, "apps")
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress).NotTo(BeNil())
			Expect(ingress.ID()).To(Equal("default-ingress"))
		})

		It("returns the non-default ingress for apps2", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/ingresses"),
					RespondWithJSON(http.StatusOK, ingressesBody),
				),
			)

			ingress, err := ocmClient.GetIngress(clusterID, "apps2")
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress).NotTo(BeNil())
			Expect(ingress.ID()).To(Equal("custom-ingress"))
		})

		It("returns ingress by explicit id lookup", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/ingresses"),
					RespondWithJSON(http.StatusOK, ingressesBody),
				),
			)

			ingress, err := ocmClient.GetIngress(clusterID, "custom-ingress")
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress).NotTo(BeNil())
			Expect(ingress.ID()).To(Equal("custom-ingress"))
		})

		It("returns not found error when ingress key has no match", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/ingresses"),
					RespondWithJSON(http.StatusOK, ingressesBody),
				),
			)

			ingress, err := ocmClient.GetIngress(clusterID, "missing")
			Expect(err).To(HaveOccurred())
			Expect(ingress).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("Failed to get ingress 'missing' for cluster 'cluster-1'"))
		})
	})

	Describe("GetIngresses", func() {
		It("returns ingress list", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/ingresses"),
					RespondWithJSON(http.StatusOK, `{
						"kind": "IngressList",
						"page": 1,
						"size": 1,
						"total": 1,
						"items": [{"id": "ingress-1", "default": true}]
					}`),
				),
			)

			ingresses, err := ocmClient.GetIngresses(clusterID)
			Expect(err).NotTo(HaveOccurred())
			Expect(ingresses).To(HaveLen(1))
			Expect(ingresses[0].ID()).To(Equal("ingress-1"))
		})
	})

	Describe("UpdateIngress", func() {
		It("updates and returns ingress", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPatch, "/api/clusters_mgmt/v1/clusters/cluster-1/ingresses/ingress-1"),
					func(_ http.ResponseWriter, request *http.Request) {
						payload := map[string]interface{}{}
						Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
						Expect(payload).To(HaveKeyWithValue("id", "ingress-1"))
						Expect(payload).To(HaveKeyWithValue("default", false))
					},
					RespondWithJSON(http.StatusOK, `{
						"id": "ingress-1",
						"default": true
					}`),
				),
			)

			ingress, err := cmv1.NewIngress().
				ID("ingress-1").
				Default(false).
				Build()
			Expect(err).NotTo(HaveOccurred())

			updated, err := ocmClient.UpdateIngress(clusterID, ingress)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).NotTo(BeNil())
			Expect(updated.ID()).To(Equal("ingress-1"))
			Expect(updated.Default()).To(BeTrue())
		})
	})

	Describe("DeleteIngress", func() {
		It("deletes ingress", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, "/api/clusters_mgmt/v1/clusters/cluster-1/ingresses/ingress-1"),
					RespondWithJSON(http.StatusNoContent, ""),
				),
			)

			err := ocmClient.DeleteIngress(clusterID, "ingress-1")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
