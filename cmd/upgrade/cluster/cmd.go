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

package cluster

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/upgrade/roles"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	version              string
	scheduleDate         string
	scheduleTime         string
	nodeDrainGracePeriod string
	controlPlane         bool
}

var nodeDrainOptions = []string{
	"15 minutes",
	"30 minutes",
	"45 minutes",
	"1 hour",
	"2 hours",
	"4 hours",
	"8 hours",
}

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Upgrade cluster",
	Long:  "Upgrade cluster to a new available version",
	Example: `  # Interactively schedule an upgrade on the cluster named "mycluster"
  rosa upgrade cluster --cluster=mycluster --interactive

  # Schedule a cluster upgrade within the hour
  rosa upgrade cluster -c mycluster --version 4.5.20`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)
	aws.AddModeFlag(Cmd)

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

	flags.StringVar(
		&args.nodeDrainGracePeriod,
		"node-drain-grace-period",
		"1 hour",
		fmt.Sprintf("You may set a grace period for how long Pod Disruption Budget-protected workloads will be "+
			"respected during upgrades.\nAfter this grace period, any workloads protected by Pod Disruption "+
			"Budgets that have not been successfully drained from a node will be forcibly evicted.\nValid "+
			"options are ['%s']", strings.Join(nodeDrainOptions, "','")),
	)

	flags.BoolVar(
		&args.controlPlane,
		"control-plane",
		false,
		"For Hosted Control Plane, whether the upgrade should cover only the control plane",
	)

	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()
	scheduleDate := args.scheduleDate
	scheduleTime := args.scheduleTime
	isHypershift := cluster.Hypershift().Enabled()

	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	_, isSTS := cluster.AWS().STS().GetRoleARN()
	if !isSTS && mode != "" {
		r.Reporter.Errorf("The 'mode' option is only supported for STS clusters")
		os.Exit(1)
	}

	if args.controlPlane && !isHypershift {
		r.Reporter.Errorf("The '--control-plane' option is only supported for Hosted Control Planes")
		os.Exit(1)
	}

	if !args.controlPlane && isHypershift {
		r.Reporter.Errorf("The '--control-plane' option is currently mandatory for Hosted Control Planes")
		os.Exit(1)
	}

	checkExistingScheduledUpgrade(r, cluster, clusterKey)

	availableUpgrades, version := buildVersion(r, cmd, cluster, args.version)
	err = r.OCMClient.CheckUpgradeClusterVersion(availableUpgrades, version, cluster)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	if scheduleDate == "" || scheduleTime == "" {
		interactive.Enable()
	}

	if isSTS && mode == "" {
		mode = setMode(r, cmd)
	}

	// if cluster is sts validate roles are compatible with upgrade version
	if isSTS {
		checkSTSRolesCompatibility(r, cluster, mode, version, clusterKey)
	}

	version, err = ocm.CheckAndParseVersion(availableUpgrades, version)
	if err != nil {
		r.Reporter.Errorf("Error parsing version to upgrade to")
		os.Exit(1)
	}

	// Compute drain grace period config
	clusterSpec := buildNodeDrainGracePeriod(r, cmd, cluster)

	if !confirm.Confirm("upgrade cluster to version '%s'", version) {
		os.Exit(0)
	}

	if isHypershift {
		err = createUpgradePolicyHypershift(r, cmd, clusterKey, cluster, version, scheduleDate, scheduleTime)
	} else {
		err = createUpgradePolicyClassic(r, cmd, clusterKey, cluster, version, scheduleDate, scheduleTime)
	}
	if err != nil {
		r.Reporter.Errorf("Failed to schedule upgrade for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Update cluster with grace period configuration
	err = r.OCMClient.UpdateCluster(cluster.ID(), r.Creator, clusterSpec)
	if err != nil {
		r.Reporter.Errorf("Failed to update cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Infof("Upgrade successfully scheduled for cluster '%s'", clusterKey)
}

func createUpgradePolicyHypershift(r *rosa.Runtime, cmd *cobra.Command, clusterKey string,
	cluster *cmv1.Cluster, version string, scheduleDate string, scheduleTime string) error {
	upgradePolicyBuilder := cmv1.NewControlPlaneUpgradePolicy().ScheduleType("manual").
		UpgradeType("ControlPlane").Version(version)
	// TODO: remove forceNow when we support scheduling
	nextRun := buildUpgradeSchedule(r, cmd, scheduleDate, scheduleTime, true)
	upgradePolicyBuilder = upgradePolicyBuilder.NextRun(nextRun)
	upgradePolicy, err := upgradePolicyBuilder.Build()
	if err != nil {
		return err
	}
	err = checkAndAckMissingAgreementsHypershift(r, cluster, upgradePolicy, clusterKey)
	if err != nil {
		return err
	}

	err = r.OCMClient.ScheduleHypershiftControlPlaneUpgrade(cluster.ID(), upgradePolicy)
	if err != nil {
		return err
	}
	return nil
}

func createUpgradePolicyClassic(r *rosa.Runtime, cmd *cobra.Command, clusterKey string,
	cluster *cmv1.Cluster, version string, scheduleDate string, scheduleTime string) error {
	upgradePolicyBuilder := cmv1.NewUpgradePolicy().
		ScheduleType("manual").
		Version(version)
	upgradePolicy, err := upgradePolicyBuilder.Build()
	if err != nil {
		return err
	}
	err = checkAndAckMissingAgreementsClassic(r, cluster, upgradePolicy, clusterKey)
	if err != nil {
		return err
	}

	nextRun := buildUpgradeSchedule(r, cmd, scheduleDate, scheduleTime, false)
	upgradePolicyBuilder = upgradePolicyBuilder.NextRun(nextRun)
	upgradePolicy, err = upgradePolicyBuilder.Build()
	if err != nil {
		return err
	}
	err = r.OCMClient.ScheduleUpgrade(cluster.ID(), upgradePolicy)
	if err != nil {
		return err
	}
	return nil
}

func buildVersion(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster, version string) ([]string, string) {
	availableUpgrades, err := r.OCMClient.GetAvailableUpgrades(ocm.GetVersionID(cluster))
	if err != nil {
		r.Reporter.Errorf("Failed to find available upgrades: %v", err)
		os.Exit(1)
	}
	if len(availableUpgrades) == 0 {
		r.Reporter.Warnf("There are no available upgrades")
		os.Exit(0)
	}
	if version == "" || interactive.Enabled() {
		if version == "" {
			version = availableUpgrades[0]
		}
		version, err = interactive.GetOption(interactive.Input{
			Question: "Version",
			Help:     cmd.Flags().Lookup("version").Usage,
			Options:  availableUpgrades,
			Default:  version,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid version to upgrade to: %s", err)
			os.Exit(1)
		}
	}
	return availableUpgrades, version
}

func setMode(r *rosa.Runtime, cmd *cobra.Command) string {
	mode, err := interactive.GetOption(interactive.Input{
		Question: "IAM Roles/Policies upgrade mode",
		Help:     cmd.Flags().Lookup("mode").Usage,
		Default:  aws.ModeAuto,
		Options:  aws.Modes,
		Required: true,
	})
	if err != nil {
		r.Reporter.Errorf("Expected a valid role upgrade mode: %v", err)
		os.Exit(1)
	}
	aws.SetModeKey(mode)
	return mode
}

func checkExistingScheduledUpgrade(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string) {
	scheduledUpgrade, upgradeState, err := r.OCMClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	if scheduledUpgrade != nil {
		r.Reporter.Warnf("There is already a %s upgrade to version %s on %s",
			upgradeState.Value(),
			scheduledUpgrade.Version(),
			scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
		)
		os.Exit(0)
	}
}

func checkSTSRolesCompatibility(r *rosa.Runtime, cluster *cmv1.Cluster, mode string,
	version string, clusterKey string) {
	r.Reporter.Infof("Ensuring account and operator role policies for cluster '%s'"+
		" are compatible with upgrade.", cluster.ID())
	err := roles.Cmd.RunE(roles.Cmd, []string{mode, cluster.ID(), version, cluster.Version().ChannelGroup()})
	if err != nil {
		rolesStr := fmt.Sprintf("rosa upgrade roles -c %s --cluster-version=%s --mode=%s", clusterKey, version, mode)
		upgradeClusterStr := fmt.Sprintf("rosa upgrade cluster -c %s", clusterKey)

		r.Reporter.Infof("Account/Operator Role policies are not valid with upgrade version %s. "+
			"Run the following command(s) to upgrade the roles and run the upgrade command again:\n\n"+
			"\t%s\n"+
			"\t%s\n", version, rolesStr, upgradeClusterStr)
		os.Exit(0)
	}
	r.Reporter.Infof("Account and operator roles for cluster '%s' are compatible with upgrade", clusterKey)
}

func buildUpgradeSchedule(r *rosa.Runtime, cmd *cobra.Command, scheduleDate string, scheduleTime string,
	forceNow bool) time.Time {
	// Set the default next run within the next 10 minutes
	now := time.Now().UTC().Add(time.Minute * 10)
	if forceNow || scheduleDate == "" {
		scheduleDate = now.Format("2006-01-02")
	}
	if forceNow || scheduleTime == "" {
		scheduleTime = now.Format("15:04")
	}

	if !forceNow && interactive.Enabled() {
		// If datetimes are set, use them in the interactive form, otherwise fallback to 'now'
		scheduleParsed, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", scheduleDate, scheduleTime))
		if err != nil {
			r.Reporter.Errorf("Schedule date should use the format 'yyyy-mm-dd'\n" +
				"   Schedule time should use the format 'HH:mm'")
			os.Exit(1)
		}
		if scheduleParsed.IsZero() {
			scheduleParsed = now
		}
		scheduleDate = scheduleParsed.Format("2006-01-02")
		scheduleTime = scheduleParsed.Format("15:04")

		scheduleDate, err = interactive.GetString(interactive.Input{
			Question: "Please input desired date in format yyyy-mm-dd",
			Help:     cmd.Flags().Lookup("schedule-date").Usage,
			Default:  scheduleDate,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid date: %s", err)
			os.Exit(1)
		}
		_, err = time.Parse("2006-01-02", scheduleDate)
		if err != nil {
			r.Reporter.Errorf("Date format '%s' invalid", scheduleDate)
			os.Exit(1)
		}

		scheduleTime, err = interactive.GetString(interactive.Input{
			Question: "Please input desired UTC time in format HH:mm",
			Help:     cmd.Flags().Lookup("schedule-time").Usage,
			Default:  scheduleTime,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid time: %s", err)
			os.Exit(1)
		}
		_, err = time.Parse("15:04", scheduleTime)
		if err != nil {
			r.Reporter.Errorf("Time format '%s' invalid", scheduleTime)
			os.Exit(1)
		}
	}

	// Parse next run to time.Time
	nextRun, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", scheduleDate, scheduleTime))
	if err != nil {
		r.Reporter.Errorf("Schedule date should use the format 'yyyy-mm-dd'\n" +
			"   Schedule time should use the format 'HH:mm'")
		os.Exit(1)
	}
	return nextRun
}

func buildNodeDrainGracePeriod(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster) ocm.Spec {
	nodeDrainGracePeriod := ""
	// Determine if the cluster already has a node drain grace period set and use that as the default
	nd := cluster.NodeDrainGracePeriod()
	if _, ok := nd.GetValue(); ok {
		// Convert larger times to hours, since the API only stores minutes
		val := int(nd.Value())
		unit := nd.Unit()
		if val >= 60 {
			val = val / 60
			if val == 1 {
				unit = "hour"
			} else {
				unit = "hours"
			}
		}
		nodeDrainGracePeriod = fmt.Sprintf("%d %s", val, unit)
	}
	// If node drain grace period is not set, or the user sent it as a CLI argument, use that instead
	if nodeDrainGracePeriod == "" || cmd.Flags().Changed("node-drain-grace-period") {
		nodeDrainGracePeriod = args.nodeDrainGracePeriod
	}
	if interactive.Enabled() {
		var err error
		nodeDrainGracePeriod, err = interactive.GetOption(interactive.Input{
			Question: "Node draining",
			Help:     cmd.Flags().Lookup("node-drain-grace-period").Usage,
			Options:  nodeDrainOptions,
			Default:  nodeDrainGracePeriod,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid node drain grace period: %s", err)
			os.Exit(1)
		}
	}
	isValidNodeDrainGracePeriod := false
	for _, nodeDrainOption := range nodeDrainOptions {
		if nodeDrainGracePeriod == nodeDrainOption {
			isValidNodeDrainGracePeriod = true
			break
		}
	}
	if !isValidNodeDrainGracePeriod {
		r.Reporter.Errorf("Expected a valid node drain grace period. Options are [%s]",
			strings.Join(nodeDrainOptions, ", "))
		os.Exit(1)
	}
	nodeDrainParsed := strings.Split(nodeDrainGracePeriod, " ")
	nodeDrainValue, err := strconv.ParseFloat(nodeDrainParsed[0], 64)
	if err != nil {
		r.Reporter.Errorf("Expected a valid node drain grace period: %s", err)
		os.Exit(1)
	}
	if nodeDrainParsed[1] == "hours" || nodeDrainParsed[1] == "hour" {
		nodeDrainValue = nodeDrainValue * 60
	}
	clusterSpec := ocm.Spec{
		NodeDrainGracePeriodInMinutes: nodeDrainValue,
	}
	return clusterSpec
}

func checkAndAckMissingAgreementsClassic(r *rosa.Runtime, cluster *cmv1.Cluster, upgradePolicy *cmv1.UpgradePolicy,
	clusterKey string) error {
	// check if the cluster upgrade requires gate agreements
	gates, err := r.OCMClient.GetMissingGateAgreementsClassic(cluster.ID(), upgradePolicy)
	if err != nil {
		return fmt.Errorf("failed to check for missing gate agreements upgrade for "+
			"cluster '%s': %v", clusterKey, err)
	}
	return checkGates(r, cluster, gates, clusterKey)
}

func checkAndAckMissingAgreementsHypershift(r *rosa.Runtime, cluster *cmv1.Cluster,
	upgradePolicy *cmv1.ControlPlaneUpgradePolicy, clusterKey string) error {
	// check if the cluster upgrade requires gate agreements
	gates, err := r.OCMClient.GetMissingGateAgreementsHypershift(cluster.ID(), upgradePolicy)
	if err != nil {
		return fmt.Errorf("failed to check for missing gate agreements upgrade for "+
			"cluster '%s': %v", clusterKey, err)
	}
	return checkGates(r, cluster, gates, clusterKey)
}

func checkGates(r *rosa.Runtime, cluster *cmv1.Cluster, gates []*cmv1.VersionGate, clusterKey string) error {
	isWarningDisplayed := false
	for _, gate := range gates {
		if !gate.STSOnly() {
			if !isWarningDisplayed {
				r.Reporter.Warnf("Missing required acknowledgements to schedule upgrade. \n")
				isWarningDisplayed = true
			}
			str := fmt.Sprintf("Description: %s\n", gate.Description())

			if gate.WarningMessage() != "" {
				str = fmt.Sprintf("%s"+
					"    Warning:     %s\n", str, gate.WarningMessage())
			}
			str = fmt.Sprintf("%s"+
				"    URL:         %s\n", str, gate.DocumentationURL())

			err := interactive.PrintHelp(interactive.Help{
				Message: "Read the below description and acknowledge to proceed with upgrade",
				Steps:   []string{str},
			})
			if err != nil {
				return fmt.Errorf("failed to get version gate '%s' for cluster '%s': %v",
					gate.ID(), clusterKey, err)
			}
			// for non sts gates we require user agreement
			if !confirm.Prompt(true, "I acknowledge") {
				os.Exit(0)
			}
		}
		err := r.OCMClient.AckVersionGate(cluster.ID(), gate.ID())
		if err != nil {
			return fmt.Errorf("failed to acknowledge version gate '%s' for cluster '%s': %v",
				gate.ID(), clusterKey, err)
		}
	}
	return nil
}
