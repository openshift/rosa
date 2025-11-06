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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/input"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	version                  string
	scheduleDate             string
	scheduleTime             string
	schedule                 string
	allowMinorVersionUpdates bool
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
	Run:  run,
	Args: machinepool.NewMachinepoolArgsFunction(false),
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.version,
		"version",
		"",
		"Version of OpenShift that the machine pool will be upgraded to",
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

	flags.StringVar(
		&args.schedule,
		"schedule",
		"",
		"cron expression in UTC which will be the time when an upgrade to the latest release will be "+
			"automatically scheduled and repeated at each occurrence. Mutually exclusive with --schedule-date and "+
			"--schedule-time. ",
	)

	flags.BoolVar(
		&args.allowMinorVersionUpdates,
		"allow-minor-version-updates",
		false,
		"When using automatic scheduling with --schedule parameter, if true it will also update to latest "+
			"minor release, e.g. 4.12.20 -> 4.13.2. By default only z-stream updates will be scheduled. ",
	)
	// Hidden for now as not supported yet
	flags.MarkHidden("allow-minor-version-updates")

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
	var err error
	currentUpgradeScheduling := ocm.UpgradeScheduling{
		Schedule:                 args.schedule,
		ScheduleDate:             args.scheduleDate,
		ScheduleTime:             args.scheduleTime,
		AllowMinorVersionUpdates: args.allowMinorVersionUpdates,
		AutomaticUpgrades:        args.schedule != "",
	}
	isVersionSet := cmd.Flags().Changed("version")

	// Check parameters preconditions
	if currentUpgradeScheduling.Schedule == "" && currentUpgradeScheduling.AllowMinorVersionUpdates {
		return fmt.Errorf("the '--allow-minor-version-upgrades' option needs to be used with --schedule")
	}

	if (currentUpgradeScheduling.ScheduleDate != "" || currentUpgradeScheduling.ScheduleTime != "") &&
		currentUpgradeScheduling.Schedule != "" {
		return fmt.Errorf("the '--schedule-date' and '--schedule-time' options are mutually exclusive with" +
			" '--schedule'")
	}

	if currentUpgradeScheduling.Schedule != "" && args.version != "" {
		return fmt.Errorf("the '--schedule' option is mutually exclusive with '--version'")
	}

	// Validate cluster state
	input.CheckIfHypershiftClusterOrExit(r, cluster)
	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
	}

	if !machinepool.MachinePoolKeyRE.MatchString(machinePoolID) {
		return fmt.Errorf("expected a valid identifier for the machine pool")
	}
	_, exists, err := r.OCMClient.GetNodePool(cluster.ID(), machinePoolID)
	if err != nil {
		return fmt.Errorf("failed to get machine pools for hosted cluster '%s': %v", clusterKey, err)
	}
	if !exists {
		return fmt.Errorf("machine pool '%s' does not exist for hosted cluster '%s'", machinePoolID, clusterKey)
	}

	// Enable interactive mode if needed
	// We need to specify either both date and time or nothing
	if (currentUpgradeScheduling.ScheduleDate != "" && currentUpgradeScheduling.ScheduleTime == "") ||
		currentUpgradeScheduling.ScheduleDate == "" && currentUpgradeScheduling.ScheduleTime != "" {
		interactive.Enable()
	}

	if interactive.Enabled() {
		r.Reporter.Infof("Interactive mode enabled.\n" +
			"Any optional fields can be left empty and a default will be selected.")
	}

	// Get upgrade type
	if interactive.Enabled() {
		currentUpgradeScheduling.AutomaticUpgrades, err = interactive.GetBool(interactive.Input{
			Question: "Enable automatic upgrades",
			Help: "Whether the upgrade is automatic or manual.\n" +
				"With automatic upgrades, user defines the schedule of the upgrade with a cron expression.\n" +
				"The target version will always be the latest available version at the moment of the schedule\n" +
				"occurrence. In the manual upgrades, user defines the schedule and a target version",
			Default:  currentUpgradeScheduling.AutomaticUpgrades,
			Required: true,
		})
		if err != nil {
			return fmt.Errorf("expected an upgrade type: %s", err)
		}

		if currentUpgradeScheduling.AutomaticUpgrades {
			currentUpgradeScheduling.AllowMinorVersionUpdates, err = interactive.GetBool(interactive.Input{
				Question: "Allow minor upgrades",
				Help:     cmd.Flags().Lookup("allow-minor-version-updates").Usage,
				Default:  currentUpgradeScheduling.AllowMinorVersionUpdates,
				Required: false,
			})
			if err != nil {
				return fmt.Errorf("expected an choice on the versions to target: %s", err)
			}
		}
	}

	// Check if any upgrade already exists
	nodePool, exists, err := checkExistingUpgrades(r, clusterKey, cluster, machinePoolID)
	if err != nil {
		return err
	}
	if exists {
		r.Reporter.Infof("An upgrade already exists for machine pool '%s' in cluster '%s'", machinePoolID, clusterKey)
		return nil
	}

	// Build the upgrade policy if it is a manual or automatic upgrade
	var upgradePolicy *cmv1.NodePoolUpgradePolicy
	if currentUpgradeScheduling.AutomaticUpgrades {
		upgradePolicy, err = buildAutomaticUpgradePolicy(r, cmd, currentUpgradeScheduling, clusterKey, nodePool)
	} else {
		upgradePolicy, err = buildManualUpgradePolicy(r, cmd, currentUpgradeScheduling, clusterKey,
			cluster, nodePool, isVersionSet, args.version)
	}
	if err != nil {
		return err
	}
	if upgradePolicy == nil {
		// Nothing to do
		return nil
	}

	// Schedule the built upgrade policy
	r.Reporter.Debugf("Scheduling the upgrade policy")
	_, err = r.OCMClient.ScheduleNodePoolUpgrade(cluster.ID(), machinePoolID, upgradePolicy)
	if err != nil {
		return errors.Wrapf(err, "Failed to schedule upgrade for machine pool '%s' in cluster '%s'",
			machinePoolID, clusterKey)
	}

	r.Reporter.Infof("Upgrade successfully scheduled for the machine pool '%s' on cluster '%s'", machinePoolID,
		clusterKey)
	return nil
}

