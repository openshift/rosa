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
	"os"
	"strings"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"

	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, and hfCreateOidcConfig are package-level
// vars so tests can stub the hyperfleet dispatch path.
var (
	hfEnabled          = hyperfleet.Enabled
	hfExitFn           = func(code int) { os.Exit(code) }
	hfDeleteOidcConfig = func() {
		r := rosa.NewRuntime().WithHyperFleet().WithAWSOnly()
		defer r.Cleanup()
		runHyperfleetDeleteOidcConfig(r)
	}
)

// runHyperfleetDeleteOidcConfig deletes an OIDC Config via the Platform API v2.
func runHyperfleetDeleteOidcConfig(r *rosa.Runtime) {
	ctx := context.Background()

	// Validate required flag
	if args.oidcConfigId == "" {
		r.Reporter.Errorf("--oidc-config-id is required")
		hfExitFn(1)
		return
	}

	// Get mode for OIDC provider deletion
	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		hfExitFn(1)
		return
	}

	// Get OIDC config from Platform API
	r.Reporter.Debugf("Retrieving OIDC config '%s' from Platform API", args.oidcConfigId)
	oidcConfig, err := r.HyperFleetClient.HyperfleetV1alpha1().OidcConfigs().Get(
		ctx,
		args.oidcConfigId,
		platform.GetOptions{},
	)
	if err != nil {
		r.Reporter.Errorf("Failed to retrieve OIDC config '%s': %v", args.oidcConfigId, err)
		hfExitFn(1)
		return
	}

	r.Reporter.Infof("Deleting OIDC config '%s' (Type: %s)", oidcConfig.Name, oidcConfig.Spec.Type)

	// Delete OIDC provider from AWS if issuer URL exists
	if oidcConfig.Spec.IssuerUrl != "" {
		err = deleteOidcProvider(ctx, r, oidcConfig.Spec.IssuerUrl, mode)
		if err != nil {
			r.Reporter.Errorf("Failed to delete OIDC provider: %v", err)
			hfExitFn(1)
			return
		}
	}

	// For unmanaged configs, delete S3 bucket and secrets manager secret
	if oidcConfig.Spec.Type == "unmanaged" && oidcConfig.Spec.SecretArn != "" {
		err = deleteUnmanagedResources(r, oidcConfig.Spec.SecretArn, mode)
		if err != nil {
			r.Reporter.Errorf("Failed to delete unmanaged resources: %v", err)
			hfExitFn(1)
			return
		}
	}

	// Delete OIDC config from Platform API
	r.Reporter.Debugf("Deleting OIDC config '%s' from Platform API", args.oidcConfigId)
	err = r.HyperFleetClient.HyperfleetV1alpha1().OidcConfigs().Delete(
		ctx,
		args.oidcConfigId,
		platform.DeleteOptions{},
	)
	if err != nil {
		r.Reporter.Errorf("Failed to delete OIDC config from Platform API: %v", err)
		hfExitFn(1)
		return
	}

	r.Reporter.Infof("OIDC config '%s' has been deleted successfully", args.oidcConfigId)
	if mode == interactive.ModeManual {
		r.Reporter.Infof("Remember to run the provided commands to clean up AWS resources")
	}
}

// deleteOidcProvider deletes the AWS OIDC provider
func deleteOidcProvider(ctx context.Context, r *rosa.Runtime, issuerUrl, mode string) error {
	// Build the provider ARN
	issuerDomain := issuerUrl
	if strings.HasPrefix(issuerUrl, "https://") {
		issuerDomain = issuerUrl[8:]
	}

	providerARN := fmt.Sprintf("arn:%s:iam::%s:oidc-provider/%s",
		r.Creator.Partition,
		r.Creator.AccountID,
		issuerDomain,
	)

	// Check if OIDC provider exists
	oidcProviderExists, err := r.AWSClient.HasOpenIDConnectProvider(
		issuerUrl,
		r.Creator.Partition,
		r.Creator.AccountID,
	)
	if err != nil {
		r.Reporter.Warnf("Failed to verify if OIDC provider exists: %s", err)
	}
	if !oidcProviderExists {
		r.Reporter.Infof("OIDC provider does not exist, skipping deletion")
		return nil
	}

	switch mode {
	case interactive.ModeAuto:
		// Delete the OIDC provider in AWS
		r.Reporter.Infof("Deleting OIDC provider with ARN '%s'", providerARN)
		err := r.AWSClient.DeleteOpenIDConnectProvider(providerARN)
		if err != nil {
			return fmt.Errorf("failed to delete OIDC provider: %v", err)
		}
		r.Reporter.Infof("Deleted OIDC provider successfully")

	case interactive.ModeManual:
		// Print manual commands
		command := buildDeleteOidcProviderCommand(providerARN)
		r.Reporter.Infof("Run the following command to delete the OIDC provider:\n")
		fmt.Println(command)

	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}

	return nil
}

