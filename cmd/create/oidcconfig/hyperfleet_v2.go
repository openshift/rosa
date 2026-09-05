package oidcconfig

import (
	"context"
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, and hfCreateOidcConfig are package-level
// vars so tests can stub the hyperfleet dispatch path.
var (
	hfEnabled          = hyperfleet.Enabled
	hfExitFn           = func(code int) { os.Exit(code) }
	hfCreateOidcConfig = func() {
		r := rosa.NewRuntime().WithAWS().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetCreate(r)
	}
)

// runHyperfleetCreate creates an OIDC Config via the Platform API v2.
func runHyperfleetCreate(r *rosa.Runtime) {
	ctx := context.Background()

	// Get mode for OIDC provider creation
	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		hfExitFn(1)
		return
	}

	// Validate required fields based on managed/unmanaged mode
	oidcConfigType := "managed"
	if !args.managed {
		oidcConfigType = "unmanaged"
	}

	// For unmanaged configs, validate required fields
	// if !args.managed {
	// 	if args.installerRoleArn == "" {
	// 		r.Reporter.Errorf("--installer-role-arn is required for unmanaged OIDC configs")
	// 		hfExitFn(1)
	// 		return
	// 	}
	// }

	// Build OIDC Config spec
	spec := v1alpha1.OidcConfigSpec{
		Type: oidcConfigType,
	}

	// For unmanaged configs, set the required fields
	if !args.managed {
		spec.InstallerRoleArn = args.installerRoleArn
		// Note: SecretArn and IssuerUrl will be set by the platform-api/operator
		// based on the S3 bucket and secrets manager resources created
	}

	// Generate a unique name if prefix is provided
	configName := ""
	if args.userPrefix != "" {
		// Use prefix for the config name
		configName = args.userPrefix
	}

	// Debug: Log the API call details
	r.Reporter.Debugf("Creating OIDC config via Platform API")
	r.Reporter.Debugf("  Resource Path: /api/v0/oidc_configs")
	r.Reporter.Debugf("  Config Name: '%s'", configName)
	r.Reporter.Debugf("  Config Type: %s", oidcConfigType)
	if !args.managed {
		r.Reporter.Debugf("  Installer Role ARN: %s", args.installerRoleArn)
	}
	r.Reporter.Debugf("  Platform API Host: %s", r.HyperFleetClient)

	oidcConfig, err := r.HyperFleetClient.HyperfleetV1alpha1().OidcConfigs().Create(
		ctx,
		&v1alpha1.OidcConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: configName,
			},
			Spec: spec,
		},
		platform.CreateOptions{},
	)
	if err != nil {
		r.Reporter.Debugf("Platform API error: %v", err)
		r.Reporter.Debugf("This likely means the /oidc_configs route is not implemented in the Platform API backend")
		r.Reporter.Errorf("Failed to create OIDC config: %v", err)
		hfExitFn(1)
		return
	}

	if output.HasFlag() {
		err = output.Print(oidcConfig)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			hfExitFn(1)
		}
		hfExitFn(0)
		return
	}

	r.Reporter.Infof("OIDC config '%s' created successfully", oidcConfig.Name)
	r.Reporter.Infof("  Type: %s", oidcConfig.Spec.Type)
	if oidcConfig.Spec.IssuerUrl != "" {
		r.Reporter.Infof("  Issuer URL: %s", oidcConfig.Spec.IssuerUrl)
	}
	if oidcConfig.Spec.SecretArn != "" {
		r.Reporter.Infof("  Secret ARN: %s", oidcConfig.Spec.SecretArn)
	}

	// Wait for thumbprint to be available and create OIDC provider
	if oidcConfig.Spec.IssuerUrl != "" {
		err = createOidcProvider(ctx, r, oidcConfig, mode)
		if err != nil {
			r.Reporter.Errorf("Failed to create OIDC provider: %v", err)
			hfExitFn(1)
			return
		}
	}
}

// createOidcProvider creates the AWS OIDC provider for the given OIDC config
func createOidcProvider(ctx context.Context, r *rosa.Runtime, oidcConfig *v1alpha1.OidcConfig, mode string) error {
	// Compute the thumbprint from the issuer URL
	r.Reporter.Debugf("Computing thumbprint from issuer URL: %s", oidcConfig.Spec.IssuerUrl)
	thumbprint, err := aws.GetOIDCThumbprint(oidcConfig.Spec.IssuerUrl)
	if err != nil {
		return fmt.Errorf("failed to compute OIDC thumbprint: %v", err)
	}

	r.Reporter.Debugf("Using thumbprint '%s'", thumbprint)

	// Check if OIDC provider already exists
	oidcProviderExists, err := r.AWSClient.HasOpenIDConnectProvider(
		oidcConfig.Spec.IssuerUrl,
		r.Creator.Partition,
		r.Creator.AccountID,
	)
	if err != nil {
		r.Reporter.Debugf("Failed to verify if OIDC provider exists: %s", err)
	}
	if oidcProviderExists {
		r.Reporter.Infof("OIDC provider already exists")
		return nil
	}

	switch mode {
	case interactive.ModeAuto:
		// Create the OIDC provider in AWS
		r.Reporter.Infof("Creating OIDC provider using '%s'", r.Creator.ARN)
		oidcProviderARN, err := r.AWSClient.CreateOpenIDConnectProvider(
			oidcConfig.Spec.IssuerUrl,
			thumbprint,
			"", // No cluster ID for standalone OIDC config
		)
		if err != nil {
			return fmt.Errorf("failed to create OIDC provider: %v", err)
		}
		r.Reporter.Infof("Created OIDC provider with ARN '%s'", oidcProviderARN)

	case interactive.ModeManual:
		// Print manual commands
		commands, err := buildOidcProviderCommands(oidcConfig.Spec.IssuerUrl, thumbprint)
		if err != nil {
			return fmt.Errorf("failed to build OIDC provider commands: %v", err)
		}
		r.Reporter.Infof("Run the following commands to create the OIDC provider:\n")
		fmt.Println(commands)

	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}

	return nil
}

// buildOidcProviderCommands builds the AWS CLI commands for manual OIDC provider creation
func buildOidcProviderCommands(issuerUrl, thumbprint string) (string, error) {
	clientIdList := strings.Join([]string{aws.OIDCClientIDOpenShift, aws.OIDCClientIDSTSAWS}, " ")

	iamTags := map[string]string{
		tags.RedHatManaged: tags.True,
	}

	createOpenIDConnectProvider := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreateOpenIdConnectProvider).
		AddParam(awscb.Url, issuerUrl).
		AddParam(awscb.ClientIdList, clientIdList).
		AddParam(awscb.ThumbprintList, thumbprint).
		AddTags(iamTags).
		Build()

	return createOpenIDConnectProvider, nil
}
