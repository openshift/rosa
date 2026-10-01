package oidcconfig

import (
	"context"
	"fmt"
	"os"
	"strings"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/constants"
	"github.com/openshift/rosa/pkg/hyperfleet"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfOidcConfigInput is the backing store for all hyperfleet-specific create oidcconfig flags.
var hfOidcConfigInput hfpathbind.OidcConfigCreateInput

// hfEnabled, hfExitFn, and hfCreateOidcConfig are package-level
// vars so tests can stub the hyperfleet dispatch path.
var (
	hfEnabled = hyperfleet.Enabled
	hfExitFn  = func(code int) { os.Exit(code) }

	hfCreateOidcConfig = func(cmd *cobra.Command) {
		r := rosa.NewRuntime().WithHyperFleet().WithAWSOnly()
		defer r.Cleanup()
		if err := prepareHyperfleetOidcConfig(cmd); err != nil {
			r.Reporter.Errorf("Failed to create OIDC config: %v", err)
			hfExitFn(1)
			return
		}
		if err := hfpathbind.RunCreateOidcConfig(context.Background(), r, cmd, &hfOidcConfigInput,
			&hyperfleetOidcConfigCreate{},
		); err != nil {
			r.Reporter.Errorf("OIDC config operation failed: %v", err)
			hfExitFn(1)
		}
	}
)

// hyperfleetOidcConfigCreate implements hfpathbind.OidcConfigCreateHandler for rosa create oidcconfig.
type hyperfleetOidcConfigCreate struct {
	// interactive prompting for required fields
	hfpathbind.GeneratedOidcConfigCreatePrompt
}

// prepareHyperfleetOidcConfig validates flags and resolves interactive inputs before
// creating the Platform API resource. This avoids creating a config and then failing
// because provider mode or required unmanaged config inputs were missing.
func prepareHyperfleetOidcConfig(cmd *cobra.Command) error {
	if args.rawFiles {
		return fmt.Errorf("--%s is not supported with Platform API v2", rawFilesFlag)
	}

	mode, err := interactive.GetMode()
	if err != nil {
		return err
	}
	if mode == "" {
		interactive.Enable()
	}
	if interactive.Enabled() && !cmd.Flags().Changed(managedFlag) {
		args.managed = confirm.Prompt(true, "Would you like to create a Managed (Red Hat hosted) OIDC Configuration")
	}
	if mode == "" {
		mode, err = interactive.GetOptionMode(cmd, mode, "OIDC Provider creation mode")
		if err != nil {
			return err
		}
	}
	interactive.SetModeKey(mode)

	if output.HasFlag() && mode != interactive.ModeAuto {
		return fmt.Errorf("--output is not supported with manual mode")
	}
	if args.managed && args.installerRoleArn != "" {
		return fmt.Errorf("--%s is not supported for managed OIDC config", constants.InstallerRoleArnFlag)
	}
	if !args.managed && args.installerRoleArn == "" && interactive.Enabled() {
		args.installerRoleArn, err = interactive.GetString(interactive.Input{
			Question: "Installer role ARN",
			Help:     cmd.Flags().Lookup(constants.InstallerRoleArnFlag).Usage,
			Required: true,
		})
		if err != nil {
			return fmt.Errorf("expected a valid installer role ARN: %w", err)
		}
	}
	if !args.managed && args.installerRoleArn == "" {
		return fmt.Errorf("--%s is required for unmanaged OIDC configs", constants.InstallerRoleArnFlag)
	}
	if !args.managed {
		if err := aws.ARNValidator(args.installerRoleArn); err != nil {
			return fmt.Errorf("expected a valid installer role ARN: %w", err)
		}
	}

	if interactive.Enabled() && !args.managed && !cmd.Flags().Changed(userPrefixFlag) {
		args.userPrefix, err = interactive.GetString(interactive.Input{
			Question:   "Prefix for OIDC",
			Help:       cmd.Flags().Lookup(userPrefixFlag).Usage,
			Default:    args.userPrefix,
			Validators: []interactive.Validator{interactive.MaxLength(maxLengthUserPrefix)},
		})
		if err != nil {
			return fmt.Errorf("expected a valid prefix for the configuration: %w", err)
		}
	}
	args.userPrefix = strings.Trim(args.userPrefix, " \t")
	if len([]rune(args.userPrefix)) > maxLengthUserPrefix {
		return fmt.Errorf("prefix length is limited to %d characters", maxLengthUserPrefix)
	}
	return nil
}

func (h *hyperfleetOidcConfigCreate) PreRequest(
	ctx context.Context,
	r *rosa.Runtime,
	input *hfpathbind.OidcConfigCreateInput,
) error {
	// Bridge flags that conflict with OCM v1 registrations (if any)
	// input.Name comes from --name (or could be from --prefix)
	if input.Name == "" && args.userPrefix != "" {
		input.Name = args.userPrefix
	}

	// Set OIDC config type based on --managed flag
	if args.managed {
		input.Type = "managed"
	} else {
		input.Type = "unmanaged"
		// For unmanaged configs, installer role ARN is required
		if args.installerRoleArn == "" {
			return fmt.Errorf("--installer-role-arn is required for unmanaged OIDC configs")
		}
		input.InstallerRoleArn = args.installerRoleArn
	}

	r.Reporter.Debugf("Creating OIDC config via Platform API")
	r.Reporter.Debugf("  Config Name: '%s'", input.Name)
	r.Reporter.Debugf("  Config Type: %s", input.Type)
	if !args.managed {
		r.Reporter.Debugf("  Installer Role ARN: %s", input.InstallerRoleArn)
	}

	return nil
}

func (h *hyperfleetOidcConfigCreate) PostExpand(
	_ context.Context,
	r *rosa.Runtime,
	input *hfpathbind.OidcConfigCreateInput,
	obj *v1alpha1.OidcConfig,
) error {
	// Nothing additional to set - pathbind.Expand handles the field mapping
	return nil
}

func (h *hyperfleetOidcConfigCreate) PostResponse(
	ctx context.Context,
	r *rosa.Runtime,
	oidcConfig *v1alpha1.OidcConfig,
) error {
	if output.HasFlag() {
		if err := output.Print(oidcConfig); err != nil {
			return err
		}
	}

	if oidcConfig.Spec.IssuerUrl != "" {
		mode, err := interactive.GetMode()
		if err != nil {
			return err
		}
		if err := createOidcProviderFn(ctx, r, oidcConfig, mode); err != nil {
			return fmt.Errorf("failed to create OIDC provider for config '%s' (the config was created): %w",
				oidcConfig.Name, err)
		}
	}

	if output.HasFlag() {
		return nil
	}

	r.Reporter.Infof("OIDC config '%s' created successfully", oidcConfig.Name)
	r.Reporter.Infof("  Type: %s", oidcConfig.Spec.Type)
	if oidcConfig.Spec.IssuerUrl != "" {
		r.Reporter.Infof("  Issuer URL: %s", oidcConfig.Spec.IssuerUrl)
	}
	if oidcConfig.Spec.SecretArn != "" {
		r.Reporter.Infof("  Secret ARN: %s", oidcConfig.Spec.SecretArn)
	}

	return nil
}

var createOidcProviderFn = createOidcProvider

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
		oidcProviderLogf(r, "OIDC provider already exists")
		return nil
	}

	switch mode {
	case interactive.ModeAuto:
		// Create the OIDC provider in AWS
		oidcProviderLogf(r, "Creating OIDC provider using '%s'", r.Creator.ARN)
		oidcProviderARN, err := r.AWSClient.CreateOpenIDConnectProvider(
			oidcConfig.Spec.IssuerUrl,
			thumbprint,
			"", // No cluster ID for standalone OIDC config
		)
		if err != nil {
			return fmt.Errorf("failed to create OIDC provider: %v", err)
		}
		oidcProviderLogf(r, "Created OIDC provider with ARN '%s'", oidcProviderARN)

	case interactive.ModeManual:
		// Print manual commands
		commands, err := buildOidcProviderCommands(oidcConfig.Spec.IssuerUrl, thumbprint)
		if err != nil {
			return fmt.Errorf("failed to build OIDC provider commands: %v", err)
		}
		oidcProviderLogf(r, "Run the following commands to create the OIDC provider:\n")
		if output.HasFlag() {
			r.Reporter.Debugf("%s", commands)
		} else {
			fmt.Println(commands)
		}

	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}

	return nil
}

func oidcProviderLogf(r *rosa.Runtime, format string, args ...any) {
	if output.HasFlag() {
		r.Reporter.Debugf(format, args...)
		return
	}
	r.Reporter.Infof(format, args...)
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
