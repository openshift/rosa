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

package oidcprovider

import (
	"fmt"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"
	"os"
)

var modes []string = []string{"auto", "manual"}

var args struct {
	clusterKey string
	mode       string
}

var Cmd = &cobra.Command{
	Use:     "oidc-provider",
	Aliases: []string{"oidcprovider"},
	Short:   "Delete OIDC Provider",
	Long:    "Cleans up OIDC provider of deleted STS cluster.",
	Example: `  # Delete OIDC provider for cluster named "mycluster"
  rosa delete oidc-provider --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"ID of the cluster (deleted/archived) to delete the OIDC provider from (required).",
	)
	Cmd.MarkFlagRequired("cluster")

	flags.StringVar(
		&args.mode,
		"mode",
		modes[0],
		"How to perform the operation. Valid options are:\n"+
			"auto: OIDC provider will be deleted using the current AWS account\n"+
			"manual: Command to delete the OIDC provider will be output",
	)
	Cmd.RegisterFlagCompletionFunc("mode", modeCompletion)
	confirm.AddFlag(flags)
}

func modeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return modes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)
	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		args.clusterKey = argv[0]
	}
	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !ocm.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster identifier '%s' isn't valid",
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

	creator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get IAM credentials: %s", err)
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

	c, err := ocmClient.GetClusterByID(clusterKey, creator)
	if err != nil {
		if errors.GetType(err) != errors.NotFound {
			reporter.Errorf("Error validating cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	}
	if c != nil && c.ID() != "" {
		reporter.Errorf("Cluster '%s' is in '%s' state. OIDC provider can be deleted only for the "+
			"uninstalled clusters", c.ID(), c.State())
		os.Exit(1)
	}

	providerARN, err := awsClient.GetOpenIDConnectProvider(clusterKey)
	if err != nil {
		reporter.Errorf("Failed to get the OIDC provider for cluster '%s'.", clusterKey)
		os.Exit(1)
	}
	if providerARN == "" {
		reporter.Infof("Cluster '%s' doesn't have OIDC provider associated with it.", clusterKey)
		return
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}
	mode := args.mode
	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "OIDC provider deletion mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  mode,
			Options:  modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid OIDC provider deletion mode: %s", err)
			os.Exit(1)
		}
	}

	switch mode {
	case "auto":
		ocmClient.LogEvent("ROSADeleteOIDCProviderModeAuto")

		if !confirm.Prompt(true, "Delete the OIDC provider '%s'?", providerARN) {
			os.Exit(1)
		}
		err := awsClient.DeleteOpenIDConnectProvider(providerARN)
		if err != nil {
			reporter.Errorf("There was an error deleting the OIDC provider: %s", err)
			os.Exit(1)
		}

	case "manual":
		ocmClient.LogEvent("ROSADeleteOIDCProviderModeManual")
		commands := buildCommand(providerARN)
		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to delete the OIDC provider:\n")
		}
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", modes)
		os.Exit(1)
	}
}

func buildCommand(providerARN string) string {
	return fmt.Sprintf("aws iam delete-open-id-connect-provider \\\n"+
		"\t--open-id-connect-provider-arn %s \n\n",
		providerARN)
}
