package hyperfleet

import (
	"fmt"
	"sort"
	"strings"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// FormatNodePoolLabels returns labels in deterministic order for CLI output.
func FormatNodePoolLabels(labels map[string]string) string {
	pairs := make([]string, 0, len(labels))
	for key, value := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, value))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ", ")
}

// FormatNodePoolTaints preserves the API's taint order and empty values.
func FormatNodePoolTaints(taints []hypershiftv1beta1.Taint) string {
	entries := make([]string, 0, len(taints))
	for _, taint := range taints {
		entries = append(entries, fmt.Sprintf("%s=%s:%s", taint.Key, taint.Value, taint.Effect))
	}
	return strings.Join(entries, ", ")
}

// FormatNodePoolReplicas shows observed/desired replicas or observed/min-max.
// A dash means that the API has not reported an observed count yet.
func FormatNodePoolReplicas(np *v1alpha1.NodePool) string {
	current := "-"
	if np.Status.Replicas != nil {
		current = fmt.Sprintf("%d", *np.Status.Replicas)
	}
	if scaling := np.Spec.NodePool.AutoScaling; scaling != nil {
		min := int32(0)
		if scaling.Min != nil {
			min = *scaling.Min
		}
		return fmt.Sprintf("%s/%d-%d", current, min, scaling.Max)
	}
	desired := int32(0)
	if np.Spec.NodePool.Replicas != nil {
		desired = *np.Spec.NodePool.Replicas
	}
	return fmt.Sprintf("%s/%d", current, desired)
}

func FormatNodePoolAutoscaling(np *v1alpha1.NodePool) string {
	if np.Spec.NodePool.AutoScaling != nil {
		return "Yes"
	}
	return "No"
}

// FormatNodePoolAutorepair prefers the customer setting over rendered management.
func FormatNodePoolAutorepair(np *v1alpha1.NodePool) string {
	enabled := np.Spec.NodePool.Management.AutoRepair
	if np.Spec.AutoRepair != nil {
		enabled = *np.Spec.AutoRepair
	}
	if enabled {
		return "Yes"
	}
	return "No"
}
