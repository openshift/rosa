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

	ocm.AddClusterFlag(Cmd)
	aws.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)
	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
	}

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
	sub, err := ocmClient.GetClusterUsingSubscription(clusterKey, creator.AccountID)
	if err != nil {
		if errors.GetType(err) == errors.Conflict {
			reporter.Errorf("More than one cluster found with the same name '%s'. Please "+
				"use cluster ID instead", clusterKey)
			os.Exit(1)
		}
		reporter.Errorf("Error validating cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	clusterID := clusterKey
	if sub != nil {
		clusterID = sub.ClusterID()
	}
	c, err := ocmClient.GetClusterByID(clusterID, creator.AccountID)
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

	providerARN, err := awsClient.GetOpenIDConnectProvider(sub.ClusterID())
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

	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "OIDC provider deletion mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid OIDC provider deletion mode: %s", err)
			os.Exit(1)
		}
	}
	switch mode {
	case aws.ModeAuto:
		ocmClient.LogEvent("ROSADeleteOIDCProviderModeAuto", nil)
		if !confirm.Prompt(true, "Delete the OIDC provider '%s'?", providerARN) {
			os.Exit(1)
		}
		err := awsClient.DeleteOpenIDConnectProvider(providerARN)
		if err != nil {
			reporter.Errorf("There was an error deleting the OIDC provider: %s", err)
			os.Exit(1)
		}
		reporter.Infof("Successfully deleted the OIDC provider %s", providerARN)
	case aws.ModeManual:
		ocmClient.LogEvent("ROSADeleteOIDCProviderModeManual", nil)
		commands := buildCommand(providerARN)
		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to delete the OIDC provider:\n")
		}
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func buildCommand(providerARN string) string {
	return fmt.Sprintf("aws iam delete-open-id-connect-provider \\\n"+
		"\t--open-id-connect-provider-arn %s \n\n",
		providerARN)
}
