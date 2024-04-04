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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openshift-online/ocm-common/pkg/rosa/oidcconfigs"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	interactiveOidc "github.com/openshift/rosa/pkg/interactive/oidc"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	OidcConfigIdFlag = "oidc-config-id"
)

var args struct {
	oidcConfigId    string
	oidcEndpointUrl string
}

type CreateOidcProviderStruct struct{}

var CreateOidcProvider rosa.CommandInterface

func (c *CreateOidcProviderStruct) NewCommand() *cobra.Command {
	return NewCreateOidcProviderCommand(c)
}

func NewCreateOidcProviderCommand(i rosa.CommandInterface) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "oidc-provider",
		Aliases: []string{"oidcprovider"},
		Short:   "Create OIDC provider for an STS cluster.",
		Long:    "Create OIDC provider for operators to authenticate against in an STS cluster.",
		Example: `  # Create OIDC provider for cluster named "mycluster"
	rosa create oidc-provider --cluster=mycluster`,
		Run:  rosa.DefaultRunner(rosa.RuntimeWithOCM(), i.Runner()),
		Args: cobra.MaximumNArgs(3),
	}

	flags := cmd.Flags()

	flags.StringVar(
		&args.oidcConfigId,
		OidcConfigIdFlag,
		"",
		"Registered OIDC configuration ID to retrieve its issuer URL. "+
			"Not to be used alongside --cluster flag.",
	)

	ocm.AddOptionalClusterFlag(cmd)
	interactive.AddModeFlag(cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)

	return cmd
}

func (c *CreateOidcProviderStruct) Runner() rosa.CommandRunner {
	return NewRunner(c)
}

func NewRunner(i rosa.CommandInterface) rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {

		// Allow the command to be called programmatically
		isProgmaticallyCalled := false
		shouldUseClusterKey := true
		if len(argv) == 3 && !cmd.Flag("cluster").Changed {
			ocm.SetClusterKey(argv[0])
			interactive.SetModeKey(argv[1])

			if argv[1] != "" {
				isProgmaticallyCalled = true
			}

			if argv[2] != "" {
				args.oidcEndpointUrl = argv[2]
				shouldUseClusterKey = false
			}
		}

		if cmd.Flag("cluster").Changed && cmd.Flag(OidcConfigIdFlag).Changed {
			return fmt.Errorf("A cluster key for STS cluster and an OIDC Config ID " +
				"cannot be specified alongside each other.")
		}

		mode, err := interactive.GetMode()
		if err != nil {
			return err
		}

		// Determine if interactive mode is needed
		if !isProgmaticallyCalled && !interactive.Enabled() &&
			(!cmd.Flags().Changed("cluster") || !cmd.Flags().Changed("mode")) {
			interactive.Enable()
		}

		var cluster *cmv1.Cluster
		clusterKey := ""
		if cmd.Flags().Changed("cluster") || (isProgmaticallyCalled && shouldUseClusterKey) {
			clusterKey = runtime.GetClusterKey()
			cluster = runtime.FetchCluster()
			if !ocm.IsSts(cluster) {
				return fmt.Errorf("Cluster '%s' is not an STS clusteruntime.", clusterKey)
			}
		}

		if !cmd.Flags().Changed("mode") && interactive.Enabled() && !isProgmaticallyCalled {
			mode, err = interactive.GetOptionMode(cmd, mode, "OIDC provider creation mode")
			if err != nil {
				return fmt.Errorf("Expected a valid OIDC provider creation mode: %s", err)
			}
		}

		var thumbprint *v1.AwsOidcThumbprint
		if isProgmaticallyCalled && args.oidcEndpointUrl != "" {
			// In the case of specifying a OIDC endpoint URL explicitly, fetch the
			// thumbprint directly
			sha1Thumbprint, err := oidcconfigs.FetchThumbprint(args.oidcEndpointUrl)
			if err != nil {
				return fmt.Errorf("Unable to get OIDC thumbprint: %s", err)
			}
			thumbprint, err = v1.NewAwsOidcThumbprint().
				Thumbprint(sha1Thumbprint).
				IssuerUrl(args.oidcEndpointUrl).
				Build()
			if err != nil {
				return fmt.Errorf("There was an error creating OIDC provider: %s", err)
			}
		} else if cluster != nil {
			if cluster.AWS().STS().OIDCEndpointURL() == "" {
				return fmt.Errorf("Cluster '%s' does not have an OIDC endpoint URL; provider cannot be created.", clusterKey)
			}

			thumbprint, err = runtime.OCMClient.GetThumbprintByClusterId(cluster.ID())
			if err != nil {
				return fmt.Errorf("Unable to get OIDC thumbprint: %s", err)
			}
		} else {
			if args.oidcConfigId == "" {
				args.oidcConfigId = interactiveOidc.GetOidcConfigID(runtime, cmd)
			}
			oidcConfig, err := runtime.OCMClient.GetOidcConfig(args.oidcConfigId)
			if err != nil {
				return fmt.Errorf("There was a problem retrieving OIDC Config '%s': %v", args.oidcConfigId, err)
			}
			if oidcConfig.IssuerUrl() == "" {
				return fmt.Errorf("OIDC config '%s' does not have an OIDC endpoint URL; provider cannot be created.",
					args.oidcConfigId)
			}
			thumbprint, err = runtime.OCMClient.GetThumbprintByOidcConfigId(oidcConfig.ID())
			if err != nil {
				return fmt.Errorf("Unable to get OIDC thumbprint: %s", err)
			}
		}

		clusterId := ""
		if !ocm.IsOidcConfigReusable(cluster) {
			clusterId = cluster.ID()
		}

		oidcProviderExists, err := runtime.AWSClient.HasOpenIDConnectProvider(thumbprint.IssuerUrl(),
			runtime.Creator.Partition, runtime.Creator.AccountID)
		if err != nil {
			if strings.Contains(err.Error(), "AccessDenied") {
				runtime.Reporter.Debugf("Failed to verify if OIDC provider exists: %s", err)
			} else {
				return fmt.Errorf("Failed to verify if OIDC provider exists: %s", err)
			}
		}
		if oidcProviderExists {
			if cluster != nil &&
				cluster.AWS().STS().OidcConfig() != nil && !cluster.AWS().STS().OidcConfig().Reusable() {
				return fmt.Errorf("Cluster '%s' already has OIDC provider but has not yet started installation. "+
					"Verify that the cluster operator roles exist and are configured correctly.", clusterKey)
			}
			// Returns so that when called from create cluster does not interrupt flow
			runtime.Reporter.Infof("OIDC provider already exists")
			return nil
		}

		switch mode {
		case interactive.ModeAuto:
			if !output.HasFlag() || runtime.Reporter.IsTerminal() {
				runtime.Reporter.Infof("Creating OIDC provider using '%s'", runtime.Creator.ARN)
			}
			confirmPromptMessage := "Create the OIDC provider?"
			if clusterKey != "" {
				confirmPromptMessage = fmt.Sprintf("Create the OIDC provider for cluster '%s'?", clusterKey)
			}
			if !confirm.Prompt(true, confirmPromptMessage) {
				os.Exit(0)
			}
			err = createProvider(runtime, thumbprint, clusterId)
			if err != nil {
				runtime.OCMClient.LogEvent("ROSACreateOIDCProviderModeAuto", map[string]string{
					ocm.ClusterID: clusterKey,
					ocm.Response:  ocm.Failure,
				})
				return fmt.Errorf("There was an error creating the OIDC provider: %s", err)
			}
			runtime.OCMClient.LogEvent("ROSACreateOIDCProviderModeAuto", map[string]string{
				ocm.ClusterID: clusterKey,
				ocm.Response:  ocm.Success,
			})
		case interactive.ModeManual:
			commands, err := buildCommands(runtime, thumbprint, clusterId)
			if err != nil {
				runtime.OCMClient.LogEvent("ROSACreateOIDCProviderModeManual", map[string]string{
					ocm.ClusterID: clusterKey,
					ocm.Response:  ocm.Failure,
				})
				return fmt.Errorf("There was an error building the list of resources: %s", err)
			}
			if runtime.Reporter.IsTerminal() {
				runtime.Reporter.Infof("Run the following commands to create the OIDC provider:\n")
			}
			runtime.OCMClient.LogEvent("ROSACreateOIDCProviderModeManual", map[string]string{
				ocm.ClusterID: clusterKey,
			})
			fmt.Println(commands)
		default:
			return fmt.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		}

		return nil
	}
}

