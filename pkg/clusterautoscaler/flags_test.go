package clusterautoscaler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Cluster Autoscaler flags", func() {

	It("Validate PrefillAutoscalerArgs function", func() {
		args := &AutoscalerArgs{}
		cmd := &cobra.Command{}
		flags := cmd.Flags()
		flags.IntVar(
			&args.ResourceLimits.Cores.Min,
			"min-cores",
			0,
			"Minimum limit for the amount of cores to deploy in the cluster.",
		)

		flags.IntVar(
			&args.ResourceLimits.Cores.Max,
			"max-cores",
			180*64,
			"Maximum limit for the amount of cores to deploy in the cluster.",
		)
		flags.Set("max-cores", "20")
		flags.Set("min-cores", "10")
		autoscaler := test.MockAutoscaler(func(a *cmv1.ClusterAutoscalerBuilder) {
			sd := &cmv1.AutoscalerScaleDownConfigBuilder{}
			sd.UtilizationThreshold("0.5")
			a.ScaleDown(sd)

			rl := &cmv1.AutoscalerResourceLimitsBuilder{}
			rl.MaxNodesTotal(10)
			cores := &cmv1.ResourceRangeBuilder{}
			cores.Min(20).Max(30)
			rl.Cores(cores)
			a.ResourceLimits(rl)
		})
		prefilledArgs, err := PrefillAutoscalerArgs(cmd, args, autoscaler)
		Expect(err).NotTo(HaveOccurred())
		Expect(prefilledArgs.ResourceLimits.MaxNodesTotal).To(Equal(10))
		Expect(prefilledArgs.ResourceLimits.Cores.Max).To(Equal(20))
		Expect(prefilledArgs.ResourceLimits.Cores.Min).To(Equal(10))
	})
})
