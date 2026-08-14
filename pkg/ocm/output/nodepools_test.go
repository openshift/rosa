package output

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Validate node drain grace period print output", func() {
	zeroValue, _ := cmv1.NewValue().Value(0).Build()
	oneValue, _ := cmv1.NewValue().Value(1).Build()
	twoValue, _ := cmv1.NewValue().Value(2).Build()

	DescribeTable("Validate node drain grace period print output",
		func(period *cmv1.Value, expectedOutput string) {
			output := PrintNodeDrainGracePeriod(period)
			Expect(output).To(Equal(expectedOutput))
		},
		Entry("Should return empty string", nil,
			"",
		),
		Entry("Should return empty string", zeroValue,
			"",
		),
		Entry("Should return 1 minute", oneValue,
			"1 minute",
		),
		Entry("Should return 2 minutes", twoValue,
			"2 minutes",
		),
	)
})

var _ = Describe("PrintNodePoolSpot", func() {
	It("returns No when spot is not configured", func() {
		awsNodePool, err := cmv1.NewAWSNodePool().Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolSpot(awsNodePool)).To(Equal("No"))
	})

	It("returns on-demand when spot is configured without max price", func() {
		awsNodePool, err := cmv1.NewAWSNodePool().
			SpotMarketOptions(cmv1.NewAwsNodePoolSpotMarketOptions()).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolSpot(awsNodePool)).To(Equal("Yes (on-demand)"))
	})

	It("returns the max price when spot max price is configured", func() {
		awsNodePool, err := cmv1.NewAWSNodePool().
			SpotMarketOptions(cmv1.NewAwsNodePoolSpotMarketOptions().MaxPrice("1.00")).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolSpot(awsNodePool)).To(Equal("Yes (max $1.00)"))
	})
})

var _ = Describe("PrintNodePoolReplicasInline", func() {
	It("Should print the correct output if autoscaling exists", func() {
		autoscaling := cmv1.NewNodePoolAutoscaling().MinReplica(2).MaxReplica(6)
		output := PrintNodePoolReplicasInline((*cmv1.NodePoolAutoscaling)(autoscaling), 2)
		Expect(output).To(Equal("2-6"))
	})

	It("Should print the correct output if autoscaling is nill", func() {
		output := PrintNodePoolReplicasInline(nil, 2)
		Expect(output).To(Equal("2"))
	})

})
