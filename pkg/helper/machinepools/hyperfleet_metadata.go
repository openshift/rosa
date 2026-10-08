package machinepools

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// ParseHyperfleetLabels validates customer labels using the shared machinepool validators.
func ParseHyperfleetLabels(input string) (map[string]string, error) {
	if input != "" && input != `""` {
		entries := strings.Split(input, ",")
		for i, label := range entries {
			if label == "" && i == len(entries)-1 {
				continue
			}
			if strings.Count(label, "=") != 1 {
				return nil, fmt.Errorf("expected key=value format for labels")
			}
		}
	}
	return ParseLabels(input)
}

// ParseHyperfleetTaints validates and converts customer taints for HyperFleet requests.
func ParseHyperfleetTaints(input string) ([]hypershiftv1beta1.Taint, error) {
	if input == "" || input == `""` {
		return nil, nil
	}

	entries := strings.Split(input, ",")
	trimmedEntries := make([]string, 0, len(entries))
	for i, taint := range entries {
		if taint == "" && i == len(entries)-1 && i > 0 {
			continue
		}
		originalTaint := taint
		taint = strings.TrimSpace(taint)
		if !strings.Contains(taint, "=") || !strings.Contains(taint, ":") {
			return nil, fmt.Errorf("expected key=value:scheduleType format for taints. Got '%s'", originalTaint)
		}
		if strings.Count(taint, "=") != 1 || strings.Count(taint, ":") != 1 {
			return nil, fmt.Errorf("invalid taint format: '%s'. Expected format is '<key>=<value>:<effect>'", taint)
		}
		trimmedEntries = append(trimmedEntries, taint)
	}

	parsedTaints, err := ParseTaints(strings.Join(trimmedEntries, ","))
	if err != nil {
		return nil, err
	}

	taints := make([]hypershiftv1beta1.Taint, 0, len(parsedTaints))
	for _, taintBuilder := range parsedTaints {
		parsed, err := taintBuilder.Build()
		if err != nil {
			return nil, err
		}
		taints = append(taints, hypershiftv1beta1.Taint{
			Key:    parsed.Key(),
			Value:  parsed.Value(),
			Effect: corev1.TaintEffect(parsed.Effect()),
		})
	}
	return taints, nil
}
