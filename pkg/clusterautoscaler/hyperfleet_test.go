package clusterautoscaler

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildHyperfleetAutoscalingConfig", func() {
	It("maps only explicitly changed supported values", func() {
		args := &AutoscalerArgs{
			MaxNodeProvisionTime: "20m",
			ResourceLimits:       ResourceLimits{MaxNodesTotal: 25},
			MaxPodGracePeriod:    30,
			PodPriorityThreshold: -10,
		}
		config, err := BuildHyperfleetAutoscalingConfig(args, "autoscaler-", HyperfleetAutoscalingChanges{
			MaxNodeProvisionTime: true,
			MaxNodesTotal:        true,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(config).NotTo(BeNil())
		Expect(config.MaxNodeProvisionTime).NotTo(BeNil())
		Expect(*config.MaxNodeProvisionTime).To(Equal("20m"))
		Expect(config.MaxNodesTotal).NotTo(BeNil())
		Expect(*config.MaxNodesTotal).To(Equal(int32(25)))
		Expect(config.MaxPodGracePeriod).To(BeNil())
		Expect(config.PodPriorityThreshold).To(BeNil())
	})

	It("uses the flag prefix in int32 range errors", func() {
		args := &AutoscalerArgs{ResourceLimits: ResourceLimits{MaxNodesTotal: math.MaxInt32 + 1}}
		_, err := BuildHyperfleetAutoscalingConfig(args, "autoscaler-", HyperfleetAutoscalingChanges{
			MaxNodesTotal: true,
		})
		Expect(err).To(MatchError(ContainSubstring("--autoscaler-max-nodes-total must be between")))
	})

	It("returns nil when no supported flag changed", func() {
		config, err := BuildHyperfleetAutoscalingConfig(&AutoscalerArgs{}, "autoscaler-", HyperfleetAutoscalingChanges{})
		Expect(err).NotTo(HaveOccurred())
		Expect(config).To(BeNil())
	})
})
