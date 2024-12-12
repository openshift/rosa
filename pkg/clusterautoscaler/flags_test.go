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

var _ = Describe("getAutoscalerMaxNodesTotalDefaultValue function", func() {
	// Classic cluster maxNodesTotal default value calculation
	//180 + (default flavour) 3 master + (multi az) 3 infra
	It("returns 186 worker nodes for classic cluster under v4.14.14", func() {
		previousValue := 0
		autoscalerValidationArgs := &AutoscalerValidationArgs{
			ClusterVersion: "v4.14.13",
			MultiAz:        true,
			IsHostedCp:     false,
		}
		maxNodesTotalDefaultValue := getAutoscalerMaxNodesTotalDefaultValue(previousValue, autoscalerValidationArgs)
		Expect(maxNodesTotalDefaultValue).To(Equal(186))
	})
	//180 + (default flavour) 3 master + (single az) 2 infra
	It("returns 185 worker nodes for classic cluster under v4.14.14", func() {
		previousValue := 0
		autoscalerValidationArgs := &AutoscalerValidationArgs{
			ClusterVersion: "v4.14.13",
			MultiAz:        false,
			IsHostedCp:     false,
		}
		maxNodesTotalDefaultValue := getAutoscalerMaxNodesTotalDefaultValue(previousValue, autoscalerValidationArgs)
		Expect(maxNodesTotalDefaultValue).To(Equal(185))
	})
	//249 + (default flavour) 3 master + (single az) 2 infra
	It("returns 255 worker nodes for classic cluster at or above v4.14.14", func() {
		previousValue := 0
		autoscalerValidationArgs := &AutoscalerValidationArgs{
			ClusterVersion: "v4.14.14",
			MultiAz:        true,
			IsHostedCp:     false,
		}
		maxNodesTotalDefaultValue := getAutoscalerMaxNodesTotalDefaultValue(previousValue, autoscalerValidationArgs)
		Expect(maxNodesTotalDefaultValue).To(Equal(255))
	})
	//249 + (default flavour) 3 master + (single az) 2 infra
	It("returns 254 worker nodes for classic cluster at or above v4.14.14", func() {
		previousValue := 0
		autoscalerValidationArgs := &AutoscalerValidationArgs{
			ClusterVersion: "v4.14.14",
			MultiAz:        false,
			IsHostedCp:     false,
		}
		maxNodesTotalDefaultValue := getAutoscalerMaxNodesTotalDefaultValue(previousValue, autoscalerValidationArgs)
		Expect(maxNodesTotalDefaultValue).To(Equal(254))
	})
	//returns value which was set previously
	It("returns prefilled value worker node count for classic cluster", func() {
		previousValue := 100
		autoscalerValidationArgs := &AutoscalerValidationArgs{
			ClusterVersion: "v4.14.14",
			MultiAz:        true,
		}
		maxNodesTotalDefaultValue := getAutoscalerMaxNodesTotalDefaultValue(previousValue, autoscalerValidationArgs)
		Expect(maxNodesTotalDefaultValue).To(Equal(100))
	})

	// Hosted CP cluster maxNodesTotal default value calculation (MultiAz bool does not affect outcome)
	It("returns 500 worker nodes for hosted cp cluster", func() {
		previousValue := 0
		autoscalerValidationArgs := &AutoscalerValidationArgs{
			ClusterVersion: "v4.14.14",
			MultiAz:        true,
			IsHostedCp:     true,
		}
		maxNodesTotalDefaultValue := getAutoscalerMaxNodesTotalDefaultValue(previousValue, autoscalerValidationArgs)
		Expect(maxNodesTotalDefaultValue).To(Equal(500))
	})
	//returns value which was set previously
	It("returns prefilled value worker node count for classic cluster", func() {
		previousValue := 100
		autoscalerValidationArgs := &AutoscalerValidationArgs{
			ClusterVersion: "v4.14.14",
			MultiAz:        true,
			IsHostedCp:     true,
		}
		maxNodesTotalDefaultValue := getAutoscalerMaxNodesTotalDefaultValue(previousValue, autoscalerValidationArgs)
		Expect(maxNodesTotalDefaultValue).To(Equal(100))
	})
})
