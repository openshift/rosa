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
	"os"
	"strings"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/clusterregistryconfig"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/helper/roles"
	"github.com/openshift/rosa/pkg/input"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const enableDeleteProtectionFlagName = "enable-delete-protection"

// SDN -> OVN Migration
const networkTypeFlagName = "network-type"
const ovnInternalSubnetsFlagName = "ovn-internal-subnets"
const networkTypeOvn = "OVNKubernetes"
const subnetConfigTransit = "transit"
const subnetConfigJoin = "join"
const subnetConfigMasquerade = "masquerade"

var args struct {
	// Basic options
	expirationTime         string
	expirationDuration     time.Duration
	enableDeleteProtection bool

	// Networking options
	private                   bool
	disableWorkloadMonitoring bool
	httpProxy                 string
	httpsProxy                string
	noProxySlice              []string
	additionalTrustBundleFile string

	// Audit log forwarding
	auditLogRoleARN string

	// HCP options:
	billingAccount string

	// Other options
	additionalAllowedPrincipals []string

	// Cluster Registry
	allowedRegistries          []string
	blockedRegistries          []string
	insecureRegistries         []string
	allowedRegistriesForImport string
	platformAllowlist          string
	additionalTrustedCa        string

	// SDN -> OVN Migration
	networkType        string
	ovnInternalSubnets string
}

