package clusterautoscaler

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildSeaplaneAutoscalingConfig", func() {
	It("maps only explicitly changed supported values", func() {
		args := &AutoscalerArgs{
			MaxNodeProvisionTime: "20m",
			ResourceLimits:       ResourceLimits{MaxNodesTotal: 25},
			MaxPodGracePeriod:    30,
			PodPriorityThreshold: -10,
		}
		config, err := BuildSeaplaneAutoscalingConfig(args, "autoscaler-", SeaplaneAutoscalingChanges{
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
		_, err := BuildSeaplaneAutoscalingConfig(args, "autoscaler-", SeaplaneAutoscalingChanges{
			MaxNodesTotal: true,
		})
		Expect(err).To(MatchError(ContainSubstring("--autoscaler-max-nodes-total must be between")))
	})

	It("returns nil when no supported flag changed", func() {
		config, err := BuildSeaplaneAutoscalingConfig(&AutoscalerArgs{}, "autoscaler-", SeaplaneAutoscalingChanges{})
		Expect(err).NotTo(HaveOccurred())
		Expect(config).To(BeNil())
	})
})
