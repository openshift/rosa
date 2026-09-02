package ocm

import (
	"encoding/json"
	"io"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

const expectedRetryAttempts = 3

func receivedGETRequestCount(apiServer *ghttp.Server, path string) int {
	count := 0
	for _, request := range apiServer.ReceivedRequests() {
		if request.Method == http.MethodGet && request.URL.Path == path {
			count++
		}
	}
	return count
}

var _ = Describe("Addons API client behavior", func() {
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

	It("rejects unsupported billing models during add-on installation", func() {
		err := ocmClient.InstallAddOn("cluster-1", "addon-a", nil, AddOnBilling{
			BillingModel:     "invalid-model",
			BillingAccountID: "123456789012",
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not an valid billing model"))
	})

	It("sends add-on billing and parameter payload during installation", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/addons_mgmt/v1/clusters/cluster-1/addons"),
				func(_ http.ResponseWriter, request *http.Request) {
					body, err := io.ReadAll(request.Body)
					Expect(err).NotTo(HaveOccurred())

					payload := map[string]interface{}{}
					Expect(json.Unmarshal(body, &payload)).To(Succeed())

					addon := payload["addon"].(map[string]interface{})
					Expect(addon["id"]).To(Equal("addon-a"))

					billing := payload["billing"].(map[string]interface{})
					Expect(billing["billing_model"]).To(Equal("marketplace"))
					Expect(billing["billing_marketplace_account"]).To(Equal("123456789012"))

					parameters := payload["parameters"].(map[string]interface{})
					items := parameters["items"].([]interface{})
					Expect(items).To(HaveLen(2))

					serializedParameters := map[string]string{}
					for _, rawItem := range items {
						item := rawItem.(map[string]interface{})
						serializedParameters[item["id"].(string)] = item["value"].(string)
					}
					Expect(serializedParameters).To(Equal(map[string]string{
						"foo": "bar",
						"baz": "qux",
					}))
				},
				RespondWithJSON(http.StatusCreated, `{}`),
			),
		)

		err := ocmClient.InstallAddOn("cluster-1", "addon-a", []AddOnParam{
			{Key: "foo", Val: "bar"},
			{Key: "baz", Val: "qux"},
		}, AddOnBilling{
			BillingModel:     "marketplace",
			BillingAccountID: "123456789012",
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns add-on installation details", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/clusters/cluster-1/addons/addon-a"),
				RespondWithJSON(http.StatusOK, `{
					"id":"addon-a",
					"addon":{"id":"addon-a"},
					"billing":{"billing_model":"standard","billing_marketplace_account":"123456789012"}
				}`),
			),
		)

		installation, err := ocmClient.GetAddOnInstallation("cluster-1", "addon-a")
		Expect(err).NotTo(HaveOccurred())
		Expect(installation).NotTo(BeNil())
		Expect(installation.ID()).To(Equal("addon-a"))
		Expect(installation.Addon().ID()).To(Equal("addon-a"))
	})

	It("filters incompatible add-ons and keeps quota-constrained entries", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"QuotaCostList",
					"page":1,
					"size":1,
					"total":1,
					"items":[{
						"allowed":1,
						"consumed":1,
						"related_resources":[
							{
								"resource_name":"addon-a",
								"cost":1,
								"availability_zone_type":"single",
								"product":"rosa",
								"cloud_provider":"aws",
								"byoc":"byoc"
							},
							{
								"resource_name":"addon-c",
								"cost":1,
								"availability_zone_type":"single",
								"product":"other",
								"cloud_provider":"aws",
								"byoc":"byoc"
							}
						]
					}]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonList",
					"page":1,
					"size":3,
					"total":3,
					"items":[
						{"id":"addon-a","name":"Addon A","resource_name":"addon-a","resource_cost":1},
						{"id":"addon-free","name":"Addon Free","resource_name":"addon-free","resource_cost":0},
						{"id":"addon-c","name":"Addon C","resource_name":"addon-c","resource_cost":1}
					]
				}`),
			),
		)

		addons, err := ocmClient.GetAvailableAddOns()
		Expect(err).NotTo(HaveOccurred())
		Expect(addons).To(HaveLen(2))
		Expect(addons[0].AddOn.ID()).To(Equal("addon-a"))
		Expect(addons[0].Available).To(BeFalse())
		Expect(addons[0].AZType).To(Equal("single"))
		Expect(addons[1].AddOn.ID()).To(Equal("addon-free"))
		Expect(addons[1].Available).To(BeTrue())
		Expect(addons[1].AZType).To(Equal(ANY))
	})

	It("excludes add-ons incompatible with cluster AZ topology", func() {
		cluster, err := cmv1.NewCluster().ID("cluster-1").MultiAZ(false).Build()
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"QuotaCostList",
					"page":1,
					"size":1,
					"total":1,
					"items":[{
						"allowed":2,
						"consumed":0,
						"related_resources":[
							{
								"resource_name":"addon-single",
								"cost":1,
								"availability_zone_type":"single",
								"product":"rosa",
								"cloud_provider":"aws",
								"byoc":"byoc"
							},
							{
								"resource_name":"addon-multi",
								"cost":1,
								"availability_zone_type":"multi",
								"product":"rosa",
								"cloud_provider":"aws",
								"byoc":"byoc"
							}
						]
					}]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonList",
					"page":1,
					"size":2,
					"total":2,
					"items":[
						{"id":"addon-single","name":"Addon Single","resource_name":"addon-single","resource_cost":1},
						{"id":"addon-multi","name":"Addon Multi","resource_name":"addon-multi","resource_cost":1}
					]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/clusters/cluster-1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonInstallationList",
					"page":1,
					"size":1,
					"total":1,
					"items":[{"addon":{"id":"addon-single"}}]
				}`),
			),
		)

		clusterAddons, err := ocmClient.GetClusterAddOns(cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(clusterAddons).To(HaveLen(1))
		Expect(clusterAddons[0].ID).To(Equal("addon-single"))
		Expect(clusterAddons[0].State).To(Equal("installing"))
	})

	It("keeps free add-ons in cluster add-on results without quota metadata", func() {
		cluster, err := cmv1.NewCluster().ID("cluster-1").MultiAZ(false).Build()
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"QuotaCostList",
					"page":1,
					"size":0,
					"total":0,
					"items":[]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonList",
					"page":1,
					"size":1,
					"total":1,
					"items":[
						{"id":"addon-free","name":"Addon Free","resource_name":"addon-free","resource_cost":0}
					]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/clusters/cluster-1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonInstallationList",
					"page":1,
					"size":0,
					"total":0,
					"items":[]
				}`),
			),
		)

		clusterAddons, err := ocmClient.GetClusterAddOns(cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(clusterAddons).To(HaveLen(1))
		Expect(clusterAddons[0].ID).To(Equal("addon-free"))
		Expect(clusterAddons[0].State).To(Equal("not installed"))
	})

	It("keeps free add-ons in cluster add-on results for multi-AZ clusters without quota metadata", func() {
		cluster, err := cmv1.NewCluster().ID("cluster-1").MultiAZ(true).Build()
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"QuotaCostList",
					"page":1,
					"size":0,
					"total":0,
					"items":[]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonList",
					"page":1,
					"size":1,
					"total":1,
					"items":[
						{"id":"addon-free","name":"Addon Free","resource_name":"addon-free","resource_cost":0}
					]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/clusters/cluster-1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonInstallationList",
					"page":1,
					"size":0,
					"total":0,
					"items":[]
				}`),
			),
		)

		clusterAddons, err := ocmClient.GetClusterAddOns(cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(clusterAddons).To(HaveLen(1))
		Expect(clusterAddons[0].ID).To(Equal("addon-free"))
		Expect(clusterAddons[0].State).To(Equal("not installed"))
	})

	It("lets matching quota metadata override free add-on defaults", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"QuotaCostList",
					"page":1,
					"size":1,
					"total":1,
					"items":[{
						"allowed":0,
						"consumed":0,
						"related_resources":[
							{
								"resource_name":"addon-free",
								"cost":1,
								"availability_zone_type":"single",
								"product":"rosa",
								"cloud_provider":"aws",
								"byoc":"byoc"
							}
						]
					}]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonList",
					"page":1,
					"size":1,
					"total":1,
					"items":[
						{"id":"addon-free","name":"Addon Free","resource_name":"addon-free","resource_cost":0}
					]
				}`),
			),
		)

		addons, err := ocmClient.GetAvailableAddOns()
		Expect(err).NotTo(HaveOccurred())
		Expect(addons).To(HaveLen(1))
		Expect(addons[0].AddOn.ID()).To(Equal("addon-free"))
		Expect(addons[0].Available).To(BeFalse())
		Expect(addons[0].AZType).To(Equal("single"))
	})

	It("keeps multi-AZ add-ons for multi-AZ clusters", func() {
		cluster, err := cmv1.NewCluster().ID("cluster-1").MultiAZ(true).Build()
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"QuotaCostList",
					"page":1,
					"size":1,
					"total":1,
					"items":[{
						"allowed":2,
						"consumed":0,
						"related_resources":[
							{
								"resource_name":"addon-single",
								"cost":1,
								"availability_zone_type":"single",
								"product":"rosa",
								"cloud_provider":"aws",
								"byoc":"byoc"
							},
							{
								"resource_name":"addon-multi",
								"cost":1,
								"availability_zone_type":"multi",
								"product":"rosa",
								"cloud_provider":"aws",
								"byoc":"byoc"
							}
						]
					}]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonList",
					"page":1,
					"size":2,
					"total":2,
					"items":[
						{"id":"addon-single","name":"Addon Single","resource_name":"addon-single","resource_cost":1},
						{"id":"addon-multi","name":"Addon Multi","resource_name":"addon-multi","resource_cost":1}
					]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/clusters/cluster-1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonInstallationList",
					"page":1,
					"size":0,
					"total":0,
					"items":[]
				}`),
			),
		)

		clusterAddons, err := ocmClient.GetClusterAddOns(cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(clusterAddons).To(HaveLen(1))
		Expect(clusterAddons[0].ID).To(Equal("addon-multi"))
		Expect(clusterAddons[0].State).To(Equal("not installed"))
	})

	It("returns error when current account lookup fails for available add-ons", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"ACCOUNTS-MGMT-500",
					"reason":"failed to load current account"
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"ACCOUNTS-MGMT-500",
					"reason":"failed to load current account"
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"ACCOUNTS-MGMT-500",
					"reason":"failed to load current account"
				}`),
			),
		)

		addons, err := ocmClient.GetAvailableAddOns()
		Expect(err).To(HaveOccurred())
		Expect(addons).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("failed to load current account"))
		Expect(receivedGETRequestCount(apiServer, "/api/accounts_mgmt/v1/current_account")).To(Equal(expectedRetryAttempts))
	})

	It("returns error when quota cost lookup fails for available add-ons", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"ACCOUNTS-MGMT-500",
					"reason":"failed to load quota cost"
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"ACCOUNTS-MGMT-500",
					"reason":"failed to load quota cost"
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusInternalServerError, `{
					"kind":"Error",
					"id":"500",
					"href":"/api/errors/500",
					"code":"ACCOUNTS-MGMT-500",
					"reason":"failed to load quota cost"
				}`),
			),
		)

		addons, err := ocmClient.GetAvailableAddOns()
		Expect(err).To(HaveOccurred())
		Expect(addons).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("failed to load quota cost"))
		Expect(receivedGETRequestCount(apiServer, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost")).To(Equal(expectedRetryAttempts))
	})

	It("returns error when listing installed add-ons fails", func() {
		cluster, err := cmv1.NewCluster().ID("cluster-1").MultiAZ(false).Build()
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/quota_cost"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"QuotaCostList",
					"page":1,
					"size":0,
					"total":0,
					"items":[]
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/addons"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"AddonList",
					"page":1,
					"size":1,
					"total":1,
					"items":[
						{"id":"addon-free","name":"Addon Free","resource_name":"addon-free","resource_cost":0}
					]
				}`),
			),
		)
		for i := 0; i < expectedRetryAttempts; i++ {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/addons_mgmt/v1/clusters/cluster-1/addons"),
					RespondWithJSON(http.StatusInternalServerError, `{
						"kind":"Error",
						"id":"500",
						"href":"/api/errors/500",
						"code":"ADDONS-MGMT-500",
						"reason":"failed to list installations"
					}`),
				),
			)
		}

		clusterAddOns, err := ocmClient.GetClusterAddOns(cluster)
		Expect(err).To(HaveOccurred())
		Expect(clusterAddOns).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("failed to list installations"))
		Expect(receivedGETRequestCount(apiServer, "/api/addons_mgmt/v1/clusters/cluster-1/addons")).To(Equal(expectedRetryAttempts))
	})
})
