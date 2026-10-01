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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	interactiveOidc "github.com/openshift/rosa/pkg/interactive/oidc"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "oidc-provider",
	Aliases: []string{"oidcprovider"},
	Short:   "Create OpenID Connect (OIDC) provider for an AWS Security Token Service (STS) cluster.",
	Long: "Create OpenID Connect (OIDC) provider for operators to authenticate against in an " +
		"AWS Security Token Service (STS) cluster.",
	Example: ` # Create OIDC provider for cluster named "mycluster"
  rosa create oidc-provider --cluster=mycluster`,
	Run:  run,
	Args: cobra.MaximumNArgs(3),
}

const (
	OidcConfigIdFlag = "oidc-config-id"
)

var args struct {
	oidcConfigId    string
	oidcEndpointUrl string
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.oidcConfigId,
		OidcConfigIdFlag,
		"",
		"Registered OIDC configuration ID to retrieve its issuer URL. "+
			"Not to be used alongside --cluster flag.",
	)

	ocm.AddOptionalClusterFlag(Cmd)
	interactive.AddModeFlag(Cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	// Check for hyperfleet mode first and route to separate handler
	if hyperfleet.Enabled() {
		runHyperfleet(cmd, argv)
		return
	}

	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Allow the command to be called programmatically
	isProgrammaticallyCalled := false
	shouldUseClusterKey := true
	if len(argv) >= 3 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
		interactive.SetModeKey(argv[1])

		if argv[1] != "" {
			isProgrammaticallyCalled = true
		}

		if argv[2] != "" {
			args.oidcEndpointUrl = argv[2]
			shouldUseClusterKey = false
		}
	}

	if cmd.Flag("cluster").Changed && cmd.Flag(OidcConfigIdFlag).Changed {
		r.Reporter.Errorf("A cluster key for STS cluster and an OIDC Config ID " +
			"cannot be specified alongside each other.")
		os.Exit(1)
	}

	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !isProgrammaticallyCalled && !interactive.Enabled() &&
		(!cmd.Flags().Changed("cluster") || !cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	var cluster *cmv1.Cluster
	clusterKey := ""
	if cmd.Flags().Changed("cluster") || (isProgrammaticallyCalled && shouldUseClusterKey) {
		clusterKey = r.GetClusterKey()

		// Platform API v2 (hyperfleet) handling
		if r.IsUsingHyperfleet() {
			handleHyperfleetOIDCProvider(r, clusterKey, args.oidcConfigId, mode)
			return
		}

		cluster = r.FetchCluster()
		if !ocm.IsSts(cluster) {
			r.Reporter.Errorf("Cluster '%s' is not an STS cluster.", clusterKey)
			os.Exit(1)
		}
	}

	if !cmd.Flags().Changed("mode") && interactive.Enabled() && !isProgrammaticallyCalled {
		mode, err = interactive.GetOptionMode(cmd, mode, "OIDC provider creation mode")
		if err != nil {
			r.Reporter.Errorf("Expected a valid OIDC provider creation mode: %s", err)
			os.Exit(1)
		}
	}

	oidcEndpointURL := ""
	if cluster != nil {
		oidcEndpointURL = cluster.AWS().STS().OIDCEndpointURL()
	} else {
		if isProgrammaticallyCalled && args.oidcEndpointUrl != "" {
			oidcEndpointURL = args.oidcEndpointUrl
		} else {
			if args.oidcConfigId == "" {
				args.oidcConfigId = interactiveOidc.GetOidcConfigID(r, cmd)
			}
			oidcConfig, err := r.OCMClient.GetOidcConfig(args.oidcConfigId)
			if err != nil {
				r.Reporter.Errorf("There was a problem retrieving OIDC Config '%s': %v", args.oidcConfigId, err)
				os.Exit(1)
			}
			oidcEndpointURL = oidcConfig.IssuerUrl()
		}
	}

	clusterId := ""
	if !ocm.IsOidcConfigReusable(cluster) {
		clusterId = cluster.ID()
	}

	oidcProviderExists, err := r.AWSClient.HasOpenIDConnectProvider(oidcEndpointURL,
		r.Creator.Partition, r.Creator.AccountID)
	if err != nil {
		if strings.Contains(err.Error(), "AccessDenied") {
			r.Reporter.Debugf("Failed to verify if OIDC provider exists: %s", err)
		} else {
			r.Reporter.Errorf("Failed to verify if OIDC provider exists: %s", err)
			os.Exit(1)
		}
	}
	if oidcProviderExists {
		if cluster != nil &&
			cluster.AWS().STS().OidcConfig() != nil && !cluster.AWS().STS().OidcConfig().Reusable() {
			r.Reporter.Warnf("Cluster '%s' already has OIDC provider but has not yet started installation. "+
				"Verify that the cluster operator roles exist and are configured correctly.", clusterKey)
			os.Exit(1)
		}
		// Returns so that when called from create cluster does not interrupt flow
		r.Reporter.Infof("OIDC provider already exists")
		return
	}

	switch mode {
	case interactive.ModeAuto:
		if !output.HasFlag() || r.Reporter.IsTerminal() {
			r.Reporter.Infof("Creating OIDC provider using '%s'", r.Creator.ARN)
		}
		confirmPromptMessage := "Create the OIDC provider?"
		if clusterKey != "" {
			confirmPromptMessage = fmt.Sprintf("Create the OIDC provider for cluster '%s'?", clusterKey)
		}
		if !confirm.Prompt(true, "%s", confirmPromptMessage) {
			os.Exit(0)
		}
		if clusterId == "" && clusterKey != "" {
			clusterId = r.FetchCluster().ID()
		}
		err = createProvider(r, oidcEndpointURL, clusterId, isProgrammaticallyCalled)
		if err != nil {
			r.Reporter.Errorf("There was an error creating the OIDC provider: %s", err)
			r.OCMClient.LogEvent("ROSACreateOIDCProviderModeAuto", map[string]string{
				ocm.ClusterID: clusterKey,
				ocm.Response:  ocm.Failure,
			})
			os.Exit(1)
		}
		r.OCMClient.LogEvent("ROSACreateOIDCProviderModeAuto", map[string]string{
			ocm.ClusterID: clusterKey,
			ocm.Response:  ocm.Success,
		})
	case interactive.ModeManual:
		commands, err := buildCommands(r, oidcEndpointURL, clusterId)
		if err != nil {
			r.Reporter.Errorf("There was an error building the list of resources: %s", err)
			os.Exit(1)
			r.OCMClient.LogEvent("ROSACreateOIDCProviderModeManual", map[string]string{
				ocm.ClusterID: clusterKey,
				ocm.Response:  ocm.Failure,
			})
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to create the OIDC provider:\n")
		}
		r.OCMClient.LogEvent("ROSACreateOIDCProviderModeManual", map[string]string{
			ocm.ClusterID: clusterKey,
		})
		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		os.Exit(1)
	}
}

