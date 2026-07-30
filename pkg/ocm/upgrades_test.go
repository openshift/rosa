package ocm

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

var _ = Describe("Upgrade policies API client behavior", func() {
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

	It("paginates upgrade policy listings", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies"),
				ghttp.VerifyFormKV("page", "1"),
				ghttp.VerifyFormKV("size", "100"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"UpgradePolicyList",
					"page":1,
					"size":100,
					"total":101,
					"items":[{"id":"policy-1","upgrade_type":"OSD"}]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies"),
				ghttp.VerifyFormKV("page", "2"),
				ghttp.VerifyFormKV("size", "100"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"UpgradePolicyList",
					"page":2,
					"size":1,
					"total":101,
					"items":[{"id":"policy-2","upgrade_type":"OSD"}]
				}`),
			),
		)

		policies, err := ocmClient.GetUpgradePolicies("cluster-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(policies).To(HaveLen(2))
		Expect(policies[0].ID()).To(Equal("policy-1"))
		Expect(policies[1].ID()).To(Equal("policy-2"))
	})

	It("returns the scheduled OSD upgrade and its state", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"UpgradePolicyList",
					"page":1,
					"size":2,
					"total":2,
					"items":[
						{"id":"policy-control-plane","upgrade_type":"CONTROL_PLANE"},
						{"id":"policy-osd","upgrade_type":"OSD"}
					]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies/policy-osd/state"),
				RespondWithJSON(http.StatusOK, `{
					"description":"scheduled for tonight",
					"value":"scheduled"
				}`),
			),
		)

		policy, state, err := ocmClient.GetScheduledUpgrade("cluster-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(policy).NotTo(BeNil())
		Expect(policy.ID()).To(Equal("policy-osd"))
		Expect(state).NotTo(BeNil())
		Expect(state.Value()).To(Equal(cmv1.UpgradePolicyStateValue("scheduled")))
	})

	It("returns false when no scheduled OSD upgrade exists", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"UpgradePolicyList",
					"page":1,
					"size":1,
					"total":1,
					"items":[{"id":"policy-control-plane","upgrade_type":"CONTROL_PLANE"}]
				}`),
			),
		)

		cancelled, err := ocmClient.CancelUpgrade("cluster-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(cancelled).To(BeFalse())
	})

	It("deletes the scheduled OSD upgrade when cancelling", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"UpgradePolicyList",
					"page":1,
					"size":1,
					"total":1,
					"items":[{"id":"policy-osd","upgrade_type":"OSD"}]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies/policy-osd/state"),
				RespondWithJSON(http.StatusOK, `{
					"description":"scheduled for tonight",
					"value":"scheduled"
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodDelete, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies/policy-osd"),
				RespondWithJSON(http.StatusOK, `{}`),
			),
		)

		cancelled, err := ocmClient.CancelUpgrade("cluster-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(cancelled).To(BeTrue())
	})

	It("parses missing classic gate agreements from dry-run errors", func() {
		upgradePolicy, err := cmv1.NewUpgradePolicy().Version("4.16.10").Build()
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies"),
				ghttp.VerifyFormKV("dryRun", "true"),
				RespondWithJSON(http.StatusBadRequest, `{
					"kind":"Error",
					"id":"400",
					"href":"/api/errors/400",
					"code":"CLUSTERS-MGMT-400",
					"reason":"missing gate agreements",
					"details":[{"id":"gate-a","sts_only":false}]
				}`),
			),
		)

		gates, err := ocmClient.GetMissingGateAgreementsClassic("cluster-1", upgradePolicy)
		Expect(err).NotTo(HaveOccurred())
		Expect(gates).To(HaveLen(1))
		Expect(gates[0].ID()).To(Equal("gate-a"))
	})

	It("returns the backend reason when classic gate details are invalid", func() {
		upgradePolicy, err := cmv1.NewUpgradePolicy().Version("4.16.10").Build()
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-1/upgrade_policies"),
				ghttp.VerifyFormKV("dryRun", "true"),
				RespondWithJSON(http.StatusBadRequest, `{
					"kind":"Error",
					"id":"400",
					"href":"/api/errors/400",
					"code":"CLUSTERS-MGMT-400",
					"reason":"invalid version gate payload",
					"details":[{"id":""}]
				}`),
			),
		)

		gates, err := ocmClient.GetMissingGateAgreementsClassic("cluster-1", upgradePolicy)
		Expect(err).To(HaveOccurred())
		Expect(gates).To(BeEmpty())
		Expect(err.Error()).To(ContainSubstring("invalid version gate payload"))
	})

	It("parses missing hypershift gate agreements from dry-run errors", func() {
		upgradePolicy, err := cmv1.NewControlPlaneUpgradePolicy().Version("4.16.10").Build()
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-1/control_plane/upgrade_policies"),
				ghttp.VerifyFormKV("dryRun", "true"),
				RespondWithJSON(http.StatusBadRequest, `{
					"kind":"Error",
					"id":"400",
					"href":"/api/errors/400",
					"code":"CLUSTERS-MGMT-400",
					"reason":"missing gate agreements",
					"details":[{"id":"gate-hcp","sts_only":true}]
				}`),
			),
		)

		gates, err := ocmClient.GetMissingGateAgreementsHypershift("cluster-1", upgradePolicy)
		Expect(err).NotTo(HaveOccurred())
		Expect(gates).To(HaveLen(1))
		Expect(gates[0].ID()).To(Equal("gate-hcp"))
	})

	It("returns backend reason when hypershift gate details are invalid", func() {
		upgradePolicy, err := cmv1.NewControlPlaneUpgradePolicy().Version("4.16.10").Build()
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-1/control_plane/upgrade_policies"),
				ghttp.VerifyFormKV("dryRun", "true"),
				RespondWithJSON(http.StatusBadRequest, `{
					"kind":"Error",
					"id":"400",
					"href":"/api/errors/400",
					"code":"CLUSTERS-MGMT-400",
					"reason":"invalid hypershift version gate payload",
					"details":[{"id":""}]
				}`),
			),
		)

		gates, err := ocmClient.GetMissingGateAgreementsHypershift("cluster-1", upgradePolicy)
		Expect(err).To(HaveOccurred())
		Expect(gates).To(BeEmpty())
		Expect(err.Error()).To(ContainSubstring("invalid hypershift version gate payload"))
	})

	It("acknowledges version gates successfully", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-1/gate_agreements"),
				RespondWithJSON(http.StatusCreated, `{"id":"agreement-1"}`),
			),
		)

		err := ocmClient.AckVersionGate("cluster-1", "gate-1")
		Expect(err).NotTo(HaveOccurred())
	})
})
