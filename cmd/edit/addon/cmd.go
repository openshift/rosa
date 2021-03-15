/*
Copyright (c) 2021 Red Hat, Inc.

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

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	clusterprovider "github.com/openshift/rosa/pkg/cluster"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "addon ID",
	Aliases: []string{"addons", "add-on", "add-ons"},
	Short:   "Edit add-on installation parameters on cluster",
	Long:    "Edit the parameters on installed Red Hat managed add-ons on a cluster",
	Example: `  # Edit the parameters of the Red Hat OpenShift logging operator add-on installation
  rosa edit addon --cluster=mycluster cluster-logging-operator`,
	Run:                run,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, argv []string) error {
		err := arguments.ParseUnknownFlags(cmd, argv)
		if err != nil {
			return fmt.Errorf("Failed to parse flags: %v", err)
		}

		if len(cmd.Flags().Args()) != 1 {
			return fmt.Errorf("Expected exactly one command line parameter containing the id of the add-on")
		}
		return nil
	},
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to edit the addon parameters of (required).",
	)
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Parse out CLI flags, then override positional arguments
	_ = cmd.Flags().Parse(argv)
	argv = cmd.Flags().Args()
	addOnID := argv[0]

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !clusterprovider.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
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
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Get the client for the OCM collection of clusters:
	ocmClient := ocmConnection.ClustersMgmt().V1()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := clusterprovider.GetCluster(ocmClient.Clusters(), clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	parameters, err := clusterprovider.GetAddOnParameters(ocmClient.Addons(), addOnID)
	if err != nil {
		reporter.Errorf("Failed to get add-on '%s' parameters: %v", addOnID, err)
		os.Exit(1)
	}

	addOnInstallation, err := clusterprovider.GetAddOnInstallation(ocmClient.Clusters(),
		clusterKey, awsCreator.ARN, addOnID)
	if err != nil {
		reporter.Errorf("Failed to get add-on '%s' installation: %v", addOnID, err)
		os.Exit(1)
	}
	if addOnInstallation.State() != cmv1.AddOnInstallationStateReady {
		reporter.Errorf("Add-on '%s' is not yet ready on cluster '%s'", addOnID, clusterKey)
		os.Exit(1)
	}

	if parameters.Len() == 0 {
		reporter.Errorf("Add-on '%s' has no parameters to edit", addOnID)
		os.Exit(1)
	}

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

	var params []clusterprovider.AddOnParam
	parameters.Each(func(param *cmv1.AddOnParameter) bool {
		// Find the installation parameter corresponding to the addon parameter
		var addOnInstallationParam *cmv1.AddOnInstallationParameter
		addOnInstallation.Parameters().Each(func(p *cmv1.AddOnInstallationParameter) bool {
			if p.ID() == param.ID() {
				addOnInstallationParam = p
				return false
			}
			return true
		})

		// If the parameter already exists in the cluster and is not editable, hide it
		if addOnInstallationParam != nil && !param.Editable() {
			return true
		}

		var val string
		// If value is already set in the CLI, ignore interactive prompt
		flag := cmd.Flags().Lookup(param.ID())
		if flag != nil {
			val = flag.Value.String()
		} else if interactive.Enabled() {
			// Set default value based on existing parameter, otherwise use parameter default
			dflt := param.DefaultValue()
			if addOnInstallationParam != nil {
				dflt = addOnInstallationParam.Value()
			}

			input := interactive.Input{
				Question: param.Name(),
				Help:     fmt.Sprintf("%s: %s", param.ID(), param.Description()),
				Required: param.Required(),
			}

			switch param.ValueType() {
			case "boolean":
				var boolVal bool
				input.Default, _ = strconv.ParseBool(dflt)
				boolVal, err = interactive.GetBool(input)
				if boolVal {
					val = "true"
				} else {
					val = "false"
				}
			case "cidr":
				var cidrVal net.IPNet
				if dflt != "" {
					_, defaultIDR, _ := net.ParseCIDR(dflt)
					input.Default = *defaultIDR
				}
				cidrVal, err = interactive.GetIPNet(input)
				val = cidrVal.String()
				if val == "<nil>" {
					val = ""
				}
			case "number":
				var numVal int
				input.Default, _ = strconv.Atoi(dflt)
				numVal, err = interactive.GetInt(input)
				val = fmt.Sprintf("%d", numVal)
			case "string":
				input.Default = dflt
				val, err = interactive.GetString(input)
			}
			if err != nil {
				reporter.Errorf("Expected a valid value for '%s': %v", param.ID(), err)
				os.Exit(1)
			}
		}

		val = strings.Trim(val, " ")
		if val != "" && param.Validation() != "" {
			isValid, err := regexp.MatchString(param.Validation(), val)
			if err != nil || !isValid {
				reporter.Errorf("Expected %v to match /%s/", val, param.Validation())
				os.Exit(1)
			}
		}

		params = append(params, clusterprovider.AddOnParam{Key: param.ID(), Val: val})

		return true
	})

	reporter.Debugf("Updating add-on parameters for '%s' on cluster '%s'", addOnID, clusterKey)
	err = clusterprovider.UpdateAddOnInstallation(ocmClient.Clusters(), clusterKey, awsCreator.ARN, addOnID, params)
	if err != nil {
		reporter.Errorf("Failed to update add-on installation '%s' for cluster '%s': %v", addOnID, clusterKey, err)
		os.Exit(1)
	}
	reporter.Infof("Add-on '%s' is now updating. To check the status run 'rosa list addons -c %s'", addOnID, clusterKey)
}
