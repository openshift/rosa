package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
)

var _ = Describe("HyperFleet machine pool creation validation", func() {
	It("rejects an explicitly empty version", func() {
		cmd := &cobra.Command{}
		cmd.Flags().String("version", "", "")
		Expect(cmd.Flags().Set("version", "")).To(Succeed())
		handler := &hyperfleetNodePoolCreate{cmd: cmd, userOptions: &mpOpts.CreateMachinepoolUserOptions{}}
		Expect(handler.validateAndParseUserOptions()).To(MatchError(
			"expected a valid OpenShift version: --version requires a non-empty value"))
	})
	DescribeTable("rejects an explicit multi-availability-zone flag while allowing its default",
		func(value string) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("multi-availability-zone", true, "")
			if value != "" {
				Expect(cmd.Flags().Set("multi-availability-zone", value)).To(Succeed())
			}
			handler := &hyperfleetNodePoolCreate{cmd: cmd, userOptions: &mpOpts.CreateMachinepoolUserOptions{}}
			err := handler.validateAndParseUserOptions()
			if value == "" {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(MatchError("setting `multi-availability-zone` flag is not supported for HCP clusters"))
			}
		},
		Entry("implicit default", ""),
		Entry("explicit true", "true"),
		Entry("explicit false", "false"),
	)
	DescribeTable("validates names from both flags and arguments",
		func(name string, valid bool) {
			for _, handler := range []*hyperfleetNodePoolCreate{
				{userOptions: &mpOpts.CreateMachinepoolUserOptions{Name: name}},
				{userOptions: &mpOpts.CreateMachinepoolUserOptions{}, argv: []string{name}},
			} {
				err := handler.validateAndParseUserOptions()
				if valid {
					Expect(err).NotTo(HaveOccurred())
				} else {
					Expect(err).To(MatchError("expected a valid name for the machine pool"))
				}
			}
		},
		Entry("letters", "workers", true),
		Entry("hyphens and digits", "workers-2", true),
		Entry("single letter", "a", true),
		Entry("punctuation from 56786", "anything%^#@", false),
		Entry("uppercase", "Workers", false),
		Entry("leading digit", "2workers", false),
		Entry("leading hyphen", "-workers", false),
		Entry("trailing hyphen", "workers-", false),
		Entry("underscore", "worker_pool", false),
	)

	DescribeTable("validates fixed replicas before converting to int32",
		func(replicas int, expectedError string) {
			handler := &hyperfleetNodePoolCreate{userOptions: &mpOpts.CreateMachinepoolUserOptions{Replicas: replicas}}
			err := handler.validateAndParseUserOptions()
			if expectedError == "" {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(MatchError(ContainSubstring(expectedError)))
			}
		},
		Entry("negative replicas", -9, "replicas must be a non-negative integer"),
		Entry("zero replicas", 0, ""),
		Entry("maximum replicas", 500, ""),
		Entry("over the maximum", 501, "less than or equal to '500'"),
		Entry("overflow", int(1<<32), "less than or equal to '500'"),
	)

	DescribeTable("validates autoscaling bounds",
		func(minimum, maximum int, expectedError string) {
			handler := &hyperfleetNodePoolCreate{userOptions: &mpOpts.CreateMachinepoolUserOptions{
				AutoscalingEnabled: true, MinReplicas: minimum, MaxReplicas: maximum,
			}}
			err := handler.validateAndParseUserOptions()
			if expectedError == "" {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(MatchError(ContainSubstring(expectedError)))
			}
		},
		Entry("valid bounds", 0, 500, ""),
		Entry("negative minimum", -1, 3, "min-replicas must be a non-negative"),
		Entry("zero maximum", 0, 0, "max-replicas must be greater than zero"),
		Entry("reversed bounds", 6, 3, "max-replicas must be greater or equal to min-replicas"),
		Entry("over the maximum", 3, 501, "less than or equal to '500'"),
	)

	It("rejects explicitly set replicas with autoscaling", func() {
		cmd := &cobra.Command{}
		cmd.Flags().Int("replicas", 2, "")
		Expect(cmd.Flags().Set("replicas", "2")).To(Succeed())
		handler := &hyperfleetNodePoolCreate{cmd: cmd, userOptions: &mpOpts.CreateMachinepoolUserOptions{
			Replicas: 2, AutoscalingEnabled: true, MinReplicas: 3, MaxReplicas: 3,
		}}
		Expect(handler.validateAndParseUserOptions()).To(MatchError("replicas can't be set when autoscaling is enabled"))
	})

	It("rejects autoscaling bounds without enabling autoscaling", func() {
		cmd := &cobra.Command{}
		cmd.Flags().Int("min-replicas", 0, "")
		Expect(cmd.Flags().Set("min-replicas", "3")).To(Succeed())
		handler := &hyperfleetNodePoolCreate{cmd: cmd, userOptions: &mpOpts.CreateMachinepoolUserOptions{MinReplicas: 3}}
		Expect(handler.validateAndParseUserOptions()).To(MatchError("autoscaling must be enabled in order to set min and max replicas"))
	})
})
