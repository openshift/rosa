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

package addon

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:     "addon ID",
	Aliases: []string{"addons", "add-on", "add-ons"},
	Short:   "Install add-ons on cluster",
	Long:    "Install Red Hat managed add-ons on a cluster",
	Example: `  # Add the CodeReady Workspaces add-on installation to the cluster
  rosa install addon --cluster=mycluster codeready-workspaces`,
	Run:                run,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, argv []string) error {
		err := arguments.ParseUnknownFlags(cmd, argv)
		if err != nil {
			return err
		}

		if len(cmd.Flags().Args()) != 1 {
			return fmt.Errorf("Expected exactly one command line parameter containing the id of the add-on")
		}
		return nil
	},
}

func init() {
	flags := Cmd.Flags()
	confirm.AddFlag(flags)
	ocm.AddClusterFlag(Cmd)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Parse out CLI flags, then override positional arguments
	_ = cmd.Flags().Parse(argv)
	argv = cmd.Flags().Args()
	addOnID := argv[0]

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient := aws.CreateNewClientOrExit(logger, reporter)

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmClient := ocm.CreateNewClientOrExit(logger, reporter)
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

	addOn, err := ocmClient.GetAddOnInstallation(cluster.ID(), addOnID)
	if err != nil && errors.GetType(err) != errors.NotFound {
		reporter.Errorf("An error occurred while trying to get addon installation : %v", err)
		os.Exit(1)
	}
	if addOn != nil {
		reporter.Warnf("Addon '%s' is already installed on cluster '%s'", addOnID, clusterKey)
		os.Exit(0)
	}

	if !confirm.Confirm("install add-on '%s' on cluster '%s'", addOnID, clusterKey) {
		os.Exit(0)
	}

	parameters, err := ocmClient.GetAddOnParameters(cluster.ID(), addOnID)
	if err != nil {
		reporter.Errorf("Failed to get add-on '%s' parameters: %v", addOnID, err)
		os.Exit(1)
	}

	var params []ocm.AddOnParam
	if parameters.Len() > 0 {
		// Determine if all required parameters have already been set as flags and ensure
		// that interactive mode is enabled if they have not. If there are no parameters
		// set as flags, then we also ensure that interactive mode is enabled so that the
		// user gets prompted.
		if arguments.HasUnknownFlags() {
			parameters.Each(func(param *cmv1.AddOnParameter) bool {
				flag := cmd.Flags().Lookup(param.ID())
				if param.Required() && (flag == nil || flag.Value.String() == "") {
					interactive.Enable()
					return false
				}
				return true
			})
		} else {
			interactive.Enable()
		}

		parameters.Each(func(param *cmv1.AddOnParameter) bool {
			var val string
			var hasVal bool
			// If value is already set in the CLI, ignore interactive prompt
			flag := cmd.Flags().Lookup(param.ID())
			if flag != nil {
				val = flag.Value.String()
				hasVal = true
			} else if interactive.Enabled() {
				input := interactive.Input{
					Question: param.Name(),
					Help:     fmt.Sprintf("%s: %s", param.ID(), param.Description()),
					Required: param.Required(),
				}
				// add a prompt to question name to indicate if the boolean param is required and check validation
				if param.ValueType() == "boolean" && param.Validation() == "^true$" && param.Required() {
					input.Question = fmt.Sprintf("%s (required)", param.Name())
					input.Validators = []interactive.Validator{
						interactive.RegExpBoolean(param.Validation()),
					}
				}
				switch param.ValueType() {
				case "boolean":
					var boolVal bool
					input.Default, _ = strconv.ParseBool(param.DefaultValue())
					boolVal, err = interactive.GetBool(input)
					if boolVal {
						val = "true"
					} else {
						val = "false"
					}
				case "cidr":
					var cidrVal net.IPNet
					if param.DefaultValue() != "" {
						_, defaultIDR, _ := net.ParseCIDR(param.DefaultValue())
						input.Default = *defaultIDR
					}
					cidrVal, err = interactive.GetIPNet(input)
					val = cidrVal.String()
					if val == "<nil>" {
						val = ""
					}
				case "number", "resource":
					var numVal int
					input.Default, _ = strconv.Atoi(param.DefaultValue())
					numVal, err = interactive.GetInt(input)
					val = fmt.Sprintf("%d", numVal)
				case "string":
					input.Default = param.DefaultValue()
					val, err = interactive.GetString(input)
				}
				if err != nil {
					reporter.Errorf("Expected a valid value for '%s': %v", param.Name(), err)
					os.Exit(1)
				}
				hasVal = true
			}

			if hasVal {
				val = strings.Trim(val, " ")
				if val != "" && param.Validation() != "" {
					isValid, err := regexp.MatchString(param.Validation(), val)
					if err != nil || !isValid {
						reporter.Errorf("Expected %v to match /%s/", val, param.Validation())
						os.Exit(1)
					}
				}
				params = append(params, ocm.AddOnParam{Key: param.ID(), Val: val})
			}

			return true
		})
	}

	reporter.Debugf("Installing add-on '%s' on cluster '%s'", addOnID, clusterKey)
	err = ocmClient.InstallAddOn(cluster.ID(), addOnID, params)
	if err != nil {
		reporter.Errorf("Failed to add add-on installation '%s' for cluster '%s': %v", addOnID, clusterKey, err)
		os.Exit(1)
	}
	reporter.Infof("Add-on '%s' is now installing. To check the status run 'rosa list addons -c %s'", addOnID, clusterKey)
	if interactive.Enabled() {
		reporter.Infof("To install this addOn again in the future, you can run:\n   %s",
			buildCommand(cluster.Name(), addOnID, params))
	}
}

func buildCommand(clusterName string, addonName string, params []ocm.AddOnParam) string {
	command := fmt.Sprintf("rosa install addon --cluster %s %s -y", clusterName, addonName)

	for _, param := range params {
		if param.Val != "" {
			command += fmt.Sprintf(" --%s %s", param.Key, param.Val)
		}
	}

	return command
}
