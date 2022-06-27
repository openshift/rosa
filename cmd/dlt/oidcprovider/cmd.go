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

	"os"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"
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
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Try to find the cluster:
	r.Reporter.Debugf("Loading cluster '%s'", clusterKey)
	sub, err := r.OCMClient.GetClusterUsingSubscription(clusterKey, r.Creator)
	if err != nil {
		if errors.GetType(err) == errors.Conflict {
			r.Reporter.Errorf("More than one cluster found with the same name '%s'. Please "+
				"use cluster ID instead", clusterKey)
			os.Exit(1)
		}
		r.Reporter.Errorf("Error validating cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	clusterID := clusterKey
	if sub != nil {
		clusterID = sub.ClusterID()
	}
	c, err := r.OCMClient.GetClusterByID(clusterID, r.Creator)
	if err != nil {
		if errors.GetType(err) != errors.NotFound {
			r.Reporter.Errorf("Error validating cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	}
	if c != nil && c.ID() != "" {
		r.Reporter.Errorf("Cluster '%s' is in '%s' state. OIDC provider can be deleted only for the "+
			"uninstalled clusters", c.ID(), c.State())
		os.Exit(1)
	}

	providerARN, err := r.AWSClient.GetOpenIDConnectProvider(sub.ClusterID())
	if err != nil {
		r.Reporter.Errorf("Failed to get the OIDC provider for cluster '%s'.", clusterKey)
		os.Exit(1)
	}
	if providerARN == "" {
		r.Reporter.Infof("Cluster '%s' doesn't have OIDC provider associated with it.", clusterKey)
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
			r.Reporter.Errorf("Expected a valid OIDC provider deletion mode: %s", err)
			os.Exit(1)
		}
	}
	switch mode {
	case aws.ModeAuto:
		r.OCMClient.LogEvent("ROSADeleteOIDCProviderModeAuto", nil)
		if !confirm.Prompt(true, "Delete the OIDC provider '%s'?", providerARN) {
			os.Exit(1)
		}
		err := r.AWSClient.DeleteOpenIDConnectProvider(providerARN)
		if err != nil {
			r.Reporter.Errorf("There was an error deleting the OIDC provider: %s", err)
			os.Exit(1)
		}
		r.Reporter.Infof("Successfully deleted the OIDC provider %s", providerARN)
	case aws.ModeManual:
		r.OCMClient.LogEvent("ROSADeleteOIDCProviderModeManual", nil)
		commands := buildCommand(providerARN)
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to delete the OIDC provider:\n")
		}
		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func buildCommand(providerARN string) string {
	return fmt.Sprintf("aws iam delete-open-id-connect-provider \\\n"+
		"\t--open-id-connect-provider-arn %s \n\n",
		providerARN)
}
