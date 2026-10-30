/*
Copyright (c) 2026 Red Hat, Inc.

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

package eligibility

import "time"

type Severity string

const (
	SeverityBlocker Severity = "BLOCKER"
	SeverityWarning Severity = "WARNING"
	SeverityOK      Severity = "OK"
)

type Owner string

const (
	OwnerCustomer Owner = "customer"
	OwnerService  Owner = "service-team"
)

type ReasonCode string

const (
	ReasonClusterNotReady     ReasonCode = "CLUSTER_NOT_READY"
	ReasonLimitedSupport      ReasonCode = "LIMITED_SUPPORT"
	ReasonLimitedSupportEOL   ReasonCode = "LIMITED_SUPPORT_EOL_VERSION"
	ReasonUpgradeInFlight     ReasonCode = "UPGRADE_IN_FLIGHT"
	ReasonGateUnacked         ReasonCode = "GATE_UNACKED"
	ReasonNoTargetsAvailable  ReasonCode = "NO_TARGETS_AVAILABLE"
	ReasonTargetNotAvailable  ReasonCode = "TARGET_NOT_AVAILABLE"
	ReasonAtLatestVersion     ReasonCode = "AT_LATEST_VERSION"
)

type Finding struct {
	Severity   Severity   `json:"severity"`
	ReasonCode ReasonCode `json:"reason_code"`
	Owner      Owner      `json:"owner"`
	Summary    string     `json:"summary"`
	Detail     string     `json:"detail,omitempty"`
	Action     string     `json:"action,omitempty"`
}

type GateDetail struct {
	ID               string `json:"id"`
	Description      string `json:"description"`
	WarningMessage   string `json:"warning_message,omitempty"`
	DocumentationURL string `json:"documentation_url,omitempty"`
	STSOnly          bool   `json:"sts_only"`
	Acknowledged     bool   `json:"acknowledged"`
}

type ClusterInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Product      string `json:"product"`
	Version      string `json:"version"`
	ChannelGroup string `json:"channel_group"`
	Region       string `json:"region"`
	State        string `json:"state"`
	Hypershift   bool   `json:"hypershift"`
}

type Result struct {
	Cluster           ClusterInfo  `json:"cluster"`
	TargetVersion     string       `json:"target_version,omitempty"`
	Eligible          bool         `json:"eligible"`
	Findings          []Finding    `json:"findings"`
	Gates             []GateDetail `json:"gates,omitempty"`
	AvailableUpgrades []string     `json:"available_upgrades,omitempty"`
	EvaluatedAt       time.Time    `json:"evaluated_at"`
	atLatestVersion   bool
}

// AddFinding appends a finding and marks the result ineligible if it is a blocker.
func (r *Result) AddFinding(f Finding) {
	r.Findings = append(r.Findings, f)
	if f.Severity == SeverityBlocker {
		r.Eligible = false
	}
}

// BlockerCount returns the number of blocker-severity findings.
func (r *Result) BlockerCount() int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == SeverityBlocker {
			n++
		}
	}
	return n
}

// ReasonCodes returns deduplicated blocker reason codes.
func (r *Result) ReasonCodes() []ReasonCode {
	codes := make([]ReasonCode, 0, len(r.Findings))
	seen := make(map[ReasonCode]bool)
	for _, f := range r.Findings {
		if f.Severity == SeverityBlocker && !seen[f.ReasonCode] {
			codes = append(codes, f.ReasonCode)
			seen[f.ReasonCode] = true
		}
	}
	return codes
}
