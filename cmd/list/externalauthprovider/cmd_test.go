package externalauthprovider

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("list external-auth-provider", func() {

	var testRuntime test.TestingRuntime

	Context("List external authentication provider command", func() {

		mockClusterReady, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
			c.ExternalAuthConfig(cmv1.NewExternalAuthConfig().Enabled(true))
		})
		Expect(err).To(BeNil())
		hypershiftClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		externalAuths := make([]*cmv1.ExternalAuth, 0)
		externalAuths = append(externalAuths, test.BuildExternalAuth())

		BeforeEach(func() {
			testRuntime.InitRuntime()
			// Reset flag to avoid any side effect on other tests
			Cmd.Flags().Set("output", "")
		})

		It("Fails with zero results", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(
				ContainSubstring("there are no external authentication providers for this cluster"))
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal(""))
		})

		It("Succeeds", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatExternalAuthList(externalAuths)))
			_, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(err).To(BeNil())
		})

	})

})
