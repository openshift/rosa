package ocm

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

var _ = Describe("Cluster logs API client behavior", func() {
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

	It("passes the requested tail value when retrieving install logs", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/logs/install"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("tail")).To(Equal("42"))
				},
				RespondWithJSON(http.StatusOK, `{"content":"install log line"}`),
			),
		)

		logs, err := ocmClient.GetInstallLogs("cluster-1", 42)
		Expect(err).NotTo(HaveOccurred())
		Expect(logs).NotTo(BeNil())
		Expect(logs.Content()).To(ContainSubstring("install log line"))
	})

	It("translates install log 404 responses into user-facing not-found errors", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-404/logs/install"),
				RespondWithJSON(http.StatusNotFound, `{
					"kind":"Error",
					"id":"404",
					"href":"/api/errors/404",
					"code":"CLUSTERS-MGMT-404",
					"reason":"missing logs"
				}`),
			),
		)

		_, err := ocmClient.GetInstallLogs("cluster-404", 5)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Failed to get logs for cluster 'cluster-404'"))
	})

	It("passes the requested tail value when retrieving uninstall logs", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/logs/uninstall"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("tail")).To(Equal("27"))
				},
				RespondWithJSON(http.StatusOK, `{"content":"uninstall log line"}`),
			),
		)

		logs, err := ocmClient.GetUninstallLogs("cluster-1", 27)
		Expect(err).NotTo(HaveOccurred())
		Expect(logs).NotTo(BeNil())
		Expect(logs.Content()).To(ContainSubstring("uninstall log line"))
	})

	It("translates uninstall log 404 responses into user-facing not-found errors", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-404/logs/uninstall"),
				RespondWithJSON(http.StatusNotFound, `{
					"kind":"Error",
					"id":"404",
					"href":"/api/errors/404",
					"code":"CLUSTERS-MGMT-404",
					"reason":"missing logs"
				}`),
			),
		)

		_, err := ocmClient.GetUninstallLogs("cluster-404", 5)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Failed to get logs for cluster 'cluster-404'"))
	})

	It("uses fixed polling parameters and returns install logs", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/logs/install"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("tail")).To(Equal("100"))
				},
				RespondWithJSON(http.StatusOK, `{"content":"polled install log"}`),
			),
		)

		callback := func(response *cmv1.LogGetResponse) bool {
			return true
		}
		logs, err := ocmClient.PollInstallLogs("cluster-1", callback)
		Expect(err).NotTo(HaveOccurred())
		Expect(logs).NotTo(BeNil())
		Expect(logs.Content()).To(ContainSubstring("polled install log"))
	})

	It("translates poll 404 responses into user-facing not-found errors", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-404/logs/install"),
				RespondWithJSON(http.StatusNotFound, `{
					"kind":"Error",
					"id":"404",
					"href":"/api/errors/404",
					"code":"CLUSTERS-MGMT-404",
					"reason":"missing logs"
				}`),
			),
		)

		callback := func(response *cmv1.LogGetResponse) bool {
			return true
		}
		_, err := ocmClient.PollInstallLogs("cluster-404", callback)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Failed to poll logs for cluster 'cluster-404'"))
	})

	It("translates uninstall poll 404 responses into user-facing not-found errors", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-404/logs/uninstall"),
				RespondWithJSON(http.StatusNotFound, `{
					"kind":"Error",
					"id":"404",
					"href":"/api/errors/404",
					"code":"CLUSTERS-MGMT-404",
					"reason":"missing logs"
				}`),
			),
		)

		callback := func(response *cmv1.LogGetResponse) bool {
			return true
		}
		_, err := ocmClient.PollUninstallLogs("cluster-404", callback)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Failed to poll logs for cluster 'cluster-404'"))
	})

	It("wraps polling errors for uninstall logs", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/logs/uninstall"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"CLUSTERS-MGMT-500",
					"reason":"poll backend failure"
				}`),
			),
		)

		callback := func(response *cmv1.LogGetResponse) bool {
			return true
		}
		_, err := ocmClient.PollUninstallLogs("cluster-1", callback)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Failed to poll logs for cluster 'cluster-1'"))
	})
})
