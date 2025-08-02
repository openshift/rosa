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
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
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
		var err error
		options := NewCreateIamServiceAccountOptions()
		options.args = userOptions
		clusterKey := r.GetClusterKey()

		// Get interactive mode
		mode, err := interactive.GetMode()
		if err != nil {
			_ = r.Reporter.Errorf("%s", err)
			return err
		}

		// Determine if interactive mode is needed
		if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
			interactive.Enable()
		}

		cluster := r.FetchCluster()
		if cluster.Name() != clusterKey && cluster.ID() != clusterKey {
			cluster, err = r.OCMClient.GetCluster(clusterKey, r.Creator)
			if err != nil {
				_ = r.Reporter.Errorf("Failed to get cluster '%s': %s", clusterKey, err)
				return err
			}
		}

		// Validate cluster has STS enabled
		if cluster.AWS().STS().RoleARN() == "" {
			return fmt.Errorf("cluster '%s' is not an STS cluster", cluster.Name())
		}

		// Get OIDC configuration
		oidcConfig := cluster.AWS().STS().OidcConfig()
		if oidcConfig == nil {
			return fmt.Errorf("cluster '%s' does not have OIDC configuration", cluster.Name())
		}

		// Get OIDC provider ARN
		oidcProviderARN, err := getOIDCProviderARN(r, cluster)
		if err != nil {
			_ = r.Reporter.Errorf("Failed to get OIDC provider ARN: %s", err)
			return err
		}

		// Validate service account names
		serviceAccountNames := userOptions.ServiceAccountNames
		if len(serviceAccountNames) == 0 {
			if interactive.Enabled() {
				// In interactive mode, ask for at least one service account
				saName, err := interactive.GetString(interactive.Input{
					Question: "Service account name",
					Help:     cmd.Flags().Lookup("name").Usage,
					Required: true,
					Validators: []interactive.Validator{
						func(val interface{}) error {
							return iamserviceaccount.ValidateServiceAccountName(val.(string))
						},
					},
				})
				if err != nil {
					_ = r.Reporter.Errorf("Expected a valid service account name: %s", err)
					return err
				}
				serviceAccountNames = []string{saName}

				// Ask if user wants to add more service accounts
				for {
					addMore, err := interactive.GetBool(interactive.Input{
						Question: "Add another service account to this role?",
						Default:  false,
					})
					if err != nil || !addMore {
						break
					}

					saName, err = interactive.GetString(interactive.Input{
						Question: "Additional service account name",
						Required: true,
						Validators: []interactive.Validator{
							func(val interface{}) error {
								return iamserviceaccount.ValidateServiceAccountName(val.(string))
							},
						},
					})
					if err != nil {
						_ = r.Reporter.Errorf("Expected a valid service account name: %s", err)
						return err
					}
					serviceAccountNames = append(serviceAccountNames, saName)
				}
			} else {
				return fmt.Errorf("at least one service account name is required")
			}
		}

		// Validate all service account names
		for _, saName := range serviceAccountNames {
			if err := iamserviceaccount.ValidateServiceAccountName(saName); err != nil {
				_ = r.Reporter.Errorf("Invalid service account name '%s': %s", saName, err)
				return fmt.Errorf("invalid service account name '%s': %s", saName, err)
			}
		}

		// Validate namespace
		namespace := userOptions.Namespace
		if interactive.Enabled() {
			namespace, err = interactive.GetString(interactive.Input{
				Question: "Namespace",
				Help:     cmd.Flags().Lookup("namespace").Usage,
				Default:  namespace,
				Required: true,
				Validators: []interactive.Validator{
					func(val interface{}) error {
						return iamserviceaccount.ValidateNamespaceName(val.(string))
					},
				},
			})
			if err != nil {
				_ = r.Reporter.Errorf("Expected a valid namespace: %s", err)
				return err
			}
		}

		if err := iamserviceaccount.ValidateNamespaceName(namespace); err != nil {
			_ = r.Reporter.Errorf("Invalid namespace: %s", err)
			return fmt.Errorf("invalid namespace: %s", err)
		}

		// Generate or validate role name
		roleName := userOptions.RoleName
		if roleName == "" {
			if len(serviceAccountNames) == 1 {
				// Single service account - auto-generate role name
				roleName = iamserviceaccount.GenerateRoleName(cluster.Name(), namespace, serviceAccountNames[0])
			} else if !interactive.Enabled() {
				// Multiple service accounts in non-interactive mode - require explicit role name
				return fmt.Errorf("role name (--role-name) is required when specifying multiple service accounts")
			}
			// In interactive mode with multiple service accounts, roleName will be empty and we'll prompt below
		}

		if interactive.Enabled() {
			// For multiple service accounts with no role name, make it clear it's required
			question := "IAM role name"
			if len(serviceAccountNames) > 1 && userOptions.RoleName == "" {
				question = "IAM role name (required for multiple service accounts)"
			}

			roleName, err = interactive.GetString(interactive.Input{
				Question: question,
				Help:     cmd.Flags().Lookup("role-name").Usage,
				Default:  roleName,
				Required: true,
				Validators: []interactive.Validator{
					func(val interface{}) error {
						if !aws.RoleNameRE.MatchString(val.(string)) {
							return fmt.Errorf("invalid IAM role name")
						}
						return nil
					},
				},
			})
			if err != nil {
				_ = r.Reporter.Errorf("Expected a valid role name: %s", err)
				return err
			}
		}

		// Validate policy ARNs
		policyArns := userOptions.PolicyArns
		if interactive.Enabled() && len(policyArns) == 0 {
			policyArnsStr, err := interactive.GetString(interactive.Input{
				Question: "Policy ARNs (comma-separated)",
				Help:     cmd.Flags().Lookup("policy-arns").Usage,
			})
			if err != nil {
				_ = r.Reporter.Errorf("Expected valid policy ARNs: %s", err)
				return err
			}
			if policyArnsStr != "" {
				policyArns = strings.Split(policyArnsStr, ",")
				for i, arn := range policyArns {
					policyArns[i] = strings.TrimSpace(arn)
				}
			}
		}

		// Validate each policy ARN
		for _, arn := range policyArns {
			if err := aws.ARNValidator(arn); err != nil {
				_ = r.Reporter.Errorf("Invalid policy ARN '%s': %s", arn, err)
				return fmt.Errorf("invalid policy ARN '%s': %s", arn, err)
			}
		}

		// Handle inline policy
		inlinePolicy := userOptions.InlinePolicy
		if interactive.Enabled() && inlinePolicy == "" && len(policyArns) == 0 {
			inlinePolicy, err = interactive.GetString(interactive.Input{
				Question: "Inline policy (JSON document or file://path)",
				Help:     cmd.Flags().Lookup("inline-policy").Usage,
			})
			if err != nil {
				_ = r.Reporter.Errorf("Expected valid inline policy: %s", err)
				return err
			}
		}

		// Process inline policy if it's a file reference
		if strings.HasPrefix(inlinePolicy, "file://") {
			policyPath := strings.TrimPrefix(inlinePolicy, "file://")
			policyBytes, err := os.ReadFile(policyPath)
			if err != nil {
				_ = r.Reporter.Errorf("Failed to read policy file '%s': %s", policyPath, err)
				return fmt.Errorf("failed to read policy file '%s': %s", policyPath, err)
			}
			inlinePolicy = string(policyBytes)
		}

		// Validate permissions boundary
		permissionsBoundary := userOptions.PermissionsBoundary
		if interactive.Enabled() && permissionsBoundary == "" {
			permissionsBoundary, err = interactive.GetString(interactive.Input{
				Question: "Permissions boundary ARN",
				Help:     cmd.Flags().Lookup("permissions-boundary").Usage,
				Validators: []interactive.Validator{
					func(val interface{}) error {
						if val.(string) != "" {
							return aws.ARNValidator(val.(string))
						}
						return nil
					},
				},
			})
			if err != nil {
				_ = r.Reporter.Errorf("Expected valid permissions boundary ARN: %s", err)
				return err
			}
		}

		if permissionsBoundary != "" {
			if err := aws.ARNValidator(permissionsBoundary); err != nil {
				_ = r.Reporter.Errorf("Invalid permissions boundary ARN: %s", err)
				return fmt.Errorf("invalid permissions boundary ARN: %s", err)
			}
		}

		// Validate that at least one policy is specified
		if len(policyArns) == 0 && inlinePolicy == "" {
			return fmt.Errorf("at least one policy ARN or inline policy must be specified")
		}

		// Get interactive mode confirmation
		if interactive.Enabled() {
			mode, err = interactive.GetOptionMode(cmd, mode, "IAM service account role creation mode")
			if err != nil {
				_ = r.Reporter.Errorf("Expected a valid creation mode: %s", err)
				return err
			}
		}

		// Check if role already exists
		exists, existingRoleARN, err := r.AWSClient.CheckRoleExists(roleName)
		if err != nil {
			_ = r.Reporter.Errorf("Failed to check if role exists: %s", err)
			return err
		}

		if exists {
			r.Reporter.Warnf("Role '%s' already exists with ARN '%s'", roleName, existingRoleARN)
			if !confirm.Prompt(false, "Role already exists. Continue with existing role?") {
				r.Reporter.Infof("Operation cancelled")
				return nil
			}
			return nil
		}

		// Generate trust policy
		serviceAccounts := make([]iamserviceaccount.ServiceAccountIdentifier, len(serviceAccountNames))
		for i, saName := range serviceAccountNames {
			serviceAccounts[i] = iamserviceaccount.ServiceAccountIdentifier{
				Name:      saName,
				Namespace: namespace,
			}
		}

		trustPolicy := iamserviceaccount.GenerateTrustPolicyMultiple(oidcProviderARN, serviceAccounts)
		if trustPolicy == "" {
			_ = r.Reporter.Errorf("Failed to generate trust policy")
			return fmt.Errorf("failed to generate trust policy")
		}

		// Generate tags - use first service account for backwards compatibility
		tags := iamserviceaccount.GenerateDefaultTags(cluster.Name(), namespace, serviceAccountNames[0])
		if len(serviceAccountNames) > 1 {
			// Add a tag indicating multiple service accounts
			// Use space as separator since comma is not allowed in AWS tag values
			tags["rosa_service_accounts"] = strings.Join(serviceAccountNames, " ")
		}

		switch mode {
		case interactive.ModeAuto:
			r.Reporter.Infof("This will create the following resources:")
			r.Reporter.Infof("  - IAM role: %s", roleName)
			if len(serviceAccountNames) == 1 {
				r.Reporter.Infof("  - Service account: %s/%s", namespace, serviceAccountNames[0])
			} else {
				r.Reporter.Infof("  - Service accounts:")
				for _, saName := range serviceAccountNames {
					r.Reporter.Infof("    - %s/%s", namespace, saName)
				}
			}
			if len(policyArns) > 0 {
				r.Reporter.Infof("  - Attached policies: %d", len(policyArns))
			}
			if !confirm.Prompt(false, "Create IAM role for service account?") {
				r.Reporter.Infof("Operation cancelled")
				return nil
			}

			// Create the role
			roleARN, err := r.AWSClient.CreateServiceAccountRole(roleName, trustPolicy, permissionsBoundary, userOptions.Path, tags)
			if err != nil {
				_ = r.Reporter.Errorf("Failed to create IAM role: %s", err)
				return err
			}

			r.Reporter.Infof("Created IAM role '%s' with ARN '%s'", roleName, roleARN)

			// Attach managed policies
			if len(policyArns) > 0 {
				err = r.AWSClient.AttachPoliciesToServiceAccountRole(roleName, policyArns)
				if err != nil {
					_ = r.Reporter.Errorf("Failed to attach policies: %s", err)
					return err
				}
				r.Reporter.Infof("Attached %d policies to role", len(policyArns))
			}

			// Add inline policy
			if inlinePolicy != "" {
				policyName := fmt.Sprintf("%s-inline-policy", roleName)
				err = r.AWSClient.PutInlinePolicyOnServiceAccountRole(roleName, policyName, inlinePolicy)
				if err != nil {
					_ = r.Reporter.Errorf("Failed to add inline policy: %s", err)
					return err
				}
				r.Reporter.Infof("Added inline policy to role")
			}

			r.Reporter.Infof("Successfully created IAM service account role")
			r.Reporter.Infof("")
			r.Reporter.Infof("Role ARN: %s", roleARN)
			r.Reporter.Infof("")
			r.Reporter.Infof("To use this role, configure it according to your workload type:")
			r.Reporter.Infof("")
			r.Reporter.Infof("For applications using direct service account annotation:")
			if len(serviceAccountNames) == 1 {
				r.Reporter.Infof("  oc annotate serviceaccount/%s -n %s eks.amazonaws.com/role-arn=%s", serviceAccountNames[0], namespace, roleARN)
			} else {
				r.Reporter.Infof("  # Annotate each service account:")
				for _, saName := range serviceAccountNames {
					r.Reporter.Infof("  oc annotate serviceaccount/%s -n %s eks.amazonaws.com/role-arn=%s", saName, namespace, roleARN)
				}
			}
			r.Reporter.Infof("")
			r.Reporter.Infof("For operators and services that support IAM roles:")
			r.Reporter.Infof("• Create a secret with role_arn and configure in the operator's custom resource")
			r.Reporter.Infof("• Configure IAM role integration in ConfigMaps or secrets as required")
			r.Reporter.Infof("• Check the specific operator's documentation for IAM role integration details")
			r.Reporter.Infof("")
			r.Reporter.Infof("For detailed integration patterns, see the documentation for your specific operator or service.")

		case interactive.ModeManual:
			r.Reporter.Infof("Run the following AWS CLI commands to create the IAM role manually:")
			r.Reporter.Infof("")

			// Generate manual commands
			commands := generateManualCommands(roleName, trustPolicy, permissionsBoundary, userOptions.Path, tags, policyArns, inlinePolicy)
			fmt.Println(commands)

		default:
			return fmt.Errorf("invalid mode. Allowed values are %s", interactive.Modes)
		}

		return nil
	}
}

