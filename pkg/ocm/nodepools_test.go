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

var _ = Describe("NodePools", Ordered, func() {
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

	It("finds node pools using a kubelet config name", func() {
		apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, `{
			"kind": "NodePoolList",
			"page": 1,
			"size": 3,
			"total": 3,
			"items": [
				{"id":"np-1","kubelet_configs":["kc-a","kc-b"]},
				{"id":"np-2","kubelet_configs":["kc-b"]},
				{"id":"np-3","kubelet_configs":[]}
			]
		}`))

		found, err := ocmClient.FindNodePoolsUsingKubeletConfig(clusterID, "kc-b")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(HaveLen(2))
		Expect(found[0].ID()).To(Equal("np-1"))
		Expect(found[1].ID()).To(Equal("np-2"))
	})

	It("returns an error when listing node pools fails during kubelet search", func() {
		apiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, `{"reason":"broken"}`))

		found, err := ocmClient.FindNodePoolsUsingKubeletConfig(clusterID, "kc-b")
		Expect(err).To(HaveOccurred())
		Expect(found).To(BeEmpty())
	})

	It("returns an empty list when no node pool uses kubelet config", func() {
		apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, `{
			"kind": "NodePoolList",
			"page": 1,
			"size": 2,
			"total": 2,
			"items": [
				{"id":"np-1","kubelet_configs":["kc-a"]},
				{"id":"np-2","kubelet_configs":["kc-c"]}
			]
		}`))

		found, err := ocmClient.FindNodePoolsUsingKubeletConfig(clusterID, "kc-b")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeEmpty())
	})

	It("returns exists=false,nil error when node pool is not found", func() {
		apiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, `{"reason":"not found"}`))

		nodePool, exists, err := ocmClient.GetNodePool(clusterID, "missing")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse())
		Expect(nodePool).To(BeNil())
	})

	It("returns node pool and exists=true when node pool exists", func() {
		apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, `{"id":"np-1","kubelet_configs":["kc-a"]}`))

		nodePool, exists, err := ocmClient.GetNodePool(clusterID, "np-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
		Expect(nodePool).NotTo(BeNil())
		Expect(nodePool.ID()).To(Equal("np-1"))
	})

	It("supports basic node pool CRUD wrappers", func() {
		apiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, `{"id":"np-created","kubelet_configs":["kc-a"]}`))
		apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, `{
			"kind":"NodePoolList",
			"page":1,
			"size":1,
			"total":1,
			"items":[{"id":"np-created","kubelet_configs":["kc-a"]}]
		}`))
		apiServer.AppendHandlers(RespondWithJSON(http.StatusOK, `{"id":"np-created","kubelet_configs":["kc-a","kc-b"]}`))
		apiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ``))

		input, err := cmv1.NewNodePool().ID("np-created").Build()
		Expect(err).NotTo(HaveOccurred())

		created, err := ocmClient.CreateNodePool(clusterID, input)
		Expect(err).NotTo(HaveOccurred())
		Expect(created).NotTo(BeNil())
		Expect(created.ID()).To(Equal("np-created"))

		nodePools, err := ocmClient.GetNodePools(clusterID)
		Expect(err).NotTo(HaveOccurred())
		Expect(nodePools).To(HaveLen(1))
		Expect(nodePools[0].ID()).To(Equal("np-created"))

		updateInput, err := cmv1.NewNodePool().ID("np-created").KubeletConfigs("kc-a", "kc-b").Build()
		Expect(err).NotTo(HaveOccurred())

		updated, err := ocmClient.UpdateNodePool(clusterID, updateInput)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).NotTo(BeNil())
		Expect(updated.KubeletConfigs()).To(Equal([]string{"kc-a", "kc-b"}))

		err = ocmClient.DeleteNodePool(clusterID, "np-created")
		Expect(err).NotTo(HaveOccurred())
	})
})
