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

var _ = Describe("Hyperfleet node pool setup", func() {
	It("uses configured replicas on every pool", func() {
		n, err := hyperfleetNodePoolReplicas(&ClusterConfig{WorkerPoolReplicas: 3}, 3)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(3))
	})

	It("uses the ROSA single-AZ default", func() {
		n, err := hyperfleetNodePoolReplicas(&ClusterConfig{}, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(2))
	})

	It("rejects unsupported autoscaling profiles", func() {
		_, err := hyperfleetNodePoolReplicas(&ClusterConfig{Autoscale: true}, 1)
		Expect(err).To(MatchError("HyperFleet e2e setup does not support autoscaled default node pools"))
	})
})

var _ = Describe("GenerateClusterCreateFlags hyperfleet", func() {
	AfterEach(func() {
		hyperfleet.Reset()
	})

	DescribeTable("rejects custom KMS before preparing any resources",
		func(kmsKey, etcdKMS bool) {
			hyperfleet.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
			GinkgoT().Setenv("CLUSTER_NAME", "hf-kms-test")
			ch := &clusterHandler{
				profile: &Profile{ClusterConfig: &ClusterConfig{
					KMSKey: kmsKey, EtcdKMS: etcdKMS, STS: true, BYOVPC: true, OIDCConfig: "managed",
				}},
				clusterConfig: &ClusterConfigure.ClusterConfig{},
			}
			flags, err := ch.GenerateClusterCreateFlags()
			Expect(err).To(MatchError(hyperfleetCustomKMSError))
			Expect(flags).To(BeEmpty())
		},
		Entry("EBS key", true, false),
		Entry("etcd key", false, true),
		Entry("both keys", true, true),
	)

	DescribeTable("returns an error instead of parsing OCM KMS data on HyperFleet",
		func(etcdKMS bool) {
			hyperfleet.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
			ch := &clusterHandler{}
			Expect(ch.elaborateKMSKeyForSTSCluster(etcdKMS)).To(MatchError(hyperfleetCustomKMSError))
		},
		Entry("EBS key", false),
		Entry("etcd key", true),
	)

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

	It("passes the release image configured in the profile", func() {
		const releaseImage = "quay.io/openshift-release-dev/ocp-release:5.0.0-ec.6-multi"
		hyperfleet.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		GinkgoT().Setenv("CLUSTER_NAME", "hf-e2e-12345")
		GinkgoT().Setenv("HYPERFLEET_VERSION", "")
		ch := &clusterHandler{
			profile:          &Profile{Version: releaseImage, Region: "us-east-1", ClusterConfig: &ClusterConfig{}},
			clusterConfig:    &ClusterConfigure.ClusterConfig{},
			clusterDetail:    &ClusterDetail{},
			resourcesHandler: &resourcesHandler{persist: false, resources: &Resources{}},
		}
		flags, err := ch.GenerateClusterCreateFlags()
		Expect(err).ToNot(HaveOccurred())
		Expect(flags).To(ContainElements("--version", releaseImage))
	})
})
