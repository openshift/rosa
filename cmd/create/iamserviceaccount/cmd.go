/*
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

		// Get AWS creator information to determine partition for FedRAMP
		creator, err := r.AWSClient.GetCreator()
		if err != nil {
			return fmt.Errorf("failed to get AWS creator information: %s", err)
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

		managedPolicies := false
		roleARN, err := r.AWSClient.EnsureRole(r.Reporter, roleName, trustPolicy, userOptions.PermissionsBoundary, "", tags, userOptions.Path, managedPolicies)
		if err != nil {
			return fmt.Errorf("failed to create role: %s", err)
		}

		// For FedRAMP environments, update the role ARN to use the correct partition
		if creator.IsGovcloud {
			roleARN = iamserviceaccount.GetRoleARN(creator.AccountID, roleName, userOptions.Path, creator.Partition)
		}

		r.Reporter.Infof("Created IAM role '%s' with ARN '%s' using OIDC '%s'", roleName, roleARN, oidcProviderARN)

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
			r.Reporter.Infof("Attached inline policy '%s' to role '%s'", policyName, roleName)
		}

		return nil
	}
}

func getOIDCProviderARN(r *rosa.Runtime, cluster *cmv1.Cluster) (string, error) {
	oidcConfig := cluster.AWS().STS().OidcConfig()
	oidcEndpointURL := cluster.AWS().STS().OIDCEndpointURL()

	// For clusters with OIDC configuration, try the standard approach first
	if oidcConfig != nil {
		providers, err := r.AWSClient.ListOidcProviders(cluster.ID(), oidcConfig)
		if err != nil {
			return "", fmt.Errorf("failed to list OIDC providers: %w", err)
		}

		if len(providers) > 0 {
			return providers[0].Arn, nil
		}
	}

	// Fallback: For classic OIDC or BYO OIDC where cluster ID-based search fails,
	// use the OIDC endpoint URL to check for existing providers
	if oidcEndpointURL != "" {
		creator, err := r.AWSClient.GetCreator()
		if err != nil {
			return "", fmt.Errorf("failed to get AWS creator information: %s", err)
		}

		// Check if OIDC provider exists using endpoint URL
		exists, err := r.AWSClient.HasOpenIDConnectProvider(oidcEndpointURL, creator.Partition, creator.AccountID)
		if err != nil {
			return "", fmt.Errorf("failed to check OIDC provider existence: %w", err)
		}

		if exists {
			// Generate the provider ARN using the endpoint URL
			return generateOIDCProviderARN(oidcEndpointURL, creator.Partition, creator.AccountID), nil
		}
	}

	// If we reach here, no OIDC provider was found
	if oidcConfig == nil && oidcEndpointURL == "" {
		return "", fmt.Errorf("cluster does not have OIDC configuration")
	}

	return "", fmt.Errorf("no OIDC provider found for cluster")
}

func generateOIDCProviderARN(oidcEndpointURL, partition, accountID string) string {
	// Parse the endpoint URL to extract the host part
	// e.g., "https://example.com/path" -> "example.com/path"
	if strings.HasPrefix(oidcEndpointURL, "https://") {
		oidcEndpointURL = strings.TrimPrefix(oidcEndpointURL, "https://")
	} else if strings.HasPrefix(oidcEndpointURL, "http://") {
		oidcEndpointURL = strings.TrimPrefix(oidcEndpointURL, "http://")
	}

	return fmt.Sprintf("arn:%s:iam::%s:oidc-provider/%s", partition, accountID, oidcEndpointURL)
}
