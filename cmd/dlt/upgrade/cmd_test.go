package upgrade

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Delete upgrade", func() {
	var testRuntime test.TestingRuntime

	mockClusterError := test.MockCluster(func(c *cmv1.ClusterBuilder) {
		c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
		c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
		c.State(cmv1.ClusterStateError)
		c.Hypershift(cmv1.NewHypershift().Enabled(true))
	})
	var hypershiftClusterNotReady = test.FormatClusterList([]*cmv1.Cluster{mockClusterError})

	mockClassicCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
		c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
		c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
		c.State(cmv1.ClusterStateReady)
		c.Hypershift(cmv1.NewHypershift().Enabled(false))
	})

	var classicCluster = test.FormatClusterList([]*cmv1.Cluster{mockClassicCluster})

	BeforeEach(func() {
		testRuntime.InitRuntime()
	})
	It("Fails if cluster is not ready", func() {
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterNotReady))
		err := runWithRuntime(testRuntime.RosaRuntime)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("Cluster 'cluster1' is not yet ready"))
	})
	It("Fails if cluster is not hypershift and we are using hypershift specific flags", func() {
		args.nodePool = "nodepool1"
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicCluster))
		err := runWithRuntime(testRuntime.RosaRuntime)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("The '--machinepool' option is only supported for Hosted Control Planes"))
	})
})
