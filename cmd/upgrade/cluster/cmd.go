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

	commonUtils "github.com/openshift-online/ocm-common/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/upgrade/roles"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	rolesHelper "github.com/openshift/rosa/pkg/helper/roles"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	version                  string
	scheduleDate             string
	scheduleTime             string
	nodeDrainGracePeriod     string
	controlPlane             bool
	schedule                 string
	allowMinorVersionUpdates bool
	dryRun                   bool
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
	Long: "Upgrade cluster to a new available version. Use '--dry-run' to acknowledge any gates prior to attempting" +
		" an upgrade",
	Example: `  # Interactively schedule an upgrade on the cluster named "mycluster"
  rosa upgrade cluster --cluster=mycluster --interactive

  # Schedule a cluster upgrade within the hour
  rosa upgrade cluster -c mycluster --version 4.12.20

  # Check if any gates need to be acknowledged prior to attempting an upgrading
  rosa upgrade cluster -c mycluster --version 4.12.20 --dry-run`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)
	interactive.AddModeFlag(Cmd)

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
		&args.schedule,
		"schedule",
		"",
		"cron expression in UTC which will be the time when an upgrade to the latest release will be "+
			"automatically scheduled and repeated at each occurrence. Mutually exclusive with --schedule-date and "+
			"--schedule-time. "+
			"This is currently supported only for Hosted Control Planes. ",
	)

	flags.BoolVar(
		&args.allowMinorVersionUpdates,
		"allow-minor-version-updates",
		false,
		"When using automatic scheduling with --schedule parameter, if true it will also update to latest "+
			"minor release, e.g. 4.12.20 -> 4.13.2. By default only z-stream updates will be scheduled. "+
			"This is currently supported only for Hosted Control Planes. ",
	)
	// Hidden for now as not supported yet
	flags.MarkHidden("allow-minor-version-updates")

	flags.StringVar(
		&args.nodeDrainGracePeriod,
		"node-drain-grace-period",
		"1 hour",
		fmt.Sprintf("You may set a grace period for how long Pod Disruption Budget-protected workloads will be "+
			"respected during upgrades.\nAfter this grace period, any workloads protected by Pod Disruption "+
			"Budgets that have not been successfully drained from a node will be forcibly evicted.\nValid "+
			"options are ['%s']\nThis flag is not supported for Hosted Control Planes.", strings.Join(nodeDrainOptions, "','")),
	)

	flags.BoolVar(
		&args.controlPlane,
		"control-plane",
		false,
		"For Hosted Control Plane, whether the upgrade should cover only the control plane",
	)

	flags.BoolVar(
		&args.dryRun,
		"dry-run",
		false,
		"Simulate upgrading the cluster, or run through acknowledgements required to upgrade prior to upgrading"+
			" a cluster.",
	)

	flags.MarkDeprecated("control-plane", "Flag is deprecated, and can be omitted when running this "+
		"command in the future")

	confirm.AddFlag(flags)
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

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command) error {
	// Define variables
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()
	currentUpgradeScheduling := ocm.UpgradeScheduling{
		Schedule:                 args.schedule,
		ScheduleDate:             args.scheduleDate,
		ScheduleTime:             args.scheduleTime,
		AllowMinorVersionUpdates: args.allowMinorVersionUpdates,
	}
	isHypershift := cluster.Hypershift().Enabled()

	if currentUpgradeScheduling.Schedule == "" && currentUpgradeScheduling.AllowMinorVersionUpdates {
		return fmt.Errorf("The '--allow-minor-version-upgrades' option needs to be used with --schedule")
	}

	if currentUpgradeScheduling.Schedule != "" && !isHypershift {
		return fmt.Errorf("The '--schedule' option is only supported for Hosted Control Planes")
	}

	if (currentUpgradeScheduling.ScheduleDate != "" || currentUpgradeScheduling.ScheduleTime != "") &&
		currentUpgradeScheduling.Schedule != "" {
		return fmt.Errorf("The '--schedule-date' and '--schedule-time' options are mutually exclusive with" +
			" '--schedule'")
	}

	if currentUpgradeScheduling.Schedule != "" && args.version != "" {
		return fmt.Errorf("The '--schedule' option is mutually exclusive with '--version'")
	}

	if args.dryRun {
		r.Reporter.Infof("Running in dry-run mode. Will not perform cluster upgrade")
	}

	// Check cluster preconditions
	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}
	if isHypershift {
		scheduledUpgrade, err := checkExistingScheduledUpgradeHypershift(r, cluster, clusterKey)
		if err != nil {
			return err
		}
		if scheduledUpgrade != nil {
			r.Reporter.Warnf("There is already a %s upgrade to version %s on %s",
				scheduledUpgrade.State().Value(),
				scheduledUpgrade.Version(),
				scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
			)
			return nil
		}
	} else {
		scheduledUpgrade, upgradeState, err := checkExistingScheduledUpgrade(r, cluster, clusterKey)
		if err != nil {
			return err
		}
		if scheduledUpgrade != nil {
			r.Reporter.Warnf("There is already a %s upgrade to version %s on %s",
				upgradeState.Value(),
				scheduledUpgrade.Version(),
				scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
			)
			return nil
		}
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

	// Start processing parameters
	// Mode
	mode, err := interactive.GetMode()
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	_, isSTS := cluster.AWS().STS().GetRoleARN()
	if !isSTS && mode != "" {
		return fmt.Errorf("The 'mode' option is only supported for STS clusters")
	}
	if isSTS && mode == "" {
		mode, err = interactive.GetOptionMode(cmd, mode, "IAM Roles/Policies upgrade mode")
		if err != nil {
			r.Reporter.Errorf("Expected a valid role upgrade mode: %v", err)
			os.Exit(1)
		}
	}

	// Upgrade type, manual or automatic
	currentUpgradeScheduling.AutomaticUpgrades = false
	if isHypershift {
		currentUpgradeScheduling.AutomaticUpgrades = currentUpgradeScheduling.Schedule != ""
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
				return fmt.Errorf("Expected an upgrade type: %s", err)
			}

			if currentUpgradeScheduling.AutomaticUpgrades {
				currentUpgradeScheduling.AllowMinorVersionUpdates, err = interactive.GetBool(interactive.Input{
					Question: "Allow minor upgrades",
					Help:     cmd.Flags().Lookup("allow-minor-version-updates").Usage,
					Default:  currentUpgradeScheduling.AllowMinorVersionUpdates,
					Required: false,
				})
				if err != nil {
					return fmt.Errorf("Expected a choice on the versions to target: %s", err)
				}
			}
		}
		if !currentUpgradeScheduling.AutomaticUpgrades {
			nextRun, err := interactive.BuildManualUpgradeSchedule(cmd, currentUpgradeScheduling.ScheduleDate,
				currentUpgradeScheduling.ScheduleTime)
			if err != nil {
				return err
			}
			currentUpgradeScheduling.NextRun = nextRun
		} else {
			schedule, err := interactive.BuildAutomaticUpgradeSchedule(cmd, currentUpgradeScheduling.Schedule)
			if err != nil {
				return err
			}
			currentUpgradeScheduling.Schedule = schedule
		}
	}

	// Version
	availableUpgrades, version, err := buildVersion(r, cmd, cluster, args.version,
		currentUpgradeScheduling.AutomaticUpgrades)
	if err != nil {
		return err
	}
	if !currentUpgradeScheduling.AutomaticUpgrades {
		if len(availableUpgrades) == 0 {
			r.Reporter.Warnf("There are no available upgrades")
			return nil
		}
		err = r.OCMClient.CheckUpgradeClusterVersion(availableUpgrades, version, cluster)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
	}

	// if cluster is sts validate roles are compatible with upgrade version
	// for automatic upgrades, version is not available
	if isSTS {
		if currentUpgradeScheduling.AutomaticUpgrades {
			// We do not know the upgrade version client side when scheduling an
			// automatic upgrade. Passing "" will still perform the role check.
			err := checkRolesManagedPolicies(r, cluster, mode, "")
			if err != nil {
				return fmt.Errorf("%v", err)
			}
		} else {
			checkSTSRolesCompatibility(r, cluster, mode, version, clusterKey)
		}
	}

	// Compute drain grace period config
	var clusterSpec ocm.Spec
	if isHypershift {
		if cmd.Flags().Changed("node-drain-grace-period") {
			return fmt.Errorf("%s flag is not supported to hosted clusters", "node-drain-grace-period")
		}
		clusterSpec = ocm.Spec{}
	} else {
		clusterSpec = buildNodeDrainGracePeriod(r, cmd, cluster)
	}

	// Validate version
	if !currentUpgradeScheduling.AutomaticUpgrades {
		version, err = ocm.CheckAndParseVersion(availableUpgrades, version, cluster)
		if err != nil {
			return fmt.Errorf("Error parsing version to upgrade to")
		}

		if r.Reporter.IsTerminal() && !args.dryRun && !confirm.Confirm("upgrade cluster to version '%s'", version) {
			os.Exit(0)
		}
	} else {
		if r.Reporter.IsTerminal() && !confirm.Confirm("schedule automatic cluster upgrades at '%s'",
			currentUpgradeScheduling.Schedule) {
			os.Exit(0)
		}
	}

	// Create policy upgrade
	if isHypershift {
		err = createUpgradePolicyHypershift(r, clusterKey, cluster, version, currentUpgradeScheduling)
	} else {
		err = createUpgradePolicyClassic(r, cmd, clusterKey, cluster, version, currentUpgradeScheduling.ScheduleDate,
			currentUpgradeScheduling.ScheduleTime)
	}
	if err != nil {
		return fmt.Errorf("Failed to schedule upgrade for cluster '%s': %v", clusterKey, err)
	}

	if args.dryRun {
		r.Reporter.Infof(
			"Upgrading cluster '%s' should succeed. Please wait 1 to 2 minutes, then rerun this command"+
				" without the '--dry-run' flag, to allow time for the acknowledged agreements to be reflected.",
			cluster.ID())
		return nil
	}

	// Update cluster with grace period configuration
	err = r.OCMClient.UpdateCluster(cluster.ID(), r.Creator, clusterSpec)
	if err != nil {
		return fmt.Errorf("Failed to update cluster '%s': %v", clusterKey, err)
	}

	r.Reporter.Infof("Upgrade successfully scheduled for cluster '%s'", clusterKey)
	return nil
}

