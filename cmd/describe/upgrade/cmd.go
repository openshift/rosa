/*
Copyright (c) 2022 Red Hat, Inc.

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

package upgrade

import (
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"appliance", "upgrades"},
	Short:   "Show details of an upgrade",
	Long:    "Show details of an upgrade",
	Example: `  # Describe an upgrade-policy"
  rosa describe upgrade`,
	Run:    run,
	Hidden: false,
	Args:   cobra.NoArgs,
}

var args struct {
	nodePool string
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.nodePool,
		"machinepool",
		"",
		"Machine pool of the cluster to target",
	)

	confirm.AddFlag(flags)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime) error {
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}

	if args.nodePool != "" && !ocm.IsHyperShiftCluster(cluster) {
		return fmt.Errorf("The '--machinepool' option is only supported for Hosted Control Planes")
	}

	r.Reporter.Debugf("Loading upgrades for cluster id '%s'", cluster.ID())
	if ocm.IsHyperShiftCluster(cluster) {
		return describeHypershiftUpgrades(r, cluster.ID(), args.nodePool)
	} else {
		return describeClassicUpgrades(r, cluster.ID())
	}
}

func describeHypershiftUpgrades(r *rosa.Runtime, clusterID string, nodePoolID string) error {
	clusterKey := r.GetClusterKey()
	if args.nodePool == "" {
		upgrades, err := r.OCMClient.GetControlPlaneUpgradePolicies(clusterID)
		if err != nil {
			return fmt.Errorf("Failed to get upgrades for cluster '%s': %v", clusterKey, err)
		}
		if len(upgrades) < 1 {
			r.Reporter.Infof("No scheduled upgrades for cluster '%s'", clusterKey)
			return nil
		}

		for _, upgrade := range upgrades {
			fmt.Print(formatHypershiftUpgrade(upgrade))
		}
	} else {
		_, upgrades, err := r.OCMClient.GetHypershiftNodePoolUpgrades(clusterID, clusterKey, nodePoolID)
		if err != nil {
			return fmt.Errorf("Failed to get upgrades for machine pool '%s' in cluster '%s': %v", nodePoolID,
				clusterKey, err)
		}
		if upgrades == nil || len(upgrades) < 1 {
			r.Reporter.Infof("No scheduled upgrades for machine pool '%s' in cluster '%s'", nodePoolID, clusterKey)
			return nil
		}

		for _, upgrade := range upgrades {
			fmt.Print(formatHypershiftUpgrade(upgrade))
		}
	}
	return nil
}

// formatHypershiftUpgrade is a generic printer for Hypershift Upgrades types
func formatHypershiftUpgrade(upgrade ocm.HypershiftUpgrader) string {
	builder := make([]string, 0)
	builder = append(builder, fmt.Sprintf(`
%-35s%s
%-35s%s
%-35s%s
%-35s%s
%-35s%s
%-35s%s
`,
		"ID:", upgrade.ID(),
		"Cluster ID:", upgrade.ClusterID(),
		"Schedule Type:", upgrade.ScheduleType(),
		"Next Run:", upgrade.NextRun().Format("2006-01-02 15:04 MST"),
		"Upgrade State:", upgrade.State().Value(),
		"State Message:", upgrade.State().Description()))
	if upgrade.Schedule() != "" {
		builder = append(builder, fmt.Sprintf(`
%-35s%s
`, "Schedule At:", upgrade.Schedule()))
		builder = append(builder, fmt.Sprintf(`
%-35s%t
`, "Enable minor version upgrades:", upgrade.EnableMinorVersionUpgrades()))
	}
	if upgrade.Version() != "" {
		builder = append(builder, fmt.Sprintf(`
%-35s%s
`, "Version:", upgrade.Version()))
	}
	return strings.Join(builder, "")
}

// formatClassicUpgrade is a generic printer for classic Upgrades types
func formatClassicUpgrade(upgrade *cmv1.UpgradePolicy, upgradeState *cmv1.UpgradePolicyState) string {
	builder := make([]string, 0)
	builder = append(builder, fmt.Sprintf(`
%-35s%s
%-35s%s
%-35s%s
%-35s%s
`,
		"ID:", upgrade.ID(),
		"Cluster ID:", upgrade.ClusterID(),
		"Next Run:", upgrade.NextRun().Format("2006-01-02 15:04 MST"),
		"Upgrade State:", upgradeState.Value()))
	if upgrade.Schedule() != "" {
		builder = append(builder, fmt.Sprintf(`
%-35s%s
`, "Schedule At:", upgrade.Schedule()))
	}
	if upgrade.Version() != "" {
		builder = append(builder, fmt.Sprintf(`
%-35s%s
`, "Version:", upgrade.Version()))
	}
	return strings.Join(builder, "")
}

func describeClassicUpgrades(r *rosa.Runtime, clusterID string) error {
	upgrades, err := r.OCMClient.GetUpgradePolicies(clusterID)
	if err != nil {
		return fmt.Errorf("Failed to get upgrade with cluster id '%s': %v", clusterID, err)
	}
	_, upgradeState, err := r.OCMClient.GetScheduledUpgrade(clusterID)
	if err != nil {
		return fmt.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterID, err)
	}
	if len(upgrades) < 1 {
		r.Reporter.Infof("No scheduled upgrades for cluster id '%s'", clusterID)
		return nil
	}

	for _, upgrade := range upgrades {
		fmt.Print(formatClassicUpgrade(upgrade, upgradeState))
	}
	return nil
}
