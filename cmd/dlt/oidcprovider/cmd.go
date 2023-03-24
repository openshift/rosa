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
	"net/url"

	"os"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/helper"
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

const (
	OidcEndpointUrlFlag = "oidc-endpoint-url"
)

var args struct {
	oidcEndpointUrl string
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.oidcEndpointUrl,
		OidcEndpointUrlFlag,
		"",
		"Endpoint url for deleting OIDC provider, this flag needs to be used in case of reusable OIDC Config",
	)
	flags.MarkHidden(OidcEndpointUrlFlag)

	ocm.AddOptionalClusterFlag(Cmd)
	aws.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	isProgmaticallyCalled := false
	if len(argv) == 3 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
		aws.SetModeKey(argv[1])
		if argv[1] != "" {
			isProgmaticallyCalled = true
		}

		if argv[2] != "" {
			args.oidcEndpointUrl = argv[2]
		}
	}

	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if !cmd.Flag("cluster").Changed && !cmd.Flag(OidcEndpointUrlFlag).Changed && !isProgmaticallyCalled {
		r.Reporter.Errorf("Either a cluster key or an OIDC Endpoint URL must be specified.")
		os.Exit(1)
	}

	clusterKey := ""
	providerArn := ""
	if args.oidcEndpointUrl == "" {
		clusterKey = r.GetClusterKey()
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

		if sub != nil {
			clusterKey = sub.ClusterID()
		}
		cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
		if err != nil {
			if errors.GetType(err) != errors.NotFound {
				r.Reporter.Errorf("Error validating cluster '%s': %v", clusterKey, err)
				os.Exit(1)
			} else if sub == nil {
				r.Reporter.Errorf("Failed to get cluster '%s': %v", r.ClusterKey, err)
				os.Exit(1)
			}

		}
		if cluster != nil && cluster.ID() != "" {
			r.Reporter.Errorf("Cluster '%s' is in '%s' state. OIDC provider can be deleted only for the "+
				"uninstalled clusters", cluster.ID(), cluster.State())
			os.Exit(1)
		}

		providerArn, err = r.AWSClient.GetOpenIDConnectProviderByClusterIdTag(sub.ClusterID())
		if err != nil {
			r.Reporter.Errorf("Failed to get the OIDC provider for cluster '%s'.", clusterKey)
			os.Exit(1)
		}
		if providerArn == "" {
			r.Reporter.Infof("Cluster '%s' doesn't have OIDC provider associated with it. "+
				"In case of reusable OIDC config please use '%s' flag.",
				clusterKey, OidcEndpointUrlFlag)
			return
		}
	} else {
		oidcEndpointUrl := args.oidcEndpointUrl
		parsedURI, _ := url.ParseRequestURI(oidcEndpointUrl)
		if parsedURI.Scheme != helper.ProtocolHttps {
			r.Reporter.Errorf("Expected OIDC endpoint URL '%s' to use an https:// scheme", oidcEndpointUrl)
			os.Exit(1)
		}
		providerArn, err = r.AWSClient.GetOpenIDConnectProviderByOidcEndpointUrl(oidcEndpointUrl)
		if err != nil {
			r.Reporter.Errorf("Failed to get the OIDC provider for cluster '%s'.", clusterKey)
			os.Exit(1)
		}
		hasClusterUsingOidcProvider, err := r.OCMClient.HasAClusterUsingOidcProvider(oidcEndpointUrl)
		if err != nil {
			r.Reporter.Errorf("There was a problem checking if any clusters are using OIDC provider '%s' : %v",
				oidcEndpointUrl, err)
			os.Exit(1)
		}
		if hasClusterUsingOidcProvider {
			r.Reporter.Errorf("There are clusters using OIDC config '%s', can't delete the provider", oidcEndpointUrl)
			os.Exit(1)
		}
		if providerArn == "" {
			r.Reporter.Infof("Provider '%s' not found.", oidcEndpointUrl)
			return
		}
	}
	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}

	if interactive.Enabled() && !isProgmaticallyCalled {
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
		if !confirm.Prompt(true, "Delete the OIDC provider '%s'?", providerArn) {
			os.Exit(1)
		}
		err := r.AWSClient.DeleteOpenIDConnectProvider(providerArn)
		if err != nil {
			r.Reporter.Errorf("There was an error deleting the OIDC provider: %s", err)
			os.Exit(1)
		}
		r.Reporter.Infof("Successfully deleted the OIDC provider %s", providerArn)
	case aws.ModeManual:
		r.OCMClient.LogEvent("ROSADeleteOIDCProviderModeManual", nil)
		commands := buildCommand(providerArn)
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
	return awscb.NewIAMCommandBuilder().
		SetCommand(awscb.DeleteOpenIdConnectProvider).
		AddParam(awscb.OpenIdConnectProviderArn, providerARN).
		Build()
}
