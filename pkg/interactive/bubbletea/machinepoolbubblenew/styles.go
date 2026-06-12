/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package machinepoolbubblenew

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const totalSteps = 24

var (
	accentColor = lipgloss.AdaptiveColor{Light: "#AD58B4", Dark: "#EE6FF8"}
	mutedColor  = lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}
	textColor   = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}
	okColor     = lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}
	errColor    = lipgloss.AdaptiveColor{Light: "#FF4676", Dark: "#ED567A"}

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			Padding(0, 0, 1, 0)

	progressFillStyle = lipgloss.NewStyle().
				Foreground(accentColor)

	progressEmptyStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	summaryLabelStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Width(28)

	summaryValueStyle = lipgloss.NewStyle().
				Foreground(textColor)

	optionalBadgeStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errColor).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(okColor).
			Italic(true)

	promptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)
)

func stepNumber(step wizardStep) int {
	return int(step) + 1
}

func stepSection(step wizardStep) string {
	switch step {
	case stepName, stepImageType, stepVersion:
		return "Basics"
	case stepSelectSubnet, stepAvailabilityZone, stepSubnetID:
		return "Networking"
	case stepAutoscaling, stepMinReplicas, stepMaxReplicas:
		return "Scaling"
	case stepLabels, stepTaints, stepSecurityGroups, stepTags:
		return "Metadata"
	case stepInstanceType, stepAutorepair, stepTuningConfigs:
		return "Compute"
	case stepCapacityReservationID, stepCapacityPreference, stepKubeletConfig, stepHTTPTokens:
		return "Advanced"
	default:
		return "Upgrades"
	}
}

func renderHeader(step wizardStep) string {
	title := fmt.Sprintf("Create machine pool  ·  Step %d/%d  ·  %s",
		stepNumber(step), totalSteps, stepSection(step))
	return headerStyle.Render(title) + "\n" + renderProgressBar(stepNumber(step), totalSteps)
}

func renderProgressBar(current, total int) string {
	const width = 32
	filled := (current * width) / total
	if filled > width {
		filled = width
	}
	fill := progressFillStyle.Render(strings.Repeat("█", filled))
	empty := progressEmptyStyle.Render(strings.Repeat("░", width-filled))
	percent := (current * 100) / total
	return fmt.Sprintf("%s  %d%%", fill+empty, percent)
}

func renderSummary(completed []completedAnswer) string {
	if len(completed) == 0 {
		return ""
	}
	var b strings.Builder
	for _, answer := range completed {
		label := summaryLabelStyle.Render(answer.label)
		value := summaryValueStyle.Render(formatSummaryValue(answer.value))
		b.WriteString(fmt.Sprintf("  %s %s\n", label, value))
	}
	return b.String()
}

func formatSummaryValue(value string) string {
	if value == "" {
		return "—"
	}
	return value
}

func renderTextPrompt(title string, optional bool, help string) string {
	line := promptStyle.Render(title)
	if optional {
		line += " " + optionalBadgeStyle.Render("optional")
	}
	if help != "" {
		line += "\n" + helpStyle.Render(help)
	}
	return line
}
