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
	//nolint:lll
	describeStringOutput = `INFO: To retrieve only the kubeconfig for this credential use: 'rosa describe break-glass-credential test-id -c cluster1 --kubeconfig'

ID:                                    test-id
Username:                              username
Expire at:                             Jan  1 0001 00:00:00 UTC
Status:                                issued
`
)

var _ = Describe("Break glass credential", func() {
	var testRuntime test.TestingRuntime

	Context("Describe break glass credential command", func() {
		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
			c.ExternalAuthConfig(cmv1.NewExternalAuthConfig().Enabled(true))
		})
		hypershiftClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		credential := test.BuildBreakGlassCredential()

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
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatResource(credential)))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{})
			Expect(err).To(BeNil())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(Equal(describeStringOutput))
		})

		It("Pass a break glass credential id and --kubeconfig through parameter and it is found", func() {
			args.id = breakGlassCredentialId
			args.kubeconfig = true
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatResource(credential)))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{})
			Expect(err).To(BeNil())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(Equal(
				"INFO: The credential is not ready yet. Please wait a few minutes for it to be fully ready.\n"))
		})

		It("Pass a break glass credential id through parameter and it is found, but it has been revoked", func() {
			args.id = breakGlassCredentialId
			const breakGlassCredentialId = "test-id-1"
			revokedCredential, err := cmv1.NewBreakGlassCredential().
				ID(breakGlassCredentialId).Username("username").Status(cmv1.BreakGlassCredentialStatusRevoked).
				Build()
			Expect(err).To(BeNil())
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatResource(revokedCredential)))
			_, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{})
			Expect(err.Error()).To(Equal("Break glass credential 'test-id' for cluster 'cluster1' has been revoked."))
			Expect(stderr).To(Equal(""))
		})
	})
})
