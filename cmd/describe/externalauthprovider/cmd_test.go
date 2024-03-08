package externalauthprovider

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

const (
	externalAuthName     = "microsoft-entra-id"
	describeStringOutput = `
ID:                                    microsoft-entra-id
Cluster ID:                            24vf9iitg3p6tlml88iml6j6mu095mh8
Issuer audiences:                      
                                       - abc
Issuer Url:                            https://test.com
Claim mappings group:                  groups
Claim mappings username:               username
`
)

var _ = Describe("External authentication provider", func() {
	var testRuntime test.TestingRuntime

	Context("Describe external authentication provider command", func() {

		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
			c.ExternalAuthConfig(cmv1.NewExternalAuthConfig().Enabled(true))
		})
		hypershiftClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		externalAuths := make([]*cmv1.ExternalAuth, 0)
		externalAuths = append(externalAuths, test.BuildExternalAuth())

		BeforeEach(func() {
			testRuntime.InitRuntime()
			// Reset flag to avoid any side effect on other tests
			Cmd.Flags().Set("output", "")
		})
		It("Fails if we are not specifying an external auth provider name", func() {
			args.name = ""
			_, _, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime, Cmd, &[]string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(
				"you need to specify an external authentication provider name with '--name' parameter"))
		})

		It("Pass an external auth provider name through parameter but it is not found", func() {
			args.name = externalAuthName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("external authentication provider '%s' not found", externalAuthName)))
			Expect(stdout).To(BeEmpty())
			Expect(stderr).To(BeEmpty())
		})

		It("Pass an external auth provider name through parameter and it is found.", func() {
			args.name = externalAuthName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatExternalAuthList(externalAuths)))
			output := describeExternalAuthProviders(testRuntime.RosaRuntime,
				mockClusterReady, test.MockClusterID, test.BuildExternalAuth())
			Expect(output).To(Equal(describeStringOutput))
		})
	})
})
