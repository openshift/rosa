package clusterautoscaler

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/test"
)

var optionalFieldOutput = `
Balance Similar Node Groups:               No
Skip Nodes With Local Storage:             Yes
Log Verbosity:                             2
Labels Ignored For Node Balancing:         foo, bar
Ignore DaemonSets Utilization:             Yes
Maximum Node Provision Time:               10m
Maximum Pod Grace Period:                  10
Pod Priority Threshold:                    10
Resource Limits:
 - Maximum Nodes:                          10
 - Minimum Number of Cores:                20
 - Maximum Number of Cores:                30
 - Minimum Memory (GiB):                   5
 - Maximum Memory (GiB):                   10
 - GPU Limitations:
  - Type: nvidia.com/gpu
   - Min:  10
   - Max:  20
Scale Down:
 - Enabled:                                Yes
 - Node Unneeded Time:                     25m
 - Node Utilization Threshold:             20
 - Delay After Node Added:                 5m
 - Delay After Node Deleted:               20m
 - Delay After Node Deletion Failure:      10m
`

var mandatoryFieldOutput = `
Balance Similar Node Groups:               No
Skip Nodes With Local Storage:             Yes
Log Verbosity:                             2
Ignore DaemonSets Utilization:             Yes
Maximum Pod Grace Period:                  10
Pod Priority Threshold:                    10
Resource Limits:
 - Maximum Nodes:                          10
 - Minimum Number of Cores:                20
 - Maximum Number of Cores:                30
 - Minimum Memory (GiB):                   5
 - Maximum Memory (GiB):                   10
Scale Down:
 - Enabled:                                Yes
 - Node Utilization Threshold:             20
`

var _ = Describe("Print Autoscaler", func() {

	It("Correctly prints with optional fields set", func() {
		autoscaler := test.MockAutoscaler(func(a *cmv1.ClusterAutoscalerBuilder) {
			a.MaxNodeProvisionTime("10m")
			a.BalancingIgnoredLabels("foo", "bar")
			a.PodPriorityThreshold(10)
			a.LogVerbosity(2)
			a.MaxPodGracePeriod(10)
			a.IgnoreDaemonsetsUtilization(true)
			a.SkipNodesWithLocalStorage(true)
			a.BalanceSimilarNodeGroups(false)

			sd := &cmv1.AutoscalerScaleDownConfigBuilder{}
			sd.Enabled(true)
			sd.DelayAfterFailure("10m")
			sd.DelayAfterAdd("5m")
			sd.DelayAfterDelete("20m")
			sd.UnneededTime("25m")
			sd.UtilizationThreshold("20")
			a.ScaleDown(sd)

			rl := &cmv1.AutoscalerResourceLimitsBuilder{}
			rl.MaxNodesTotal(10)

			mem := &cmv1.ResourceRangeBuilder{}
			mem.Max(10).Min(5)
			rl.Memory(mem)

			cores := &cmv1.ResourceRangeBuilder{}
			cores.Min(20).Max(30)
			rl.Cores(cores)

			gpus := &cmv1.AutoscalerResourceLimitsGPULimitBuilder{}
			gpus.Type("nvidia.com/gpu")

			gpuRR := &cmv1.ResourceRangeBuilder{}
			gpuRR.Max(20).Min(10)
			gpus.Range(gpuRR)

			rl.GPUS(gpus)
			a.ResourceLimits(rl)
		})

		out := PrintAutoscaler(autoscaler)
		Expect(out).NotTo(BeNil())
		Expect(optionalFieldOutput).To(Equal(out))
	})

	It("Correctly sprints with mandatory fields", func() {
		autoscaler := test.MockAutoscaler(func(a *cmv1.ClusterAutoscalerBuilder) {
			a.PodPriorityThreshold(10)
			a.LogVerbosity(2)
			a.MaxPodGracePeriod(10)
			a.IgnoreDaemonsetsUtilization(true)
			a.SkipNodesWithLocalStorage(true)
			a.BalanceSimilarNodeGroups(false)

			sd := &cmv1.AutoscalerScaleDownConfigBuilder{}
			sd.Enabled(true)
			sd.UtilizationThreshold("20")
			a.ScaleDown(sd)

			rl := &cmv1.AutoscalerResourceLimitsBuilder{}
			rl.MaxNodesTotal(10)

			mem := &cmv1.ResourceRangeBuilder{}
			mem.Max(10).Min(5)
			rl.Memory(mem)

			cores := &cmv1.ResourceRangeBuilder{}
			cores.Min(20).Max(30)
			rl.Cores(cores)
			a.ResourceLimits(rl)
		})

		out := PrintAutoscaler(autoscaler)
		fmt.Println(out)
		Expect(out).NotTo(BeNil())
		Expect(mandatoryFieldOutput).To(Equal(out))
	})
})
