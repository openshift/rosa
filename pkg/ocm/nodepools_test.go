package ocm

import (
	"bytes"
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
