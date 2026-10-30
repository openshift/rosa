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
	"fmt"
	"regexp"
	"strings"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/ocm"
)

var validVersion = regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)

// CheckAll runs the Day-1 OCM-side eligibility checks and returns a structured result.
// It does not require backplane or in-cluster access.
func CheckAll(ocmClient OCMClient, cluster *cmv1.Cluster, targetVersion string) (*Result, error) {
	if targetVersion != "" && !validVersion.MatchString(targetVersion) {
		return nil, fmt.Errorf("invalid target version format %q: expected X.Y or X.Y.Z", targetVersion)
	}

	channelGroup := ""
	if version := cluster.Version(); version != nil {
		channelGroup = version.ChannelGroup()
	}

	result := &Result{
		Cluster: ClusterInfo{
			ID:           cluster.ID(),
			Name:         cluster.Name(),
			Version:      cluster.OpenshiftVersion(),
			ChannelGroup: channelGroup,
			Region:       cluster.Region().ID(),
			State:        string(cluster.State()),
			Hypershift:   cluster.Hypershift().Enabled(),
		},
		TargetVersion: targetVersion,
		Eligible:      true,
		Findings:      []Finding{},
		EvaluatedAt:   time.Now().UTC(),
	}

	if result.Cluster.Hypershift {
		result.Cluster.Product = "ROSA HCP"
	} else {
		result.Cluster.Product = "ROSA Classic"
	}

	if err := checkClusterState(result, cluster); err != nil {
		return result, err
	}

	if err := checkLimitedSupport(result, ocmClient, cluster); err != nil {
		return result, err
	}

	if err := checkActiveUpgrades(result, ocmClient, cluster); err != nil {
		return result, err
	}

	if err := checkAvailableTargets(result, ocmClient, cluster, targetVersion); err != nil {
		return result, err
	}

	// Gate checks run when a specific target is requested, or when targets are blocked
	// (not when the cluster is simply at the latest version in its channel)
	if targetVersion != "" || (len(result.AvailableUpgrades) == 0 && !result.atLatestVersion) {
		if err := checkVersionGates(result, ocmClient, cluster, targetVersion); err != nil {
			return result, err
		}
	}

	return result, nil
}

// checkClusterState verifies the cluster is in 'ready' state.
func checkClusterState(result *Result, cluster *cmv1.Cluster) error {
	if cluster.State() != cmv1.ClusterStateReady {
		result.AddFinding(Finding{
			Severity:   SeverityBlocker,
			ReasonCode: ReasonClusterNotReady,
			Owner:      OwnerService,
			Summary:    fmt.Sprintf("Cluster state is '%s', not 'ready'", cluster.State()),
			Detail:     "The cluster must be in 'ready' state before an upgrade can be scheduled.",
			Action:     "Wait for the cluster to reach 'ready' state, or contact support if it is stuck.",
		})
	}
	return nil
}

var versionEOLPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bend of life\b`),
	regexp.MustCompile(`(?i)\bend of supported life\b`),
	regexp.MustCompile(`(?i)\beol\b`),
	regexp.MustCompile(`(?i)\bunsupported version\b`),
	regexp.MustCompile(`(?i)version is no longer supported`),
}

// isVersionEOLReason returns true if the LS reason indicates version end-of-life.
func isVersionEOLReason(reason *cmv1.LimitedSupportReason) bool {
	summary := reason.Summary()
	for _, re := range versionEOLPatterns {
		if re.MatchString(summary) {
			return true
		}
	}
	return false
}

// checkLimitedSupport queries LS reasons and classifies them as version-EOL (warning) or other (blocker).
func checkLimitedSupport(result *Result, ocmClient OCMClient, cluster *cmv1.Cluster) error {
	reasons, err := ocmClient.GetLimitedSupportReasons(cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to check limited support reasons: %w", err)
	}

	if len(reasons) == 0 {
		return nil
	}

	var versionReasons, otherReasons []string
	hasVersionEOL := false
	hasOther := false

	for _, reason := range reasons {
		summary, _ := reason.GetSummary()
		line := fmt.Sprintf("  - %s", summary)
		if isVersionEOLReason(reason) {
			versionReasons = append(versionReasons, line)
			hasVersionEOL = true
		} else {
			otherReasons = append(otherReasons, line)
			hasOther = true
		}
	}

	if hasVersionEOL {
		result.AddFinding(Finding{
			Severity:   SeverityWarning,
			ReasonCode: ReasonLimitedSupportEOL,
			Owner:      OwnerCustomer,
			Summary:    fmt.Sprintf("Limited Support active due to version end-of-life (%d reason(s))", len(versionReasons)),
			Detail: fmt.Sprintf("The cluster is in Limited Support because its version has reached end of life. "+
				"Upgrading the cluster to a supported version will resolve this.\n\nReasons:\n%s",
				strings.Join(versionReasons, "\n")),
			Action: "Upgrade the cluster to a supported version. This Limited Support reason " +
				"will be automatically removed once the cluster is on a supported release.",
		})
	}

	if hasOther {
		result.AddFinding(Finding{
			Severity:   SeverityBlocker,
			ReasonCode: ReasonLimitedSupport,
			Owner:      OwnerCustomer,
			Summary:    fmt.Sprintf("Limited Support active — %d non-version reason(s)", len(otherReasons)),
			Detail: fmt.Sprintf("The cluster is in Limited Support for reasons unrelated to version end-of-life. "+
				"The console disables upgrade controls when this is active. The CLI may still accept "+
				"some operations, but they may fail during execution.\n\nReasons:\n%s",
				strings.Join(otherReasons, "\n")),
			Action: "Resolve the Limited Support reason(s) before scheduling an upgrade. " +
				"The console channel-edit and upgrade buttons will remain disabled until Limited Support is cleared.",
		})
	}

	return nil
}

// checkActiveUpgrades reports a blocker if an upgrade policy is already scheduled.
func checkActiveUpgrades(result *Result, ocmClient OCMClient, cluster *cmv1.Cluster) error {
	if cluster.Hypershift().Enabled() {
		return checkActiveUpgradesHypershift(result, ocmClient, cluster)
	}
	return checkActiveUpgradesClassic(result, ocmClient, cluster)
}

// checkActiveUpgradesClassic checks for an existing scheduled upgrade on a Classic cluster.
func checkActiveUpgradesClassic(result *Result, ocmClient OCMClient, cluster *cmv1.Cluster) error {
	policy, state, err := ocmClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to check active upgrade policies: %w", err)
	}
	if policy == nil {
		return nil
	}

	stateDesc := ""
	if state != nil {
		stateDesc = fmt.Sprintf(" (state: %s)", state.Value())
	}

	result.AddFinding(Finding{
		Severity:   SeverityBlocker,
		ReasonCode: ReasonUpgradeInFlight,
		Owner:      OwnerCustomer,
		Summary: fmt.Sprintf("Upgrade already scheduled to version %s on %s%s",
			policy.Version(), policy.NextRun().Format("2006-01-02 15:04 MST"), stateDesc),
		Detail: "Only one upgrade policy can be active at a time. Cancel the existing upgrade before scheduling a new one.",
		Action: fmt.Sprintf("rosa delete upgrade -c %s", cluster.ID()),
	})

	return nil
}

// checkActiveUpgradesHypershift checks for active upgrade policies on an HCP cluster.
func checkActiveUpgradesHypershift(result *Result, ocmClient OCMClient, cluster *cmv1.Cluster) error {
	policies, err := ocmClient.GetUpgradePolicies(cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to check active upgrade policies: %w", err)
	}

	for _, policy := range policies {
		if policy.NextRun().IsZero() {
			continue
		}
		result.AddFinding(Finding{
			Severity:   SeverityBlocker,
			ReasonCode: ReasonUpgradeInFlight,
			Owner:      OwnerCustomer,
			Summary: fmt.Sprintf("Upgrade already scheduled: type=%s version=%s on %s",
				policy.UpgradeType(), policy.Version(), policy.NextRun().Format("2006-01-02 15:04 MST")),
			Detail: "An upgrade policy is already active. Cancel or wait for it to complete before scheduling a new one.",
			Action: fmt.Sprintf("rosa delete upgrade -c %s", cluster.ID()),
		})
	}

	return nil
}

// checkAvailableTargets verifies upgrade targets exist and, if specified, that the target version is among them.
func checkAvailableTargets(result *Result, ocmClient OCMClient, cluster *cmv1.Cluster, targetVersion string) error {
	versionID := ocm.GetVersionID(cluster)
	product := ""
	if cluster.Hypershift().Enabled() {
		product = ocm.HcpProduct
	}
	available, err := ocmClient.GetAvailableUpgradesWithProduct(product, versionID)
	if err != nil {
		return fmt.Errorf("failed to get available upgrades: %w", err)
	}

	result.AvailableUpgrades = available

	if len(available) == 0 {
		rawUpgrades, rawErr := ocmClient.GetRawAvailableUpgrades(versionID)
		if rawErr != nil {
			return fmt.Errorf("failed to get raw available upgrades: %w", rawErr)
		}

		if len(rawUpgrades) == 0 {
			if targetVersion != "" {
				result.AddFinding(Finding{
					Severity:   SeverityBlocker,
					ReasonCode: ReasonTargetNotAvailable,
					Owner:      OwnerCustomer,
					Summary:    fmt.Sprintf("Requested version %s is not available — no upgrade targets exist", targetVersion),
					Detail: fmt.Sprintf("The cluster is on %s and no upgrade targets exist in the '%s' channel.",
						cluster.OpenshiftVersion(), cluster.Version().ChannelGroup()),
					Action: "Verify the target version exists for this channel group, or switch " +
						"to a different channel group (e.g., fast or candidate).",
				})
				return nil
			}
			result.atLatestVersion = true
			result.AddFinding(Finding{
				Severity:   SeverityOK,
				ReasonCode: ReasonAtLatestVersion,
				Summary: fmt.Sprintf("Already on the latest version (%s) in the '%s' channel",
					cluster.OpenshiftVersion(), cluster.Version().ChannelGroup()),
				Detail: "There are no newer versions available in this channel group. " +
					"This is expected if the cluster is up to date.",
				Action: "No action needed. To access newer versions, consider switching to " +
					"a different channel group (e.g., fast or candidate).",
			})
		} else {
			result.AddFinding(Finding{
				Severity:   SeverityBlocker,
				ReasonCode: ReasonNoTargetsAvailable,
				Owner:      OwnerCustomer,
				Summary: fmt.Sprintf(
					"No ROSA-enabled upgrade targets available (%d version(s) exist in channel but are not enabled)",
					len(rawUpgrades)),
				Detail: fmt.Sprintf("OCM lists %d version(s) in the upgrade graph (%s) but none are ROSA-enabled. "+
					"This is typically caused by unacknowledged version gates blocking the next minor, "+
					"or the target versions have not been published for ROSA yet.",
					len(rawUpgrades), strings.Join(rawUpgrades, ", ")),
				Action: "Check version gate agreements (see gates section below). If all gates are acknowledged, " +
					"the target versions may not yet be available for ROSA — try again later or contact support.",
			})
		}
		return nil
	}

	if targetVersion != "" {
		found := false
		for _, v := range available {
			if v == targetVersion {
				found = true
				break
			}
		}
		if !found {
			result.AddFinding(Finding{
				Severity:   SeverityBlocker,
				ReasonCode: ReasonTargetNotAvailable,
				Owner:      OwnerCustomer,
				Summary:    fmt.Sprintf("Requested version %s is not in the available targets", targetVersion),
				Detail:     fmt.Sprintf("Available versions: %s", strings.Join(available, ", ")),
				Action: "Choose one of the available target versions, or check whether " +
					"version gate agreements are blocking this target.",
			})
		}
	}

	return nil
}

// checkVersionGates evaluates version gate agreements for minor-version transitions.
func checkVersionGates(result *Result, ocmClient OCMClient, cluster *cmv1.Cluster, targetVersion string) error {
	targetMinor := targetVersion
	if targetMinor == "" {
		// Derive the next minor from current version
		parts := strings.SplitN(cluster.OpenshiftVersion(), ".", 3)
		if len(parts) >= 2 {
			major := parts[0]
			minor := parts[1]
			var minorInt int
			if _, err := fmt.Sscanf(minor, "%d", &minorInt); err == nil {
				targetMinor = fmt.Sprintf("%s.%d", major, minorInt+1)
			}
		}
	} else {
		// Extract major.minor from full version
		parts := strings.SplitN(targetVersion, ".", 3)
		if len(parts) >= 2 {
			targetMinor = parts[0] + "." + parts[1]
		}
	}

	if targetMinor == "" {
		return nil
	}

	// Gates guard minor-version transitions only; skip for z-stream upgrades
	curParts := strings.SplitN(cluster.OpenshiftVersion(), ".", 3)
	if len(curParts) >= 2 && targetMinor == curParts[0]+"."+curParts[1] {
		return nil
	}

	allGates, err := ocmClient.ListAllOcpGates(targetMinor)
	if err != nil {
		return fmt.Errorf("failed to list version gates for %s: %w", targetMinor, err)
	}

	if len(allGates) == 0 {
		return nil
	}

	// Get existing gate agreements for this cluster
	agreements, err := getGateAgreements(ocmClient, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to get gate agreements: %w", err)
	}

	ackedGateIDs := make(map[string]bool)
	for _, a := range agreements {
		if gate, ok := a.GetVersionGate(); ok {
			ackedGateIDs[gate.ID()] = true
		}
	}

	clusterCreation := cluster.CreationTimestamp()

	_, isSTS := cluster.AWS().STS().GetRoleARN()
	isGCP := cluster.CloudProvider() != nil && cluster.CloudProvider().ID() == "gcp"

	var unackedGates []GateDetail
	for _, gate := range allGates {
		if gate.STSOnly() && !isSTS {
			continue
		}
		if strings.Contains(gate.Label(), "gate-wif") && !isGCP {
			continue
		}

		gd := GateDetail{
			ID:               gate.ID(),
			Description:      gate.Description(),
			WarningMessage:   gate.WarningMessage(),
			DocumentationURL: gate.DocumentationURL(),
			STSOnly:          gate.STSOnly(),
		}

		if ackedGateIDs[gate.ID()] {
			gd.Acknowledged = true
			result.Gates = append(result.Gates, gd)
			continue
		}

		// Implicit acknowledgement: clusters created before a gate was introduced
		// are considered to have satisfied that gate without an explicit agreement.
		gateCreation := gate.CreationTimestamp()
		if !gateCreation.IsZero() && !clusterCreation.IsZero() && clusterCreation.Before(gateCreation) {
			gd.Acknowledged = true
			result.Gates = append(result.Gates, gd)
			continue
		}

		gd.Acknowledged = false
		unackedGates = append(unackedGates, gd)
		result.Gates = append(result.Gates, gd)
	}

	if len(unackedGates) > 0 {
		var gateLines []string
		for _, g := range unackedGates {
			line := fmt.Sprintf("  - %s", g.Description)
			if g.STSOnly {
				line += " (STS-only)"
			}
			if g.DocumentationURL != "" {
				line += fmt.Sprintf("\n    Docs: %s", g.DocumentationURL)
			}
			gateLines = append(gateLines, line)
		}

		result.AddFinding(Finding{
			Severity:   SeverityBlocker,
			ReasonCode: ReasonGateUnacked,
			Owner:      OwnerCustomer,
			Summary:    fmt.Sprintf("Version gates for %s not acknowledged — %d gate(s)", targetMinor, len(unackedGates)),
			Detail: fmt.Sprintf(
				"The following gate(s) must be acknowledged before the target is visible:\n\n%s",
				strings.Join(gateLines, "\n")),
			Action: fmt.Sprintf(
				"rosa upgrade cluster -c %s --version <target> "+
					"(the CLI will prompt for gate acknowledgements during scheduling)",
				cluster.ID()),
		})
	}

	return nil
}

func getGateAgreements(ocmClient OCMClient, clusterID string) ([]*cmv1.VersionGateAgreement, error) {
	return ocmClient.ListGateAgreements(clusterID)
}
