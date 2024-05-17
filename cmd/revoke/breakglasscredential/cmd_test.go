package breakglasscredential

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Revoke break-glass-credential", func() {
	var testRuntime test.TestingRuntime

	Context("Revoke break glass credential command", func() {
		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
			c.ExternalAuthConfig(cmv1.NewExternalAuthConfig().Enabled(true))
		})
		hypershiftClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		BeforeEach(func() {
			testRuntime.InitRuntime()
			// Reset flag to avoid any side effect on other tests
			Cmd.Flags().Set("output", "")
		})

		It("Warning with zero results", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{})
			Expect(err).To(BeNil())
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal("INFO: There are no break glass credentials for cluster 'cluster1'\n"))
		})
	})
})
