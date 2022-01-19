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

	"github.com/openshift/rosa/pkg/interactive/confirm"

	"github.com/briandowns/spinner"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/upgrade/accountroles"
	"github.com/openshift/rosa/cmd/upgrade/operatorroles"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	version              string
	scheduleDate         string
	scheduleTime         string
	nodeDrainGracePeriod string
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
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	mode, err := aws.GetMode()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	_, isSTS := cluster.AWS().STS().GetRoleARN()
	if !isSTS && mode != "" {
		reporter.Errorf("The 'mode' option is only supported for STS clusters")
		os.Exit(1)
	}

	scheduledUpgrade, upgradeState, err := ocmClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	if scheduledUpgrade != nil {
		reporter.Warnf("There is already a %s upgrade to version %s on %s",
			upgradeState.Value(),
			scheduledUpgrade.Version(),
			scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
		)
		os.Exit(0)
	}

	version := args.version
	scheduleDate := args.scheduleDate
	scheduleTime := args.scheduleTime

	availableUpgrades, err := ocmClient.GetAvailableUpgrades(ocm.GetVersionID(cluster))
	if err != nil {
		reporter.Errorf("Failed to find available upgrades: %v", err)
		os.Exit(1)
	}
	if len(availableUpgrades) == 0 {
		reporter.Warnf("There are no available upgrades")
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
			reporter.Errorf("Expected a valid version to upgrade to: %s", err)
			os.Exit(1)
		}
	}

	// Check that the version is valid
	validVersion := false
	for _, v := range availableUpgrades {
		if v == version {
			validVersion = true
			break
		}
	}
	if !validVersion {
		reporter.Errorf("Expected a valid version to upgrade to")
		os.Exit(1)
	}

	if scheduleDate == "" || scheduleTime == "" {
		interactive.Enable()
	}

	// if cluster is sts validate roles are compatible with upgrade version
	if isSTS {
		var spin *spinner.Spinner
		if reporter.IsTerminal() {
			spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		}

		reporter.Infof("Ensuring cluster roles and policies are compatible with upgrade.")
		if spin != nil {
			spin.Start()
		}

		prefix, err := aws.GetPrefixFromAccountRole(cluster)
		if err != nil {
			reporter.Errorf("Could not get role prefix for cluster '%s' : %v", clusterKey, err)
			os.Exit(1)
		}

		isAccountRoleUpgradeNeeded, err := awsClient.IsUpgradedNeededForRole(prefix, awsCreator.AccountID, version)
		if err != nil {
			reporter.Errorf("Could not validate '%s' clusters account roles : %v", clusterKey, err)
			os.Exit(1)
		}

		isOperatorRoleUpgradeNeeded, err := awsClient.IsUpgradedNeededForOperatorRole(cluster,
			awsCreator.AccountID, version)
		if err != nil {
			reporter.Errorf("Could not validate '%s' clusters operator roles : %v", clusterKey, err)
			os.Exit(1)
		}

		if spin != nil {
			spin.Stop()
		}

		if isAccountRoleUpgradeNeeded || isOperatorRoleUpgradeNeeded {
			if mode != "" {
				if isAccountRoleUpgradeNeeded {
					reporter.Infof("Preparing to upgrade account roles.")
					accountroles.Cmd.Run(accountroles.Cmd, []string{prefix, mode})
				}
				if isOperatorRoleUpgradeNeeded {
					reporter.Infof("Preparing to upgrade operator roles.")
					operatorroles.Cmd.Run(operatorroles.Cmd, []string{clusterKey, mode})
				}
				if mode == aws.ModeManual {
					reporter.Infof("Run the following command to continue scheduling cluster upgrade"+
						" once cluster roles have been upgraded : \n\n"+
						"\trosa upgrade cluster --cluster %s\n", clusterKey)
					os.Exit(0)
				}
			} else {
				reporter.Infof("Cluster Roles are not valid with upgrade version %s. "+
					"Run the following command(s) to upgrade Cluster Roles:\n\n"+
					"\t%s\n",
					version,
					buildRoleUpgradeCommand(isAccountRoleUpgradeNeeded, isOperatorRoleUpgradeNeeded, clusterKey, prefix))
				os.Exit(0)
			}
		}
	}

	// Set the default next run within the next 10 minutes
	now := time.Now().UTC().Add(time.Minute * 10)
	if scheduleDate == "" {
		scheduleDate = now.Format("2006-01-02")
	}
	if scheduleTime == "" {
		scheduleTime = now.Format("15:04")
	}

	if interactive.Enabled() {
		// If datetimes are set, use them in the interactive form, otherwise fallback to 'now'
		scheduleParsed, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", scheduleDate, scheduleTime))
		if err != nil {
			reporter.Errorf("Schedule date should use the format 'yyyy-mm-dd'\n" +
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
			reporter.Errorf("Expected a valid date: %s", err)
			os.Exit(1)
		}
		_, err = time.Parse("2006-01-02", scheduleDate)
		if err != nil {
			reporter.Errorf("Date format '%s' invalid", scheduleDate)
			os.Exit(1)
		}

		scheduleTime, err = interactive.GetString(interactive.Input{
			Question: "Please input desired UTC time in format HH:mm",
			Help:     cmd.Flags().Lookup("schedule-time").Usage,
			Default:  scheduleTime,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid time: %s", err)
			os.Exit(1)
		}
		_, err = time.Parse("15:04", scheduleTime)
		if err != nil {
			reporter.Errorf("Time format '%s' invalid", scheduleTime)
			os.Exit(1)
		}
	}

	// Parse next run to time.Time
	nextRun, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", scheduleDate, scheduleTime))
	if err != nil {
		reporter.Errorf("Schedule date should use the format 'yyyy-mm-dd'\n" +
			"   Schedule time should use the format 'HH:mm'")
		os.Exit(1)
	}

	upgradePolicyBuilder := cmv1.NewUpgradePolicy().
		ScheduleType("manual").
		Version(version).
		NextRun(nextRun)

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
		nodeDrainGracePeriod, err = interactive.GetOption(interactive.Input{
			Question: "Node draining",
			Help:     cmd.Flags().Lookup("node-drain-grace-period").Usage,
			Options:  nodeDrainOptions,
			Default:  nodeDrainGracePeriod,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid node drain grace period: %s", err)
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
		reporter.Errorf("Expected a valid node drain grace period. Options are [%s]",
			strings.Join(nodeDrainOptions, ", "))
		os.Exit(1)
	}
	nodeDrainParsed := strings.Split(nodeDrainGracePeriod, " ")
	nodeDrainValue, err := strconv.ParseFloat(nodeDrainParsed[0], 64)
	if err != nil {
		reporter.Errorf("Expected a valid node drain grace period: %s", err)
		os.Exit(1)
	}
	if nodeDrainParsed[1] == "hours" || nodeDrainParsed[1] == "hour" {
		nodeDrainValue = nodeDrainValue * 60
	}

	clusterSpec := ocm.Spec{
		NodeDrainGracePeriodInMinutes: nodeDrainValue,
	}

	upgradePolicy, err := upgradePolicyBuilder.Build()
	if err != nil {
		reporter.Errorf("Failed to schedule upgrade for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// check if the cluster upgrade requires gate agreements
	gates, err := ocmClient.GetMissingGateAgreements(cluster.ID(), upgradePolicy)
	if err != nil {
		reporter.Errorf("Failed to check for missing gate agreements upgrade for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	for _, gate := range gates {
		reporter.Infof("Gate: %v", gate)
		if !gate.STSOnly() {
			// for non sts gates we require user agreement
			if !confirm.Prompt(true,
				"I acknowledge that my workloads are no longer using removed APIs. (Background:'%s)",
				gate.Description()) {
				os.Exit(0)
			}
		}
		err = ocmClient.AckVersionGate(cluster.ID(), gate.ID())
		if err != nil {
			reporter.Errorf("Failed to acknowledge version gate '%s' for cluster '%s': %v",
				gate.ID(), clusterKey, err)
			os.Exit(1)
		}
	}

	err = ocmClient.ScheduleUpgrade(cluster.ID(), upgradePolicy)
	if err != nil {
		reporter.Errorf("Failed to schedule upgrade for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	err = ocmClient.UpdateCluster(cluster.ID(), awsCreator, clusterSpec)
	if err != nil {
		reporter.Errorf("Failed to update cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	reporter.Infof("Upgrade successfully scheduled for cluster '%s'", clusterKey)
}

func buildRoleUpgradeCommand(isAccountRoleUpgradeNeeded bool, isOperatorRoleUpgradeNeeded bool,
	clusterKey string, prefix string) string {
	accountRoleCmd := fmt.Sprintf("rosa upgrade account-roles --prefix %s", prefix)
	operatorRoleCmd := fmt.Sprintf("rosa upgrade operator-roles --cluster %s", clusterKey)

	if isAccountRoleUpgradeNeeded && isOperatorRoleUpgradeNeeded {
		return fmt.Sprintf("%s\n\t%s", accountRoleCmd, operatorRoleCmd)
	}
	if isAccountRoleUpgradeNeeded {
		return accountRoleCmd
	}
	return operatorRoleCmd
}