// deleteUnmanagedResources deletes S3 bucket and secrets manager secret for unmanaged configs
func deleteUnmanagedResources(r *rosa.Runtime, secretArn, mode string) error {
	// Extract bucket name from secret ARN
	// The secret has format: rosa-private-key-<prefix>-oidc-<random-hash>-<random-aws-hash>
	// The bucket is: <prefix>-oidc-<random-hash>
	secretResourceName := ""
	if strings.Contains(secretArn, ":secret:") {
		parts := strings.Split(secretArn, ":secret:")
		if len(parts) > 1 {
			secretResourceName = parts[1]
			// Remove the last part (AWS random hash after the last -)
			lastDash := strings.LastIndex(secretResourceName, "-")
			if lastDash != -1 {
				secretResourceName = secretResourceName[:lastDash]
			}
		}
	}

	bucketName := strings.TrimPrefix(secretResourceName, "rosa-private-key-")

	r.Reporter.Debugf("Deleting unmanaged resources: bucket=%s, secret=%s", bucketName, secretArn)

	switch mode {
	case interactive.ModeAuto:
		// Delete secrets manager secret
		r.Reporter.Infof("Deleting secret from AWS Secrets Manager")
		err := r.AWSClient.DeleteSecretInSecretsManager(secretArn)
		if err != nil {
			return fmt.Errorf("failed to delete secret from secrets manager: %v", err)
		}

		// Delete S3 bucket
		if bucketName != "" {
			r.Reporter.Infof("Deleting S3 bucket '%s'", bucketName)
			err = r.AWSClient.DeleteS3Bucket(bucketName)
			if err != nil {
				return fmt.Errorf("failed to delete S3 bucket: %v", err)
			}
		}

		r.Reporter.Infof("Deleted unmanaged OIDC resources successfully")

	case interactive.ModeManual:
		// Print manual commands
		commands := buildDeleteUnmanagedResourcesCommands(secretArn, bucketName, args.region)
		r.Reporter.Infof("Run the following commands to delete unmanaged resources:\n")
		fmt.Println(commands)

	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}

	return nil
}

// buildDeleteOidcProviderCommand builds the AWS CLI command for manual OIDC provider deletion
func buildDeleteOidcProviderCommand(providerARN string) string {
	command := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.DeleteOpenIdConnectProvider).
		AddParam(awscb.OpenIdConnectProviderArn, providerARN).
		Build()
	return command
}

// buildDeleteUnmanagedResourcesCommands builds AWS CLI commands for deleting unmanaged resources
func buildDeleteUnmanagedResourcesCommands(secretArn, bucketName, region string) string {
	commands := []string{}

	// Delete secret command
	deleteSecretCommand := awscb.NewSecretsManagerCommandBuilder().
		SetCommand(awscb.DeleteSecret).
		AddParam(awscb.SecretID, secretArn).
		AddParam(awscb.Region, region).
		Build()
	commands = append(commands, deleteSecretCommand)

	// Empty and delete S3 bucket commands
	if bucketName != "" {
		emptyS3BucketCommand := awscb.NewS3CommandBuilder().
			SetCommand(awscb.Remove).
			AddValueNoParam(fmt.Sprintf("s3://%s", bucketName)).
			AddParamNoValue(awscb.Recursive).
			Build()
		commands = append(commands, emptyS3BucketCommand)

		deleteS3BucketCommand := awscb.NewS3CommandBuilder().
			SetCommand(awscb.RemoveBucket).
			AddValueNoParam(fmt.Sprintf("s3://%s", bucketName)).
			Build()
		commands = append(commands, deleteS3BucketCommand)
	}

	return awscb.JoinCommands(commands)
}
