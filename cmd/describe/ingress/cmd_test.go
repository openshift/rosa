package ingress

import (
	"bytes"
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/aws"
	. "github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Describe ingress", func() {
	const (
		ingressOutput = `Cluster ID:                 123
Component Routes:           
    console: 
        Hostname:           console-hostname
        TLS Secret Ref:     console-secret
    downloads: 
        Hostname:           downloads-hostname
        TLS Secret Ref:     downloads-secret
    oauth: 
        Hostname:           oauth-hostname
        TLS Secret Ref:     oauth-secret
Default:                    true
Excluded Namespaces:        [excluded-ns-1, excluded-ns-2]
ID:                         a1b1
LB-Type:                    nlb
Namespace Ownership Policy: Strict
Private:                    false
Route Selectors:            map[route-1:selector-1 route-2:selector-2]
Wildcard Policy:            WildcardsAllowed
`
		privateIngressOutput = `Cluster ID:                 123
Component Routes:           
    console: 
        Hostname:           console-hostname
        TLS Secret Ref:     console-secret
    downloads: 
        Hostname:           downloads-hostname
        TLS Secret Ref:     downloads-secret
    oauth: 
        Hostname:           oauth-hostname
        TLS Secret Ref:     oauth-secret
Default:                    true
Excluded Namespaces:        [excluded-ns-1, excluded-ns-2]
ID:                         a1b1
LB-Type:                    nlb
Namespace Ownership Policy: Strict
Private:                    true
Route Selectors:            map[route-1:selector-1 route-2:selector-2]
Wildcard Policy:            WildcardsAllowed
`
	)
	Context("describe", func() {
		// Full diff for long string to help debugging
		format.TruncatedDiff = false

		mockReadyCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.ID("123")
			c.Region(cmv1.NewCloudRegion().ID(aws.DefaultRegion))
			c.State(cmv1.ClusterStateReady)
		})
		classicClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockReadyCluster})
		mockNotReadyCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.ID("123")
			c.Region(cmv1.NewCloudRegion().ID(aws.DefaultRegion))
			c.State(cmv1.ClusterStateInstalling)
		})
		classicClusterNotReady := test.FormatClusterList([]*cmv1.Cluster{mockNotReadyCluster})
		ingress, err := cmv1.NewIngress().
			ID("a1b1").
			Default(true).
			Listening(cmv1.ListeningMethodExternal).
			LoadBalancerType(cmv1.LoadBalancerFlavorNlb).
			RouteWildcardPolicy(cmv1.WildcardPolicyWildcardsAllowed).
			RouteNamespaceOwnershipPolicy(cmv1.NamespaceOwnershipPolicyStrict).
			RouteSelectors(map[string]string{
				"route-1": "selector-1",
				"route-2": "selector-2",
			}).
			ExcludedNamespaces("excluded-ns-1", "excluded-ns-2").
			ComponentRoutes(map[string]*cmv1.ComponentRouteBuilder{
				"oauth":     cmv1.NewComponentRoute().Hostname("oauth-hostname").TlsSecretRef("oauth-secret"),
				"downloads": cmv1.NewComponentRoute().Hostname("downloads-hostname").TlsSecretRef("downloads-secret"),
				"console":   cmv1.NewComponentRoute().Hostname("console-hostname").TlsSecretRef("console-secret"),
			}).Build()
		Expect(err).To(BeNil())
		ingressResponse := test.FormatIngressList([]*cmv1.Ingress{ingress})
		privateIngress, err := cmv1.NewIngress().
			ID("a1b1").
			Default(true).
			Listening(cmv1.ListeningMethodInternal).
			LoadBalancerType(cmv1.LoadBalancerFlavorNlb).
			RouteWildcardPolicy(cmv1.WildcardPolicyWildcardsAllowed).
			RouteNamespaceOwnershipPolicy(cmv1.NamespaceOwnershipPolicyStrict).
			RouteSelectors(map[string]string{
				"route-1": "selector-1",
				"route-2": "selector-2",
			}).
			ExcludedNamespaces("excluded-ns-1", "excluded-ns-2").
			ComponentRoutes(map[string]*cmv1.ComponentRouteBuilder{
				"oauth":     cmv1.NewComponentRoute().Hostname("oauth-hostname").TlsSecretRef("oauth-secret"),
				"downloads": cmv1.NewComponentRoute().Hostname("downloads-hostname").TlsSecretRef("downloads-secret"),
				"console":   cmv1.NewComponentRoute().Hostname("console-hostname").TlsSecretRef("console-secret"),
			}).Build()
		Expect(err).To(BeNil())
		privateIngressResponse := test.FormatIngressList([]*cmv1.Ingress{privateIngress})
		var t *test.TestingRuntime
		BeforeEach(func() {
			t = test.NewTestRuntime()
			SetOutput("")
		})

		It("Fails if ingress ID/alias has not been specified", func() {
			runner := DescribeIngressRunner(NewDescribeIngressUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			err = runner(context.Background(), t.RosaRuntime, NewDescribeIngressCommand(), []string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("you need to specify an ingress ID/alias"))
		})

		It("Fails if ingress ID/alias is invalid", func() {
			runner := DescribeIngressRunner(NewDescribeIngressUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeIngressCommand()
			cmd.Flag("ingress").Value.Set("A1b2")
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal(
				"Ingress identifier 'A1b2' isn't valid: it must contain between three and five lowercase letters or digits",
			))
		})

		It("Cluster not ready", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterNotReady))
			runner := DescribeIngressRunner(NewDescribeIngressUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeIngressCommand()
			cmd.Flag("ingress").Value.Set("apps")
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("cluster '123' is not yet ready"))
		})

		It("Ingress not found", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			runner := DescribeIngressRunner(NewDescribeIngressUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeIngressCommand()
			cmd.Flag("ingress").Value.Set("apps")
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Failed to get ingress 'apps' for cluster '123'"))
		})

		It("Ingress found", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ingressResponse))
			runner := DescribeIngressRunner(NewDescribeIngressUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeIngressCommand()
			cmd.Flag("ingress").Value.Set("apps")
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).ToNot(HaveOccurred())
			stdout, err := t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal(ingressOutput))
		})

		It("Ingress found through argv", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ingressResponse))
			args := NewDescribeIngressUserOptions()
			runner := DescribeIngressRunner(args)
			cmd := NewDescribeIngressCommand()
			cmd.Flag("cluster").Value.Set("123")
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			err = runner(context.Background(), t.RosaRuntime, cmd,
				[]string{
					"apps",
				})
			Expect(err).ToNot(HaveOccurred())
			stdout, err := t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal(ingressOutput))
		})

		It("Ingress found json output", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ingressResponse))
			runner := DescribeIngressRunner(NewDescribeIngressUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeIngressCommand()
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			cmd.Flag("output").Value.Set("json")
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{"apps"})
			Expect(err).ToNot(HaveOccurred())
			stdout, err := t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			var ingressJson bytes.Buffer
			cmv1.MarshalIngress(ingress, &ingressJson)
			Expect(stdout).To(Equal(ingressJson.String() + "\n"))
		})

		It("Private Ingress found", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, privateIngressResponse))
			runner := DescribeIngressRunner(NewDescribeIngressUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeIngressCommand()
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{"apps"})
			Expect(err).ToNot(HaveOccurred())
			stdout, err := t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal(privateIngressOutput))
		})
	})
})
