package clusterautoscaler

import (
	"fmt"
	"math"
)

// SeaplaneAutoscalingConfig contains the cluster-autoscaler options supported
// by Seaplane. Pointer fields preserve whether the user explicitly supplied a
// scalar value, including zero.
type SeaplaneAutoscalingConfig struct {
	MaxNodeProvisionTime *string `json:"maxNodeProvisionTime,omitempty"`
	MaxNodesTotal        *int32  `json:"maxNodesTotal,omitempty"`
	MaxPodGracePeriod    *int32  `json:"maxPodGracePeriod,omitempty"`
	PodPriorityThreshold *int32  `json:"podPriorityThreshold,omitempty"`
}

// SeaplaneAutoscalingChanges identifies the supported autoscaler flags the
// user explicitly changed. Cobra flag inspection remains in command packages.
type SeaplaneAutoscalingChanges struct {
	MaxNodeProvisionTime bool
	MaxNodesTotal        bool
	MaxPodGracePeriod    bool
	PodPriorityThreshold bool
}

// BuildSeaplaneAutoscalingConfig validates and converts the existing ROSA
// autoscaler arguments to the subset supported by Seaplane. It returns nil
// when no supported flag changed.
func BuildSeaplaneAutoscalingConfig(
	args *AutoscalerArgs,
	prefix string,
	changes SeaplaneAutoscalingChanges,
) (*SeaplaneAutoscalingConfig, error) {
	if args == nil || !changes.any() {
		return nil, nil
	}

	result := &SeaplaneAutoscalingConfig{}
	int32Value := func(name string, value int) (*int32, error) {
		if value < math.MinInt32 || value > math.MaxInt32 {
			return nil, fmt.Errorf("--%s%s must be between %d and %d", prefix, name, math.MinInt32, math.MaxInt32)
		}
		converted := int32(value)
		return &converted, nil
	}

	if changes.MaxNodeProvisionTime {
		value := args.MaxNodeProvisionTime
		result.MaxNodeProvisionTime = &value
	}
	var err error
	if changes.MaxNodesTotal {
		result.MaxNodesTotal, err = int32Value(maxNodesTotalFlag, args.ResourceLimits.MaxNodesTotal)
		if err != nil {
			return nil, err
		}
	}
	if changes.MaxPodGracePeriod {
		result.MaxPodGracePeriod, err = int32Value(maxPodGracePeriodFlag, args.MaxPodGracePeriod)
		if err != nil {
			return nil, err
		}
	}
	if changes.PodPriorityThreshold {
		result.PodPriorityThreshold, err = int32Value(podPriorityThresholdFlag, args.PodPriorityThreshold)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c SeaplaneAutoscalingChanges) any() bool {
	return c.MaxNodeProvisionTime || c.MaxNodesTotal || c.MaxPodGracePeriod || c.PodPriorityThreshold
}