func getOIDCProviderARN(r *rosa.Runtime, cluster *cmv1.Cluster) (string, error) {
	// Get OIDC config
	oidcConfig := cluster.AWS().STS().OidcConfig()
	if oidcConfig == nil {
		return "", fmt.Errorf("cluster does not have OIDC configuration")
	}

	// For managed OIDC, construct the provider ARN
	if oidcConfig.Managed() {
		issuerURL := cluster.AWS().STS().OidcConfig().IssuerUrl()
		if issuerURL == "" {
			return "", fmt.Errorf("missing OIDC issuer URL")
		}

		// Remove https:// prefix if present
		issuerURL = strings.TrimPrefix(issuerURL, "https://")

		return fmt.Sprintf("arn:%s:iam::%s:oidc-provider/%s", r.Creator.Partition, r.Creator.AccountID, issuerURL), nil
	}

	// For unmanaged OIDC, we need to find the provider ARN
	// This requires listing OIDC providers and matching by URL
	providerArns, err := r.AWSClient.ListOpenIDConnectProviderArns()
	if err != nil {
		return "", fmt.Errorf("failed to list OIDC providers: %w", err)
	}

	issuerURL := oidcConfig.IssuerUrl()
	if issuerURL == "" {
		return "", fmt.Errorf("missing OIDC issuer URL")
	}

	issuerURL = strings.TrimPrefix(issuerURL, "https://")

	for _, arn := range providerArns {
		if strings.Contains(arn, issuerURL) {
			return arn, nil
		}
	}

	return "", fmt.Errorf("OIDC provider not found for cluster")
}

