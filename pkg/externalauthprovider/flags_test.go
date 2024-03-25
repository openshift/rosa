package externalauthprovider

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("External authentication provider", func() {

	r := rosa.NewRuntime()
	clusterKey := "test-cluster"

	service := NewExternalAuthService(r.OCMClient)
	It("KO: cluster is not ready", func() {
		mockClusterNotReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateInstalling)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})

		err := service.IsExternalAuthProviderSupported(mockClusterNotReady, clusterKey)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("cluster 'test-cluster' is not yet ready"))
	})

	It("KO: a ready cluster is not enabled with external auth provider flag", func() {
		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})

		err := service.IsExternalAuthProviderSupported(mockClusterReady, clusterKey)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("External authentication configuration is not enabled for cluster 'test-cluster'\n" +
			"Create a hosted control plane with '--external-auth-providers-enabled' parameter to enabled the configuration"))
	})

	It("KO: non hcp cluster is not supported for external auth provider", func() {
		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
		})

		err := service.IsExternalAuthProviderSupported(mockClusterReady, clusterKey)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("external authentication provider is only supported for Hosted Control Planes"))
	})

	It("OK: a ready hcp cluster is enabled with external auth provider flag", func() {
		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
			c.ExternalAuthConfig(cmv1.NewExternalAuthConfig().Enabled(true))
		})

		err := service.IsExternalAuthProviderSupported(mockClusterReady, clusterKey)
		Expect(err).To(Not(HaveOccurred()))
	})
})