func CreateOIDCProvider(r *rosa.Runtime, oidcConfigId string, clusterId string, isProgrammaticallyCalled bool) error {
	args.oidcConfigId = oidcConfigId
	oidcConfig, err := r.OCMClient.GetOidcConfig(oidcConfigId)
	if err != nil {
		return fmt.Errorf("there was a problem retrieving OIDC Config '%s': %v", oidcConfigId, err)
	}
	oidcEndpointURL := oidcConfig.IssuerUrl()
	return createProvider(r, oidcEndpointURL, clusterId, isProgrammaticallyCalled)
}

func createProvider(r *rosa.Runtime, oidcEndpointUrl string, clusterId string, isProgrammaticallyCalled bool) error {
	inputBuilder := cmv1.NewOidcThumbprintInput()
	if (isProgrammaticallyCalled || clusterId == "") && args.oidcConfigId != "" {
		inputBuilder.OidcConfigId(args.oidcConfigId)
	} else {
		inputBuilder.ClusterId(clusterId)
	}
	input, err := inputBuilder.Build()
	if err != nil {
		return err
	}
	thumbprint, err := r.OCMClient.FetchOidcThumbprint(input)
	if err != nil {
		return err
	}
	r.Reporter.Debugf("Using thumbprint '%s'", thumbprint.Thumbprint())

	oidcProviderARN, err := r.AWSClient.CreateOpenIDConnectProvider(oidcEndpointUrl, thumbprint.Thumbprint(), clusterId)
	if err != nil {
		return err
	}
	if !output.HasFlag() || r.Reporter.IsTerminal() {
		r.Reporter.Infof("Created OIDC provider with ARN '%s'", oidcProviderARN)
	}

	return nil
}