func generateManualCommands(roleName, trustPolicy, permissionsBoundary, path string, tags map[string]string, policyArns []string, inlinePolicy string) string {
	commands := []string{}

	// Save trust policy to file
	commands = append(commands, "# Save the trust policy to a file")
	commands = append(commands, fmt.Sprintf("cat > %s-trust-policy.json << 'EOF'", roleName))
	commands = append(commands, trustPolicy)
	commands = append(commands, "EOF")
	commands = append(commands, "")

	// Create role command
	createRoleCmd := fmt.Sprintf("aws iam create-role --role-name %s --assume-role-policy-document file://%s-trust-policy.json", roleName, roleName)

	if path != "/" {
		createRoleCmd += fmt.Sprintf(" --path %s", path)
	}

	if permissionsBoundary != "" {
		createRoleCmd += fmt.Sprintf(" --permissions-boundary %s", permissionsBoundary)
	}

	// Add tags
	if len(tags) > 0 {
		tagPairs := []string{}
		for key, value := range tags {
			tagPairs = append(tagPairs, fmt.Sprintf("Key=%s,Value=%s", key, value))
		}
		createRoleCmd += fmt.Sprintf(" --tags %s", strings.Join(tagPairs, " "))
	}

	commands = append(commands, createRoleCmd)
	commands = append(commands, "")

	// Attach managed policies
	for _, policyArn := range policyArns {
		commands = append(commands, fmt.Sprintf("aws iam attach-role-policy --role-name %s --policy-arn %s", roleName, policyArn))
	}

	// Add inline policy
	if inlinePolicy != "" {
		commands = append(commands, "")
		commands = append(commands, "# Save the inline policy to a file")
		commands = append(commands, fmt.Sprintf("cat > %s-inline-policy.json << 'EOF'", roleName))
		commands = append(commands, inlinePolicy)
		commands = append(commands, "EOF")
		commands = append(commands, "")
		commands = append(commands, fmt.Sprintf("aws iam put-role-policy --role-name %s --policy-name %s-inline-policy --policy-document file://%s-inline-policy.json", roleName, roleName, roleName))
	}

	return strings.Join(commands, "\n")
}
