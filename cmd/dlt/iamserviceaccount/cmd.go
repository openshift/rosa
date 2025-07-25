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
	"fmt"
	"os"
	"strings"

	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/iamserviceaccount"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	serviceAccountName string
	namespace          string
	roleName           string
}

var Cmd = &cobra.Command{
	Use:     "iamserviceaccount",
	Aliases: []string{"iam-service-account"},
	Short:   "Delete IAM role for Kubernetes service account",
	Long: "Delete an IAM role that was created for a Kubernetes service account. " +
		"This will remove the role and all attached policies.",
	Example: `  # Delete by service account details
  rosa delete iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace default

  # Delete by explicit role name with approval
  rosa delete iamserviceaccount --cluster my-cluster \
    --role-name my-custom-role --approve`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.serviceAccountName,
		"name",
		"",
		"Name of the Kubernetes service account.",
	)

	flags.StringVar(
		&args.namespace,
		"namespace",
		"default",
		"Kubernetes namespace for the service account.",
	)

	flags.StringVar(
		&args.roleName,
		"role-name",
		"",
		"Name of the IAM role to delete (auto-detected if not specified).",
	)

	// Mark required flags
	_ = Cmd.MarkFlagRequired("cluster")

	interactive.AddModeFlag(Cmd)
	interactive.AddFlag(flags)
	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Get interactive mode
	mode, err := interactive.GetMode()
	if err != nil {
		_ = r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}

	// Get cluster key using OCM standard method
	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		_ = r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	cluster := r.FetchCluster()
	if cluster.Name() != clusterKey && cluster.ID() != clusterKey {
		cluster, err = r.OCMClient.GetCluster(clusterKey, r.Creator)
		if err != nil {
			_ = r.Reporter.Errorf("Failed to get cluster '%s': %s", clusterKey, err)
			os.Exit(1)
		}
	}

	// Validate cluster has STS enabled
	if cluster.AWS().STS().RoleARN() == "" {
		_ = r.Reporter.Errorf("Cluster '%s' is not an STS cluster", cluster.Name())
		os.Exit(1)
	}

	// Get role name - either explicit or derived from service account
	roleName := args.roleName
	serviceAccountName := args.serviceAccountName
	namespace := args.namespace

	if roleName == "" {
		// Need service account details to derive role name
		if interactive.Enabled() && serviceAccountName == "" {
			serviceAccountName, err = interactive.GetString(interactive.Input{
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
				os.Exit(1)
			}
		}

		if serviceAccountName == "" {
			_ = r.Reporter.Errorf("Service account name is required when role name is not specified")
			os.Exit(1)
		}

		if err := iamserviceaccount.ValidateServiceAccountName(serviceAccountName); err != nil {
			_ = r.Reporter.Errorf("Invalid service account name: %s", err)
			os.Exit(1)
		}

		// Validate namespace
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
				os.Exit(1)
			}
		}

		if err := iamserviceaccount.ValidateNamespaceName(namespace); err != nil {
			_ = r.Reporter.Errorf("Invalid namespace: %s", err)
			os.Exit(1)
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
				_ = r.Reporter.Errorf("Expected a valid role name: %s", err)
				os.Exit(1)
			}
		}
	}

	// Check if role exists
	exists, roleARN, err := r.AWSClient.CheckRoleExists(roleName)
	if err != nil {
		_ = r.Reporter.Errorf("Failed to check if role exists: %s", err)
		os.Exit(1)
	}

	if !exists {
		r.Reporter.Warnf("Role '%s' does not exist", roleName)
		os.Exit(0)
	}

	// Get role details to verify it's a service account role
	role, attachedPolicies, inlinePolicies, err := r.AWSClient.GetServiceAccountRoleDetails(roleName)
	if err != nil {
		_ = r.Reporter.Errorf("Failed to get role details: %s", err)
		os.Exit(1)
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
			os.Exit(0)
		}
	}

	// Verify cluster matches if we have the tag
	if clusterName != "" && clusterName != cluster.Name() {
		r.Reporter.Warnf("Role '%s' belongs to cluster '%s', but you specified cluster '%s'", roleName, clusterName, cluster.Name())
		if !confirm.Prompt(false, "Continue with deletion?") {
			r.Reporter.Infof("Operation cancelled")
			os.Exit(0)
		}
	}

	// Get interactive mode confirmation
	if interactive.Enabled() {
		mode, err = interactive.GetOptionMode(cmd, mode, "IAM service account role deletion mode")
		if err != nil {
			_ = r.Reporter.Errorf("Expected a valid deletion mode: %s", err)
			os.Exit(1)
		}
	}

	switch mode {
	case interactive.ModeAuto:
		// Show what will be deleted
		r.Reporter.Infof("Role details:")
		r.Reporter.Infof("  Name: %s", roleName)
		r.Reporter.Infof("  ARN: %s", roleARN)
		if roleNamespace != "" && roleServiceAccount != "" {
			r.Reporter.Infof("  Service Account: %s/%s", roleNamespace, roleServiceAccount)
		}
		if len(attachedPolicies) > 0 {
			r.Reporter.Infof("  Attached Policies: %d", len(attachedPolicies))
			for _, policy := range attachedPolicies {
				r.Reporter.Infof("    - %s", *policy.PolicyArn)
			}
		}
		if len(inlinePolicies) > 0 {
			r.Reporter.Infof("  Inline Policies: %d", len(inlinePolicies))
			for _, policyName := range inlinePolicies {
				r.Reporter.Infof("    - %s", policyName)
			}
		}

		if !confirm.Prompt(false, "Delete IAM role '%s' and all associated policies?", roleName) {
			r.Reporter.Infof("Operation cancelled")
			os.Exit(0)
		}

		// Delete the role
		err = r.AWSClient.DeleteServiceAccountRole(roleName)
		if err != nil {
			_ = r.Reporter.Errorf("Failed to delete IAM role: %s", err)
			os.Exit(1)
		}

		r.Reporter.Infof("Successfully deleted IAM service account role '%s'", roleName)

	case interactive.ModeManual:
		r.Reporter.Infof("Run the following AWS CLI commands to delete the IAM role manually:")
		r.Reporter.Infof("")

		// Generate manual commands
		commands := generateManualDeleteCommands(roleName, attachedPolicies, inlinePolicies)
		fmt.Println(commands)

	default:
		_ = r.Reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		os.Exit(1)
	}
}

func generateManualDeleteCommands(roleName string, attachedPolicies []iamtypes.AttachedPolicy, inlinePolicies []string) string {
	commands := []string{}

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