func buildManualUpgradePolicy(r *rosa.Runtime, cmd *cobra.Command, currentUpgradeScheduling ocm.UpgradeScheduling,
	clusterKey string, cluster *cmv1.Cluster, nodePool *cmv1.NodePool, isVersionSet bool,
	inputVersion string) (*cmv1.NodePoolUpgradePolicy, error) {
	var err error
	// Build schedule
	r.Reporter.Debugf("Building the upgrade schedule")
	nextRun, err := interactive.BuildManualUpgradeSchedule(cmd, currentUpgradeScheduling.ScheduleDate,
		currentUpgradeScheduling.ScheduleTime)
	if err != nil {
		return nil, err
	}
	currentUpgradeScheduling.NextRun = nextRun

	// check version
	version := inputVersion
	if isVersionSet || version == "" || interactive.Enabled() {
		version, err = ComputeNodePoolVersion(r, cmd, cluster, nodePool, version)
		if err != nil {
			return nil, err
		}
		if version == "" {
			r.Reporter.Infof("No available upgrade for the machine pool '%s' on cluster '%s'", nodePool.ID(),
				clusterKey)
			return nil, nil
		}
	}
	version = ocm.GetRawVersionId(version)

	// build the upgrade policy
	r.Reporter.Debugf("Building the upgrade policy")
	var upgradePolicy *cmv1.NodePoolUpgradePolicy
	upgradePolicy, err = r.OCMClient.BuildNodeUpgradePolicy(version, nodePool.ID(), currentUpgradeScheduling)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to build manual schedule upgrade for machine pool '%s' in cluster '%s'",
			nodePool.ID(), clusterKey)
	}

	// Ask for confirmation
	if r.Reporter.IsTerminal() && !confirm.Confirm("upgrade machine pool '%s' to version '%s'", nodePool.ID(),
		version) {
		return nil, nil
	}

	return upgradePolicy, nil
}

func buildAutomaticUpgradePolicy(r *rosa.Runtime, cmd *cobra.Command, currentUpgradeScheduling ocm.UpgradeScheduling,
	clusterKey string, nodePool *cmv1.NodePool) (*cmv1.NodePoolUpgradePolicy, error) {
	var err error
	// Build schedule
	schedule, err := interactive.BuildAutomaticUpgradeSchedule(cmd, currentUpgradeScheduling.Schedule)
	if err != nil {
		return nil, err
	}
	currentUpgradeScheduling.Schedule = schedule

	// build the upgrade policy
	r.Reporter.Debugf("Building the upgrade policy")
	var upgradePolicy *cmv1.NodePoolUpgradePolicy

	upgradePolicy, err = r.OCMClient.BuildNodeUpgradePolicy("", nodePool.ID(), currentUpgradeScheduling)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to build automatic schedule upgrade for machine pool %s in cluster '%s'",
			nodePool.ID(), clusterKey)
	}

	// Ask for confirmation
	if r.Reporter.IsTerminal() && !confirm.Confirm("schedule automatic upgrades for machine pool '%s' at '%s'",
		nodePool.ID(), currentUpgradeScheduling.Schedule) {
		return nil, nil
	}

	return upgradePolicy, nil
}

func checkExistingUpgrades(r *rosa.Runtime, clusterKey string, cluster *cmv1.Cluster,
	machinePoolID string) (*cmv1.NodePool, bool, error) {
	r.Reporter.Debugf("Checking existing upgrades for hosted cluster '%s'", clusterKey)
	return checkNodePoolExistingScheduledUpgrade(r, cluster, clusterKey, machinePoolID)
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
	var err error
	channelGroup := cluster.Version().ChannelGroup()
	filteredVersionList := ocm.GetNodePoolAvailableUpgrades(nodePool)
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
			Required: true,
		})
		if err != nil {
			return "", fmt.Errorf("expected a valid machine pool version from interactive prompt: %s", err)
		}
	}
	// This is called in HyperShift, but we don't want to exclude version which are HCP disabled for node pools
	// so we pass the relative parameter as false
	version, err = r.OCMClient.ValidateVersion(version, filteredVersionList, channelGroup, true, false)
	if err != nil {
		return "", fmt.Errorf("expected a valid machine pool version: %s", err)
	}
	return version, nil
}
