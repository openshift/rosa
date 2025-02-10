package clusterautoscaler

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Cluster Autoscaler validations", func() {

	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("Determines ready, HCP cluster can support Autoscaler", func() {

		cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			h := &cmv1.HypershiftBuilder{}
			h.Enabled(true)
			c.Hypershift(h)
			c.State(cmv1.ClusterStateReady)
		})

		err := IsAutoscalerSupported(t.RosaRuntime, cluster)
		Expect(err).To(BeNil())

	})

	It("Determines Autoscalers are not valid for clusters in non-ready state", func() {
		cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.State(cmv1.ClusterStateInstalling)
		})

		err := IsAutoscalerSupported(t.RosaRuntime, cluster)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(fmt.Sprintf(ClusterNotReadyMessage, t.RosaRuntime.ClusterKey, cluster.State())))
	})

	It("Determines ready, non-HCP cluster can support Autoscaler", func() {
		cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			h := &cmv1.HypershiftBuilder{}
			h.Enabled(false)
			c.Hypershift(h)
			c.State(cmv1.ClusterStateReady)
		})

		err := IsAutoscalerSupported(t.RosaRuntime, cluster)
		Expect(err).To(BeNil())
	})

})
