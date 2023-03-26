package machinepool

import (
	"fmt"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	Yes = "Yes"
	No  = "No"
)

func printStringSlice(in []string) string {
	if len(in) == 0 {
		return ""
	}
	return strings.Join(in, ", ")
}

func printLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	output := []string{}
	for k, v := range labels {
		output = append(output, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(output, ", ")
}

func printTaints(taints []*cmv1.Taint) string {
	if len(taints) == 0 {
		return ""
	}
	output := []string{}
	for _, taint := range taints {
		output = append(output, fmt.Sprintf("%s=%s:%s", taint.Key(), taint.Value(), taint.Effect()))
	}

	return strings.Join(output, ", ")
}
