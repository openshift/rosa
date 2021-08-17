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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const doubleQuotesToRemove = "\"\""

var args struct {
	// Basic options
	expirationTime     string
	expirationDuration time.Duration

	// Networking options
	private                   bool
	disableWorkloadMonitoring bool
	httpProxy                 string
	httpsProxy                string
	noProxySlice              []string
	additionalTrustBundleFile string

	// Upgrade schedule options
	upgradeScheduleDay  string
	upgradeScheduleTime string
}

var daysMap = map[string]string{
	"Sunday":    "SUN",
	"Monday":    "MON",
	"Tuesday":   "TUE",
	"Wednesday": "WED",
	"Thursday":  "THU",
	"Friday":    "FRI",
	"Saturday":  "SAT",
}

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Edit cluster",
	Long:  "Edit cluster.",
	Example: `  # Edit a cluster named "mycluster" to make it private
  rosa edit cluster -c mycluster --private

  # Edit a cluster to enable automatic upgrades
  rosa edit cluster -c mycluster --upgrade-schedule-day Monday --upgrade-schedule-time 15:04

  # Edit all options interactively
  rosa edit cluster -c mycluster --interactive`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)

	// Basic options
	flags.StringVar(
		&args.expirationTime,
		"expiration-time",
		"",
		"Specific time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.",
	)
	flags.DurationVar(
		&args.expirationDuration,
		"expiration",
		0,
		"Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.",
	)
	// Cluster expiration is not supported in production
	flags.MarkHidden("expiration-time")
	flags.MarkHidden("expiration")

	// Networking options
	flags.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict master API endpoint to direct, private connectivity.",
	)
	flags.BoolVar(
		&args.disableWorkloadMonitoring,
		"disable-workload-monitoring",
		false,
		"Enables you to monitor your own projects in isolation from Red Hat Site Reliability Engineer (SRE) "+
			"platform metrics.",
	)
	flags.StringVar(
		&args.httpProxy,
		"http-proxy",
		"",
		"A proxy URL to use for creating HTTP connections outside the cluster. The URL scheme must be http.",
	)

	flags.StringVar(
		&args.httpsProxy,
		"https-proxy",
		"",
		"A proxy URL to use for creating HTTPS connections outside the cluster.",
	)

	flags.StringSliceVar(
		&args.noProxySlice,
		"no-proxy",
		nil,
		"A comma-separated list of destination domain names, domains, IP addresses or "+
			"other network CIDRs to exclude proxying.",
	)

	flags.StringVar(
		&args.additionalTrustBundleFile,
		"additional-trust-bundle-file",
		"",
		"A file contains a PEM-encoded X.509 certificate bundle that will be "+
			"added to the nodes' trusted certificate store.")

	flags.StringVar(
		&args.upgradeScheduleDay,
		"upgrade-schedule-day",
		"Sunday",
		"Preferred day of the week to upgrade cluster.",
	)
	Cmd.RegisterFlagCompletionFunc("upgrade-schedule-day", upgradeScheduleDayCompletion)

	flags.StringVar(
		&args.upgradeScheduleTime,
		"upgrade-schedule-time",
		"0:00",
		"Preferred time of the day to upgrade cluster.",
	)
	Cmd.RegisterFlagCompletionFunc("upgrade-schedule-time", upgradeScheduleTimeCompletion)
}

func upgradeScheduleDayCompletion(cmd *cobra.Command,
	args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return dayOptions(), cobra.ShellCompDirectiveDefault
}

