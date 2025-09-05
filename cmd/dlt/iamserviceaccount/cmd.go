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
	"text/tabwriter"

	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/iamserviceaccount"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	iamServiceAccountOpts "github.com/openshift/rosa/pkg/options/iamserviceaccount"
	"github.com/openshift/rosa/pkg/rosa"
)

func NewDeleteIamServiceAccountCommand() *cobra.Command {
	cmd, options := iamServiceAccountOpts.BuildIamServiceAccountDeleteCommandWithOptions()
	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCMAndAWS(), DeleteIamServiceAccountRunner(options))
	return cmd
}

var Cmd = NewDeleteIamServiceAccountCommand()

func DeleteIamServiceAccountRunner(userOptions *iamServiceAccountOpts.DeleteIamServiceAccountUserOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		var err error
		clusterKey := r.GetClusterKey()

		// Get interactive mode
		mode, err := interactive.GetMode()
		if err != nil {
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
				return fmt.Errorf("failed to get cluster '%s': %s", clusterKey, err)
			}
		}

		// Validate cluster has STS enabled
		if cluster.AWS().STS().RoleARN() == "" {
			return fmt.Errorf("cluster '%s' is not an STS cluster", cluster.Name())
		}

		// Get role name - either explicit or derived from service account
		roleName := userOptions.RoleName
		serviceAccountName := userOptions.ServiceAccountName
		namespace := userOptions.Namespace

		useExplicitRoleName, err := interactive.GetBool(interactive.Input{
			Question: "Do you want to provide an explicit role name",
			Help: "Whether or not to delete based on an explicit role name. If you choose 'No' to this prompt," +
				" you will be prompted for a service account name and namespace to generate the iam service account " +
				"role name to delete.",
			Required: true,
			Default:  true,
		})
		if err != nil {
			return fmt.Errorf("expected a valid response to yes/no prompt: %s", err)
		}

		if !useExplicitRoleName {
			// Need service account details to derive role name
			if interactive.Enabled() && serviceAccountName == "" {
				serviceAccountName, err = interactive.GetString(interactive.Input{
					Question: "Service account name",
					Help:     cmd.Flags().Lookup("name").Usage,
					Required: true,
					Validators: []interactive.Validator{
						iamserviceaccount.ServiceAccountNameValidator,
					},
				})
				if err != nil {
					return fmt.Errorf("expected a valid service account name: %s", err)
				}
			}

			if serviceAccountName == "" {
				return fmt.Errorf("service account name is required when role name is not specified")
			}

			if err := iamserviceaccount.ValidateServiceAccountName(serviceAccountName); err != nil {
				return fmt.Errorf("invalid service account name: %s", err)
			}

			// Validate namespace
			if interactive.Enabled() {
				namespace, err = interactive.GetString(interactive.Input{
					Question: "Namespace",
					Help:     cmd.Flags().Lookup("namespace").Usage,
					Default:  namespace,
					Required: true,
					Validators: []interactive.Validator{
						iamserviceaccount.NamespaceNameValidator,
					},
				})
				if err != nil {
					return fmt.Errorf("expected a valid namespace: %s", err)
				}
			}

			if err := iamserviceaccount.ValidateNamespaceName(namespace); err != nil {
				return fmt.Errorf("invalid namespace: %s", err)
			}

			// Generate role name
			roleName = iamserviceaccount.GenerateRoleName(cluster.Name(), namespace, serviceAccountName)
		} else {
			// Using explicit role name
			if interactive.Enabled() {
				roleName, err = interactive.GetString(interactive.Input{
					Question: "IAM role name",
					Help:     cmd.Flags().Lookup("role-name").Usage,
					Default:  roleName,
					Required: true,
				})
				if err != nil {
					return fmt.Errorf("expected a valid role name: %s", err)
				}
			}
		}

		// Check if role exists
		exists, roleARN, err := r.AWSClient.CheckRoleExists(roleName)
		if err != nil {
			return fmt.Errorf("failed to check if role exists: %s", err)
		}

		if !exists {
			r.Reporter.Warnf("Role '%s' does not exist", roleName)
			return nil
		}

		// Get role details to verify it's a service account role
		role, attachedPolicies, inlinePolicies, err := r.AWSClient.GetServiceAccountRoleDetails(roleName)
		if err != nil {
			return fmt.Errorf("failed to get role details: %s", err)
		}

		// Verify this is a service account role by checking tags
		isServiceAccountRole := false
		clusterName := ""
		roleNamespace := ""
		roleServiceAccount := ""

		for _, tag := range role.Tags {
			switch *tag.Key {
			case iamserviceaccount.RoleTypeTagKey:
				if *tag.Value == iamserviceaccount.ServiceAccountRoleType {
					isServiceAccountRole = true
				}
			case iamserviceaccount.ClusterTagKey:
				clusterName = *tag.Value
			case iamserviceaccount.NamespaceTagKey:
				roleNamespace = *tag.Value
			case iamserviceaccount.ServiceAccountTagKey:
				roleServiceAccount = *tag.Value
			}
		}

		if !isServiceAccountRole {
			r.Reporter.Warnf("Role '%s' does not appear to be a service account role", roleName)
			if !confirm.Prompt(false, "Continue with deletion?") {
				r.Reporter.Infof("Operation cancelled")
				return nil
			}
		}

		// Verify cluster matches if we have the tag
		if clusterName != "" && clusterName != cluster.Name() {
			r.Reporter.Warnf("Role '%s' belongs to cluster '%s', but you specified cluster '%s'", roleName, clusterName, cluster.Name())
			if !confirm.Prompt(false, "Continue with deletion?") {
				r.Reporter.Infof("Operation cancelled")
				return nil
			}
		}

		// Get interactive mode confirmation
		if interactive.Enabled() {
			mode, err = interactive.GetOptionMode(cmd, mode, "IAM service account role deletion mode")
			if err != nil {
				return fmt.Errorf("expected a valid deletion mode: %s", err)
			}
		}

		switch mode {
		case interactive.ModeAuto:
			// Show what will be deleted
			r.Reporter.Infof("Role details:")
			writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(writer, "Name:\t%s\n", roleName)
			fmt.Fprintf(writer, "ARN:\t%s\n", roleARN)
			if roleNamespace != "" && roleServiceAccount != "" {
				fmt.Fprintf(writer, "Service Account:\t%s/%s\n", roleNamespace, roleServiceAccount)
			}
			if len(attachedPolicies) > 0 {
				fmt.Fprintf(writer, "Attached Policies:\t%d\n", len(attachedPolicies))
				for _, policy := range attachedPolicies {
					fmt.Fprintf(writer, "\t- %s\n", *policy.PolicyArn)
				}
			}
			if len(inlinePolicies) > 0 {
				fmt.Fprintf(writer, "Inline Policies:\t%d\n", len(inlinePolicies))
				for _, policyName := range inlinePolicies {
					fmt.Fprintf(writer, "\t- %s\n", policyName)
				}
			}
			if err := writer.Flush(); err != nil {
				return fmt.Errorf("failed to write role details: %s", err)
			}

			if !confirm.Prompt(false, "Delete IAM role '%s' and all associated policies?", roleName) {
				r.Reporter.Infof("Operation cancelled")
				return nil
			}

			// Delete the role
			err = r.AWSClient.DeleteServiceAccountRole(roleName)
			if err != nil {
				return fmt.Errorf("failed to delete IAM role: %s", err)
			}

			r.Reporter.Infof("Successfully deleted IAM service account role '%s'", roleName)

		case interactive.ModeManual:
			r.Reporter.Infof("Run the following AWS CLI commands to delete the IAM role manually:")
			r.Reporter.Infof("")

			// Generate manual commands
			commands := generateManualDeleteCommands(roleName, attachedPolicies, inlinePolicies)
			r.Reporter.Infof("%s", commands)

		default:
			return fmt.Errorf("invalid mode. Allowed values are %s", interactive.Modes)
		}

		return nil
	}
}

func generateManualDeleteCommands(roleName string, attachedPolicies []iamtypes.AttachedPolicy, inlinePolicies []string) string {
	var commands []string

	// Detach managed policies
	if len(attachedPolicies) > 0 {
		commands = append(commands, "# Detach managed policies")
		for _, policy := range attachedPolicies {
			commands = append(commands, fmt.Sprintf("aws iam detach-role-policy --role-name %s --policy-arn %s", roleName, *policy.PolicyArn))
		}
		commands = append(commands, "")
	}

	// Delete inline policies
	if len(inlinePolicies) > 0 {
		commands = append(commands, "# Delete inline policies")
		for _, policyName := range inlinePolicies {
			commands = append(commands, fmt.Sprintf("aws iam delete-role-policy --role-name %s --policy-name %s", roleName, policyName))
		}
		commands = append(commands, "")
	}

	// Delete the role
	commands = append(commands, "# Delete the role")
	commands = append(commands, fmt.Sprintf("aws iam delete-role --role-name %s", roleName))

	return strings.Join(commands, "\n")
}