func buildCommands(r *rosa.Runtime, oidcEndpointUrl string, clusterId string) (string, error) {
	commands := []string{}

	input, err := cmv1.NewOidcThumbprintInput().OidcConfigId(args.oidcConfigId).ClusterId(clusterId).Build()
	if err != nil {
		return "", err
	}
	thumbprint, err := r.OCMClient.FetchOidcThumbprint(input)
	if err != nil {
		return "", err
	}
	r.Reporter.Debugf("Using thumbprint '%s'", thumbprint.Thumbprint())

	iamTags := map[string]string{
		tags.RedHatManaged: tags.True,
	}
	if clusterId != "" {
		iamTags[tags.ClusterID] = clusterId
	}

	clientIdList := strings.Join([]string{aws.OIDCClientIDOpenShift, aws.OIDCClientIDSTSAWS}, " ")

	createOpenIDConnectProvider := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreateOpenIdConnectProvider).
		AddParam(awscb.Url, oidcEndpointUrl).
		AddParam(awscb.ClientIdList, clientIdList).
		AddParam(awscb.ThumbprintList, thumbprint.Thumbprint()).
		AddTags(iamTags).
		Build()
	commands = append(commands, createOpenIDConnectProvider)

	return awscb.JoinCommands(commands), nil
}

// runHyperfleet is the entry point for hyperfleet mode
func runHyperfleet(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithHyperFleet().WithAWSOnly()
	defer r.Cleanup()

	clusterFlagChanged := cmd.Flags().Changed("cluster")
	configIDFlagChanged := args.oidcConfigId != ""
	if configIDFlagChanged && (clusterFlagChanged || (len(argv) > 0 && argv[0] != "")) {
		r.Reporter.Errorf("A cluster key and an OIDC Config ID cannot be specified alongside each other.")
		os.Exit(1)
	}

	// Resolve a cluster key only when the caller did not provide an OIDC config ID.
	var clusterKey string
	if configIDFlagChanged {
		clusterKey = ""
	} else if len(argv) >= 1 && !clusterFlagChanged {
		clusterKey = argv[0]
	} else {
		clusterKey = r.GetClusterKey()
	}

	// Get mode
	if len(argv) >= 2 && argv[1] != "" && !cmd.Flags().Changed("mode") {
		interactive.SetModeKey(argv[1])
	}
	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if mode == "" {
		interactive.Enable()
		mode, err = interactive.GetOptionMode(cmd, mode, "OIDC provider creation mode")
		if err != nil {
			r.Reporter.Errorf("Expected a valid OIDC provider creation mode: %s", err)
			os.Exit(1)
		}
	}

	handleHyperfleetOIDCProvider(r, clusterKey, args.oidcConfigId, mode)
}

