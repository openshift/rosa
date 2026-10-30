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
	"testing"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// mockOCMClient implements OCMClient for testing.
type mockOCMClient struct {
	limitedSupportReasons []*cmv1.LimitedSupportReason
	scheduledPolicy       *cmv1.UpgradePolicy
	scheduledState        *cmv1.UpgradePolicyState
	upgradePolicies       []*cmv1.UpgradePolicy
	availableUpgrades     []string
	rawAvailableUpgrades  []string
	gates                 []*cmv1.VersionGate
	gateAgreements        []*cmv1.VersionGateAgreement
	err                   error
}

func (m *mockOCMClient) GetLimitedSupportReasons(string) ([]*cmv1.LimitedSupportReason, error) {
	return m.limitedSupportReasons, m.err
}

func (m *mockOCMClient) GetScheduledUpgrade(string) (*cmv1.UpgradePolicy, *cmv1.UpgradePolicyState, error) {
	return m.scheduledPolicy, m.scheduledState, m.err
}

func (m *mockOCMClient) GetUpgradePolicies(string) ([]*cmv1.UpgradePolicy, error) {
	return m.upgradePolicies, m.err
}

func (m *mockOCMClient) GetAvailableUpgrades(string) ([]string, error) {
	return m.availableUpgrades, m.err
}

func (m *mockOCMClient) GetRawAvailableUpgrades(string) ([]string, error) {
	return m.rawAvailableUpgrades, m.err
}

func (m *mockOCMClient) ListAllOcpGates(string) ([]*cmv1.VersionGate, error) {
	return m.gates, m.err
}

func (m *mockOCMClient) ListGateAgreements(string) ([]*cmv1.VersionGateAgreement, error) {
	return m.gateAgreements, m.err
}

// --- helpers ---

func mustBuildCluster(b *cmv1.ClusterBuilder) *cmv1.Cluster {
	c, err := b.Build()
	if err != nil {
		panic(err)
	}
	return c
}

