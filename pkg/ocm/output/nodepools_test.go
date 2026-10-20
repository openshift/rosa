// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

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

var _ = Describe("PrintNodePoolMessage", func() {
	It("returns empty string when status is nil", func() {
		Expect(PrintNodePoolMessage(nil)).To(Equal(""))
	})

	It("returns empty string when the message is empty", func() {
		status, err := cmv1.NewNodePoolStatus().Message("").Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal(""))
	})

	It("nests a machine's conditions beneath it as a two-level YAML-style list", func() {
		status, err := cmv1.NewNodePoolStatus().
			Message("* Machine foo:\n    * NodeHealthy: waiting for a Node to exist").
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal(
			"\n  - Machine foo:\n    - NodeHealthy: waiting for a Node to exist"))
	})

	It("nests every condition under its own machine when there are multiple machines", func() {
		status, err := cmv1.NewNodePoolStatus().
			Message("* Machine foo:\n" +
				"    * NodeHealthy: waiting for a Node to exist\n" +
				"* Machine bar:\n" +
				"    * NodeHealthy: waiting for a Node to exist\n" +
				"    * Ready: condition status is Unknown").
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal("\n" +
			"  - Machine foo:\n" +
			"    - NodeHealthy: waiting for a Node to exist\n" +
			"  - Machine bar:\n" +
			"    - NodeHealthy: waiting for a Node to exist\n" +
			"    - Ready: condition status is Unknown"))
	})

	It("adds a third nesting level when a bullet is indented deeper than the second level", func() {
		status, err := cmv1.NewNodePoolStatus().
			Message("* Machine foo:\n" +
				"    * NodeHealthy: waiting for a Node to exist\n" +
				"        * Detail: deeply nested condition").
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal(
			"\n  - Machine foo:\n    - NodeHealthy: waiting for a Node to exist\n      - Detail: deeply nested condition"))
	})

	It("treats a shallower bullet after a deeper one as returning to the matching ancestor level", func() {
		status, err := cmv1.NewNodePoolStatus().
			Message("* Machine foo:\n" +
				"    * NodeHealthy: waiting for a Node to exist\n" +
				"        * Detail: deeply nested condition\n" +
				"    * Ready: condition status is Unknown").
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal("\n" +
			"  - Machine foo:\n" +
			"    - NodeHealthy: waiting for a Node to exist\n" +
			"      - Detail: deeply nested condition\n" +
			"    - Ready: condition status is Unknown"))
	})

	It("preserves arbitrarily deep nesting with no fixed ceiling", func() {
		status, err := cmv1.NewNodePoolStatus().
			Message("* Level1:\n" +
				"    * Level2:\n" +
				"        * Level3:\n" +
				"            * Level4:\n" +
				"                * Level5: deepest condition\n" +
				"            * Level4-sibling: back up a level\n" +
				"    * Level2-sibling: back up several levels").
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal("\n" +
			"  - Level1:\n" +
			"    - Level2:\n" +
			"      - Level3:\n" +
			"        - Level4:\n" +
			"          - Level5: deepest condition\n" +
			"        - Level4-sibling: back up a level\n" +
			"    - Level2-sibling: back up several levels"))
	})

	It("lists a single-line message without a bullet", func() {
		status, err := cmv1.NewNodePoolStatus().Message("cluster upgrade in progress").Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal("\n  - cluster upgrade in progress"))
	})

	It("appends an indented continuation line to the preceding condition instead of a new item", func() {
		status, err := cmv1.NewNodePoolStatus().
			Message("* Machine foo:\n" +
				"    * NodeHealthy: Waiting for a Node with spec.providerID\n" +
				"      aws:///us-west-2a/i-00dfda71eb9b2bc65 to exist").
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal(
			"\n  - Machine foo:\n    - NodeHealthy: Waiting for a Node with spec.providerID " +
				"aws:///us-west-2a/i-00dfda71eb9b2bc65 to exist"))
	})

	It("appends an unindented continuation line to the preceding condition instead of a new item", func() {
		status, err := cmv1.NewNodePoolStatus().
			Message("* Machine foo:\n" +
				"    * NodeHealthy: Waiting for a Node with spec.providerID\n" +
				"aws:///us-west-2a/i-00dfda71eb9b2bc65 to exist").
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal(
			"\n  - Machine foo:\n    - NodeHealthy: Waiting for a Node with spec.providerID " +
				"aws:///us-west-2a/i-00dfda71eb9b2bc65 to exist"))
	})

	It("appends a continuation line to a top-level item when there is no preceding bullet", func() {
		status, err := cmv1.NewNodePoolStatus().
			Message("cluster upgrade in progress\ncheck back later").
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal(
			"\n  - cluster upgrade in progress check back later"))
	})

	It("skips whitespace-only lines instead of emitting an empty list item", func() {
		status, err := cmv1.NewNodePoolStatus().
			Message("* Machine foo:\n    \n    * NodeHealthy: waiting for a Node to exist").
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(PrintNodePoolMessage(status)).To(Equal(
			"\n  - Machine foo:\n    - NodeHealthy: waiting for a Node to exist"))
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