func createProvider(runtime *rosa.Runtime, thumbprint *v1.AwsOidcThumbprint, clusterId string) error {
	runtime.Reporter.Debugf("Using thumbprint '%s'", thumbprint.Thumbprint())

	oidcProviderARN, err := runtime.AWSClient.CreateOpenIDConnectProvider(thumbprint.IssuerUrl(),
		thumbprint.Thumbprint(), clusterId)
	if err != nil {
		return err
	}
	if !output.HasFlag() || runtime.Reporter.IsTerminal() {
		runtime.Reporter.Infof("Created OIDC provider with ARN '%s'", oidcProviderARN)
	}

	return nil
}

func buildCommands(runtime *rosa.Runtime, thumbprint *v1.AwsOidcThumbprint, clusterId string) (string, error) {
	commands := []string{}

	runtime.Reporter.Debugf("Using thumbprint '%s'", thumbprint.Thumbprint())

	iamTags := map[string]string{
		tags.RedHatManaged: tags.True,
	}
	if clusterId != "" {
		iamTags[tags.ClusterID] = clusterId
	}

	clientIdList := strings.Join([]string{aws.OIDCClientIDOpenShift, aws.OIDCClientIDSTSAWS}, " ")

	createOpenIDConnectProvider := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreateOpenIdConnectProvider).
		AddParam(awscb.Url, thumbprint.IssuerUrl()).
		AddParam(awscb.ClientIdList, clientIdList).
		AddParam(awscb.ThumbprintList, thumbprint.Thumbprint()).
		AddTags(iamTags).
		Build()
	commands = append(commands, createOpenIDConnectProvider)

	return awscb.JoinCommands(commands), nil
}
