package breakglasscredential

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

const (
	breakGlassCredentialId = "test-id"
	describeStringOutput   = `
ID:                                    test-id
Username:                              username
Expire at:                             Jan  1 0001 00:00:00 UTC
Status:                                issued
`
)

var _ = Describe("Break glass credential", func() {
	var testRuntime test.TestingRuntime
	var err error

	Context("Describe break glass credential command", func() {
		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
			c.ExternalAuthConfig(cmv1.NewExternalAuthConfig().Enabled(true))
		})
		hypershiftClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		breakGlassCredentials := make([]*cmv1.BreakGlassCredential, 0)
		breakGlassCredentials = append(breakGlassCredentials, test.BuildBreakGlassCredential())

		BeforeEach(func() {
			testRuntime.InitRuntime()
			// Reset flag to avoid any side effect on other tests
			Cmd.Flags().Set("output", "")
		})

		It("Fails if we are not specifying a break glass credential id", func() {
			args.id = ""
			_, _, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime, Cmd, &[]string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(
				"you need to specify a break glass credential id with '--id' parameter"))
		})

		It("Pass a break glass credential id through parameter and it is found", func() {
			args.id = breakGlassCredentialId
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound,
				test.FormatBreakGlassCredentialList(breakGlassCredentials)))
			Expect(err).To(BeNil())
			output := describeBreakGlassCredential(testRuntime.RosaRuntime,
				mockClusterReady, test.MockClusterID, test.BuildBreakGlassCredential())
			Expect(output).To(Equal(describeStringOutput))
		})

	})
})
