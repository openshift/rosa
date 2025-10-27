/*
Copyright (c) 2020 Red Hat, Inc.

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
	"strconv"
	"strings"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	nodePool string
}

var Cmd = &cobra.Command{
	Use:     "upgrades",
	Aliases: []string{"upgrade"},
	Short:   "List available cluster upgrades",
	Long:    "List available and scheduled cluster version upgrades",
	Run:     run,
	Args:    machinepool.NewMachinepoolArgsFunction(true),
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
	output.AddFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, _ *cobra.Command) error {
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()
	isNodePool := args.nodePool != ""
	isHypershift := ocm.IsHyperShiftCluster(cluster)

	if cluster.State() != cmv1.ClusterStateReady &&
		cluster.State() != cmv1.ClusterStateHibernating {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}

	if isNodePool && !ocm.IsHyperShiftCluster(cluster) {
		return fmt.Errorf("The '--machinepool' option is only supported for Hosted Control Planes")
	}

	var scheduledUpgrade *cmv1.UpgradePolicy
	var nodePoolScheduledUpgrade *cmv1.NodePoolUpgradePolicy
	var nodePool *cmv1.NodePool
	var upgradeState *cmv1.UpgradePolicyState
	var controlPlaneScheduledUpgrade *cmv1.ControlPlaneUpgradePolicy
	var availableUpgrades []string
	var err error

	if isNodePool {
		r.Reporter.Debugf("Loading available upgrades for node pool '%s' cluster '%s'", args.nodePool, clusterKey)
		nodePool, nodePoolScheduledUpgrade, err = r.OCMClient.GetHypershiftNodePoolUpgrade(cluster.ID(), clusterKey,
			args.nodePool)
		if err != nil {
			return err
		}

		// Get available node pool upgrades
		availableUpgrades = ocm.GetNodePoolAvailableUpgrades(nodePool)
		if len(availableUpgrades) == 0 {
			r.Reporter.Infof("There are no available upgrades for machine pool '%s'", args.nodePool)
			return nil
		}
	} else {
		// Control plane or cluster updates
		r.Reporter.Debugf("Loading available upgrades for cluster '%s'", clusterKey)
		availableUpgrades = ocm.GetAvailableUpgradesByCluster(cluster)

		if len(availableUpgrades) == 0 {
			r.Reporter.Infof("There are no available upgrades for cluster '%s'", clusterKey)
			return nil
		}
	}

	if output.HasFlag() {
		err := output.Print(availableUpgrades)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	latestRev := latestInCurrentMinor(ocm.GetVersionID(cluster), availableUpgrades)

	r.Reporter.Debugf("Loading scheduled upgrades for cluster '%s'", clusterKey)
	if !isHypershift {
		scheduledUpgrade, upgradeState, err = r.OCMClient.GetScheduledUpgrade(cluster.ID())
		if err != nil {
			return fmt.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
		}
	} else {
		if !isNodePool {
			controlPlaneScheduledUpgrade, err = r.OCMClient.GetControlPlaneScheduledUpgrade(cluster.ID())
			if err != nil {
				return fmt.Errorf("Failed to get scheduled control plane upgrades for cluster '%s': %v", clusterKey, err)
			}
		}
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "VERSION\tNOTES\n")
	for i, availableUpgrade := range availableUpgrades {
		notes := make([]string, 0)
		if i == 0 || availableUpgrade == latestRev {
			notes = append(notes, "recommended")
		}
		if !isHypershift {
			upgradeNotes := formatScheduledUpgrade(availableUpgrade, scheduledUpgrade, upgradeState)
			if len(upgradeNotes) != 0 {
				notes = append(notes, upgradeNotes)
			}
		} else {
			if isNodePool {
				upgradeNotes := formatScheduledUpgradeHypershift(availableUpgrade, nodePoolScheduledUpgrade)
				if len(upgradeNotes) != 0 {
					notes = append(notes, upgradeNotes)
				}
			} else {
				upgradeNotes := formatScheduledUpgradeHypershift(availableUpgrade, controlPlaneScheduledUpgrade)
				if len(upgradeNotes) != 0 {
					notes = append(notes, upgradeNotes)
				}
			}
		}
		fmt.Fprintf(writer, "%s\t%s\n", availableUpgrade, strings.Join(notes, " - "))
	}
	writer.Flush()
	return nil
}

func formatScheduledUpgrade(availableUpgrade string,
	scheduledUpgrade *cmv1.UpgradePolicy, upgradeState *cmv1.UpgradePolicyState) (notes string) {
	if availableUpgrade == scheduledUpgrade.Version() {
		notes = fmt.Sprintf("%s for %s", upgradeState.Value(),
			scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"))
	}
	return
}

func latestInCurrentMinor(current string, versions []string) string {
	latestVersion := current
	currentParts := strings.Split(current, ".")
	currentPart1, _ := strconv.Atoi(currentParts[1])
	currentPart2, _ := strconv.Atoi(currentParts[2])
	currentRev := currentPart2
	latestRev := currentRev
	for _, version := range versions {
		versionParts := strings.Split(version, ".")
		versionPart1, _ := strconv.Atoi(versionParts[1])
		versionPart2, _ := strconv.Atoi(versionParts[2])
		if currentPart1 != versionPart1 {
			continue
		}
		if versionPart2 > latestRev {
			latestRev = versionPart2
			latestVersion = version
		}
	}
	return latestVersion
}

func formatScheduledUpgradeHypershift(availableUpgrade string,
	scheduledUpgrade ocm.HypershiftUpgrader) (notes string) {
	if availableUpgrade == scheduledUpgrade.Version() {
		notes = fmt.Sprintf("%s for %s", scheduledUpgrade.State().Value(),
			scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"))
	}
	return
}