func createUpgradePolicyHypershift(r *rosa.Runtime, clusterKey string,
	cluster *cmv1.Cluster, version string, currentScheduling ocm.UpgradeScheduling) error {
	upgradePolicyBuilder := cmv1.NewControlPlaneUpgradePolicy().UpgradeType(cmv1.UpgradeTypeControlPlane)
	if currentScheduling.AutomaticUpgrades {
		upgradePolicyBuilder = upgradePolicyBuilder.ScheduleType(cmv1.ScheduleTypeAutomatic).
			Schedule(currentScheduling.Schedule).EnableMinorVersionUpgrades(currentScheduling.AllowMinorVersionUpdates)
	} else {
		upgradePolicyBuilder = upgradePolicyBuilder.ScheduleType(cmv1.ScheduleTypeManual).Version(version)
		upgradePolicyBuilder = upgradePolicyBuilder.NextRun(currentScheduling.NextRun)
	}

	upgradePolicy, err := upgradePolicyBuilder.Build()
	if err != nil {
		return err
	}
	err = checkAndAckMissingAgreementsHypershift(r, cluster, upgradePolicy, clusterKey)
	if err != nil {
		return err
	}

	if args.dryRun {
		return nil
	}

	_, err = r.OCMClient.ScheduleHypershiftControlPlaneUpgrade(cluster.ID(), upgradePolicy)
	if err != nil {
		return err
	}
	return nil
}

