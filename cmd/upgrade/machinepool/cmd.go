/*
Copyright (c) 2023 Red Hat, Inc.

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

package machinepool

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/helper/versions"
	"github.com/openshift/rosa/pkg/input"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

var args struct {
	version      string
	scheduleDate string
	scheduleTime string
}

var Cmd = &cobra.Command{
	Use:     "machinepool",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Upgrade machinepool",
	Long:    "Upgrade machinepool to a new available version. This is supported only for Hosted Control Planes.",
	Example: `  # Interactively schedule an upgrade on the cluster named "mycluster"" for a machinepool named "np1"
  rosa upgrade machinepool np1 --cluster=mycluster --interactive

  # Schedule a machinepool upgrade within the hour
  rosa upgrade machinepool np1 -c mycluster --version 4.12.20`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"expected exactly one command line parameter containing the id of the machine pool",
			)
		}
		return nil
	},
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.version,
		"version",
		"",
		"Version of OpenShift that the cluster will be upgraded to",
	)

	flags.StringVar(
		&args.scheduleDate,
		"schedule-date",
		"",
		"Next date the upgrade should run at the specified UTC time. Format should be 'yyyy-mm-dd'",
	)

	flags.StringVar(
		&args.scheduleTime,
		"schedule-time",
		"",
		"Next UTC time that the upgrade should run on the specified date. Format should be 'HH:mm'",
	)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd, argv)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()
	machinePoolID := argv[0]
	scheduleDate := args.scheduleDate
	scheduleTime := args.scheduleTime
	isVersionSet := cmd.Flags().Changed("version")

	// Validate cluster state
	input.CheckIfHypershiftClusterOrExit(r, cluster)
	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}
	if scheduleDate == "" || scheduleTime == "" {
		interactive.Enable()
	}

	// check existing upgrades
	r.Reporter.Debugf("Checking existing upgrades for hosted cluster '%s'", clusterKey)
	nodePool, exists, err := checkNodePoolExistingScheduledUpgrade(r, cluster, clusterKey, machinePoolID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// check version
	version := args.version
	if isVersionSet || interactive.Enabled() {
		version, err = ComputeNodePoolVersion(r, cmd, cluster, nodePool, version)
		if err != nil {
			return err
		}
		if version == "" {
			r.Reporter.Infof("No upgrade available for the machine pool '%s' on cluster '%s'", machinePoolID,
				clusterKey)
			return nil
		}
	}
	version = ocm.GetRawVersionId(version)

	// build the upgrade policy
	r.Reporter.Debugf("Building and scheduling the upgrade policy")
	var upgradePolicy *cmv1.NodePoolUpgradePolicy
	upgradePolicy, err = buildPolicy(r, cmd, version, machinePoolID, scheduleDate, scheduleTime)
	if err != nil {
		return fmt.Errorf("Failed to build schedule upgrade for machine pool %s in cluster '%s': %v",
			machinePoolID, clusterKey, err)
	}

	// Ask for confirmation
	if r.Reporter.IsTerminal() && !confirm.Confirm("upgrade machine pool '%s' to version '%s'", machinePoolID,
		version) {
		return nil
	}

	// schedule the upgrade policy
	err = r.OCMClient.ScheduleNodePoolUpgrade(cluster.ID(), machinePoolID, upgradePolicy)
	if err != nil {
		return fmt.Errorf("Failed to schedule upgrade for machine pool %s in cluster '%s': %v",
			machinePoolID, clusterKey, err)
	}

	r.Reporter.Infof("Upgrade successfully scheduled for the machine pool '%s' on cluster '%s'", machinePoolID,
		clusterKey)
	return nil
}

func checkNodePoolExistingScheduledUpgrade(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string,
	nodePoolId string) (*cmv1.NodePool, bool, error) {
	nodePool, scheduledUpgrade, err := r.OCMClient.GetHypershiftNodePoolUpgrade(cluster.ID(), clusterKey, nodePoolId)
	if err != nil {
		return nil, false, err
	}
	if scheduledUpgrade != nil {
		r.Reporter.Warnf("There is already a %s upgrade to version %s on %s",
			scheduledUpgrade.State().Value(),
			scheduledUpgrade.Version(),
			scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
		)
		return nil, true, nil
	}
	return nodePool, false, nil
}

func ComputeNodePoolVersion(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster,
	nodePool *cmv1.NodePool, version string) (string, error) {
	channelGroup := cluster.Version().ChannelGroup()
	filteredVersionList, err := GetAvailableVersion(r, cluster, nodePool)
	if err != nil {
		return "", err
	}
	// No updates available
	if len(filteredVersionList) == 0 {
		return "", nil
	}
	if version == "" {
		// Use as default latest version if we don't specify anything
		version = filteredVersionList[0]
	}

	if interactive.Enabled() {
		version, err = interactive.GetOption(interactive.Input{
			Question: "Machine pool version",
			Help:     cmd.Flags().Lookup("version").Usage,
			Options:  filteredVersionList,
			Default:  version,
		})
		if err != nil {
			return "", fmt.Errorf("Expected a valid machine pool version from interactive prompt: %s", err)
		}
	}
	// This is called in HyperShift, but we don't want to exclude version which are HCP disabled for node pools
	// so we pass the relative parameter as false
	version, err = r.OCMClient.ValidateVersion(version, filteredVersionList, channelGroup, true, false)
	if err != nil {
		return "", fmt.Errorf("Expected a valid machine pool version: %s", err)
	}
	return version, nil
}

func GetAvailableVersion(r *rosa.Runtime, cluster *cmv1.Cluster, nodePool *cmv1.NodePool) ([]string, error) {
	clusterVersion := cluster.Version().RawID()
	nodePoolVersion := ocm.GetRawVersionId(nodePool.Version().ID())
	// This is called in HyperShift, but we don't want to exclude version which are HCP disabled for node pools
	// so we pass the relative parameter as false
	versionList, err := versions.GetVersionList(r, cluster.Version().ChannelGroup(), true, false, false)
	if err != nil {
		return nil, err
	}

	// Filter the available list of versions for a hosted machine pool
	filteredVersionList := versions.GetFilteredVersionListForUpdate(versionList, nodePoolVersion, clusterVersion)
	if err != nil {
		return nil, err
	}
	return filteredVersionList, nil
}

func buildPolicy(r *rosa.Runtime, cmd *cobra.Command,
	version string, machinePoolID string, scheduleDate string, scheduleTime string) (*cmv1.NodePoolUpgradePolicy, error) {
	nextRun, err := interactive.BuildManualUpgradeSchedule(cmd, scheduleDate, scheduleTime)
	if err != nil {
		return nil, err
	}
	return r.OCMClient.BuildNodeUpgradePolicy(version, machinePoolID, nextRun)
}
