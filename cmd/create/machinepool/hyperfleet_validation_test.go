package machinepool

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/test"
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

var _ = Describe("HyperFleet machine pool rolling update settings", func() {
	// The flags carry the same defaults the create command registers, so the
	// tests exercise the values a user actually sends.
	newUpgradeHandler := func(maxSurge, maxUnavailable string) *hyperfleetNodePoolCreate {
		cmd := &cobra.Command{}
		cmd.Flags().String("max-surge", "1", "")
		cmd.Flags().String("max-unavailable", "0", "")
		options := &mpOpts.CreateMachinepoolUserOptions{MaxSurge: "1", MaxUnavailable: "0"}
		if maxSurge != "" {
			Expect(cmd.Flags().Set("max-surge", maxSurge)).To(Succeed())
			options.MaxSurge = maxSurge
		}
		if maxUnavailable != "" {
			Expect(cmd.Flags().Set("max-unavailable", maxUnavailable)).To(Succeed())
			options.MaxUnavailable = maxUnavailable
		}
		return &hyperfleetNodePoolCreate{cmd: cmd, userOptions: options}
	}

	It("sends no rolling update settings when neither flag is set", func() {
		handler := newUpgradeHandler("", "")
		Expect(handler.validateAndParseUserOptions()).To(Succeed())
		Expect(handler.upgrade).To(BeNil())
	})

	It("sends both values when only one flag is set", func() {
		handler := newUpgradeHandler("2", "")
		Expect(handler.validateAndParseUserOptions()).To(Succeed())
		Expect(handler.upgrade).To(Equal(&hyperfleet.NodePoolUpgrade{MaxSurge: "2", MaxUnavailable: "0"}))
	})

	It("sends percentages as the user wrote them", func() {
		handler := newUpgradeHandler("20%", "10%")
		Expect(handler.validateAndParseUserOptions()).To(Succeed())
		Expect(handler.upgrade).To(Equal(&hyperfleet.NodePoolUpgrade{MaxSurge: "20%", MaxUnavailable: "10%"}))
	})

	It("rejects a rolling update that can never make progress", func() {
		handler := newUpgradeHandler("0", "0")
		Expect(handler.validateAndParseUserOptions()).To(MatchError(
			"only one of --max-surge and --max-unavailable can be zero"))
	})

	It("rejects an invalid value", func() {
		handler := newUpgradeHandler("", "101%")
		Expect(handler.validateAndParseUserOptions()).To(MatchError(ContainSubstring("invalid --max-unavailable")))
	})

	It("treats non-empty user options as set when there is no cobra command", func() {
		handler := &hyperfleetNodePoolCreate{userOptions: &mpOpts.CreateMachinepoolUserOptions{MaxSurge: "3"}}
		Expect(handler.validateAndParseUserOptions()).To(Succeed())
		Expect(handler.upgrade).To(Equal(&hyperfleet.NodePoolUpgrade{MaxSurge: "3"}))
	})

	It("reuses the generated client when no rolling update settings are requested", func() {
		runtime := test.NewTestRuntime().RosaRuntime
		handler := newUpgradeHandler("", "")
		Expect(handler.validateAndParseUserOptions()).To(Succeed())
		Expect(handler.runtimeWithUpgrade(runtime)).To(BeIdenticalTo(runtime))
	})

	It("wraps the client when rolling update settings are requested", func() {
		runtime := test.NewTestRuntime().RosaRuntime
		runtime.HyperFleetClient = hfmocks.NewMockInterface(gomock.NewController(GinkgoT()))
		handler := newUpgradeHandler("2", "1")
		Expect(handler.validateAndParseUserOptions()).To(Succeed())

		wrapped := handler.runtimeWithUpgrade(runtime)
		Expect(wrapped).NotTo(BeIdenticalTo(runtime))
		Expect(wrapped.HyperFleetClient).NotTo(BeIdenticalTo(runtime.HyperFleetClient))
	})
})
