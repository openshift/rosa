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
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
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
	additionalTrustBundleFile string
}

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Edit cluster",
	Long:  "Edit cluster.",
	Example: `  # Edit a cluster named "mycluster" to make it private
  rosa edit cluster mycluster --private

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

	flags.StringVar(
		&args.additionalTrustBundleFile,
		"additional-trust-bundle-file",
		"",
		"A file contains a PEM-encoded X.509 certificate bundle that will be "+
			"added to the nodes' trusted certificate store.")
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Enable interactive mode if no flags have been set
	if !interactive.Enabled() {
		changedFlags := false
		for _, flag := range []string{"expiration-time", "expiration", "private",
			"disable-workload-monitoring", "http-proxy", "https-proxy", "additional-trust-bundle-file"} {
			if cmd.Flags().Changed(flag) {
				changedFlags = true
			}
		}
		if !changedFlags {
			interactive.Enable()
		}
	}

	logger := logging.CreateLoggerOrExit(reporter)

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

	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Validate flags:
	expiration, err := validateExpiration()
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	if interactive.Enabled() {
		reporter.Infof("Interactive mode enabled.\n" +
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
	var additionalTrustBundleFile *string
	var additionalTrustBundleFileValue string
	if cmd.Flags().Changed("additional-trust-bundle-file") {
		additionalTrustBundleFileValue = args.additionalTrustBundleFile
		additionalTrustBundleFile = &additionalTrustBundleFileValue
	}

	if httpProxy != nil || httpsProxy != nil || additionalTrustBundleFile != nil {
		enableProxy = true
		useExistingVPC = true
	}

	if len(cluster.AWS().SubnetIDs()) == 0 &&
		((httpProxy != nil && *httpProxy != "") || (httpsProxy != nil && *httpsProxy != "") ||
			(additionalTrustBundleFile != nil && *additionalTrustBundleFile != "")) {
		reporter.Errorf("Cluster-wide proxy is not supported on clusters using the default VPC")
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
			reporter.Errorf("Expected a valid private value: %s", err)
			os.Exit(1)
		}
		private = &privateValue
	} else if privateValue {
		reporter.Warnf("You are choosing to make your cluster API private. %s", privateWarning)
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
			reporter.Errorf("Expected a valid disable-workload-monitoring value: %v", err)
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
			reporter.Errorf("Expected a valid proxy-enabled value: %s", err)
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
			reporter.Errorf("Expected a valid http proxy: %s", err)
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
			reporter.Errorf("%s", err)
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
				// received double quotes from the iser. need to remove the existing value
				def = doubleQuotesToRemove
			}
		}
		httpsProxyValue, err = interactive.GetString(interactive.Input{
			Question: "HTTPS proxy",
			Help:     cmd.Flags().Lookup("https-proxy").Usage,
			Default:  def,
		})
		if err != nil {
			reporter.Errorf("Expected a valid https proxy: %s", err)
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
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	/*******  AdditionalTrustBundle *******/
	updateAdditionalTrustBundle := false
	if additionalTrustBundleFile != nil {
		updateAdditionalTrustBundle = true
	}
	if useExistingVPC && enableProxy && !updateAdditionalTrustBundle && additionalTrustBundleFile == nil &&
		interactive.Enabled() {
		updateAdditionalTrustBundleValue, err := interactive.GetBool(interactive.Input{
			Question: "Update additional trust bundle",
			Default:  updateAdditionalTrustBundle,
		})
		if err != nil {
			reporter.Errorf("Expected a valid -update-additional-trust-bundle value: %s", err)
			os.Exit(1)
		}
		updateAdditionalTrustBundle = updateAdditionalTrustBundleValue
	}
	if enableProxy && updateAdditionalTrustBundle && interactive.Enabled() {
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
			reporter.Errorf("Expected a valid additional trust bundle file name: %s", err)
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
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	if enableProxy && httpProxy == nil && httpsProxy == nil && additionalTrustBundleFile == nil {
		reporter.Errorf("Expected at least one of the following: http-proxy, https-proxy, additional-trust-bundle")
		os.Exit(1)
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
	if additionalTrustBundleFile != nil {
		clusterConfig.AdditionalTrustBundle = new(string)
		if *additionalTrustBundleFile == doubleQuotesToRemove {
			*clusterConfig.AdditionalTrustBundle = *additionalTrustBundleFile
		} else {
			// Get certificate contents
			if len(*additionalTrustBundleFile) > 0 {
				cert, err := ioutil.ReadFile(*additionalTrustBundleFile)
				if err != nil {
					reporter.Errorf("Failed to read additional trust bundle file: %s", err)
					os.Exit(1)
				}
				*clusterConfig.AdditionalTrustBundle = string(cert)
			}
		}
	}

	reporter.Debugf("Updating cluster '%s'", clusterKey)
	err = ocmClient.UpdateCluster(clusterKey, awsCreator, clusterConfig)
	if err != nil {
		reporter.Errorf("Failed to update cluster: %v", err)
		os.Exit(1)
	}
	reporter.Infof("Updated cluster '%s'", clusterKey)
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

// parseRFC3339 parses an RFC3339 date in either RFC3339Nano or RFC3339 format.
func parseRFC3339(s string) (time.Time, error) {
	if t, timeErr := time.Parse(time.RFC3339Nano, s); timeErr == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}
