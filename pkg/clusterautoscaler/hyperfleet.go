package clusterautoscaler

import (
	"fmt"
	"math"

	"github.com/spf13/cobra"
)

// HyperfleetAutoscalingConfig contains the cluster-autoscaler options supported
// by HyperFleet. Pointer fields preserve whether the user explicitly supplied a
// scalar value, including zero.
type HyperfleetAutoscalingConfig struct {
	MaxNodeProvisionTime *string `json:"maxNodeProvisionTime,omitempty"`
	MaxNodesTotal        *int32  `json:"maxNodesTotal,omitempty"`
	MaxPodGracePeriod    *int32  `json:"maxPodGracePeriod,omitempty"`
	PodPriorityThreshold *int32  `json:"podPriorityThreshold,omitempty"`
}

// BuildHyperfleetAutoscalingConfig validates and converts the existing ROSA
// autoscaler flags to the subset supported by HyperFleet. It returns nil when
// no autoscaler flag was supplied and requireChanges is false.
func BuildHyperfleetAutoscalingConfig(
	cmd *cobra.Command,
	prefix string,
	args *AutoscalerArgs,
	requireChanges bool,
) (*HyperfleetAutoscalingConfig, error) {
	if cmd == nil || args == nil {
		return nil, nil
	}
	if !IsAutoscalerSetViaCLI(cmd.Flags(), prefix) {
		if requireChanges {
			_, err := ValidateAutoscalerFlagsForHostedCp(prefix, cmd)
			return nil, err
		}
		return nil, nil
	}
	if ok, err := ValidateAutoscalerFlagsForHostedCp(prefix, cmd); !ok || err != nil {
		return nil, err
	}

	result := &HyperfleetAutoscalingConfig{}
	changed := func(name string) bool {
		return cmd.Flags().Changed(prefix + name)
	}
	int32Value := func(name string, value int) (*int32, error) {
		if value < math.MinInt32 || value > math.MaxInt32 {
			return nil, fmt.Errorf("--%s%s must be between %d and %d", prefix, name, math.MinInt32, math.MaxInt32)
		}
		converted := int32(value)
		return &converted, nil
	}

	if changed(maxNodeProvisionTimeFlag) {
		value := args.MaxNodeProvisionTime
		result.MaxNodeProvisionTime = &value
	}
	var err error
	if changed(maxNodesTotalFlag) {
		result.MaxNodesTotal, err = int32Value(maxNodesTotalFlag, args.ResourceLimits.MaxNodesTotal)
		if err != nil {
			return nil, err
		}
	}
	if changed(maxPodGracePeriodFlag) {
		result.MaxPodGracePeriod, err = int32Value(maxPodGracePeriodFlag, args.MaxPodGracePeriod)
		if err != nil {
			return nil, err
		}
	}
	if changed(podPriorityThresholdFlag) {
		result.PodPriorityThreshold, err = int32Value(podPriorityThresholdFlag, args.PodPriorityThreshold)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