func readyHCPCluster(version, channelGroup string) *cmv1.Cluster {
	return mustBuildCluster(
		cmv1.NewCluster().
			ID("test-cluster-id").
			Name("test-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion(version).
			Version(cmv1.NewVersion().ID("openshift-v"+version).ChannelGroup(channelGroup)).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			CreationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
	)
}

func readyClassicCluster(version, channelGroup string) *cmv1.Cluster {
	return mustBuildCluster(
		cmv1.NewCluster().
			ID("test-cluster-id").
			Name("test-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion(version).
			Version(cmv1.NewVersion().ID("openshift-v"+version).ChannelGroup(channelGroup)).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(false)).
			CreationTimestamp(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
	)
}

func mustBuildLS(summary string) *cmv1.LimitedSupportReason {
	ls, err := cmv1.NewLimitedSupportReason().ID("ls-1").Summary(summary).Build()
	if err != nil {
		panic(err)
	}
	return ls
}

func mustBuildGate(id, description string, stsOnly bool, createdAt time.Time) *cmv1.VersionGate {
	g, err := cmv1.NewVersionGate().
		ID(id).
		Description(description).
		STSOnly(stsOnly).
		CreationTimestamp(createdAt).
		Build()
	if err != nil {
		panic(err)
	}
	return g
}

func mustBuildGateWithLabel(id, description, label string, stsOnly bool, createdAt time.Time) *cmv1.VersionGate {
	g, err := cmv1.NewVersionGate().
		ID(id).
		Description(description).
		Label(label).
		STSOnly(stsOnly).
		CreationTimestamp(createdAt).
		Build()
	if err != nil {
		panic(err)
	}
	return g
}

func mustBuildGateAgreement(gateID string) *cmv1.VersionGateAgreement {
	a, err := cmv1.NewVersionGateAgreement().
		ID("agree-1").
		VersionGate(cmv1.NewVersionGate().ID(gateID)).
		Build()
	if err != nil {
		panic(err)
	}
	return a
}

func mustBuildUpgradePolicy(version string, nextRun time.Time) *cmv1.UpgradePolicy {
	p, err := cmv1.NewUpgradePolicy().
		ID("policy-1").
		Version(version).
		NextRun(nextRun).
		UpgradeType("OSD").
		Build()
	if err != nil {
		panic(err)
	}
	return p
}

func assertFinding(t *testing.T, result *Result, code ReasonCode, severity Severity) {
	t.Helper()
	for _, f := range result.Findings {
		if f.ReasonCode == code && f.Severity == severity {
			return
		}
	}
	codes := make([]string, len(result.Findings))
	for i, f := range result.Findings {
		codes[i] = string(f.ReasonCode) + ":" + string(f.Severity)
	}
	t.Errorf("expected finding %s:%s, got findings: %v", code, severity, codes)
}

func assertNoBlockers(t *testing.T, result *Result) {
	t.Helper()
	if !result.Eligible {
		codes := make([]string, 0)
		for _, f := range result.Findings {
			if f.Severity == SeverityBlocker {
				codes = append(codes, string(f.ReasonCode))
			}
		}
		t.Errorf("expected eligible, got blockers: %v", codes)
	}
}

// =============================================================================
// CATEGORY A: Gate/ack/path disagreement between surfaces
// =============================================================================

// OHSS-52583: No upgrade path visible despite Cincinnati showing one.
// OCM version gates + in-cluster admin-acks both unacknowledged.
// POC catches: YES — GATE_UNACKED
func TestOHSS52583_GateUnacknowledged(t *testing.T) {
	// Cluster created AFTER the gate — must explicitly ack
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("test-cluster-id").
			Name("test-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion("4.17.33").
			Version(cmv1.NewVersion().ID("openshift-v4.17.33").ChannelGroup("stable")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			CreationTimestamp(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)),
	)

	// Gate was created BEFORE the cluster
	gate := mustBuildGate("gate-418-api", "Kubernetes API removals in 4.18", false,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		availableUpgrades:    []string{},
		rawAvailableUpgrades: []string{"4.18.12"},
		gates:                []*cmv1.VersionGate{gate},
		gateAgreements:       []*cmv1.VersionGateAgreement{},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Eligible {
		t.Error("expected NOT eligible")
	}
	assertFinding(t, result, ReasonGateUnacked, SeverityBlocker)
}

// OHSS-58695: Upgrade stalled — API removals ack not provided.
// Same pattern as 52583 but with explicit target version.
// POC catches: YES — GATE_UNACKED
func TestOHSS58695_GateUnackedWithTarget(t *testing.T) {
	// Cluster created AFTER the gate
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("test-cluster-id").
			Name("test-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion("4.17.33").
			Version(cmv1.NewVersion().ID("openshift-v4.17.33").ChannelGroup("stable")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			CreationTimestamp(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)),
	)

	gate := mustBuildGate("gate-418-api", "API removals acknowledgement for 4.18", false,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.18.12"},
		rawAvailableUpgrades: []string{"4.18.12"},
		gates:                []*cmv1.VersionGate{gate},
		gateAgreements:       []*cmv1.VersionGateAgreement{},
	}

	result, err := CheckAll(mock, cluster, "4.18.12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Eligible {
		t.Error("expected NOT eligible")
	}
	assertFinding(t, result, ReasonGateUnacked, SeverityBlocker)
}

// OHSS-52604: spec.pausedUntil set by Terraform, opaque OCM error.
// POC catches: NO (Day 2) — pausedUntil is in-cluster state, not exposed via OCM API.
// This test verifies the POC does NOT false-positive on this case.
func TestOHSS52604_PausedUntilNotDetectable(t *testing.T) {
	cluster := readyHCPCluster("4.19.20", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.19.23"},
		rawAvailableUpgrades: []string{"4.19.23"},
	}

	result, err := CheckAll(mock, cluster, "4.19.23")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// POC reports eligible because it can't see pausedUntil — this is a known Day 2 gap
	assertNoBlockers(t, result)
}

// OHSS-55966: MUO stuck at StartedNotificationSent.
// POC catches: NO (Day 2) — MUO controller state is in-cluster only.
// But if there was an active upgrade policy, the POC would catch that.
func TestOHSS55966_MUOStuckDetectsActivePolicy(t *testing.T) {
	cluster := readyClassicCluster("4.17.33", "stable")

	policy := mustBuildUpgradePolicy("4.18.12", time.Now().Add(-2*time.Hour))

	state, _ := cmv1.NewUpgradePolicyState().Value("started").Build()

	mock := &mockOCMClient{
		scheduledPolicy:      policy,
		scheduledState:       state,
		availableUpgrades:    []string{"4.18.12"},
		rawAvailableUpgrades: []string{"4.18.12"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertFinding(t, result, ReasonUpgradeInFlight, SeverityBlocker)
}

// OHSS-52268: CLI and OCM reported conflicting upgrade state.
// POC catches: PARTIAL — detects active upgrade policy, but not the surface disagreement itself.
func TestOHSS52268_ConflictingState(t *testing.T) {
	cluster := readyClassicCluster("4.17.33", "stable")

	policy := mustBuildUpgradePolicy("4.18.12", time.Now().Add(-1*time.Hour))

	mock := &mockOCMClient{
		scheduledPolicy:      policy,
		availableUpgrades:    []string{"4.18.12"},
		rawAvailableUpgrades: []string{"4.18.12"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertFinding(t, result, ReasonUpgradeInFlight, SeverityBlocker)
}

// =============================================================================
// CATEGORY B: Limited Support / console-CLI behavioral split
// =============================================================================

// OHSS-57359: Console Save disabled, CLI succeeds. Limited Support active.
// POC catches: YES — LIMITED_SUPPORT
func TestOHSS57359_LimitedSupportBlocks(t *testing.T) {
	cluster := readyClassicCluster("4.17.33", "stable")

	mock := &mockOCMClient{
		limitedSupportReasons: []*cmv1.LimitedSupportReason{
			mustBuildLS("Network egress blocked — customer VPC has no route to required endpoints"),
		},
		availableUpgrades:    []string{"4.17.34"},
		rawAvailableUpgrades: []string{"4.17.34"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Eligible {
		t.Error("expected NOT eligible")
	}
	assertFinding(t, result, ReasonLimitedSupport, SeverityBlocker)
}

// Version EOL Limited Support should be WARNING, not BLOCKER.
// Matt's feedback: if LS reason is "version is EOL", blocking upgrade is circular.
func TestLimitedSupport_VersionEOL_IsWarning(t *testing.T) {
	cluster := readyClassicCluster("4.14.1", "stable")

	mock := &mockOCMClient{
		limitedSupportReasons: []*cmv1.LimitedSupportReason{
			mustBuildLS("Cluster version has reached end of supported life cycle"),
		},
		availableUpgrades:    []string{"4.14.77"},
		rawAvailableUpgrades: []string{"4.14.77"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Version EOL LS should NOT block — upgrading IS the fix
	assertNoBlockers(t, result)
	assertFinding(t, result, ReasonLimitedSupportEOL, SeverityWarning)
}

// Mixed LS: version EOL (WARNING) + network issue (BLOCKER).
func TestLimitedSupport_MixedEOLAndOther(t *testing.T) {
	cluster := readyClassicCluster("4.14.1", "stable")

	mock := &mockOCMClient{
		limitedSupportReasons: []*cmv1.LimitedSupportReason{
			mustBuildLS("Cluster version has reached end of supported life cycle"),
			mustBuildLS("Network egress blocked — customer VPC has no route to required endpoints"),
		},
		availableUpgrades:    []string{"4.14.77"},
		rawAvailableUpgrades: []string{"4.14.77"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Network LS blocks, but version EOL is just a warning
	if result.Eligible {
		t.Error("expected NOT eligible due to network LS")
	}
	assertFinding(t, result, ReasonLimitedSupportEOL, SeverityWarning)
	assertFinding(t, result, ReasonLimitedSupport, SeverityBlocker)
}

// Only version EOL LS — cluster should still be eligible to upgrade.
func TestLimitedSupport_OnlyEOL_StillEligible(t *testing.T) {
	cluster := readyClassicCluster("4.14.1", "stable")

	mock := &mockOCMClient{
		limitedSupportReasons: []*cmv1.LimitedSupportReason{
			mustBuildLS("Cluster version has reached end of life"),
		},
		availableUpgrades:    []string{"4.14.77"},
		rawAvailableUpgrades: []string{"4.14.77"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
	if !result.Eligible {
		t.Error("expected eligible — version EOL LS should not block upgrade")
	}
}

// =============================================================================
// CATEGORY C: RBAC/permission drift
// =============================================================================

// OHSS-58345: dedicated-admin missing UpgradeConfig aggregate ClusterRoles.
// POC catches: NO (Day 2) — requires in-cluster RBAC check via backplane.
func TestOHSS58345_RBACDriftNotDetectable(t *testing.T) {
	cluster := readyClassicCluster("4.17.33", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.17.34"},
		rawAvailableUpgrades: []string{"4.17.34"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// POC reports eligible — RBAC drift is invisible from OCM
	assertNoBlockers(t, result)
}

// =============================================================================
// CATEGORY D: Service-owned gate, no customer repair path
// =============================================================================

// OHSS-58084: Boot image skew gate, customer can't repair on OSD GCP.
// POC catches: NO (Day 2) — ClusterBootImageSkewError is an in-cluster condition.
func TestOHSS58084_BootImageSkewNotDetectable(t *testing.T) {
	cluster := readyClassicCluster("4.14.1", "eus")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.14.77"},
		rawAvailableUpgrades: []string{"4.14.77"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// OHSS-58032: WifConfig 404, scheduling fails silently.
// POC catches: NO (Day 2) — WifConfig validation is in OCM console, not in CLI/API.
func TestOHSS58032_WifConfigNotDetectable(t *testing.T) {
	cluster := readyHCPCluster("4.17.33", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.17.34"},
		rawAvailableUpgrades: []string{"4.17.34"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// =============================================================================
// CATEGORY E: Customer config invalid, not validated pre-flight
// =============================================================================

// OHSS-53365: Pull secret mismatch causes upgrade to cancel at Pre-Health Check.
// POC catches: NO (Day 2) — requires in-cluster pull secret validation.
func TestOHSS53365_PullSecretNotDetectable(t *testing.T) {
	cluster := readyClassicCluster("4.18.6", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.18.31"},
		rawAvailableUpgrades: []string{"4.18.31"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// OHSS-56299: Invalid ClusterAutoscaler utilizationThreshold: 0.
// POC catches: NO (Day 3) — requires CLI input validation enhancement.
func TestOHSS56299_InvalidAutoscalerNotDetectable(t *testing.T) {
	cluster := readyClassicCluster("4.18.10", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.18.31"},
		rawAvailableUpgrades: []string{"4.18.31"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// OHSS-53041: Incompatible cluster-logging operator blocked y-stream upgrade.
// POC catches: NO (Day 2) — IncompatibleOperatorsInstalled is in-cluster condition.
func TestOHSS53041_IncompatibleOperatorNotDetectable(t *testing.T) {
	cluster := readyHCPCluster("4.18.20", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.19.10"},
		rawAvailableUpgrades: []string{"4.19.10"},
	}

	result, err := CheckAll(mock, cluster, "4.19.10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Gates might fire, but the operator incompatibility is invisible from OCM
	// Just verify there's no false finding about operators
	for _, f := range result.Findings {
		if f.ReasonCode == "INCOMPATIBLE_OPERATOR" {
			t.Error("unexpected INCOMPATIBLE_OPERATOR finding — this is a Day 2 check")
		}
	}
}

// OHSS-55964: Misconfigured console plugins blocked CVO.
// POC catches: NO (Day 2) — in-cluster CVO state.
func TestOHSS55964_ConsolePluginsNotDetectable(t *testing.T) {
	cluster := readyClassicCluster("4.18.20", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.18.31"},
		rawAvailableUpgrades: []string{"4.18.31"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// OHSS-54695: "found 2 critical alerts" — specific alerts not named.
// POC catches: NO (Day 2) — MUO pre-health check is in-cluster.
func TestOHSS54695_CriticalAlertsNotDetectable(t *testing.T) {
	cluster := readyClassicCluster("4.18.10", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.18.31"},
		rawAvailableUpgrades: []string{"4.18.31"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// =============================================================================
// CATEGORY F: Cloud infrastructure / access blocking upgrade
// =============================================================================

// OHSS-52380: Workers can't reach quay.io, no VPC egress route.
// POC catches: NO (Day 3) — requires cloud API / network check.
func TestOHSS52380_NoEgressNotDetectable(t *testing.T) {
	cluster := readyHCPCluster("4.17.33", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.17.34"},
		rawAvailableUpgrades: []string{"4.17.34"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// OHSS-54291: SCP enforcing IMDSv2 conflicts with httpTokens:optional.
// POC catches: NO (Day 3) — requires cloud SCP/policy check.
func TestOHSS54291_IMDSv2ConflictNotDetectable(t *testing.T) {
	cluster := readyHCPCluster("4.17.33", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.17.34"},
		rawAvailableUpgrades: []string{"4.17.34"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// OHSS-57568: Invalid cloud configuration, SRE can't access cloud console.
// POC catches: NO (Day 3) — requires cloud permission check.
func TestOHSS57568_CloudConfigNotDetectable(t *testing.T) {
	cluster := readyHCPCluster("4.17.33", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.17.34"},
		rawAvailableUpgrades: []string{"4.17.34"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// =============================================================================
// CATEGORY G: Pre-existing cluster state blocking upgrade
// =============================================================================

// OHSS-54944: Degraded node, DNS operator stuck.
// POC catches: NO (Day 2) — requires in-cluster node/operator health check.
func TestOHSS54944_DegradedNodeNotDetectable(t *testing.T) {
	cluster := readyClassicCluster("4.18.10", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.18.31"},
		rawAvailableUpgrades: []string{"4.18.31"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// OHSS-52328: GPU NodePool upgrade InstanceProvisionFailed — AWS quota exhausted.
// POC catches: NO (Day 3) — requires cloud quota check.
func TestOHSS52328_GPUQuotaNotDetectable(t *testing.T) {
	cluster := readyHCPCluster("4.17.33", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.17.34"},
		rawAvailableUpgrades: []string{"4.17.34"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// =============================================================================
// CROSS-CUTTING: Version gate edge cases
// =============================================================================

// Implicit gate acknowledgement: cluster created BEFORE gate, should be auto-acked.
func TestGateImplicitAck_ClusterCreatedBeforeGate(t *testing.T) {
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("old-cluster").
			Name("old-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion("4.17.33").
			Version(cmv1.NewVersion().ID("openshift-v4.17.33").ChannelGroup("stable")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			CreationTimestamp(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
	)

	// Gate was created AFTER the cluster
	gate := mustBuildGate("gate-418", "API removals in 4.18", false,
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.18.12"},
		rawAvailableUpgrades: []string{"4.18.12"},
		gates:                []*cmv1.VersionGate{gate},
		gateAgreements:       []*cmv1.VersionGateAgreement{},
	}

	result, err := CheckAll(mock, cluster, "4.18.12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Gate should be implicitly acknowledged
	assertNoBlockers(t, result)
	if len(result.Gates) != 1 || !result.Gates[0].Acknowledged {
		t.Error("expected gate to be implicitly acknowledged")
	}
}

// Gate NOT implicitly acked: cluster created AFTER gate.
func TestGateNotImplicitAck_ClusterCreatedAfterGate(t *testing.T) {
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("new-cluster").
			Name("new-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion("4.17.33").
			Version(cmv1.NewVersion().ID("openshift-v4.17.33").ChannelGroup("stable")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			CreationTimestamp(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)),
	)

	// Gate was created BEFORE the cluster
	gate := mustBuildGate("gate-418", "API removals in 4.18", false,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.18.12"},
		rawAvailableUpgrades: []string{"4.18.12"},
		gates:                []*cmv1.VersionGate{gate},
		gateAgreements:       []*cmv1.VersionGateAgreement{},
	}

	result, err := CheckAll(mock, cluster, "4.18.12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertFinding(t, result, ReasonGateUnacked, SeverityBlocker)
}

// Explicit gate agreement overrides implicit logic.
func TestGateExplicitAck(t *testing.T) {
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("new-cluster").
			Name("new-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion("4.17.33").
			Version(cmv1.NewVersion().ID("openshift-v4.17.33").ChannelGroup("stable")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			CreationTimestamp(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)),
	)

	gate := mustBuildGate("gate-418", "API removals in 4.18", false,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.18.12"},
		rawAvailableUpgrades: []string{"4.18.12"},
		gates:                []*cmv1.VersionGate{gate},
		gateAgreements:       []*cmv1.VersionGateAgreement{mustBuildGateAgreement("gate-418")},
	}

	result, err := CheckAll(mock, cluster, "4.18.12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
}

// =============================================================================
// CROSS-CUTTING: "At latest" vs "blocked by gates"
// =============================================================================

// Cluster on latest version — should be ELIGIBLE with AT_LATEST_VERSION.
func TestAtLatestVersion(t *testing.T) {
	cluster := readyHCPCluster("4.22.20", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{},
		rawAvailableUpgrades: []string{},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
	assertFinding(t, result, ReasonAtLatestVersion, SeverityOK)
}

// No ROSA-enabled targets but raw targets exist — BLOCKED.
func TestNoRosaEnabledTargets(t *testing.T) {
	cluster := readyHCPCluster("4.18.33", "stable")

	mock := &mockOCMClient{
		availableUpgrades:    []string{},
		rawAvailableUpgrades: []string{"4.19.1", "4.19.2"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Eligible {
		t.Error("expected NOT eligible")
	}
	assertFinding(t, result, ReasonNoTargetsAvailable, SeverityBlocker)
}

// =============================================================================
// CROSS-CUTTING: Multiple blockers in one pass
// =============================================================================

// Cluster with Limited Support AND unacked gates — should report BOTH.
func TestMultipleBlockers(t *testing.T) {
	cluster := readyClassicCluster("4.17.33", "stable")

	gate := mustBuildGate("gate-418", "API removals in 4.18", false,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		limitedSupportReasons: []*cmv1.LimitedSupportReason{
			mustBuildLS("Network egress blocked"),
		},
		availableUpgrades:    []string{},
		rawAvailableUpgrades: []string{"4.18.12"},
		gates:                []*cmv1.VersionGate{gate},
		gateAgreements:       []*cmv1.VersionGateAgreement{},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Eligible {
		t.Error("expected NOT eligible")
	}
	assertFinding(t, result, ReasonLimitedSupport, SeverityBlocker)
	assertFinding(t, result, ReasonNoTargetsAvailable, SeverityBlocker)
	assertFinding(t, result, ReasonGateUnacked, SeverityBlocker)

	if result.BlockerCount() < 3 {
		t.Errorf("expected at least 3 blockers, got %d", result.BlockerCount())
	}
}

// Cluster NOT ready — should block immediately.
func TestClusterNotReady(t *testing.T) {
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("error-cluster").
			Name("error-cluster").
			State(cmv1.ClusterStateError).
			OpenshiftVersion("4.17.33").
			Version(cmv1.NewVersion().ID("openshift-v4.17.33").ChannelGroup("stable")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			CreationTimestamp(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
	)

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.17.34"},
		rawAvailableUpgrades: []string{"4.17.34"},
	}

	result, err := CheckAll(mock, cluster, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertFinding(t, result, ReasonClusterNotReady, SeverityBlocker)
}

// =============================================================================
// COVERAGE SUMMARY (validated by tests above)
// =============================================================================
//
// Day 1 POC catches (6/19):
//   OHSS-52583 — GATE_UNACKED           ✓ tested
//   OHSS-58695 — GATE_UNACKED           ✓ tested
//   OHSS-52268 — UPGRADE_IN_FLIGHT      ✓ tested (partial)
//   OHSS-55966 — UPGRADE_IN_FLIGHT      ✓ tested (partial)
//   OHSS-57359 — LIMITED_SUPPORT         ✓ tested
//   OHSS-52604 — not detectable (Day 2) ✓ tested (confirms gap)
//
// Day 2 gaps (8 tickets — require in-cluster checks):
//   OHSS-58345 — RBAC drift             ✓ tested (confirms gap)
//   OHSS-58084 — boot image skew        ✓ tested (confirms gap)
//   OHSS-58032 — WifConfig 404          ✓ tested (confirms gap)
//   OHSS-53365 — pull secret            ✓ tested (confirms gap)
//   OHSS-53041 — incompatible operator  ✓ tested (confirms gap)
//   OHSS-55964 — console plugins        ✓ tested (confirms gap)
//   OHSS-54695 — critical alerts        ✓ tested (confirms gap)
//   OHSS-54944 — degraded node          ✓ tested (confirms gap)
//
// Day 3 gaps (5 tickets — require cloud API or CLI validation):
//   OHSS-56299 — invalid autoscaler     ✓ tested (confirms gap)
//   OHSS-52380 — no egress              ✓ tested (confirms gap)
//   OHSS-54291 — IMDSv2 SCP             ✓ tested (confirms gap)
//   OHSS-57568 — cloud config           ✓ tested (confirms gap)
//   OHSS-52328 — GPU quota              ✓ tested (confirms gap)

func TestGateFiltering_STSOnlySkippedForNonSTS(t *testing.T) {
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("test-cluster-id").
			Name("test-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion("4.20.44").
			Version(cmv1.NewVersion().ID("openshift-v4.20.44").ChannelGroup("stable").
				AvailableUpgrades("4.21.40")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(false)).
			CreationTimestamp(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
	)

	stsGate := mustBuildGate("gate-sts", "STS permissions for 4.21", true,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.21.40"},
		rawAvailableUpgrades: []string{"4.21.40"},
		gates:                []*cmv1.VersionGate{stsGate},
		gateAgreements:       []*cmv1.VersionGateAgreement{},
	}

	result, err := CheckAll(mock, cluster, "4.21.40")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Eligible {
		t.Error("expected ELIGIBLE — STS-only gate should be skipped for non-STS cluster")
	}
	if len(result.Gates) != 0 {
		t.Errorf("expected 0 gates in result, got %d", len(result.Gates))
	}
}

func TestGateFiltering_WIFGateSkippedForNonGCP(t *testing.T) {
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("test-cluster-id").
			Name("test-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion("4.20.44").
			Version(cmv1.NewVersion().ID("openshift-v4.20.44").ChannelGroup("stable").
				AvailableUpgrades("4.21.40")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			CloudProvider(cmv1.NewCloudProvider().ID("aws")).
			Hypershift(cmv1.NewHypershift().Enabled(false)).
			CreationTimestamp(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
	)

	wifGate := mustBuildGateWithLabel("gate-wif", "WIF config update required", "api.openshift.com/gate-wif", false,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.21.40"},
		rawAvailableUpgrades: []string{"4.21.40"},
		gates:                []*cmv1.VersionGate{wifGate},
		gateAgreements:       []*cmv1.VersionGateAgreement{},
	}

	result, err := CheckAll(mock, cluster, "4.21.40")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Eligible {
		t.Error("expected ELIGIBLE — WIF gate should be skipped for non-GCP cluster")
	}
}

func TestGateFiltering_STSGateShownForSTSCluster(t *testing.T) {
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("test-cluster-id").
			Name("test-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion("4.20.44").
			Version(cmv1.NewVersion().ID("openshift-v4.20.44").ChannelGroup("stable").
				AvailableUpgrades("4.21.40")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			AWS(cmv1.NewAWS().STS(cmv1.NewSTS().RoleARN("arn:aws:iam::role/test"))).
			Hypershift(cmv1.NewHypershift().Enabled(false)).
			CreationTimestamp(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
	)

	stsGate := mustBuildGate("gate-sts", "STS permissions for 4.21", true,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.21.40"},
		rawAvailableUpgrades: []string{"4.21.40"},
		gates:                []*cmv1.VersionGate{stsGate},
		gateAgreements:       []*cmv1.VersionGateAgreement{},
	}

	result, err := CheckAll(mock, cluster, "4.21.40")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Eligible {
		t.Error("expected NOT eligible — STS gate should block STS cluster")
	}
	assertFinding(t, result, ReasonGateUnacked, SeverityBlocker)
}

// Z-stream upgrade should skip gate evaluation entirely.
func TestGateFiltering_ZStreamSkipsGates(t *testing.T) {
	cluster := mustBuildCluster(
		cmv1.NewCluster().
			ID("test-cluster-id").
			Name("test-cluster").
			State(cmv1.ClusterStateReady).
			OpenshiftVersion("4.17.33").
			Version(cmv1.NewVersion().ID("openshift-v4.17.33").ChannelGroup("stable")).
			Region(cmv1.NewCloudRegion().ID("us-east-1")).
			Hypershift(cmv1.NewHypershift().Enabled(true)).
			CreationTimestamp(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)),
	)

	gate := mustBuildGate("gate-417-zstream", "Should not appear for z-stream", false,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	mock := &mockOCMClient{
		availableUpgrades:    []string{"4.17.34"},
		rawAvailableUpgrades: []string{"4.17.34"},
		gates:                []*cmv1.VersionGate{gate},
		gateAgreements:       []*cmv1.VersionGateAgreement{},
	}

	result, err := CheckAll(mock, cluster, "4.17.34")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBlockers(t, result)
	if len(result.Gates) != 0 {
		t.Errorf("expected no gates for z-stream upgrade, got %d", len(result.Gates))
	}
}
