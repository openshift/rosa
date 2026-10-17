// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

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
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("cluster day-2 AWS tag edit", func() {
	var cmd *cobra.Command
	var runtime *rosa.Runtime
	var server *ghttp.Server

	BeforeEach(func() {
		interactive.SetEnabled(false)
		DeferCleanup(func() { interactive.SetEnabled(false) })
		cmd = makeCmd()
		initFlags(cmd)
		server = MakeTCPServer()
		DeferCleanup(server.Close)
		logger, err := logging.NewGoLoggerBuilder().Debug(false).Build()
		Expect(err).NotTo(HaveOccurred(), "build test logger for cluster tag edits")
		connection, err := sdk.NewConnectionBuilder().Logger(logger).
			Tokens(MakeTokenString("Bearer", 15*time.Minute)).URL(server.URL()).Build()
		Expect(err).NotTo(HaveOccurred(), "connect fake OCM server for cluster tag edits")
		runtime = rosa.NewRuntime()
		runtime.OCMClient = ocm.NewClientWithConnection(connection)
		DeferCleanup(runtime.Cleanup)
	})

	cluster := func(hosted bool, version string) *cmv1.Cluster {
		object, err := cmv1.NewCluster().ID("c1").
			Hypershift(cmv1.NewHypershift().Enabled(hosted)).
			Version(cmv1.NewVersion().RawID(version)).Build()
		Expect(err).NotTo(HaveOccurred(), "build %q cluster fixture", version)
		return object
	}

	It("accepts an empty flag as a request to clear tags", func() {
		Expect(cmd.Flags().Set("tags", "")).To(Succeed(), "set empty cluster tag flag")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1",
				"fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK, `{"aws":{"tags":{"old":"value"}}}`),
		))
		changes, err := prepareClusterTagEdit(runtime, cmd, cluster(true, "openshift-4.20.0"))
		Expect(err).NotTo(HaveOccurred(), "prepare cluster tag clear")
		Expect(changes).To(HaveKeyWithValue("old", BeNil()), "clear must delete the old tag")
		Expect(server.ReceivedRequests()).To(HaveLen(1), "clear should only read customer tags")
	})

	It("returns no changes for an identical desired set", func() {
		Expect(cmd.Flags().Set("tags", "team:platform")).To(Succeed(), "set unchanged cluster tags")
		server.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"aws":{"tags":{"team":"platform"}}}`))
		changes, err := prepareClusterTagEdit(runtime, cmd, cluster(true, "4.20.0"))
		Expect(err).NotTo(HaveOccurred(), "prepare unchanged cluster tags")
		Expect(changes).To(BeNil(), "identical desired tags must produce no diff")
		Expect(server.ReceivedRequests()).To(HaveLen(1), "no-op should only read customer tags")
		Expect(onlyClusterTagsChanged(cmd)).To(BeTrue(), "tags are the only edit flag")
	})

	It("skips the cluster PATCH for an unchanged tag-only edit", func() {
		Expect(cmd.Flags().Set("cluster", "c1")).To(Succeed(), "select cluster c1")
		Expect(cmd.Flags().Set("tags", "team:platform")).To(Succeed(), "set unchanged cluster tags")
		runtime.Cluster = cluster(true, "openshift-4.20.0")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/c1",
				"fetchUserTagsOnly=true"),
			RespondWithJSON(http.StatusOK, `{"aws":{"tags":{"team":"platform"}}}`),
		))
		Expect(runWithRuntime(runtime, cmd)).To(Succeed(), "run unchanged cluster tag edit")
		Expect(server.ReceivedRequests()).To(HaveLen(1), "unchanged edit must not PATCH")
	})

	It("preserves a spaced key and value from the --tags flag", func() {
		Expect(cmd.Flags().Set("tags", `"Cost Center":"Finance Team"`)).To(Succeed(),
			"set quoted cluster tag containing spaces")
		server.AppendHandlers(RespondWithJSON(http.StatusOK, `{"aws":{"tags":{}}}`))
		changes, err := prepareClusterTagEdit(runtime, cmd, cluster(true, "4.20.0"))
		Expect(err).NotTo(HaveOccurred(), "parse quoted cluster tag")
		Expect(changes).To(HaveKey("Cost Center"), "preserve spaced tag key")
		Expect(*changes["Cost Center"]).To(Equal("Finance Team"), "preserve spaced tag value")
	})

	It("keeps ordinary edits in a combined tag PATCH", func() {
		Expect(cmd.Flags().Set("tags", "team:platform")).To(Succeed(), "set cluster tags")
		Expect(cmd.Flags().Set("billing-account", "billing")).To(Succeed(), "set billing account")
		Expect(onlyClusterTagsChanged(cmd)).To(BeFalse(), "billing edit must not be treated as tag-only")
	})

	DescribeTable("rejects ineligible clusters before reading tags",
		func(hosted bool, version string) {
			Expect(cmd.Flags().Set("tags", "team:platform")).To(Succeed(), "set tags for %q", version)
			changes, err := prepareClusterTagEdit(runtime, cmd, cluster(hosted, version))
			Expect(err).To(HaveOccurred(), "reject hosted=%t version=%q", hosted, version)
			Expect(changes).To(BeNil(), "rejected cluster must have no tag diff")
			Expect(server.ReceivedRequests()).To(BeEmpty(), "reject before reading tags")
		},
		Entry("classic", false, "4.20.0"),
		Entry("older HCP", true, "4.19.20"),
		Entry("unknown version", true, ""),
		Entry("invalid version", true, "garbage"),
	)
})
