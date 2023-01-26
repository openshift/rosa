package ocm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("IDPs", func() {
	Context("OAuthURL", func() {
		Context("BuildOAuthURL", func() {
			It("Checks Hive cluster", func() {
				consoleURL := cmv1.NewClusterConsole().
					URL("https://console-openshift-console.apps.cluster.example.com")
				cluster, err := cmv1.NewCluster().Name("cluster1").ID("id1").Console(consoleURL).Build()
				Expect(err).To(BeNil())
				url, err := BuildOAuthURL(cluster, cmv1.IdentityProviderTypeGithub)
				Expect(url).To(Equal("https://oauth-openshift.apps.cluster.example.com"))
				Expect(err).To(BeNil())
			})
			It("Checks HyperShift cluster - Empty API URL", func() {
				hypershift := cmv1.NewHypershift().Enabled(true)
				apiURL := cmv1.NewClusterAPI().URL("")
				cluster, err := cmv1.NewCluster().Name("cluster1").
					ID("id1").Hypershift(hypershift).API(apiURL).Build()
				Expect(err).To(BeNil())
				url, err := BuildOAuthURL(cluster, cmv1.IdentityProviderTypeGithub)
				Expect(url).To(Equal(""))
				Expect(err).To(Not(BeNil()))
			})
			It("Checks HyperShift cluster - Valid API URL", func() {
				hypershift := cmv1.NewHypershift().Enabled(true)
				apiURL := cmv1.NewClusterAPI().URL("https://api.example.com:443")
				cluster, err := cmv1.NewCluster().Name("cluster1").
					ID("id1").Hypershift(hypershift).API(apiURL).Build()
				Expect(err).To(BeNil())
				url, err := BuildOAuthURL(cluster, cmv1.IdentityProviderTypeGithub)
				Expect(url).To(Equal("https://oauth.example.com"))
				Expect(err).To(BeNil())
			})
			It("Checks HyperShift cluster - Valid API URL and keep the port", func() {
				hypershift := cmv1.NewHypershift().Enabled(true)
				apiURL := cmv1.NewClusterAPI().URL("https://api.example.com:443")
				cluster, err := cmv1.NewCluster().Name("cluster1").
					ID("id1").Hypershift(hypershift).API(apiURL).Build()
				Expect(err).To(BeNil())
				url, err := BuildOAuthURL(cluster, cmv1.IdentityProviderTypeOpenID)
				Expect(url).To(Equal("https://oauth.example.com:443"))
				Expect(err).To(BeNil())
			})
		})
	})
})
