package ocm

import (
	"bytes"
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

var _ = Describe("Node pool warnings", func() {
	It("returns normalized warning headers from create node pool", func() {
		apiServer := MakeTCPServer()
		DeferCleanup(apiServer.Close)

		accessToken := MakeTokenString("Bearer", 15*time.Minute)
		logger, err := logging.NewGoLoggerBuilder().
			Debug(false).
			Build()
		Expect(err).ToNot(HaveOccurred())

		connection, err := sdk.NewConnectionBuilder().
			Logger(logger).
			Tokens(accessToken).
			URL(apiServer.URL()).
			Build()
		Expect(err).ToNot(HaveOccurred())
		client := NewClientWithConnection(connection)
		DeferCleanup(func() {
			Expect(client.Close()).To(Succeed())
		})

		nodePool, err := cmv1.NewNodePool().
			ID("test-nodepool").
			Build()
		Expect(err).ToNot(HaveOccurred())

		var nodePoolBody bytes.Buffer
		err = cmv1.MarshalNodePool(nodePool, &nodePoolBody)
		Expect(err).ToNot(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				func(w http.ResponseWriter, req *http.Request) {
					w.Header().Add("Warning",
						`199 - "Spot NodePool created without termination handler configuration. Nodes will not be gracefully drained on interruptions."`)
				},
				RespondWithJSON(http.StatusCreated, nodePoolBody.String()),
			),
		)

		result, err := client.CreateNodePoolWithWarnings("test-cluster", nodePool)
		Expect(err).ToNot(HaveOccurred())
		Expect(result.NodePool).ToNot(BeNil())
		Expect(result.NodePool.ID()).To(Equal("test-nodepool"))
		Expect(result.Warnings).To(Equal([]string{
			"Spot NodePool created without termination handler configuration. Nodes will not be gracefully drained on interruptions.",
		}))
	})

	It("CreateNodePool preserves the existing body-only behavior", func() {
		apiServer := MakeTCPServer()
		DeferCleanup(apiServer.Close)

		accessToken := MakeTokenString("Bearer", 15*time.Minute)
		logger, err := logging.NewGoLoggerBuilder().
			Debug(false).
			Build()
		Expect(err).ToNot(HaveOccurred())

		connection, err := sdk.NewConnectionBuilder().
			Logger(logger).
			Tokens(accessToken).
			URL(apiServer.URL()).
			Build()
		Expect(err).ToNot(HaveOccurred())
		client := NewClientWithConnection(connection)
		DeferCleanup(func() {
			Expect(client.Close()).To(Succeed())
		})

		nodePool, err := cmv1.NewNodePool().
			ID("test-nodepool").
			Build()
		Expect(err).ToNot(HaveOccurred())

		var nodePoolBody bytes.Buffer
		err = cmv1.MarshalNodePool(nodePool, &nodePoolBody)
		Expect(err).ToNot(HaveOccurred())

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				func(w http.ResponseWriter, req *http.Request) {
					w.Header().Add("Warning", `199 - "warning that should be ignored by CreateNodePool"`)
				},
				RespondWithJSON(http.StatusCreated, nodePoolBody.String()),
			),
		)

		createdNodePool, err := client.CreateNodePool("test-cluster", nodePool)
		Expect(err).ToNot(HaveOccurred())
		Expect(createdNodePool).ToNot(BeNil())
		Expect(createdNodePool.ID()).To(Equal("test-nodepool"))
	})
})

