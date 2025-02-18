package clusterautoscaler

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/output"
)

func PrintAutoscaler(a *cmv1.ClusterAutoscaler) string {

	out := "\n"
	out += fmt.Sprintf("Balance Similar Node Groups:               %s\n",
		output.PrintBool(a.BalanceSimilarNodeGroups()))
	out += fmt.Sprintf("Skip Nodes With Local Storage:             %s\n",
		output.PrintBool(a.SkipNodesWithLocalStorage()))
	out += fmt.Sprintf("Log Verbosity:                             %d\n",
		a.LogVerbosity())

	if len(a.BalancingIgnoredLabels()) > 0 {
		out += fmt.Sprintf("Labels Ignored For Node Balancing:         %s\n",
			output.PrintStringSlice(a.BalancingIgnoredLabels()))
	}

	out += fmt.Sprintf("Ignore DaemonSets Utilization:             %s\n",
		output.PrintBool(a.IgnoreDaemonsetsUtilization()))

	if a.MaxNodeProvisionTime() != "" {
		out += fmt.Sprintf("Maximum Node Provision Time:               %s\n",
			a.MaxNodeProvisionTime())
	}

	out += fmt.Sprintf("Maximum Pod Grace Period:                  %d\n",
		a.MaxPodGracePeriod())
	out += fmt.Sprintf("Pod Priority Threshold:                    %d\n",
		a.PodPriorityThreshold())

	//Resource Limits
	out += "Resource Limits:\n"
	out += fmt.Sprintf(" - Maximum Nodes:                          %d\n",
		a.ResourceLimits().MaxNodesTotal())
	out += fmt.Sprintf(" - Minimum Number of Cores:                %d\n",
		a.ResourceLimits().Cores().Min())
	out += fmt.Sprintf(" - Maximum Number of Cores:                %d\n",
		a.ResourceLimits().Cores().Max())
	out += fmt.Sprintf(" - Minimum Memory (GiB):                   %d\n",
		a.ResourceLimits().Memory().Min())
	out += fmt.Sprintf(" - Maximum Memory (GiB):                   %d\n",
		a.ResourceLimits().Memory().Max())

	if len(a.ResourceLimits().GPUS()) > 0 {
		out += " - GPU Limitations:\n"
		for _, limitation := range a.ResourceLimits().GPUS() {
			out += fmt.Sprintf("  - Type: %s\n", limitation.Type())
			out += fmt.Sprintf("   - Min:  %d\n", limitation.Range().Min())
			out += fmt.Sprintf("   - Max:  %d\n", limitation.Range().Max())
		}
	}

	//Scale Down
	out += "Scale Down:\n"
	out += fmt.Sprintf(" - Enabled:                                %s\n",
		output.PrintBool(a.ScaleDown().Enabled()))

	if a.ScaleDown().UnneededTime() != "" {
		out += fmt.Sprintf(" - Node Unneeded Time:                     %s\n",
			a.ScaleDown().UnneededTime())
	}
	out += fmt.Sprintf(" - Node Utilization Threshold:             %s\n",
		a.ScaleDown().UtilizationThreshold())

	if a.ScaleDown().DelayAfterAdd() != "" {
		out += fmt.Sprintf(" - Delay After Node Added:                 %s\n",
			a.ScaleDown().DelayAfterAdd())
	}

	if a.ScaleDown().DelayAfterDelete() != "" {
		out += fmt.Sprintf(" - Delay After Node Deleted:               %s\n",
			a.ScaleDown().DelayAfterDelete())
	}

	if a.ScaleDown().DelayAfterFailure() != "" {
		out += fmt.Sprintf(" - Delay After Node Deletion Failure:      %s\n",
			a.ScaleDown().DelayAfterFailure())
	}

	return out
}

func PrintHypershiftAutoscaler(a *cmv1.ClusterAutoscaler) string {

	out := "\n"

	if a.MaxNodeProvisionTime() != "" {
		out += fmt.Sprintf("Maximum Node Provision Time:               %s\n",
			a.MaxNodeProvisionTime())
	}

	out += fmt.Sprintf("Maximum Pod Grace Period:                  %d\n",
		a.MaxPodGracePeriod())
	out += fmt.Sprintf("Pod Priority Threshold:                    %d\n",
		a.PodPriorityThreshold())

	//Resource Limits
	out += "Resource Limits:\n"
	out += fmt.Sprintf(" - Maximum Nodes:                          %d\n",
		a.ResourceLimits().MaxNodesTotal())

	return out
}