func createUpgradePolicyClassic(r *rosa.Runtime, cmd *cobra.Command, clusterKey string,
	cluster *cmv1.Cluster, version string, scheduleDate string, scheduleTime string) error {
	upgradePolicyBuilder := cmv1.NewUpgradePolicy().
		ScheduleType(cmv1.ScheduleTypeManual).
		Version(version)
	upgradePolicy, err := upgradePolicyBuilder.Build()
	if err != nil {
		return err
	}
	err = checkAndAckMissingAgreementsClassic(r, cluster, upgradePolicy, clusterKey)
	if err != nil {
		return err
	}

	if args.dryRun {
		return nil
	}

	nextRun, err := interactive.BuildManualUpgradeSchedule(cmd, scheduleDate, scheduleTime)
	if err != nil {
		return err
	}
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

func buildVersion(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster,
	version string, isAutomaticUpgrade bool) ([]string, string, error) {
	var availableUpgrades []string
	var err error
	if ocm.IsHyperShiftCluster(cluster) {
		availableUpgrades = ocm.GetAvailableUpgradesByCluster(cluster)
	} else {
		availableUpgrades, err = r.OCMClient.GetAvailableUpgrades(ocm.GetVersionID(cluster))
		if err != nil {
			return availableUpgrades, version, fmt.Errorf("Failed to find available upgrades: %v", err)
		}
	}
	if len(availableUpgrades) == 0 {
		return availableUpgrades, version, nil
	}
	if !isAutomaticUpgrade && (version == "" || interactive.Enabled()) {
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
			return availableUpgrades, version, fmt.Errorf("Expected a valid version to upgrade to: %s", err)
		}
	}
	return availableUpgrades, version, nil
}