var _ = Describe("NodePools", Ordered, func() {
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

	It("finds node pools using a kubelet config name", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/node_pools"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "NodePoolList",
					"page": 1,
					"size": 3,
					"total": 3,
					"items": [
						{"id":"np-1","kubelet_configs":["kc-a","kc-b"]},
						{"id":"np-2","kubelet_configs":["kc-b"]},
						{"id":"np-3","kubelet_configs":[]}
					]
				}`),
			),
		)

		found, err := ocmClient.FindNodePoolsUsingKubeletConfig(clusterID, "kc-b")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(HaveLen(2))
		Expect(found[0].ID()).To(Equal("np-1"))
		Expect(found[1].ID()).To(Equal("np-2"))
	})

	It("returns an error when listing node pools fails during kubelet search", func() {
		for i := 0; i < 3; i++ {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/node_pools"),
					RespondWithJSON(http.StatusInternalServerError, `{"reason":"broken"}`),
				),
			)
		}

		found, err := ocmClient.FindNodePoolsUsingKubeletConfig(clusterID, "kc-b")
		Expect(err).To(HaveOccurred())
		Expect(found).To(BeEmpty())
	})

	It("returns an empty list when no node pool uses kubelet config", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/node_pools"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "NodePoolList",
					"page": 1,
					"size": 2,
					"total": 2,
					"items": [
						{"id":"np-1","kubelet_configs":["kc-a"]},
						{"id":"np-2","kubelet_configs":["kc-c"]}
					]
				}`),
			),
		)

		found, err := ocmClient.FindNodePoolsUsingKubeletConfig(clusterID, "kc-b")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeEmpty())
	})

	It("returns exists=false,nil error when node pool is not found", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/node_pools/missing"),
				RespondWithJSON(http.StatusNotFound, `{"reason":"not found"}`),
			),
		)

		nodePool, exists, err := ocmClient.GetNodePool(clusterID, "missing")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse())
		Expect(nodePool).To(BeNil())
	})

	It("returns node pool and exists=true when node pool exists", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/node_pools/np-1"),
				RespondWithJSON(http.StatusOK, `{"id":"np-1","kubelet_configs":["kc-a"]}`),
			),
		)

		nodePool, exists, err := ocmClient.GetNodePool(clusterID, "np-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
		Expect(nodePool).NotTo(BeNil())
		Expect(nodePool.ID()).To(Equal("np-1"))
		Expect(nodePool.KubeletConfigs()).To(Equal([]string{"kc-a"}))
	})

	It("creates node pools", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-1/node_pools"),
				func(_ http.ResponseWriter, request *http.Request) {
					payload := map[string]interface{}{}
					Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
					Expect(payload).To(HaveKeyWithValue("id", "np-created"))
					Expect(payload).To(HaveKeyWithValue("kubelet_configs", []interface{}{"kc-a"}))
				},
				RespondWithJSON(http.StatusCreated, `{"id":"np-created","kubelet_configs":["kc-a","kc-b"]}`),
			),
		)
		input, err := cmv1.NewNodePool().ID("np-created").KubeletConfigs("kc-a").Build()
		Expect(err).NotTo(HaveOccurred())

		created, err := ocmClient.CreateNodePool(clusterID, input)
		Expect(err).NotTo(HaveOccurred())
		Expect(created).NotTo(BeNil())
		Expect(created.ID()).To(Equal("np-created"))
		Expect(created.KubeletConfigs()).To(Equal([]string{"kc-a", "kc-b"}))
	})

	It("lists node pools", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/node_pools"),
				ghttp.VerifyFormKV("page", "1"),
				ghttp.VerifyFormKV("size", "-1"),
				RespondWithJSON(http.StatusOK, `{
					"kind":"NodePoolList",
					"page":1,
					"size":1,
					"total":1,
					"items":[{"id":"np-created","kubelet_configs":["kc-a"]}]
				}`),
			),
		)

		nodePools, err := ocmClient.GetNodePools(clusterID)
		Expect(err).NotTo(HaveOccurred())
		Expect(nodePools).To(HaveLen(1))
		Expect(nodePools[0].ID()).To(Equal("np-created"))
		Expect(nodePools[0].KubeletConfigs()).To(Equal([]string{"kc-a"}))
	})

	It("updates node pools", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPatch, "/api/clusters_mgmt/v1/clusters/cluster-1/node_pools/np-created"),
				func(_ http.ResponseWriter, request *http.Request) {
					payload := map[string]interface{}{}
					Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
					Expect(payload).To(HaveKeyWithValue("id", "np-created"))
					Expect(payload).To(HaveKeyWithValue("kubelet_configs", []interface{}{"kc-a", "kc-b"}))
				},
				RespondWithJSON(http.StatusOK, `{"id":"np-created","kubelet_configs":["kc-a","kc-b","kc-c"]}`),
			),
		)

		updateInput, err := cmv1.NewNodePool().ID("np-created").KubeletConfigs("kc-a", "kc-b").Build()
		Expect(err).NotTo(HaveOccurred())

		updated, err := ocmClient.UpdateNodePool(clusterID, updateInput)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).NotTo(BeNil())
		Expect(updated.ID()).To(Equal("np-created"))
		Expect(updated.KubeletConfigs()).To(Equal([]string{"kc-a", "kc-b", "kc-c"}))
	})

	It("deletes node pools", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodDelete, "/api/clusters_mgmt/v1/clusters/cluster-1/node_pools/np-created"),
				RespondWithJSON(http.StatusNoContent, ``),
			),
		)

		err := ocmClient.DeleteNodePool(clusterID, "np-created")
		Expect(err).NotTo(HaveOccurred())
	})
})