var clusterRegistryConfigArgs *clusterregistryconfig.ClusterRegistryConfigArgs

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Edit cluster",
	Long:  "Edit cluster.",
	Example: `  # Edit a cluster named "mycluster" to make it private
  rosa edit cluster -c mycluster --private

  # Edit a cluster named "mycluster" to enable User Workload Monitoring
  rosa edit cluster -c mycluster --disable-workload-monitoring=false

  # Edit all options interactively
  rosa edit cluster -c mycluster --interactive`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)
	confirm.AddFlag(Cmd.Flags())

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
	flags.BoolVar(
		&args.enableDeleteProtection,
		enableDeleteProtectionFlagName,
		false,
		"Toggle cluster deletion protection against accidental cluster deletion.",
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
		&args.auditLogRoleARN,
		"audit-log-arn",
		"",
		"The ARN of the role that is used to forward audit logs to AWS CloudWatch.",
	)

	flags.StringSliceVar(
		&args.additionalAllowedPrincipals,
		"additional-allowed-principals",
		nil,
		"A comma-separated list of additional allowed principal ARNs "+
			"to be added to the Hosted Control Plane's VPC Endpoint Service to enable additional "+
			"VPC Endpoint connection requests to be automatically accepted.",
	)

	clusterRegistryConfigArgs = clusterregistryconfig.AddClusterRegistryConfigFlags(Cmd)

	flags.StringVar(
		&args.billingAccount,
		"billing-account",
		"",
		"Account ID used for billing subscriptions purchased through the AWS console for ROSA",
	)

	flags.StringVar(
		&args.networkType,
		"network-type",
		"",
		"Migrate a cluster's network type from OpenShiftSDN to OVN-Kubernetes",
	)

	flags.StringVar(
		&args.ovnInternalSubnets,
		ovnInternalSubnetsFlagName,
		"",
		"OVN-Kubernetes internal subnet configuration for migrating 'network-type' from OpenShiftSDN -> "+
			"OVN-Kubernetes. Choices consist of 'transit', 'join', or 'masquerade'",
	)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	// Enable interactive mode if no flags have been set
	if !interactive.Enabled() {
		changedFlags := false
		for _, flag := range []string{"expiration-time", "expiration", "private",
			"disable-workload-monitoring", "http-proxy", "https-proxy", "no-proxy",
			"additional-trust-bundle-file", "additional-allowed-principals", "audit-log-arn",
			"registry-config-allowed-registries", "registry-config-blocked-registries",
			"registry-config-insecure-registries", "allowed-registries-for-import",
			"registry-config-platform-allowlist", "registry-config-additional-trusted-ca", "billing-account",
			"registry-config-allowed-registries-for-import", "enable-delete-protection"} {
			if cmd.Flags().Changed(flag) {
				changedFlags = true
				break
			}
		}
		if !changedFlags {
			interactive.Enable()
		}
	}

	cluster := r.FetchCluster()

	// Validate flags:
	expiration, err := validateExpiration()
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	ovnInternalSubnets, err := validateOvnInternalSubnetConfiguration()
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	networkType, err := validateNetworkType()
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

	var additionalAllowedPrincipals []string
	if cmd.Flags().Changed("additional-allowed-principals") {
		additionalAllowedPrincipals = args.additionalAllowedPrincipals
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
		"endpoints, use the 'rosa edit ingress' command. "
	privateWarning, err = warnUserForOAuthHCPVisibility(r, clusterKey, cluster, privateWarning)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}
	if interactive.Enabled() {
		privateValue, err = interactive.GetBool(interactive.Input{
			Question: "Private cluster, check this command's help for possible impacts",
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
				def = input.DoubleQuotesToRemove
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

		if def != httpProxyValue && httpProxyValue == input.DoubleQuotesToRemove {
			//user entered double quotes ("") to remove the existing value
			httpProxy = new(string)
			*httpProxy = ""
		} else {
			httpProxy = &httpProxyValue
		}
	}
	if httpProxy != nil && *httpProxy != input.DoubleQuotesToRemove {
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
				def = input.DoubleQuotesToRemove
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
		if def != httpsProxyValue && httpsProxyValue == input.DoubleQuotesToRemove {
			//user entered double quotes ("") to remove the existing value
			httpsProxy = new(string)
			*httpsProxy = ""
		} else {
			httpsProxy = &httpsProxyValue
		}
	}
	if httpsProxy != nil && *httpsProxy != input.DoubleQuotesToRemove {
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
		if len(noProxySlice) == 1 && noProxySlice[0] == input.DoubleQuotesToRemove {
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
				def = input.DoubleQuotesToRemove
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
		} else if additionalTrustBundleFileValue == input.DoubleQuotesToRemove {
			//user entered double quotes ("") to remove the existing value
			additionalTrustBundleFile = new(string)
			*additionalTrustBundleFile = ""
		} else {
			additionalTrustBundleFile = &additionalTrustBundleFileValue
		}
	}
	if additionalTrustBundleFile != nil && *additionalTrustBundleFile != input.DoubleQuotesToRemove {
		err = ocm.ValidateAdditionalTrustBundle(*additionalTrustBundleFile)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	/*******  AdditionalAllowedPrincipals *******/
	updateAdditionalAllowedPrincipals := false
	if additionalAllowedPrincipals != nil {
		updateAdditionalAllowedPrincipals = true
	}
	if !updateAdditionalAllowedPrincipals && additionalAllowedPrincipals == nil &&
		interactive.Enabled() {
		updateAdditionalAllowedPrincipalsValue, err := interactive.GetBool(interactive.Input{
			Question: "Update additional allowed principals",
			Default:  updateAdditionalAllowedPrincipals,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid update-additional-allowed-principals value: %s", err)
			os.Exit(1)
		}
		updateAdditionalAllowedPrincipals = updateAdditionalAllowedPrincipalsValue
	}
	if updateAdditionalAllowedPrincipals && interactive.Enabled() {
		aapInputs, err := interactive.GetString(interactive.Input{
			Question: "Additional Allowed Principal ARNs",
			Help:     cmd.Flags().Lookup("additional-allowed-principals").Usage,
			Default:  cluster.AWS().AdditionalAllowedPrincipals(),
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for Additional Allowed Principal ARNs: %s", err)
			os.Exit(1)
		}
		additionalAllowedPrincipals = helper.HandleEmptyStringOnSlice(strings.Split(aapInputs, ","))
	}
	if len(additionalAllowedPrincipals) > 0 {
		if len(additionalAllowedPrincipals) == 1 &&
			additionalAllowedPrincipals[0] == input.DoubleQuotesToRemove {
			additionalAllowedPrincipals = []string{}
		} else {
			if err := roles.ValidateAdditionalAllowedPrincipals(additionalAllowedPrincipals); err != nil {
				r.Reporter.Errorf(err.Error())
				os.Exit(1)
			}
		}
	}

	// Audit Log Forwarding
	auditLogRole, err := setAuditLogForwarding(r, cmd, cluster, args.auditLogRoleARN)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if interactive.Enabled() && aws.IsHostedCP(cluster) {
		auditLogRole, err = auditLogInteractivePrompt(r, cmd, cluster)
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
		if *additionalTrustBundleFile == input.DoubleQuotesToRemove {
			*clusterConfig.AdditionalTrustBundle = *additionalTrustBundleFile
		} else {
			// Get certificate contents
			if len(*additionalTrustBundleFile) > 0 {
				cert, err := os.ReadFile(*additionalTrustBundleFile)
				if err != nil {
					r.Reporter.Errorf("Failed to read additional trust bundle file: %s", err)
					os.Exit(1)
				}
				*clusterConfig.AdditionalTrustBundle = string(cert)
			}
		}
	}

	if additionalAllowedPrincipals != nil {
		clusterConfig.AdditionalAllowedPrincipals = additionalAllowedPrincipals
	}

	clusterRegistryConfigArgs, err = clusterregistryconfig.GetClusterRegistryConfigOptions(
		cmd.Flags(), clusterRegistryConfigArgs, aws.IsHostedCP(cluster), cluster)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if clusterRegistryConfigArgs != nil {
		allowedRegistries, blockedRegistries, insecureRegistries,
			additionalTrustedCa, allowedRegistriesForImport,
			platformAllowlist := clusterregistryconfig.GetClusterRegistryConfigArgs(
			clusterRegistryConfigArgs)

		// prompt for a warning if any registry config field is set
		if allowedRegistries != nil || blockedRegistries != nil || insecureRegistries != nil ||
			additionalTrustedCa != "" || allowedRegistriesForImport != "" || platformAllowlist != "" {
			if PromptUserToAcceptRegistryChange(r) {
				clusterConfig, err = BuildClusterConfigWithRegistry(clusterConfig, allowedRegistries,
					blockedRegistries, insecureRegistries,
					additionalTrustedCa, allowedRegistriesForImport, platformAllowlist)
			}
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
	}
	if auditLogRole != nil {
		clusterConfig.AuditLogRoleARN = new(string)
		*clusterConfig.AuditLogRoleARN = *auditLogRole
	}

	// Deletion Protection
	var deleteProtection bool
	if !cmd.Flags().Changed(enableDeleteProtectionFlagName) {
		deleteProtection = cluster.DeleteProtection().Enabled()
	} else {
		deleteProtection = args.enableDeleteProtection
	}

	if interactive.Enabled() {
		deleteProtection, err = interactive.GetBool(interactive.Input{
			Question: "Enable cluster deletion protection",
			Help:     cmd.Flags().Lookup(enableDeleteProtectionFlagName).Usage,
			Default:  deleteProtection,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value: %v", err)
			os.Exit(1)
		}
	}

	if cluster.DeleteProtection().Enabled() != deleteProtection {
		r.Reporter.Debugf("Updating cluster deletion protection to : %t", deleteProtection)
		newDeleteProtection, err := cmv1.NewDeleteProtection().Enabled(deleteProtection).Build()
		if err != nil {
			r.Reporter.Errorf("Failed to build delete protection: %v", err)
			os.Exit(1)
		}

		if err := r.OCMClient.UpdateClusterDeletionProtection(cluster.ID(), newDeleteProtection); err != nil {
			r.Reporter.Errorf("Failed to update cluster delete protection: %v", err)
			os.Exit(1)
		}
	}

	// SDN -> OVN Migration
	var clusterNetworkType string
	if !cmd.Flags().Changed(networkTypeFlagName) {
		var ok bool
		if cluster.Network() == nil {
			ok = false
		} else {
			networkType, ok = cluster.Network().GetType()
			clusterNetworkType = networkType // Store the cluster's current network type for interactive usage
		}
		if !ok {
			r.Reporter.Errorf("Unable to get cluster's network type")
			os.Exit(1)
		}
	}

	var migrateNetworkType bool
	// Only prompt user with migrating the cluster's network type when it is not OVN-Kubernetes
	if interactive.Enabled() && clusterNetworkType != "" && clusterNetworkType != networkTypeOvn {
		migrateNetworkType, err = interactive.GetBool(interactive.Input{
			Question: "Migrate cluster network type from OpenShiftSDN -> OVN-Kubernetes",
			Help: "Clusters are required to migrate from network type 'OpenShiftSDN' to 'OVN-Kubernetes', this allows " +
				"you to do this along with your cluster changes",
			Default: false,
		})

		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if migrateNetworkType {
			migrateNetworkType, err = confirmMigration()

			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}

		if migrateNetworkType {
			networkType, err = interactive.GetString(interactive.Input{
				Question: "Network type for cluster",
				Help:     cmd.Flags().Lookup(networkTypeFlagName).Usage,
				Default:  networkTypeOvn,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value: %v", err)
				os.Exit(1)
			}

			ovnInternalSubnets, err = interactive.GetString(interactive.Input{
				Question: "OVN-Kubernetes internal subnet configuration for cluster",
				Help:     cmd.Flags().Lookup(ovnInternalSubnetsFlagName).Usage,
				Default:  ovnInternalSubnets,
				Options:  []string{subnetConfigTransit, subnetConfigJoin, subnetConfigMasquerade},
				Required: false,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value: %v", err)
			}
		}
	}

	if cmd.Flags().Changed(networkTypeFlagName) && networkType == networkTypeOvn {
		migrateNetworkType, err = confirmMigration()

		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	if networkType == networkTypeOvn && migrateNetworkType {
		clusterConfig.NetworkType = networkType

		if helper.Contains([]string{subnetConfigTransit, subnetConfigJoin, subnetConfigMasquerade}, ovnInternalSubnets) {
			clusterConfig.SubnetConfiguration = ovnInternalSubnets
		}
	}

	var billingAccount string
	if cmd.Flags().Changed("billing-account") {
		billingAccount = args.billingAccount

		if billingAccount != "" && !aws.IsHostedCP(cluster) {
			r.Reporter.Errorf("Billing accounts are only supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		if billingAccount != "" && !ocm.IsValidAWSAccount(billingAccount) {
			r.Reporter.Errorf("Provided billing account number %s is not valid. "+
				"Rerun the command with a valid billing account number", billingAccount)
			os.Exit(1)
		}
	} else {
		billingAccount = cluster.AWS().BillingAccountID()
	}

	if interactive.Enabled() && aws.IsHostedCP(cluster) {
		cloudAccounts, err := r.OCMClient.GetBillingAccounts()
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		billingAccounts := ocm.GenerateBillingAccountsList(cloudAccounts)
		if len(billingAccounts) > 0 {
			billingAccount, err = interactive.GetOption(interactive.Input{
				Question:       "Update billing account",
				Help:           cmd.Flags().Lookup("billing-account").Usage,
				Default:        billingAccount,
				DefaultMessage: fmt.Sprintf("current = '%s'", cluster.AWS().BillingAccountID()),
				Required:       true,
				Options:        billingAccounts,
			})

			if err != nil {
				r.Reporter.Errorf("Expected a valid billing account: '%s'", err)
				os.Exit(1)
			}

			billingAccount = aws.ParseOption(billingAccount)
		}

		err = ocm.ValidateBillingAccount(billingAccount)
		if err != nil {
			r.Reporter.Errorf("%v", err)
			os.Exit(1)
		}

		// Get contract info
		contracts, isContractEnabled := ocm.GetBillingAccountContracts(cloudAccounts, billingAccount)

		if billingAccount != r.Creator.AccountID {
			r.Reporter.Infof(
				"The AWS billing account you selected is different from your AWS infrastructure account. " +
					"The AWS billing account will be charged for subscription usage. " +
					"The AWS infrastructure account contains the ROSA infrastructure.",
			)
		}

		if isContractEnabled && len(contracts) > 0 {
			//currently, an AWS account will have only one ROSA HCP active contract at a time
			contractDisplay := ocm.GenerateContractDisplay(contracts[0])
			r.Reporter.Infof(contractDisplay)
		}
	}

	// sets the billing account only if it has changed
	if billingAccount != "" && billingAccount != cluster.AWS().BillingAccountID() {
		clusterConfig.BillingAccount = billingAccount
	}

	r.Reporter.Debugf("Updating cluster '%s'", clusterKey)
	err = r.OCMClient.UpdateCluster(cluster.ID(), r.Creator, clusterConfig)
	if err != nil {
		r.Reporter.Errorf("Failed to update cluster: %v", err)
		os.Exit(1)
	}
	r.Reporter.Infof("Updated cluster '%s'", clusterKey)
}

// warnUserForOAuthHCPVisibility is a method for HCP only that checks if the user has public ingress and warns them
// about how changing cluster visibility may impact them
func warnUserForOAuthHCPVisibility(r *rosa.Runtime, clusterKey string, cluster *cmv1.Cluster,
	privateWarning string) (string, error) {
	if !cluster.Hypershift().Enabled() {
		return privateWarning, nil
	}
	// if ingress visibility public, warning
	r.Reporter.Debugf("Loading ingresses for cluster '%s'", clusterKey)
	ingresses, err := r.OCMClient.GetIngresses(cluster.ID())
	if err != nil {
		return "", fmt.Errorf("failed to get ingresses for cluster '%s': %v", clusterKey, err)
	}
	publicIngresses := make([]string, 0)
	for _, ingress := range ingresses {
		if ingress.Listening() == cmv1.ListeningMethodExternal {
			publicIngresses = append(publicIngresses, ingress.ID())
		}
	}
	// No public ingresses, nothing to report back
	if len(publicIngresses) == 0 {
		return privateWarning, nil
	}

	privateWarning += fmt.Sprintf("OAuth visibility will be affected by cluster visibility change. "+
		"Any application using OAuth behind a public ingress like the OpenShift Console will not be accessible "+
		"anymore unless the user already has access to the private network. List of affected public ingresses: %s",
		strings.Join(publicIngresses, ","))

	return privateWarning, nil

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

// SDN -> OVN migration subnet configuration validator
func validateOvnInternalSubnetConfiguration() (ovnInternalSubnets string, err error) {
	if len(args.ovnInternalSubnets) > 0 {
		if args.networkType == "" {
			err = fmt.Errorf("Expected a value for %s when supplying the flag %s", networkTypeFlagName,
				ovnInternalSubnetsFlagName)
			return
		}
		if !helper.Contains([]string{
			subnetConfigTransit,
			subnetConfigJoin,
			subnetConfigMasquerade,
		}, args.ovnInternalSubnets) {
			err = fmt.Errorf("Incorrect option for '%s', please use one of '%s', '%s', or '%s'",
				ovnInternalSubnetsFlagName, subnetConfigTransit, subnetConfigJoin, subnetConfigMasquerade)
		} else {
			ovnInternalSubnets = args.ovnInternalSubnets
		}
	}
	return
}

// SDN -> OVN migration network type validator (one option)
func validateNetworkType() (networkConfig string, err error) {
	if len(args.networkType) > 0 {
		if args.networkType != networkTypeOvn {
			err = fmt.Errorf("Incorrect network type '%s', please use '%s' or remove the flag",
				args.networkType, networkTypeOvn)
		} else {
			networkConfig = args.networkType
		}
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

func isExpectedHTTPProxyOrHTTPSProxy(httpProxy, httpsProxy *string, noProxySlice []string, cluster *cmv1.Cluster) bool {
	return httpProxy == nil && httpsProxy == nil && len(noProxySlice) > 0 && cluster.Proxy() == nil
}

func auditLogRoleExists(cluster *cmv1.Cluster) bool {
	return cluster.AWS().AuditLog().RoleArn() != ""
}

func auditLogEnableOrUpdatePrompt(currentAuditLogRole string) string {
	if currentAuditLogRole == "" {
		return "Enable audit log forwarding to AWS CloudWatch"
	}
	return fmt.Sprintf("Update existing audit log forwarding role '%s'", currentAuditLogRole)
}

func setAuditLogForwarding(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster, auditLogArn string) (
	argValuePtr *string, err error) {
	if cmd.Flags().Changed("audit-log-arn") {
		if !aws.IsHostedCP(cluster) {
			return nil, fmt.Errorf("Audit log forwarding to AWS CloudWatch is only supported for Hosted Control Plane clusters")

		}
		if auditLogArn != "" && !aws.RoleArnRE.MatchString(auditLogArn) {
			return nil, fmt.Errorf("Expected a valid value for audit-log-arn matching %s", aws.RoleArnRE.String())
		}
		argValuePtr := new(string)
		*argValuePtr = auditLogArn

		confirmAuditLogForwarding(r, argValuePtr)
		return argValuePtr, nil
	}
	return nil, nil
}

func confirmAuditLogForwarding(r *rosa.Runtime, auditLogArn *string) {

	if *auditLogArn != "" {
		r.Reporter.Warnf("You are choosing to enable audit log forwarding")
		if !confirm.Confirm("enable audit log forwarding for cluster with the provided role arn '%s'", *auditLogArn) {
			os.Exit(0)
		}
		return
	}
	r.Reporter.Warnf("You are choosing to disable audit log forwarding.")
	if !confirm.Confirm("disable audit log forwarding for cluster") {
		os.Exit(0)
	}
}

func auditLogInteractivePrompt(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster) (
	argValuePtr *string, err error) {

	auditLogRolePtr := new(string)

	requestAuditLogForwarding, err := interactive.GetBool(interactive.Input{
		Question: auditLogEnableOrUpdatePrompt(cluster.AWS().AuditLog().RoleArn()),
		Default:  false,
		Required: true,
	})
	if err != nil {
		return nil, fmt.Errorf("Expected a valid value: %s", err)
	}
	if requestAuditLogForwarding {

		r.Reporter.Infof("To configure the audit log forwarding role in your AWS account, " +
			"please refer to steps 1 through 6: https://access.redhat.com/solutions/7002219")

		auditLogRoleValue, err := interactive.GetString(interactive.Input{
			Question: "Audit log forwarding role ARN",
			Help:     cmd.Flags().Lookup("audit-log-arn").Usage,
			Default:  "",
			Required: true,
			Validators: []interactive.Validator{
				interactive.RegExp(aws.RoleArnRE.String()),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("Expected a valid value for audit-log-arn: %s", err)
		}
		*auditLogRolePtr = auditLogRoleValue
		return auditLogRolePtr, nil
	}

	if auditLogRoleExists(cluster) && !requestAuditLogForwarding {
		disableAuditLog, err := interactive.GetBool(interactive.Input{
			Question: "Disable Audit Log",
			Help:     "Disable AWS CloudWatch audit log forwarding that is currently enabled.",
			Default:  false,
		})
		if err != nil {
			return nil, fmt.Errorf("Expected a valid value: %s", err)
		}
		if disableAuditLog {
			*auditLogRolePtr = ""
			return auditLogRolePtr, nil
		}
	}
	return
}

func PromptUserToAcceptRegistryChange(r *rosa.Runtime) bool {
	prompt := "Changing any registry related parameter will trigger a rollout across all machinepools " +
		"(all machinepool nodes will be recreated, following pod draining from each node). Do you want to proceed?"
	if !confirm.ConfirmRaw(prompt) {
		r.Reporter.Warnf("No changes to registry configuration.")
		return false
	}
	return true
}

func BuildClusterConfigWithRegistry(clusterConfig ocm.Spec, allowedRegistries []string,
	blockedRegistries []string, insecureRegistries []string, additionalTrustedCa string,
	allowedRegistriesForImport string, platformAllowlist string) (ocm.Spec, error) {
	clusterConfig.AllowedRegistries = allowedRegistries
	clusterConfig.BlockedRegistries = blockedRegistries
	clusterConfig.InsecureRegistries = insecureRegistries
	clusterConfig.PlatformAllowlist = platformAllowlist
	if additionalTrustedCa != "" {
		ca, err := clusterregistryconfig.BuildAdditionalTrustedCAFromInputFile(additionalTrustedCa)
		if err != nil {
			return clusterConfig, fmt.Errorf(
				"Failed to build the additional trusted ca from file %s, got error: %s",
				additionalTrustedCa, err)
		}
		clusterConfig.AdditionalTrustedCa = ca
		clusterConfig.AdditionalTrustedCaFile = additionalTrustedCa
	}
	clusterConfig.AllowedRegistriesForImport = allowedRegistriesForImport
	return clusterConfig, nil
}

func confirmMigration() (bool, error) {
	return interactive.GetBool(interactive.Input{
		Question: "Changing the network plugin will reboot cluster nodes, can not be interrupted or rolled " +
			"back, and can not be combined with other operations such as cluster upgrades. \n\nConfirm that " +
			"you want to proceed with migrating from 'OpenShiftSDN' to 'OVN-Kubernetes",
		Help: "Confirm that you are wanting to migrate your cluster's network type, it may be safer to do " +
			"this migration with no other changes",
		Default: false,
	})
}