func checkExistingScheduledUpgrade(r *rosa.Runtime, cluster *cmv1.Cluster,
	clusterKey string) (*cmv1.UpgradePolicy, *cmv1.UpgradePolicyState, error) {
	scheduledUpgrade, upgradeState, err := r.OCMClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
	}

	return scheduledUpgrade, upgradeState, nil
}

func checkExistingScheduledUpgradeHypershift(r *rosa.Runtime, cluster *cmv1.Cluster,
	clusterKey string) (*cmv1.ControlPlaneUpgradePolicy, error) {
	scheduledUpgrade, err := r.OCMClient.GetControlPlaneScheduledUpgrade(cluster.ID())
	if err != nil {
		return nil, fmt.Errorf("Failed to get scheduled control plane upgrades for cluster '%s': %v", clusterKey, err)
	}
	return scheduledUpgrade, nil
}

func checkSTSRolesCompatibility(r *rosa.Runtime, cluster *cmv1.Cluster, mode string,
	version string, clusterKey string) {
	r.Reporter.Infof("Ensuring account and operator role policies for cluster '%s'"+
		" are compatible with upgrade.", cluster.ID())
	arguments.DisableRegionDeprecationWarning = true // disable region deprecation warning
	roles.Cmd.Run(roles.Cmd, []string{mode, cluster.ID(), version, cluster.Version().ChannelGroup()})
	arguments.DisableRegionDeprecationWarning = false // enable region deprecation again
	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Account and operator roles for cluster '%s' are compatible with upgrade", clusterKey)
	}
}

func checkRolesManagedPolicies(r *rosa.Runtime, cluster *cmv1.Cluster, mode string,
	upgradeVersion string) error {
	ocmClient := r.OCMClient

	credRequests, err := ocmClient.GetCredRequests(cluster.Hypershift().Enabled())
	if err != nil {
		return fmt.Errorf("Error getting operator credential request from OCM %v", err)
	}

	unifiedPath, err := aws.GetPathFromAccountRole(cluster, aws.AccountRoles[aws.InstallerAccountRole].Name)
	if err != nil {
		return fmt.Errorf("Expected a valid path for '%s': %v", cluster.AWS().STS().RoleARN(), err)
	}

	err = rolesHelper.ValidateAccountAndOperatorRolesManagedPolicies(r, cluster, credRequests, unifiedPath,
		mode, upgradeVersion)
	if err != nil {
		return err
	}

	return nil
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
	nodeDrainValue, err := strconv.ParseFloat(nodeDrainParsed[0], commonUtils.MaxByteSize)
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
		return err
	}
	return checkGates(r, cluster, gates, clusterKey)
}

func checkGates(r *rosa.Runtime, cluster *cmv1.Cluster, gates []*cmv1.VersionGate, clusterKey string) error {
	isWarningDisplayed := false
	if !args.dryRun {
		r.Reporter.Warnf("To check and acknowledge gates prior to scheduling an upgrade, run this command with " +
			"'--dry-run'")
	}
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
			} else {
				r.Reporter.Infof("Gate %s acknowledged", gate.ID())
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
