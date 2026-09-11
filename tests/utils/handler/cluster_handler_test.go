package handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/hyperfleet"
	ClusterConfigure "github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
)

var _ = Describe("classifyClusterState", func() {
	It("recognizes ready state", func() {
		Expect(classifyClusterState(constants.Ready)).To(Equal(clusterStateReady))
	})

	It("recognizes uninstalling as terminal", func() {
		Expect(classifyClusterState(constants.Uninstalling)).To(Equal(clusterStateTerminal))
	})

	It("recognizes error as terminal", func() {
		Expect(classifyClusterState(constants.Error)).To(Equal(clusterStateTerminal))
	})

	It("recognizes waiting state", func() {
		Expect(classifyClusterState(constants.Waiting)).To(Equal(clusterStateWaiting))
	})

	It("recognizes installing as transient", func() {
		Expect(classifyClusterState(constants.Installing)).To(Equal(clusterStateTransient))
	})

	It("recognizes pending as transient", func() {
		Expect(classifyClusterState(constants.Pending)).To(Equal(clusterStateTransient))
	})

	It("recognizes validating as transient", func() {
		Expect(classifyClusterState(constants.Validating)).To(Equal(clusterStateTransient))
	})

	It("returns unknown for unrecognized states", func() {
		Expect(classifyClusterState("hibernating")).To(Equal(clusterStateUnknown))
	})

	It("returns unknown for empty state", func() {
		Expect(classifyClusterState("")).To(Equal(clusterStateUnknown))
	})
})

var _ = Describe("GenerateClusterCreateFlags hyperfleet", func() {
	AfterEach(func() {
		hyperfleet.Reset()
	})

	It("skips NAME_PREFIX when CLUSTER_NAME is set", func() {
		hyperfleet.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		GinkgoT().Setenv("CLUSTER_NAME", "hf-e2e-12345")
		ch := &clusterHandler{
			profile:          &Profile{Region: "us-east-1", ClusterConfig: &ClusterConfig{}},
			clusterConfig:    &ClusterConfigure.ClusterConfig{},
			clusterDetail:    &ClusterDetail{},
			resourcesHandler: &resourcesHandler{persist: false, resources: &Resources{}},
		}
		flags, err := ch.GenerateClusterCreateFlags()
		Expect(err).ToNot(HaveOccurred())
		Expect(ch.profile.ClusterConfig.Name).To(Equal("hf-e2e-12345"))
		Expect(flags).To(ContainElement("hf-e2e-12345"))
	})
})
