// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

type tagWarningReporter struct {
	reporter.Logger
	warnings []string
}

func (r *tagWarningReporter) Warnf(format string, args ...any) {
	r.warnings = append(r.warnings, fmt.Sprintf(format, args...))
}

var _ = Describe("node-pool day-2 AWS tag edit", func() {
	var cmd *cobra.Command
	var runtime *rosa.Runtime
	var server *ghttp.Server
	var pool *cmv1.NodePool

	BeforeEach(func() {
		interactive.SetEnabled(false)
		DeferCleanup(func() { interactive.SetEnabled(false) })
		cmd = &cobra.Command{}
		cmd.Flags().StringArray("tags", nil, "")
		server = MakeTCPServer()
		DeferCleanup(server.Close)
		logger, err := logging.NewGoLoggerBuilder().Debug(false).Build()
		Expect(err).NotTo(HaveOccurred(), "build test logger for node-pool tag edits")
		connection, err := sdk.NewConnectionBuilder().Logger(logger).
			Tokens(MakeTokenString("Bearer", 15*time.Minute)).URL(server.URL()).Build()
		Expect(err).NotTo(HaveOccurred(), "connect fake OCM server for node-pool tag edits")
		runtime = rosa.NewRuntime()
		runtime.OCMClient = ocm.NewClientWithConnection(connection)
		DeferCleanup(runtime.Cleanup)
		pool, err = cmv1.NewNodePool().ID("p1").Build()
		Expect(err).NotTo(HaveOccurred(), "build node-pool p1 fixture")
	})

	cluster := func(hosted bool, version string) *cmv1.Cluster {
		object, err := cmv1.NewCluster().ID("c1").
			Hypershift(cmv1.NewHypershift().Enabled(hosted)).
			Version(cmv1.NewVersion().RawID(version)).Build()
		Expect(err).NotTo(HaveOccurred(), "build hosted=%t version=%q cluster fixture", hosted, version)
		return object
	}

	It("replaces overrides and clears possible equal-value pool tags", func() {
		warnings := &tagWarningReporter{Logger: runtime.Reporter}
		runtime.Reporter = warnings
		Expect(cmd.Flags().Set("tags", "team:platform,new:yes")).To(Succeed(), "set desired pool tags")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1",
				"fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK, `{"aws":{"tags":{"team":"platform"}}}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1/node_pools/p1",
				"fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK,
				`{"aws_node_pool":{"tags":{"team":"platform","old":"yes","deleted":null}}}`),
		))
		changes, err := prepareNodePoolTagEdit(cmd, cluster(true, "openshift-4.20.0"), pool, runtime)
		Expect(err).NotTo(HaveOccurred(), "prepare pool overrides")
		Expect(changes).To(HaveLen(3), "add a pool tag and remove known or possible overrides")
		Expect(*changes["new"]).To(Equal("yes"), "add new pool tag")
		Expect(changes).To(HaveKeyWithValue("old", BeNil()), "delete old pool tag")
		Expect(changes).To(HaveKeyWithValue("team", BeNil()),
			"remove a possible stored override equal to the inherited cluster tag")
		Expect(warnings.warnings).To(HaveLen(1), "warn once for a pool tag matching the cluster tag")
		Expect(warnings.warnings[0]).To(Equal(`Tag "team:platform" matches an existing cluster tag and won't be added as a machine pool override; a later cluster tag change may change it.`),
			"show the matching tag value and explain why it will not become an override")
		Expect(server.ReceivedRequests()).To(HaveLen(2), "read both cluster and pool customer tags")
	})

	It("does not patch when the desired overrides match without inherited ambiguity", func() {
		warnings := &tagWarningReporter{Logger: runtime.Reporter}
		runtime.Reporter = warnings
		Expect(cmd.Flags().Set("tags", "own:value")).To(Succeed(), "set unchanged pool override")
		server.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"aws":{"tags":{}}}`))
		server.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"aws_node_pool":{"tags":{"own":"value"}}}`))
		changes, err := prepareNodePoolTagEdit(cmd, cluster(true, "4.20.0"), pool, runtime)
		Expect(err).NotTo(HaveOccurred(), "prepare unchanged pool overrides")
		Expect(changes).To(BeNil(), "matching overrides must produce no diff")
		Expect(warnings.warnings).To(BeEmpty(), "do not warn when no tag matches a cluster tag")
		Expect(server.ReceivedRequests()).To(HaveLen(2), "no-op should only read tags")
		Expect(onlyMachinePoolTagsChanged(cmd)).To(BeTrue(), "tags are the only pool edit flag")
	})

	It("clears a stored pool override that later matches the cluster tag", func() {
		clusterTags := map[string]string{}
		storedPoolTags := map[string]string{}
		var patches []map[string]*string
		server.RouteToHandler(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1",
			func(writer http.ResponseWriter, request *http.Request) {
				Expect(request.URL.Query().Get("fetchUserTagsOnly")).To(Equal("true"),
					"read customer-only cluster tags")
				writer.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(writer).Encode(map[string]any{
					"aws": map[string]any{"tags": clusterTags},
				})).To(Succeed(), "respond with current cluster tags")
			})
		server.RouteToHandler(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1/node_pools/p1",
			func(writer http.ResponseWriter, request *http.Request) {
				Expect(request.URL.Query().Get("fetchUserTagsOnly")).To(Equal("true"),
					"read customer-only effective pool tags")
				effective := maps.Clone(clusterTags)
				maps.Copy(effective, storedPoolTags)
				writer.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(writer).Encode(map[string]any{
					"aws_node_pool": map[string]any{"tags": effective},
				})).To(Succeed(), "respond with merged cluster and stored pool tags")
			})
		server.RouteToHandler(http.MethodPatch, "/api/clusters_mgmt/v1/clusters/c1/node_pools/p1",
			func(writer http.ResponseWriter, request *http.Request) {
				var patch struct {
					AWSNodePool struct {
						Tags map[string]*string `json:"tags"`
					} `json:"aws_node_pool"`
				}
				Expect(json.NewDecoder(request.Body).Decode(&patch)).To(Succeed(),
					"decode pool tag PATCH")
				patches = append(patches, patch.AWSNodePool.Tags)
				for key, value := range patch.AWSNodePool.Tags {
					if value == nil {
						delete(storedPoolTags, key)
					} else {
						storedPoolTags[key] = *value
					}
				}
				writer.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(writer).Encode(map[string]string{"id": "p1"})).To(Succeed(),
					"respond to pool tag PATCH")
			})

		Expect(cmd.Flags().Set("tags", "team:platform")).To(Succeed(), "set pool override")
		changes, err := prepareNodePoolTagEdit(cmd, cluster(true, "4.20.0"), pool, runtime)
		Expect(err).NotTo(HaveOccurred(), "prepare initial pool override")
		_, err = runtime.OCMClient.UpdateNodePoolWithTags("c1", pool, changes)
		Expect(err).NotTo(HaveOccurred(), "store initial pool override")
		Expect(storedPoolTags).To(Equal(map[string]string{"team": "platform"}),
			"pool should store its own team tag")

		clusterTags["team"] = "platform"
		effective, err := runtime.OCMClient.GetNodePoolUserTags("c1", "p1")
		Expect(err).NotTo(HaveOccurred(), "read pool tags after cluster value matches")
		Expect(effective).To(Equal(map[string]string{"team": "platform"}),
			"merged read cannot distinguish the stored override from inheritance")

		clearCmd := &cobra.Command{}
		clearCmd.Flags().StringArray("tags", nil, "")
		Expect(clearCmd.Flags().Set("tags", "")).To(Succeed(), "clear pool-specific tags")
		changes, err = prepareNodePoolTagEdit(clearCmd, cluster(true, "4.20.0"), pool, runtime)
		Expect(err).NotTo(HaveOccurred(), "prepare clear of matching stored pool override")
		Expect(changes).To(HaveKeyWithValue("team", BeNil()),
			"send null for the possible equal-value pool override")
		_, err = runtime.OCMClient.UpdateNodePoolWithTags("c1", pool, changes)
		Expect(err).NotTo(HaveOccurred(), "remove stored pool override")
		Expect(patches).To(HaveLen(2), "send one add and one removal PATCH")
		Expect(patches[1]).To(HaveKeyWithValue("team", BeNil()),
			"encode pool override removal as JSON null")
		Expect(storedPoolTags).To(BeEmpty(), "pool should no longer store the equal-value override")

		clusterTags["team"] = "finance"
		effective, err = runtime.OCMClient.GetNodePoolUserTags("c1", "p1")
		Expect(err).NotTo(HaveOccurred(), "read pool tags after cluster tag changes")
		Expect(effective).To(Equal(map[string]string{"team": "finance"}),
			"pool should inherit the later cluster tag value after override removal")
	})

	It("preserves a spaced pool tag key and value from the --tags flag", func() {
		Expect(cmd.Flags().Set("tags", `"Cost Center":"Finance Team"`)).To(Succeed(),
			"set quoted pool tag containing spaces")
		server.AppendHandlers(RespondWithJSON(http.StatusOK, `{"aws":{"tags":{}}}`))
		server.AppendHandlers(RespondWithJSON(http.StatusOK, `{"aws_node_pool":{"tags":{}}}`))
		changes, err := prepareNodePoolTagEdit(cmd, cluster(true, "4.20.0"), pool, runtime)
		Expect(err).NotTo(HaveOccurred(), "parse quoted pool tag")
		Expect(changes).To(HaveKey("Cost Center"), "preserve spaced pool tag key")
		Expect(*changes["Cost Center"]).To(Equal("Finance Team"), "preserve spaced pool tag value")
	})

	It("recognizes a tag edit combined with another pool edit", func() {
		cmd.Flags().Int("replicas", 0, "")
		Expect(cmd.Flags().Set("tags", "own:value")).To(Succeed(), "set pool tag")
		Expect(cmd.Flags().Set("replicas", "2")).To(Succeed(), "set pool replicas")
		Expect(onlyMachinePoolTagsChanged(cmd)).To(BeFalse(), "replica edit requires combined PATCH")
	})

	It("keeps enabled interactive edits out of the tag-only no-op shortcut", func() {
		interactive.AddFlag(cmd.Flags())
		Expect(cmd.Flags().Set("tags", "own:value")).To(Succeed(), "set pool tags")
		Expect(cmd.Flags().Set("interactive", "true")).To(Succeed(), "request interactive editing")
		Expect(interactive.Enabled()).To(BeTrue(), "interactive flag should enable prompts")
		Expect(onlyMachinePoolTagsChanged(cmd)).To(BeFalse(),
			"an explicit interactive flag must allow later machine-pool edit prompts")
	})

	It("keeps the tag-only no-op shortcut for interactive=false", func() {
		interactive.AddFlag(cmd.Flags())
		Expect(cmd.Flags().Set("tags", "own:value")).To(Succeed(), "set pool tags")
		Expect(cmd.Flags().Set("interactive", "false")).To(Succeed(), "disable interactive editing explicitly")
		Expect(interactive.Enabled()).To(BeFalse(), "interactive=false should leave prompts disabled")
		Expect(onlyMachinePoolTagsChanged(cmd)).To(BeTrue(),
			"explicit interactive=false must not prevent a tag-only no-op")
	})

	It("keeps globally enabled interactive edits out of the tag-only no-op shortcut", func() {
		Expect(cmd.Flags().Set("tags", "own:value")).To(Succeed(), "set pool tags")
		interactive.SetEnabled(true)
		Expect(onlyMachinePoolTagsChanged(cmd)).To(BeFalse(),
			"global interactive mode must allow later machine-pool edit prompts")
	})

	It("rejects classic edits before an OCM request", func() {
		Expect(cmd.Flags().Set("tags", "team:platform")).To(Succeed(), "set classic pool tag edit")
		readyClassic, err := cmv1.NewCluster().ID("c1").State(cmv1.ClusterStateReady).
			Hypershift(cmv1.NewHypershift().Enabled(false)).
			Version(cmv1.NewVersion().RawID("4.20.0")).Build()
		Expect(err).NotTo(HaveOccurred(), "build ready classic cluster fixture")
		err = (&machinePool{}).EditMachinePool(cmd, "p1", "c1",
			readyClassic, runtime)
		Expect(err).To(HaveOccurred(), "classic machine pool must reject tag edits")
		Expect(err.Error()).To(ContainSubstring("Hosted Control Plane"),
			"reject classic tagging because the cluster is not HCP")
		Expect(server.ReceivedRequests()).To(BeEmpty(), "reject classic edit before OCM calls")
	})

	It("skips the PATCH for unchanged tag-only node-pool edits", func() {
		Expect(cmd.Flags().Set("tags", "own:value")).To(Succeed(), "set unchanged pool tag")
		readyCluster, err := cmv1.NewCluster().ID("c1").State(cmv1.ClusterStateReady).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			Version(cmv1.NewVersion().RawID("openshift-4.20.0")).Build()
		Expect(err).NotTo(HaveOccurred(), "build ready HCP cluster fixture")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1/autoscaler"),
			RespondWithJSON(http.StatusNotFound, `{"kind":"Error","reason":"not found"}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1/node_pools/p1"),
			RespondWithJSON(http.StatusOK, `{"id":"p1","replicas":1}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1",
				"fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK, `{"aws":{"tags":{}}}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1/node_pools/p1",
				"fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK, `{"aws_node_pool":{"tags":{"own":"value"}}}`),
		))
		Expect((&machinePool{}).EditMachinePool(cmd, "p1", "c1", readyCluster, runtime)).To(Succeed(),
			"run unchanged pool tag edit")
		Expect(server.ReceivedRequests()).To(HaveLen(4), "unchanged edit must not PATCH")
	})

	It("PATCHes only changed tags for a tag-only node-pool edit", func() {
		Expect(cmd.Flags().Set("tags", `"Cost Center":"Finance Team"`)).To(Succeed(),
			"set quoted pool tag for PATCH")
		readyCluster, err := cmv1.NewCluster().ID("c1").State(cmv1.ClusterStateReady).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			Version(cmv1.NewVersion().RawID("openshift-4.20.0")).Build()
		Expect(err).NotTo(HaveOccurred(), "build ready HCP cluster fixture")
		server.AppendHandlers(RespondWithJSON(http.StatusNotFound,
			`{"kind":"Error","reason":"not found"}`))
		server.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"id":"p1","replicas":1}`))
		server.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"aws":{"tags":{}}}`))
		server.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"aws_node_pool":{"tags":{"old":"value"}}}`))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodPatch,
				"/api/clusters_mgmt/v1/clusters/c1/node_pools/p1"),
			func(_ http.ResponseWriter, request *http.Request) {
				body, readErr := io.ReadAll(request.Body)
				Expect(readErr).NotTo(HaveOccurred(), "read node-pool PATCH body")
				var patch struct {
					AWSNodePool struct {
						Tags map[string]*string `json:"tags"`
					} `json:"aws_node_pool"`
				}
				Expect(json.Unmarshal(body, &patch)).To(Succeed(), "decode node-pool PATCH body")
				Expect(patch.AWSNodePool.Tags).To(HaveLen(2), "PATCH only changed pool tags")
				Expect(patch.AWSNodePool.Tags).To(HaveKeyWithValue("old", BeNil()), "encode deletion as null")
				Expect(*patch.AWSNodePool.Tags["Cost Center"]).To(Equal("Finance Team"),
					"encode spaced pool tag value")
			},
			RespondWithJSON(http.StatusOK, `{"id":"p1"}`),
		))
		Expect((&machinePool{}).EditMachinePool(cmd, "p1", "c1", readyCluster, runtime)).To(Succeed(),
			"PATCH changed pool tags")
		Expect(server.ReceivedRequests()).To(HaveLen(5), "issue one PATCH after four setup reads")
	})

	It("combines a pool tag replacement and replica edit in one PATCH", func() {
		cmd.Flags().Int("replicas", 0, "")
		cmd.Flags().Int("min-replicas", 0, "")
		cmd.Flags().Int("max-replicas", 0, "")
		cmd.Flags().Bool("enable-autoscaling", false, "")
		cmd.Flags().String("labels", "", "")
		cmd.Flags().String("taints", "", "")
		Expect(cmd.Flags().Set("tags", `"Cost Center":"Finance Team"`)).To(Succeed(),
			"set replacement pool tag with spaces")
		Expect(cmd.Flags().Set("replicas", "3")).To(Succeed(), "set requested replica count")
		readyCluster, err := cmv1.NewCluster().ID("c1").State(cmv1.ClusterStateReady).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			Version(cmv1.NewVersion().RawID("openshift-4.20.0")).Build()
		Expect(err).NotTo(HaveOccurred(), "build ready HCP fixture for combined edit")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1/autoscaler"),
			RespondWithJSON(http.StatusNotFound, `{"kind":"Error","reason":"not found"}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1/node_pools/p1"),
			RespondWithJSON(http.StatusOK, `{"id":"p1","replicas":1}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1",
				"fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK, `{"aws":{"tags":{}}}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1/node_pools/p1",
				"fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK, `{"aws_node_pool":{"tags":{"old":"value"}}}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1/node_pools"),
			RespondWithJSON(http.StatusOK,
				`{"kind":"NodePoolList","page":1,"size":0,"total":0,"items":[]}`),
		))
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodPatch, "/api/clusters_mgmt/v1/clusters/c1/node_pools/p1"),
			func(_ http.ResponseWriter, request *http.Request) {
				body, readErr := io.ReadAll(request.Body)
				Expect(readErr).NotTo(HaveOccurred(), "read combined node-pool PATCH body")
				var patch struct {
					Replicas    int `json:"replicas"`
					AWSNodePool struct {
						Tags map[string]*string `json:"tags"`
					} `json:"aws_node_pool"`
				}
				Expect(json.Unmarshal(body, &patch)).To(Succeed(), "decode combined node-pool PATCH")
				Expect(patch.Replicas).To(Equal(3), "retain requested replica edit")
				Expect(patch.AWSNodePool.Tags).To(HaveLen(2), "send only changed pool tags")
				Expect(patch.AWSNodePool.Tags).To(HaveKeyWithValue("old", BeNil()),
					"encode omitted pool tag as null")
				Expect(*patch.AWSNodePool.Tags["Cost Center"]).To(Equal("Finance Team"),
					"preserve spaced pool tag key and value")
			},
			RespondWithJSON(http.StatusOK, `{"id":"p1","replicas":3}`),
		))
		Expect((&machinePool{}).EditMachinePool(cmd, "p1", "c1", readyCluster, runtime)).To(Succeed(),
			"combine pool tag replacement and replica change")
		Expect(server.ReceivedRequests()).To(HaveLen(6), "issue exactly one PATCH after setup reads")
	})

	DescribeTable("rejects unsupported clusters before reading tags",
		func(hosted bool, version string) {
			Expect(cmd.Flags().Set("tags", "team:platform")).To(Succeed(), "set tags for version %q", version)
			changes, err := prepareNodePoolTagEdit(cmd, cluster(hosted, version), pool, runtime)
			Expect(err).To(HaveOccurred(), "reject hosted=%t version=%q", hosted, version)
			Expect(changes).To(BeNil(), "rejected pool must have no tag diff")
			Expect(server.ReceivedRequests()).To(BeEmpty(), "reject before reading tags")
		},
		Entry("classic", false, "4.20.0"),
		Entry("older HCP", true, "4.19.20"),
		Entry("unknown version", true, ""),
		Entry("invalid version", true, "garbage"),
	)
})