// handleHyperfleetOIDCProvider handles OIDC provider operations for Platform API v2 (hyperfleet) clusters.
func handleHyperfleetOIDCProvider(r *rosa.Runtime, clusterKey, oidcConfigID, mode string) {
	r.Reporter.Debugf("Handling HyperFleet OIDC provider for cluster '%s' and config '%s'", clusterKey, oidcConfigID)

	ctx := context.Background()

	var (
		clusterUID    string
		oidcIssuerURL string
		err           error
	)
	if oidcConfigID != "" {
		// An OIDC config ID is an alternate issuer source. The provider check
		// and auto/manual recovery below are shared with the cluster path.
		config, getErr := r.HyperFleetClient.HyperfleetV1alpha1().OidcConfigs().Get(
			ctx, oidcConfigID, platform.GetOptions{})
		if getErr != nil {
			r.Reporter.Errorf("Failed to get OIDC config '%s': %v", oidcConfigID, getErr)
			os.Exit(1)
		}
		oidcIssuerURL = config.Spec.IssuerUrl
	} else {
		// Resolve the key and retain the cluster returned by the list operation.
		// Fetching it again by UID would make a second Platform API request.
		cluster, getErr := hyperfleet.GetCluster(ctx, r.HyperFleetClient, clusterKey)
		if getErr != nil {
			r.Reporter.Errorf("Failed to resolve cluster '%s': %v", clusterKey, getErr)
			os.Exit(1)
		}
		if cluster == nil {
			r.Reporter.Errorf("Failed to resolve cluster '%s': cluster '%s' not found", clusterKey, clusterKey)
			os.Exit(1)
		}
		clusterUID = string(cluster.UID)
		oidcIssuerURL = extractOIDCIssuerFromHyperfleetCluster(cluster)
	}

	if oidcIssuerURL == "" {
		if oidcConfigID != "" {
			r.Reporter.Errorf("OIDC config '%s' does not have an issuer URL configured", oidcConfigID)
		} else {
			r.Reporter.Errorf("Cluster '%s' does not have an OIDC issuer URL configured", clusterKey)
		}
		os.Exit(1)
	}

	r.Reporter.Debugf("Found OIDC issuer URL: %s", oidcIssuerURL)

	// Check if OIDC provider exists in AWS
	providerExists, err := r.AWSClient.HasOpenIDConnectProvider(
		oidcIssuerURL,
		r.Creator.Partition,
		r.Creator.AccountID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "AccessDenied") {
			r.Reporter.Debugf("Failed to verify if OIDC provider exists: %s", err)
		} else {
			r.Reporter.Errorf("Failed to verify if OIDC provider exists: %s", err)
			os.Exit(1)
		}
	}

	if providerExists {
		r.Reporter.Infof("OIDC provider already exists")
		return
	}

	// The provider may need to be recreated after manual deletion. Match the V1
	// command's auto/manual behavior for both a cluster issuer and an OIDC config.
	switch mode {
	case interactive.ModeAuto:
		confirmMessage := "Create the OIDC provider?"
		if clusterKey != "" {
			confirmMessage = fmt.Sprintf("Create the OIDC provider for cluster '%s'?", clusterKey)
		}
		if !confirm.Prompt(true, "%s", confirmMessage) {
			return
		}

		thumbprint, err := aws.GetOIDCThumbprint(oidcIssuerURL)
		if err != nil {
			r.Reporter.Errorf("Failed to compute OIDC provider thumbprint: %v", err)
			os.Exit(1)
		}
		providerARN, err := r.AWSClient.CreateOpenIDConnectProvider(oidcIssuerURL, thumbprint, clusterUID)
		if err != nil {
			r.Reporter.Errorf("There was an error creating the OIDC provider: %v", err)
			os.Exit(1)
		}
		r.Reporter.Infof("Created OIDC provider with ARN '%s'", providerARN)
	case interactive.ModeManual:
		thumbprint, err := aws.GetOIDCThumbprint(oidcIssuerURL)
		if err != nil {
			r.Reporter.Errorf("Failed to compute OIDC provider thumbprint: %v", err)
			os.Exit(1)
		}
		commands := buildHyperfleetOIDCProviderCommand(oidcIssuerURL, thumbprint, clusterUID)
		r.Reporter.Infof("Run the following commands to create the OIDC provider:\n")
		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		os.Exit(1)
	}
}

func buildHyperfleetOIDCProviderCommand(issuerURL, thumbprint, clusterID string) string {
	iamTags := map[string]string{tags.RedHatManaged: tags.True}
	if clusterID != "" {
		iamTags[tags.ClusterID] = clusterID
	}
	clientIDList := strings.Join([]string{aws.OIDCClientIDOpenShift, aws.OIDCClientIDSTSAWS}, " ")

	return awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreateOpenIdConnectProvider).
		AddParam(awscb.Url, issuerURL).
		AddParam(awscb.ClientIdList, clientIDList).
		AddParam(awscb.ThumbprintList, thumbprint).
		AddTags(iamTags).
		Build()
}

// extractOIDCIssuerFromHyperfleetCluster extracts the OIDC issuer URL from a hyperfleet cluster object.
func extractOIDCIssuerFromHyperfleetCluster(cluster *v1alpha1.Cluster) string {
	// The OIDC issuer is in the HostedCluster spec
	if cluster.Spec.HostedCluster.IssuerURL != "" {
		return cluster.Spec.HostedCluster.IssuerURL
	}

	return ""
}
