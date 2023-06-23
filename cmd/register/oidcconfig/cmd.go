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
	// nolint:gosec

	//#nosec GSC-G505 -- Import blacklist: crypto/sha1

	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/briandowns/spinner"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/cmd/create/oidcprovider"
	"github.com/openshift/rosa/pkg/aws"
	. "github.com/openshift/rosa/pkg/constants"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

var args struct {
	installerRoleArn string
	issuerUrl        string
	secretArn        string
}

var Cmd = &cobra.Command{
	Use:     "oidc-config",
	Aliases: []string{"oidcconfig", "oidcconfig"},
	Short:   "Registers unmanaged OIDC config with Openshift Clusters Manager.",
	Long:    "Registers unmanaged OIDC config with Openshift Clusters Manager.",
	Example: `  # Register OIDC config
	rosa register oidc-config`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

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

	aws.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
	output.AddFlag(Cmd)
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

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if !cmd.Flags().Changed("mode") {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "OIDC Provider creation mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid OIDC Provider creation mode: %s", err)
			os.Exit(1)
		}
	}

	checkInteractiveModeNeeded(cmd)

	if interactive.Enabled() || (confirm.Yes() && args.installerRoleArn == "") {
		args.installerRoleArn = interactive.GetInstallerRoleArn(r, cmd, args.installerRoleArn, MinorVersionForGetSecret)
	}
	roleName, _ := aws.GetResourceIdFromARN(args.installerRoleArn)
	if !output.HasFlag() && r.Reporter.IsTerminal() {
		r.Reporter.Infof("Using %s for the installer role", args.installerRoleArn)
	}
	err = aws.ARNValidator(args.installerRoleArn)
	if err != nil {
		r.Reporter.Errorf("Expected a valid ARN: %s", err)
		os.Exit(1)
	}
	roleExists, _, err := r.AWSClient.CheckRoleExists(roleName)
	if err != nil {
		r.Reporter.Errorf("There was a problem checking if role '%s' exists: %v", args.installerRoleArn, err)
		os.Exit(1)
	}
	if !roleExists {
		r.Reporter.Errorf("Role '%s' does not exist", args.installerRoleArn)
		os.Exit(1)
	}
	isValid, err := r.AWSClient.ValidateAccountRoleVersionCompatibility(
		roleName, aws.InstallerAccountRole, MinorVersionForGetSecret)
	if err != nil {
		r.Reporter.Errorf("There was a problem listing role tags: %v", err)
		os.Exit(1)
	}
	if !isValid {
		r.Reporter.Errorf("Role '%s' is not of minimum version '%s'", args.installerRoleArn, MinorVersionForGetSecret)
		os.Exit(1)
	}

	if interactive.Enabled() && args.issuerUrl == "" {
		issuerUrl, err := interactive.GetString(interactive.Input{
			Question: "Issuer URL (please include 'https://')",
			Help:     cmd.Flags().Lookup(IssuerUrlFlag).Usage,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a issuer URL: %s", err)
			os.Exit(1)
		}
		args.issuerUrl = issuerUrl
	}
	parsedURI, err := url.ParseRequestURI(args.issuerUrl)
	if err != nil {
		r.Reporter.Errorf("Invalid issuer URL: %s", err)
		os.Exit(1)
	}
	if parsedURI.Scheme != helper.ProtocolHttps {
		r.Reporter.Errorf("Expected OIDC endpoint URL '%s' to use an https:// scheme", args.issuerUrl)
		os.Exit(1)
	}

	if interactive.Enabled() && args.secretArn == "" {
		secretArn, err := interactive.GetString(interactive.Input{
			Question: "Secret ARN",
			Help:     cmd.Flags().Lookup(SecretArnFlag).Usage,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a secret ARN: %s", err)
			os.Exit(1)
		}
		args.secretArn = secretArn
	}
	if !arn.IsARN(args.secretArn) {
		r.Reporter.Errorf("Secret ARN '%s' is not a valid ARN", args.secretArn)
		os.Exit(1)
	}
	parsedSecretArn, err := arn.Parse(args.secretArn)
	if err != nil {
		r.Reporter.Errorf("Secret ARN '%s' is not a valid ARN", args.secretArn)
		os.Exit(1)
	}
	if parsedSecretArn.Service != SecretsManagerService {
		r.Reporter.Errorf("Secret ARN '%s' is not a valid secrets manager ARN", args.secretArn)
		os.Exit(1)
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
		oidcConfig, err = r.OCMClient.CreateOidcConfig(oidcConfig)
	}
	if err != nil {
		if spin != nil {
			spin.Stop()
		}
		r.Reporter.Errorf("There was a problem building your unmanaged OIDC Configuration: %v", err)
		os.Exit(1)
	}
	if output.HasFlag() {
		err = output.Print(oidcConfig)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if r.Reporter.IsTerminal() {
		if spin != nil {
			spin.Stop()
		}
		output := fmt.Sprintf(InformOperatorRolesOutput, oidcConfig.ID())
		r.Reporter.Infof(output)
	}
	oidcprovider.Cmd.Run(oidcprovider.Cmd, []string{"", mode, args.issuerUrl})
}
