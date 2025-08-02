/*
Copyright (c) 2025 Red Hat, Inc.

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

package iamserviceaccount

import (
	"context"
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/iamserviceaccount"
	iamServiceAccountOpts "github.com/openshift/rosa/pkg/options/iamserviceaccount"
	"github.com/openshift/rosa/pkg/rosa"
)

func NewCreateIamServiceAccountCommand() *cobra.Command {
	cmd, options := iamServiceAccountOpts.BuildIamServiceAccountCreateCommandWithOptions()
	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCMAndAWS(), CreateIamServiceAccountRunner(options))
	return cmd
}

var Cmd = NewCreateIamServiceAccountCommand()

func CreateIamServiceAccountRunner(userOptions *iamServiceAccountOpts.CreateIamServiceAccountUserOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		cluster := r.FetchCluster()

		// Validate cluster has STS enabled
		if cluster.AWS().STS().RoleARN() == "" {
			return fmt.Errorf("cluster '%s' is not an STS cluster", cluster.Name())
		}

		// Validate service account names
		if len(userOptions.ServiceAccountNames) == 0 {
			return fmt.Errorf("at least one service account name is required")
		}

		// Validate that at least one policy is specified
		if len(userOptions.PolicyArns) == 0 && userOptions.InlinePolicy == "" {
			return fmt.Errorf("at least one policy ARN or inline policy must be specified")
		}

		// Validate policy ARNs
		for _, arn := range userOptions.PolicyArns {
			if err := aws.ARNValidator(arn); err != nil {
				return fmt.Errorf("invalid policy ARN '%s': %s", arn, err)
			}
		}

		// Generate role name if not provided
		roleName := userOptions.RoleName
		if roleName == "" {
			if len(userOptions.ServiceAccountNames) > 1 {
				return fmt.Errorf("role name is required when specifying multiple service accounts")
			}
			roleName = iamserviceaccount.GenerateRoleName(cluster.Name(), userOptions.Namespace, userOptions.ServiceAccountNames[0])
		}

		// Generate trust policy using existing helpers
		oidcConfig := cluster.AWS().STS().OidcConfig()
		if oidcConfig == nil {
			return fmt.Errorf("cluster '%s' does not have OIDC configuration", cluster.Name())
		}

		// Use existing AWS client functions
		serviceAccounts := make([]iamserviceaccount.ServiceAccountIdentifier, len(userOptions.ServiceAccountNames))
		for i, name := range userOptions.ServiceAccountNames {
			serviceAccounts[i] = iamserviceaccount.ServiceAccountIdentifier{
				Name:      name,
				Namespace: userOptions.Namespace,
			}
		}

		oidcProviderARN, err := getOIDCProviderARN(r, cluster)
		if err != nil {
			return fmt.Errorf("failed to get OIDC provider ARN: %s", err)
		}

		trustPolicy := iamserviceaccount.GenerateTrustPolicyMultiple(oidcProviderARN, serviceAccounts)
		tags := iamserviceaccount.GenerateDefaultTags(cluster.Name(), userOptions.Namespace, userOptions.ServiceAccountNames[0])

		// Use existing EnsureRole function
		roleARN, err := r.AWSClient.EnsureRole(r.Reporter, roleName, trustPolicy, userOptions.PermissionsBoundary, "", tags, userOptions.Path, false)
		if err != nil {
			return fmt.Errorf("failed to create role: %s", err)
		}

		r.Reporter.Infof("Created IAM role '%s' with ARN '%s'", roleName, roleARN)

		// Attach managed policies
		for _, policyARN := range userOptions.PolicyArns {
			err = r.AWSClient.AttachRolePolicy(r.Reporter, roleName, policyARN)
			if err != nil {
				return fmt.Errorf("failed to attach policy '%s' to role '%s': %s", policyARN, roleName, err)
			}
		}

		// Handle inline policy
		if userOptions.InlinePolicy != "" {
			inlinePolicy := userOptions.InlinePolicy

			// Process inline policy if it's a file reference
			if strings.HasPrefix(inlinePolicy, "file://") {
				policyPath := strings.TrimPrefix(inlinePolicy, "file://")
				policyBytes, err := os.ReadFile(policyPath)
				if err != nil {
					return fmt.Errorf("failed to read policy file '%s': %s", policyPath, err)
				}
				inlinePolicy = string(policyBytes)
			}

			// Generate inline policy name
			policyName := fmt.Sprintf("%s-inline-policy", roleName)
			err = r.AWSClient.PutRolePolicy(roleName, policyName, inlinePolicy)
			if err != nil {
				return fmt.Errorf("failed to attach inline policy to role '%s': %s", roleName, err)
			}
			r.Reporter.Infof("Attached inline policy '%s' to role '%s(https://console.aws.amazon.com/iam/home?#/roles/%s)'", policyName, roleName, roleName)
		}

		return nil
	}
}

func getOIDCProviderARN(r *rosa.Runtime, cluster *cmv1.Cluster) (string, error) {
	oidcConfig := cluster.AWS().STS().OidcConfig()
	if oidcConfig == nil {
		return "", fmt.Errorf("cluster does not have OIDC configuration")
	}

	providers, err := r.AWSClient.ListOidcProviders("", oidcConfig)
	if err != nil {
		return "", fmt.Errorf("failed to list OIDC providers: %w", err)
	}

	if len(providers) == 0 {
		return "", fmt.Errorf("no OIDC provider found for cluster")
	}

	return providers[0].Arn, nil
}
