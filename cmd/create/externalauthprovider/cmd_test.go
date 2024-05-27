package externalauthprovider

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("ExternalAuthProvider Create Tests", func() {
	var testRuntime test.TestingRuntime

	Context("Create break glass credential command", func() {
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

		It("KO: Should ask for interactive mode if params are not passed", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime, Cmd, &[]string{})
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal("INFO: Enabling interactive mode\n\x1b[0G\x1b[2K? " +
				"Name: [? for help] \x1b[?25l\x1b[?25l\x1b7\x1b[999;999f\x1b[6n\x1b8\x1b[?25h\x1b[6n\x1b[?25h"))
		})

		It("KO: Should ask for interactive mode if all mandatory fields are not given", func() {
			Cmd.Flags().Set("name", "test-name")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime, Cmd, &[]string{})
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal("\x1b[0G\x1b[2K? Issuer audiences: [? for help]" +
				" \x1b[?25l\x1b[?25l\x1b7\x1b[999;999f\x1b[6n\x1b8\x1b[?25h\x1b[6n\x1b[?25h"))
		})
	})
})