func upgradeScheduleTimeCompletion(cmd *cobra.Command,
	args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getTimeOptions(), cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	// Enable interactive mode if no flags have been set
	if !interactive.Enabled() {
		changedFlags := false
		for _, flag := range []string{
			"additional-trust-bundle-file",
			"disable-workload-monitoring",
			"expiration-time",
			"expiration",
			"http-proxy",
			"https-proxy",
			"no-proxy",
			"private",
			"upgrade-schedule-day",
			"upgrade-schedule-time",
		} {
			if cmd.Flags().Changed(flag) {
				changedFlags = true
			}
		}
		if !changedFlags {
			interactive.Enable()
		}
	}

	cluster := r.FetchCluster()

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is in '%s' state, can't update", clusterKey, cluster.State())
		os.Exit(1)
	}

	// Validate flags:
	expiration, err := validateExpiration()
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	if interactive.Enabled() {
		r.Reporter.Infof("Interactive mode enabled.\n" +
			"Any optional fields can be ignored and will not be updated.")
	}

	/*There are three possible options of input from the user when a prompt shows up:
	1) The user presses the 'enter' button ---> interactive 'getString' method returns either an existing value if exists
	   (the one that shows up as part of the question, i.e. - ? HTTP proxy: http://site.com),
	   or double quotes ("") if no existing value. In that case, we send to OCM nil as we do not want any change.
	2) In case the user wants to remove an existing value, an empty string ("") should be entered by the user -->
	   interactive 'getString' method returns "\"\"". In that case, we send OCM double quotes to remove the existing value.
	3) The user enters any other value ---> a simple and straightforward case. */

	enableProxy := false
	useExistingVPC := false
	var httpProxy *string
	var httpProxyValue string
	if cmd.Flags().Changed("http-proxy") {
		httpProxyValue = args.httpProxy
		httpProxy = &httpProxyValue
	}
	var httpsProxy *string
	var httpsProxyValue string
	if cmd.Flags().Changed("https-proxy") {
		httpsProxyValue = args.httpsProxy
		httpsProxy = &httpsProxyValue
	}
	var noProxySlice []string
	if cmd.Flags().Changed("no-proxy") {
		noProxySlice = args.noProxySlice
	}
	var additionalTrustBundleFile *string
	var additionalTrustBundleFileValue string
	if cmd.Flags().Changed("additional-trust-bundle-file") {
		additionalTrustBundleFileValue = args.additionalTrustBundleFile
		additionalTrustBundleFile = &additionalTrustBundleFileValue
	}

	if httpProxy != nil || httpsProxy != nil || len(noProxySlice) > 0 || additionalTrustBundleFile != nil {
		enableProxy = true
		useExistingVPC = true
	}

	if len(cluster.AWS().SubnetIDs()) == 0 &&
		((httpProxy != nil && *httpProxy != "") || (httpsProxy != nil && *httpsProxy != "") ||
			len(noProxySlice) > 0 ||
			(additionalTrustBundleFile != nil && *additionalTrustBundleFile != "")) {
		r.Reporter.Errorf("Cluster-wide proxy is not supported on clusters using the default VPC")
		os.Exit(1)
	}

	var private *bool
	var privateValue bool
	if cmd.Flags().Changed("private") {
		privateValue = args.private
		private = &privateValue
	} else if interactive.Enabled() {
		privateValue = cluster.API().Listening() == cmv1.ListeningMethodInternal
	}

	privateWarning := "You will not be able to access your cluster until you edit network settings " +
		"in your cloud provider. To also change the privacy setting of the application router " +
		"endpoints, use the 'rosa edit ingress' command."
	if interactive.Enabled() {
		privateValue, err = interactive.GetBool(interactive.Input{
			Question: "Private cluster",
			Help:     fmt.Sprintf("%s %s", cmd.Flags().Lookup("private").Usage, privateWarning),
			Default:  privateValue,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid private value: %s", err)
			os.Exit(1)
		}
		private = &privateValue
	} else if privateValue {
		r.Reporter.Warnf("You are choosing to make your cluster API private. %s", privateWarning)
		if !confirm.Confirm("set cluster '%s' as private", clusterKey) {
			os.Exit(0)
		}
	}

	var disableWorkloadMonitoring *bool
	var disableWorkloadMonitoringValue bool

	if cmd.Flags().Changed("disable-workload-monitoring") {
		disableWorkloadMonitoringValue = args.disableWorkloadMonitoring
		disableWorkloadMonitoring = &disableWorkloadMonitoringValue
	} else if interactive.Enabled() {
		disableWorkloadMonitoringValue = cluster.DisableUserWorkloadMonitoring()
	}

	if interactive.Enabled() {
		disableWorkloadMonitoringValue, err = interactive.GetBool(interactive.Input{
			Question: "Disable Workload monitoring",
			Help:     cmd.Flags().Lookup("disable-workload-monitoring").Usage,
			Default:  disableWorkloadMonitoringValue,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid disable-workload-monitoring value: %v", err)
			os.Exit(1)
		}
		disableWorkloadMonitoring = &disableWorkloadMonitoringValue
	} else if disableWorkloadMonitoringValue {
		if !confirm.Confirm("disable workload monitoring for your cluster %s", clusterKey) {
			os.Exit(0)
		}
	}

	if len(cluster.AWS().SubnetIDs()) > 0 {
		useExistingVPC = true
	}
	if useExistingVPC && !enableProxy && interactive.Enabled() {
		enableProxyValue, err := interactive.GetBool(interactive.Input{
			Question: "Update cluster-wide proxy",
			Help: "To install cluster-wide proxy, you need to set one of the following attributes: 'http-proxy', " +
				"'https-proxy', additional-trust-bundle",
			Default: enableProxy,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid proxy-enabled value: %s", err)
			os.Exit(1)
		}
		enableProxy = enableProxyValue
	}
	if enableProxy && interactive.Enabled() {
		err = interactive.PrintHelp(interactive.Help{
			Message: "To remove any existing cluster-wide proxy value or an existing additional-trust-bundle value, " +
				"enter a set of double quotes (\"\")",
		})
		if err != nil {
			return
		}
	}

	/*******  HTTPProxy *******/
	if enableProxy && interactive.Enabled() {
		var def string
		if cluster.Proxy() != nil {
			def = cluster.Proxy().HTTPProxy()
		}
		if httpProxy != nil {
			def = *httpProxy
			if def == "" {
				// received double quotes from the user. need to remove the existing value
				def = doubleQuotesToRemove
			}
		}
		httpProxyValue, err = interactive.GetString(interactive.Input{
			Question: "HTTP proxy",
			Help:     cmd.Flags().Lookup("http-proxy").Usage,
			Default:  def,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid http proxy: %s", err)
			os.Exit(1)
		}

		if len(httpProxyValue) == 0 {
			//user skipped the prompt by pressing 'enter'
			httpProxy = nil
		} else if httpProxyValue == doubleQuotesToRemove {
			//user entered double quotes ("") to remove the existing value
			httpProxy = new(string)
			*httpProxy = ""
		} else {
			httpProxy = &httpProxyValue
		}
	}
	if httpProxy != nil && *httpProxy != doubleQuotesToRemove {
		err = ocm.ValidateHTTPProxy(*httpProxy)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	/******* HTTPSProxy *******/
	if enableProxy && interactive.Enabled() {
		var def string
		if cluster.Proxy() != nil {
			def = cluster.Proxy().HTTPSProxy()
		}
		if httpsProxy != nil {
			def = *httpsProxy
			if def == "" {
				// received double quotes from the user. need to remove the existing value
				def = doubleQuotesToRemove
			}
		}
		httpsProxyValue, err = interactive.GetString(interactive.Input{
			Question: "HTTPS proxy",
			Help:     cmd.Flags().Lookup("https-proxy").Usage,
			Default:  def,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid https proxy: %s", err)
			os.Exit(1)
		}
		if len(httpsProxyValue) == 0 {
			//user skipped the prompt by pressing 'enter'
			httpsProxy = nil
		} else if httpsProxyValue == doubleQuotesToRemove {
			//user entered double quotes ("") to remove the existing value
			httpsProxy = new(string)
			*httpsProxy = ""
		} else {
			httpsProxy = &httpsProxyValue
		}
	}
	if httpsProxy != nil && *httpsProxy != doubleQuotesToRemove {
		err = interactive.IsURL(*httpsProxy)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	///******* NoProxy *******/
	if enableProxy && interactive.Enabled() {
		noProxyInput, err := interactive.GetString(interactive.Input{
			Question: "No proxy",
			Help:     cmd.Flags().Lookup("no-proxy").Usage,
			Default:  cluster.Proxy().NoProxy(),
			Validators: []interactive.Validator{
				aws.UserNoProxyValidator,
				aws.UserNoProxyDuplicateValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid set of no proxy domains/CIDR's: %s", err)
			os.Exit(1)
		}
		noProxySlice = helper.HandleEmptyStringOnSlice(strings.Split(noProxyInput, ","))
	}
	if isExpectedHTTPProxyOrHTTPSProxy(httpProxy, httpsProxy, noProxySlice, cluster) {
		r.Reporter.Errorf("Expected at least one of the following: http-proxy, https-proxy")
		os.Exit(1)
	}

	if len(noProxySlice) > 0 {
		if len(noProxySlice) == 1 && noProxySlice[0] == doubleQuotesToRemove {
			noProxySlice[0] = ""
		}

		duplicate, found := aws.HasDuplicates(noProxySlice)
		if found {
			r.Reporter.Errorf("Invalid no-proxy list, duplicate key '%s' found", duplicate)
			os.Exit(1)
		}
		for _, domain := range noProxySlice {
			err := aws.UserNoProxyValidator(domain)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
	}

	/*******  AdditionalTrustBundle *******/
	updateAdditionalTrustBundle := false
	if additionalTrustBundleFile != nil {
		updateAdditionalTrustBundle = true
	}
	if useExistingVPC && !updateAdditionalTrustBundle && additionalTrustBundleFile == nil &&
		interactive.Enabled() {
		updateAdditionalTrustBundleValue, err := interactive.GetBool(interactive.Input{
			Question: "Update additional trust bundle",
			Default:  updateAdditionalTrustBundle,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid -update-additional-trust-bundle value: %s", err)
			os.Exit(1)
		}
		updateAdditionalTrustBundle = updateAdditionalTrustBundleValue
	}
	if updateAdditionalTrustBundle && interactive.Enabled() {
		var def string
		if cluster.AdditionalTrustBundle() == "REDACTED" {
			def = "REDACTED"
		}
		if additionalTrustBundleFile != nil {
			def = *additionalTrustBundleFile
			if def == "" {
				// received double quotes from the iser. need to remove the existing value
				def = doubleQuotesToRemove
			}
		}
		additionalTrustBundleFileValue, err = interactive.GetCert(interactive.Input{
			Question: "Additional trust bundle file path",
			Help:     cmd.Flags().Lookup("additional-trust-bundle-file").Usage,
			Default:  def,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid additional trust bundle file name: %s", err)
			os.Exit(1)
		}

		if len(additionalTrustBundleFileValue) == 0 {
			//user skipped the prompt by pressing 'enter'
			additionalTrustBundleFile = nil
		} else if additionalTrustBundleFileValue == doubleQuotesToRemove {
			//user entered double quotes ("") to remove the existing value
			additionalTrustBundleFile = new(string)
			*additionalTrustBundleFile = ""
		} else {
			additionalTrustBundleFile = &additionalTrustBundleFileValue
		}
	}
	if additionalTrustBundleFile != nil && *additionalTrustBundleFile != doubleQuotesToRemove {
		err = ocm.ValidateAdditionalTrustBundle(*additionalTrustBundleFile)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	upgradeScheduleDay := args.upgradeScheduleDay
	upgradeScheduleTime := args.upgradeScheduleTime
	enableAutomaticUpgradeSchedule := false
	isSTS := cluster.AWS().STS().RoleARN() != ""

	if cmd.Flags().Changed("upgrade-schedule-day") || cmd.Flags().Changed("upgrade-schedule-time") {
		enableAutomaticUpgradeSchedule = true
	}
	if interactive.Enabled() && !enableAutomaticUpgradeSchedule && !isSTS {
		enableAutomaticUpgradeSchedule, _ = interactive.GetBool(interactive.Input{
			Question: "Enable automatic upgrades?",
			Help: "Clusters will be automatically upgraded based on your defined " +
				"day and start time when new versions are available",
			Required: true,
		})
	}

	if enableAutomaticUpgradeSchedule && isSTS {
		r.Reporter.Errorf("Automatic upgrades are not currently supported on STS clusters")
		os.Exit(1)
	}

	scheduledUpgrade, upgradeState, err := r.OCMClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	if scheduledUpgrade != nil {
		r.Reporter.Warnf("There is already a scheduled %s upgrade on this cluster. To change the "+
			"upgrade policy first delete the existing one with 'rosa delete upgrade -c %s'",
			upgradeState.Value(),
			clusterKey,
		)
		os.Exit(1)
	}

	if interactive.Enabled() && enableAutomaticUpgradeSchedule {
		upgradeScheduleDay, err = interactive.GetOption(interactive.Input{
			Question: "Preferred day",
			Help:     cmd.Flags().Lookup("upgrade-schedule-day").Usage,
			Options:  dayOptions(),
			Default:  upgradeScheduleDay,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid day of the week: %s", err)
			os.Exit(1)
		}

		upgradeScheduleTime, err = interactive.GetOption(interactive.Input{
			Question: "Preferred time",
			Help:     cmd.Flags().Lookup("upgrade-schedule-time").Usage,
			Options:  getTimeOptions(),
			Default:  upgradeScheduleTime,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid time of the day: %s", err)
			os.Exit(1)
		}
	}

	upgradeSchedule := ""
	if enableAutomaticUpgradeSchedule {
		if upgradeScheduleDay == "" {
			r.Reporter.Errorf("Automatic upgrade schedule requires a day of the week")
			os.Exit(1)
		}
		if daysMap[upgradeScheduleDay] == "" {
			r.Reporter.Errorf("Invalid day of the week. Valid options are %s", dayOptions())
			os.Exit(1)
		}
		if upgradeScheduleTime == "" {
			r.Reporter.Errorf("Automatic upgrade schedule requires a time of the day")
			os.Exit(1)
		}
		timeOfDay, err := time.Parse("15:04", upgradeScheduleTime)
		if err != nil {
			r.Reporter.Errorf("Invalid time of the day. Use the format HH:mm")
			os.Exit(1)
		}
		upgradeSchedule = fmt.Sprintf("%d %d * * %s",
			timeOfDay.Minute(), timeOfDay.Hour(), daysMap[upgradeScheduleDay])

		err = cronValidator(upgradeSchedule)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	clusterConfig := ocm.Spec{
		Expiration:                expiration,
		Private:                   private,
		DisableWorkloadMonitoring: disableWorkloadMonitoring,
	}

	if httpProxy != nil {
		clusterConfig.HTTPProxy = httpProxy
	}
	if httpsProxy != nil {
		clusterConfig.HTTPSProxy = httpsProxy
	}
	if noProxySlice != nil {
		str := strings.Join(noProxySlice, ",")
		clusterConfig.NoProxy = &str
	}
	if additionalTrustBundleFile != nil {
		clusterConfig.AdditionalTrustBundle = new(string)
		if *additionalTrustBundleFile == doubleQuotesToRemove {
			*clusterConfig.AdditionalTrustBundle = *additionalTrustBundleFile
		} else {
			// Get certificate contents
			if len(*additionalTrustBundleFile) > 0 {
				cert, err := ioutil.ReadFile(*additionalTrustBundleFile)
				if err != nil {
					r.Reporter.Errorf("Failed to read additional trust bundle file: %s", err)
					os.Exit(1)
				}
				*clusterConfig.AdditionalTrustBundle = string(cert)
			}
		}
	}

	r.Reporter.Debugf("Updating cluster '%s'", clusterKey)
	err = r.OCMClient.UpdateCluster(clusterKey, r.Creator, clusterConfig)
	if err != nil {
		r.Reporter.Errorf("Failed to update cluster: %v", err)
		os.Exit(1)
	}

	if upgradeSchedule != "" {
		upgradePolicy, err := cmv1.NewUpgradePolicy().
			ScheduleType("automatic").
			Schedule(upgradeSchedule).
			Build()
		if err != nil {
			r.Reporter.Errorf("Failed to schedule upgrade for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
		err = r.OCMClient.ScheduleUpgrade(cluster.ID(), upgradePolicy)
		if err != nil {
			r.Reporter.Errorf("Failed to set automatic upgrade policy for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	}

	r.Reporter.Infof("Updated cluster '%s'", clusterKey)
}

func dayOptions() []string {
	keys := reflect.ValueOf(daysMap).MapKeys()
	daySlice := make([]string, len(keys))
	for i, v := range keys {
		daySlice[i] = v.Interface().(string)
	}
	return daySlice
}

func validateExpiration() (expiration time.Time, err error) {
	// Validate options
	if len(args.expirationTime) > 0 && args.expirationDuration != 0 {
		err = errors.New("At most one of 'expiration-time' or 'expiration' may be specified")
		return
	}

	// Parse the expiration options
	if len(args.expirationTime) > 0 {
		t, err := parseRFC3339(args.expirationTime)
		if err != nil {
			err = fmt.Errorf("Failed to parse expiration-time: %s", err)
			return expiration, err
		}

		expiration = t
	}
	if args.expirationDuration != 0 {
		// round up to the nearest second
		expiration = time.Now().Add(args.expirationDuration).Round(time.Second)
	}

	return
}

func getTimeOptions() []string {
	timeOptions := []string{}
	for time := 0; time < 24; time++ {
		timeOptions = append(timeOptions, fmt.Sprintf("%d:00", time))
	}
	return timeOptions
}

// parseRFC3339 parses an RFC3339 date in either RFC3339Nano or RFC3339 format.
func parseRFC3339(s string) (time.Time, error) {
	if t, timeErr := time.Parse(time.RFC3339Nano, s); timeErr == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func isExpectedHTTPProxyOrHTTPSProxy(httpProxy, httpsProxy *string, noProxySlice []string, cluster *cmv1.Cluster) bool {
	return httpProxy == nil && httpsProxy == nil && len(noProxySlice) > 0 && cluster.Proxy() == nil
}

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func cronValidator(val interface{}) error {
	if schedule, ok := val.(string); ok {
		if schedule == "" {
			return nil
		}
		_, err := cronParser.Parse(fmt.Sprintf("CRON_TZ=UTC %s", schedule))
		if err != nil {
			return err
		}
		parts := strings.Fields(schedule)
		if _, err := strconv.Atoi(parts[0]); err != nil {
			return fmt.Errorf("The minute value '%s' must be a valid number", parts[0])
		}
		if _, err := strconv.Atoi(parts[1]); err != nil {
			return fmt.Errorf("The hour value '%s' must be a valid number", parts[1])
		}
		if parts[2] != "*" {
			return fmt.Errorf("Setting day of month in the schedule expression is not supported")
		}
		if parts[3] != "*" {
			return fmt.Errorf("Setting a month in the schedule expression is not supported")
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}
