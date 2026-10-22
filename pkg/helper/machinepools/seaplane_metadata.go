// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepools

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// ParseSeaplaneLabels validates customer labels using the shared machinepool validators.
func ParseSeaplaneLabels(input string) (map[string]string, error) {
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

// ParseSeaplaneTaints validates and converts customer taints for Seaplane requests.
func ParseSeaplaneTaints(input string) ([]hypershiftv1beta1.Taint, error) {
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

// ValidateSeaplaneUpgrade validates the rolling-update values for Seaplane
// requests. It reuses the value rules of the OCM path so both paths reject the
// same input, and adds the HyperShift constraint that maxSurge and
// maxUnavailable cannot both be zero: a rolling update that neither surges nor
// takes a node out can never make progress.
func ValidateSeaplaneUpgrade(maxSurge string, maxUnavailable string) error {
	if err := ValidateUpgradeMaxSurgeUnavailable(maxSurge); err != nil {
		return fmt.Errorf("invalid --max-surge: %v", err)
	}
	if err := ValidateUpgradeMaxSurgeUnavailable(maxUnavailable); err != nil {
		return fmt.Errorf("invalid --max-unavailable: %v", err)
	}
	if isZeroUpgradeValue(maxSurge) && isZeroUpgradeValue(maxUnavailable) {
		return fmt.Errorf("only one of --max-surge and --max-unavailable can be zero")
	}
	return nil
}

// isZeroUpgradeValue reports whether a rolling-update value is an explicit
// zero, in absolute ("0") or percentage ("0%") form. Empty means unset.
func isZeroUpgradeValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed == "0" || trimmed == "0%"
}

// ParseSeaplaneImageType maps a CLI image type onto the value the Platform
// API accepts. The OCM path spells the Linux default "Default" while
// HyperShift spells it "Linux", so both are accepted here and normalised.
// An empty value stays empty and leaves the API default in place.
func ParseSeaplaneImageType(imageType string) (string, error) {
	switch imageType {
	case "":
		return "", nil
	case string(cmv1.ImageTypeDefault), string(hypershiftv1beta1.ImageTypeLinux):
		return string(hypershiftv1beta1.ImageTypeLinux), nil
	case string(cmv1.ImageTypeWindows):
		return string(hypershiftv1beta1.ImageTypeWindows), nil
	}
	// Same wording as the OCM path, so both reject an unknown type alike.
	return "", fmt.Errorf("expected a valid image type for the machine pool: '%s'", imageType)
}

// ParseSeaplaneNodeDrainGracePeriod parses and validates a node drain grace period
// duration string and returns a metav1.Duration pointer suitable for the API.
// Empty input returns nil. Accepts format "N minutes" or "N hours" where N is
// a non-negative integer, with a maximum of 1 week (10080 minutes / 168 hours).
func ParseSeaplaneNodeDrainGracePeriod(input string) (*metav1.Duration, error) {
	if input == "" {
		return nil, nil
	}
	// Validate using the same rules as the OCM path
	if err := ValidateNodeDrainGracePeriod(input); err != nil {
		return nil, err
	}

	// Parse the duration string: "N minutes" or "N hours"
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid node drain timeout format: %q", input)
	}

	// If only a number is provided, default to minutes
	if len(parts) == 1 {
		parts = append(parts, "minutes")
	}

	var duration time.Duration
	switch parts[1] {
	case "minute", "minutes":
		d, err := time.ParseDuration(parts[0] + "m")
		if err != nil {
			return nil, fmt.Errorf("invalid node drain timeout value: %w", err)
		}
		duration = d
	case "hour", "hours":
		d, err := time.ParseDuration(parts[0] + "h")
		if err != nil {
			return nil, fmt.Errorf("invalid node drain timeout value: %w", err)
		}
		duration = d
	default:
		return nil, fmt.Errorf("invalid node drain timeout unit: %q, expected 'minute(s)' or 'hour(s)'", parts[1])
	}

	return &metav1.Duration{Duration: duration}, nil
}

// FormatSeaplaneNodeDrainGracePeriod formats a metav1.Duration as a human-readable
// string for display. Returns empty string for nil or zero duration.
func FormatSeaplaneNodeDrainGracePeriod(d *metav1.Duration) string {
	if d == nil || d.Duration == 0 {
		return ""
	}
	return d.Duration.String()
}
