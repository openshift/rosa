// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

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

var _ = Describe("day-2 AWS tag transport", func() {
	var server *ghttp.Server
	var client *Client
	BeforeEach(func() {
		server = MakeTCPServer()
		DeferCleanup(server.Close)
		client = buildTestOCMClient(server.URL())
		DeferCleanup(func() { Expect(client.Close()).To(Succeed(), "close fake OCM connection") })
	})

	It("reads user-only tags and ignores null tombstones", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1", "fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK, `{"aws":{"tags":{"team":"platform"}}}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet,
				"/api/clusters_mgmt/v1/clusters/c1/node_pools/p1", "fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK, `{"aws_node_pool":{"tags":{"team":"platform","old":null}}}`),
		))
		clusterTags, err := client.GetClusterUserTags("c1")
		Expect(err).NotTo(HaveOccurred(), "read customer-only cluster tags")
		Expect(clusterTags).To(Equal(map[string]string{"team": "platform"}), "exclude system cluster tags")
		poolTags, err := client.GetNodePoolUserTags("c1", "p1")
		Expect(err).NotTo(HaveOccurred(), "read customer-only pool tags")
		Expect(poolTags).To(Equal(map[string]string{"team": "platform"}), "ignore null pool tombstone")
	})

	It("combines nullable tag changes with a typed cluster edit", func() {
		server.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"kind":"ClusterList","page":1,"size":1,"total":1,"items":[{"id":"c1"}]}`))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodPatch, "/api/clusters_mgmt/v1/clusters/c1"),
			func(_ http.ResponseWriter, request *http.Request) {
				body, err := io.ReadAll(request.Body)
				Expect(err).NotTo(HaveOccurred(), "read cluster PATCH request body")
				var payload struct {
					AWS struct {
						BillingAccountID string             `json:"billing_account_id"`
						Tags             map[string]*string `json:"tags"`
					} `json:"aws"`
				}
				Expect(json.Unmarshal(body, &payload)).To(Succeed(), "decode cluster PATCH body")
				Expect(payload.AWS.BillingAccountID).To(Equal("billing"), "preserve billing account edit")
				Expect(payload.AWS.Tags).To(HaveKey("remove"), "include removed cluster tag")
				Expect(payload.AWS.Tags["remove"]).To(BeNil(), "encode cluster tag deletion as null")
				Expect(*payload.AWS.Tags["add"]).To(Equal("value"), "include added cluster tag")
				Expect(*payload.AWS.Tags["Cost Center"]).To(Equal("Finance Team"),
					"preserve spaced cluster tag key and value")
			},
			RespondWithJSON(http.StatusOK, `{"id":"c1"}`),
		))
		value := "value"
		spacedValue := "Finance Team"
		Expect(client.UpdateCluster("c1", nil, Spec{BillingAccount: "billing",
			TagChanges: map[string]*string{"add": &value, "remove": nil,
				"Cost Center": &spacedValue}})).To(Succeed(), "combine cluster edits into one PATCH")
	})

	It("combines node-pool changes and reports API validation errors", func() {
		pool, err := cmv1.NewNodePool().ID("p1").Replicas(3).
			AWSNodePool(cmv1.NewAWSNodePool().InstanceType("m5.xlarge")).Build()
		Expect(err).NotTo(HaveOccurred(), "build node-pool update fixture")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodPatch, "/api/clusters_mgmt/v1/clusters/c1/node_pools/p1"),
			func(_ http.ResponseWriter, request *http.Request) {
				body, err := io.ReadAll(request.Body)
				Expect(err).NotTo(HaveOccurred(), "read node-pool PATCH request body")
				var payload struct {
					Replicas    int `json:"replicas"`
					AWSNodePool struct {
						InstanceType string             `json:"instance_type"`
						Tags         map[string]*string `json:"tags"`
					} `json:"aws_node_pool"`
				}
				Expect(json.Unmarshal(body, &payload)).To(Succeed(), "decode node-pool PATCH body")
				Expect(payload.Replicas).To(Equal(3), "preserve replica edit")
				Expect(payload.AWSNodePool.InstanceType).To(Equal("m5.xlarge"), "preserve instance type")
				Expect(payload.AWSNodePool.Tags).To(HaveKeyWithValue("remove", BeNil()),
					"encode pool tag deletion as null")
				Expect(*payload.AWSNodePool.Tags["Cost Center"]).To(Equal("Finance Team"),
					"preserve spaced pool tag key and value")
			},
			RespondWithJSON(http.StatusBadRequest,
				`{"kind":"Error","reason":"reserved tag Cost Center=Finance Team"}`),
		))
		spacedValue := "Finance Team"
		_, err = client.UpdateNodePoolWithTags("c1", pool,
			map[string]*string{"remove": nil, "Cost Center": &spacedValue})
		Expect(err).To(HaveOccurred(), "propagate node-pool API validation failure")
		Expect(err.Error()).To(ContainSubstring("reserved tag"), "retain API validation reason")
		Expect(err.Error()).To(ContainSubstring("Cost Center"), "retain tag key in API validation reason")
		Expect(err.Error()).To(ContainSubstring("Finance Team"), "retain tag value in API validation reason")
	})

})
