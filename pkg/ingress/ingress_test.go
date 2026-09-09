package ingress

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Get min width for output", func() {
	It("retrieves the min width", func() {
		minWidth := getMinWidth([]string{"a", "ab", "abc", "def"})
		Expect(minWidth).To(Equal(3))
	})
	When("empty slice", func() {
		It("retrieves the min width as 0", func() {
			minWidth := getMinWidth([]string{})
			Expect(minWidth).To(Equal(0))
		})
	})
})

var _ = Describe("Retrieve map of entries for output", func() {
	It("retrieves map", func() {
		cluster, err := cmv1.NewCluster().ID("123").Build()
		Expect(err).To(BeNil())
		ingress, err := cmv1.NewIngress().
			ID("123").
			Default(true).
			Listening(cmv1.ListeningMethodExternal).
			LoadBalancerType(cmv1.LoadBalancerFlavorNlb).
			RouteWildcardPolicy(cmv1.WildcardPolicyWildcardsAllowed).
			RouteNamespaceOwnershipPolicy(cmv1.NamespaceOwnershipPolicyStrict).
			RouteSelectors(map[string]string{
				"test-route": "test-selector",
			}).
			ExcludedNamespaces("test", "test2").
			ComponentRoutes(map[string]*cmv1.ComponentRouteBuilder{
				string(cmv1.ComponentRouteTypeOauth): v1.NewComponentRoute().
					Hostname("oauth-hostname").TlsSecretRef("oauth-secret"),
			}).
			Build()
		Expect(err).To(BeNil())
		mapOutput := generateEntriesOutput(cluster, ingress)
		Expect(mapOutput).To(HaveLen(10))
	})
})

var _ = Describe("Describe ingress", func() {
	const (
		ingressOutput = "Cluster ID:                 123\n" +
			"Component Routes:           \n" +
			"    console: \n" +
			"        Hostname:       console-hostname\n" +
			"        TLS Secret Ref: console-secret\n" +
			"    downloads: \n" +
			"        Hostname:       downloads-hostname\n" +
			"        TLS Secret Ref: downloads-secret\n" +
			"    oauth: \n" +
			"        Hostname:       oauth-hostname\n" +
			"        TLS Secret Ref: oauth-secret\n" +
			"Default:                    true\n" +
			"Excluded Namespaces:        [excluded-ns-1, excluded-ns-2]\n" +
			"ID:                         a1b1\n" +
			"LB-Type:                    nlb\n" +
			"Namespace Ownership Policy: Strict\n" +
			"Private:                    false\n" +
			"Route Selectors:            map[route-1:selector-1 route-2:selector-2]\n" +
			"Wildcard Policy:            WildcardsAllowed\n"
	)
	It("Ingress found", func() {
		// Full diff for long string to help debugging
		format.TruncatedDiff = false
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
		t := test.NewTestRuntime()
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ingressResponse))
		err = t.StdOutReader.Record()
		Expect(err).ToNot(HaveOccurred())
		mockReadyCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.ID("123")
			c.Region(cmv1.NewCloudRegion().ID(aws.DefaultRegion))
			c.State(cmv1.ClusterStateReady)
		})
		err = NewIngressService().DescribeIngress(t.RosaRuntime, mockReadyCluster, "apps")
		Expect(err).ToNot(HaveOccurred())
		stdout, err := t.StdOutReader.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(stdout).To(Equal(ingressOutput))
	})

	It("Ingress found with component routes and no optional fields", func() {
		format.TruncatedDiff = false
		expectedOutput := "Cluster ID:       123\n" +
			"Component Routes: \n" +
			"    downloads: \n" +
			"        Hostname:       downloads.banana.company.com\n" +
			"        TLS Secret Ref: downloads-tls-secret\n" +
			"Default:          true\n" +
			"ID:               a1q0\n" +
			"LB-Type:          nlb\n" +
			"Private:          false\n"
		ingress, err := cmv1.NewIngress().
			ID("a1q0").
			Default(true).
			Listening(cmv1.ListeningMethodExternal).
			LoadBalancerType(cmv1.LoadBalancerFlavorNlb).
			ComponentRoutes(map[string]*cmv1.ComponentRouteBuilder{
				"downloads": cmv1.NewComponentRoute().
					Hostname("downloads.banana.company.com").
					TlsSecretRef("downloads-tls-secret"),
			}).Build()
		Expect(err).To(BeNil())
		ingressResponse := test.FormatIngressList([]*cmv1.Ingress{ingress})
		t := test.NewTestRuntime()
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ingressResponse))
		err = t.StdOutReader.Record()
		Expect(err).ToNot(HaveOccurred())
		mockReadyCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.ID("123")
			c.Region(cmv1.NewCloudRegion().ID(aws.DefaultRegion))
			c.State(cmv1.ClusterStateReady)
		})
		err = NewIngressService().DescribeIngress(t.RosaRuntime, mockReadyCluster, "apps")
		Expect(err).ToNot(HaveOccurred())
		stdout, err := t.StdOutReader.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(stdout).To(Equal(expectedOutput))
	})

})
