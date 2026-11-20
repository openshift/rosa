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

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResultAddFinding(t *testing.T) {
	r := &Result{Eligible: true, EvaluatedAt: time.Now().UTC()}

	r.AddFinding(Finding{
		Severity:   SeverityOK,
		ReasonCode: "TEST_OK",
		Summary:    "All good",
	})
	if !r.Eligible {
		t.Error("expected Eligible to remain true after OK finding")
	}

	r.AddFinding(Finding{
		Severity:   SeverityBlocker,
		ReasonCode: ReasonLimitedSupport,
		Owner:      OwnerCustomer,
		Summary:    "Limited Support active",
	})
	if r.Eligible {
		t.Error("expected Eligible to become false after BLOCKER finding")
	}
}

func TestResultBlockerCount(t *testing.T) {
	r := &Result{Eligible: true}
	r.AddFinding(Finding{Severity: SeverityBlocker, ReasonCode: ReasonLimitedSupport})
	r.AddFinding(Finding{Severity: SeverityWarning, ReasonCode: "WARN"})
	r.AddFinding(Finding{Severity: SeverityBlocker, ReasonCode: ReasonGateUnacked})

	if r.BlockerCount() != 2 {
		t.Errorf("expected 2 blockers, got %d", r.BlockerCount())
	}
}

func TestResultReasonCodes(t *testing.T) {
	r := &Result{Eligible: true}
	r.AddFinding(Finding{Severity: SeverityBlocker, ReasonCode: ReasonLimitedSupport})
	r.AddFinding(Finding{Severity: SeverityBlocker, ReasonCode: ReasonGateUnacked})
	r.AddFinding(Finding{Severity: SeverityBlocker, ReasonCode: ReasonLimitedSupport}) // duplicate
	r.AddFinding(Finding{Severity: SeverityWarning, ReasonCode: "WARN"})

	codes := r.ReasonCodes()
	if len(codes) != 2 {
		t.Errorf("expected 2 unique reason codes, got %d: %v", len(codes), codes)
	}
	if codes[0] != ReasonLimitedSupport {
		t.Errorf("expected first code to be %s, got %s", ReasonLimitedSupport, codes[0])
	}
	if codes[1] != ReasonGateUnacked {
		t.Errorf("expected second code to be %s, got %s", ReasonGateUnacked, codes[1])
	}
}

func TestResultJSONSerialization(t *testing.T) {
	r := &Result{
		Cluster: ClusterInfo{
			ID:      "abc123",
			Name:    "test-cluster",
			Product: "ROSA Classic",
			Version: "4.17.33",
			Region:  "us-east-1",
			State:   "ready",
		},
		TargetVersion:     "4.18.12",
		Eligible:          false,
		AvailableUpgrades: []string{"4.17.34", "4.18.12"},
		EvaluatedAt:       time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC),
		Findings: []Finding{
			{
				Severity:   SeverityBlocker,
				ReasonCode: ReasonGateUnacked,
				Owner:      OwnerCustomer,
				Summary:    "Version gates for 4.18 not acknowledged",
			},
		},
		Gates: []GateDetail{
			{
				ID:           "gate-ocp-418",
				Description:  "Kubernetes API removals in 4.18",
				Acknowledged: false,
			},
		},
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed Result
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Cluster.ID != "abc123" {
		t.Errorf("expected cluster ID abc123, got %s", parsed.Cluster.ID)
	}
	if parsed.Eligible {
		t.Error("expected Eligible to be false")
	}
	if len(parsed.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(parsed.Findings))
	}
	if parsed.Findings[0].ReasonCode != ReasonGateUnacked {
		t.Errorf("expected reason code %s, got %s", ReasonGateUnacked, parsed.Findings[0].ReasonCode)
	}
}
