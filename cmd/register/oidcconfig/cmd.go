/*
Copyright (c) 2023 Red Hat, Inc.

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

package oidcconfig

import (
	"context"
	"fmt"

	"github.com/briandowns/spinner"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/create/oidcprovider"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	. "github.com/openshift/rosa/pkg/constants"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	interactiveRoles "github.com/openshift/rosa/pkg/interactive/roles"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	installerRoleArn string
	issuerUrl        string
	secretArn        string
}

func SetCreateOidcProviderCommand(cmd rosa.CommandInterface) {
	oidcprovider.CreateOidcProvider = cmd
}

func NewRegisterOidcConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "oidc-config",
		Aliases: []string{"oidcconfig", "oidcconfig"},
		Short:   "Registers unmanaged OIDC config with Openshift Clusters Manager.",
		Long:    "Registers unmanaged OIDC config with Openshift Clusters Manager.",
		Example: `  # Register OIDC config
		rosa register oidc-config`,
		Run:  rosa.DefaultRunner(rosa.RuntimeWithOCM(), RegisterOidcConfigRunner()),
		Args: cobra.NoArgs,
	}

	flags := cmd.Flags()

	// normalizing installer role argument to support deprecated flag
	flags.SetNormalizeFunc(arguments.NormalizeFlags)
	flags.StringVar(
		&args.installerRoleArn,
		InstallerRoleArnFlag,
		"",
		"STS Role ARN with get secrets permission.",
	)

	flags.StringVar(
		&args.issuerUrl,
		IssuerUrlFlag,
		"",
		"Issuer/Bucket URL.",
	)

	flags.StringVar(
		&args.secretArn,
		SecretArnFlag,
		"",
		"Secrets Manager ARN with private key secret.",
	)

	interactive.AddModeFlag(cmd)
	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
	output.AddFlag(cmd)

	return cmd
}

func checkInteractiveModeNeeded(cmd *cobra.Command) {
	installerRoleArnNotSet := (!cmd.Flags().Changed(InstallerRoleArnFlag) || args.installerRoleArn == "") && !confirm.Yes()
	issuerUrlNotSet := (!cmd.Flags().Changed(IssuerUrlFlag) || args.issuerUrl == "")
	secretArnNotSet := (!cmd.Flags().Changed(SecretArnFlag) || args.secretArn == "")
	if installerRoleArnNotSet || issuerUrlNotSet || secretArnNotSet {
		interactive.Enable()
		return
	}
}

func RegisterOidcConfigRunner() rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		mode, err := interactive.GetMode()
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		if !cmd.Flags().Changed("mode") {
			mode, err = interactive.GetOptionMode(cmd, mode, "OIDC Provider creation mode")
			if err != nil {
				return fmt.Errorf("Expected a valid OIDC Provider creation mode: %s", err)
			}
		}

		checkInteractiveModeNeeded(cmd)

		if !cmd.Flags().Changed(InstallerRoleArnFlag) && (interactive.Enabled() || confirm.Yes()) {
			args.installerRoleArn = interactiveRoles.
				GetInstallerRoleArn(runtime, cmd, args.installerRoleArn, MinorVersionForGetSecret, runtime.AWSClient.FindRoleARNs)
		}
		roleName, _ := aws.GetResourceIdFromARN(args.installerRoleArn)
		if !output.HasFlag() && runtime.Reporter.IsTerminal() {
			runtime.Reporter.Infof("Using %s for the installer role", args.installerRoleArn)
		}
		err = aws.ARNValidator(args.installerRoleArn)
		if err != nil {
			return fmt.Errorf("Expected a valid ARN: %s", err)
		}
		roleExists, _, err := runtime.AWSClient.CheckRoleExists(roleName)
		if err != nil {
			return fmt.Errorf("There was a problem checking if role '%s' exists: %v", args.installerRoleArn, err)
		}
		if !roleExists {
			return fmt.Errorf("Role '%s' does not exist", args.installerRoleArn)
		}
		isValid, err := runtime.AWSClient.ValidateAccountRoleVersionCompatibility(
			roleName, aws.InstallerAccountRole, MinorVersionForGetSecret)
		if err != nil {
			return fmt.Errorf("There was a problem listing role tags: %v", err)
		}
		if !isValid {
			return fmt.Errorf("Role '%s' is not of minimum version '%s'", args.installerRoleArn, MinorVersionForGetSecret)
		}

		if interactive.Enabled() && !cmd.Flags().Changed(IssuerUrlFlag) {
			issuerUrl, err := interactive.GetString(interactive.Input{
				Question:   "Issuer URL (please include 'https://')",
				Help:       cmd.Flags().Lookup(IssuerUrlFlag).Usage,
				Required:   true,
				Validators: []interactive.Validator{interactive.IsURLHttps},
			})
			if err != nil {
				return fmt.Errorf("Expected an issuer URL: %s", err)
			}
			args.issuerUrl = issuerUrl
		}
		if err := interactive.IsURLHttps(args.issuerUrl); err != nil {
			return fmt.Errorf("%v", err)
		}

		if interactive.Enabled() && !cmd.Flags().Changed(SecretArnFlag) {
			secretArn, err := interactive.GetString(interactive.Input{
				Question:   "Secret ARN",
				Help:       cmd.Flags().Lookup(SecretArnFlag).Usage,
				Required:   true,
				Validators: []interactive.Validator{aws.SecretManagerArnValidator},
			})
			if err != nil {
				return fmt.Errorf("Expected a secret ARN: %s", err)
			}
			args.secretArn = secretArn
		}
		if err := aws.SecretManagerArnValidator(args.secretArn); err != nil {
			return fmt.Errorf("%v", err)
		}

		var spin *spinner.Spinner
		if spin != nil {
			spin.Start()
		}
		installerRoleArn := args.installerRoleArn
		oidcConfig, err := v1.NewOidcConfig().
			Managed(false).
			SecretArn(args.secretArn).
			IssuerUrl(args.issuerUrl).
			InstallerRoleArn(installerRoleArn).
			Build()
		if err == nil {
			oidcConfig, err = runtime.OCMClient.CreateOidcConfig(oidcConfig)
		}
		if err != nil {
			if spin != nil {
				spin.Stop()
			}
			return fmt.Errorf("There was a problem building your unmanaged OIDC Configuration: %v", err)
		}
		if output.HasFlag() {
			err = output.Print(oidcConfig)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			return nil
		}
		if runtime.Reporter.IsTerminal() {
			if spin != nil {
				spin.Stop()
			}
			output := fmt.Sprintf(InformOperatorRolesOutput, oidcConfig.ID())
			runtime.Reporter.Infof(output)
		}
		arguments.DisableRegionDeprecationWarning = true // disable region deprecation warning
		cmd = oidcprovider.CreateOidcProvider.NewCommand()
		args := []string{"", mode, args.issuerUrl}
		cmd.ParseFlags(args)
		runner := oidcprovider.CreateOidcProvider.Runner()
		arguments.DisableRegionDeprecationWarning = false // enable region deprecation again
		err = runner(nil, runtime, cmd, args)
		if err != nil {
			return fmt.Errorf("%s", err)
		}

		return nil
	}
}
