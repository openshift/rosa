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
	"fmt"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/iamserviceaccount"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

type DescribeIamServiceAccountUserOptions struct {
	ServiceAccountName string
	Namespace          string
	RoleName           string
}

const (
	describeUse   = "iamserviceaccount"
	describeShort = "Describe IAM role for Kubernetes service account"
	describeLong  = "Show detailed information about an IAM role that was created for a " +
		"Kubernetes service account, including trust policy and attached permissions."
	describeExample = `  # Describe IAM role for service account
  rosa describe iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace default`
)

func NewDescribeIamServiceAccountUserOptions() *DescribeIamServiceAccountUserOptions {
	return &DescribeIamServiceAccountUserOptions{
		Namespace: "default",
	}
}

func BuildIamServiceAccountDescribeCommandWithOptions() (*cobra.Command, *DescribeIamServiceAccountUserOptions) {
	options := NewDescribeIamServiceAccountUserOptions()
	cmd := &cobra.Command{
		Use:     describeUse,
		Aliases: []string{"iam-service-account"},
		Short:   describeShort,
		Long:    describeLong,
		Example: describeExample,
		Args:    cobra.NoArgs,
	}

	flags := cmd.Flags()
	ocm.AddClusterFlag(cmd)

	flags.StringVar(
		&options.ServiceAccountName,
		"name",
		"",
		"Name of the Kubernetes service account.",
	)

	flags.StringVar(
		&options.Namespace,
		"namespace",
		"default",
		"Kubernetes namespace for the service account.",
	)

	flags.StringVar(
		&options.RoleName,
		"role-name",
		"",
		"Name of the IAM role to describe (auto-detected if not specified).",
	)

	interactive.AddFlag(flags)
	output.AddFlag(cmd)
	return cmd, options
}

// ValidateCluster validates that the cluster supports IAM service accounts
func (opts *DescribeIamServiceAccountUserOptions) ValidateCluster(cluster *v1.Cluster) error {
	if cluster.AWS().STS().RoleARN() == "" {
		return fmt.Errorf("cluster '%s' is not an STS cluster", cluster.Name())
	}
	return nil
}

// ValidateAndPromptForInputs validates user inputs and prompts for missing required values
func (opts *DescribeIamServiceAccountUserOptions) ValidateAndPromptForInputs(r *rosa.Runtime, cmd *cobra.Command) error {
	// If role name is not provided, we need service account details
	if opts.RoleName == "" {
		// Validate and prompt for service account name
		if err := opts.validateAndPromptServiceAccountName(cmd); err != nil {
			return err
		}

		// Validate and prompt for namespace
		if err := opts.validateAndPromptNamespace(cmd); err != nil {
			return err
		}

		// Generate role name from service account details
		if err := opts.GenerateRoleNameIfNeeded(r.FetchCluster().Name()); err != nil {
			return err
		}
	} else {
		// Using explicit role name - validate and prompt if in interactive mode
		if err := opts.validateAndPromptRoleName(cmd); err != nil {
			return err
		}
	}

	return nil
}

// GenerateRoleNameIfNeeded generates a role name based on cluster, namespace, and service account
func (opts *DescribeIamServiceAccountUserOptions) GenerateRoleNameIfNeeded(clusterName string) error {
	if opts.RoleName == "" && opts.ServiceAccountName != "" && opts.Namespace != "" {
		opts.RoleName = iamserviceaccount.GenerateRoleName(clusterName, opts.Namespace, opts.ServiceAccountName)
	}
	return nil
}

// validateAndPromptServiceAccountName validates and prompts for service account name
func (opts *DescribeIamServiceAccountUserOptions) validateAndPromptServiceAccountName(cmd *cobra.Command) error {
	if interactive.Enabled() && opts.ServiceAccountName == "" {
		var err error
		opts.ServiceAccountName, err = interactive.GetString(interactive.Input{
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

	if opts.ServiceAccountName == "" {
		return fmt.Errorf("service account name is required when role name is not specified")
	}

	if err := iamserviceaccount.ValidateServiceAccountName(opts.ServiceAccountName); err != nil {
		return fmt.Errorf("invalid service account name: %s", err)
	}

	return nil
}

// validateAndPromptNamespace validates and prompts for namespace
func (opts *DescribeIamServiceAccountUserOptions) validateAndPromptNamespace(cmd *cobra.Command) error {
	if interactive.Enabled() {
		var err error
		opts.Namespace, err = interactive.GetString(interactive.Input{
			Question: "Namespace",
			Help:     cmd.Flags().Lookup("namespace").Usage,
			Default:  opts.Namespace,
			Required: true,
			Validators: []interactive.Validator{
				iamserviceaccount.NamespaceNameValidator,
			},
		})
		if err != nil {
			return fmt.Errorf("expected a valid namespace: %s", err)
		}
	}

	if err := iamserviceaccount.ValidateNamespaceName(opts.Namespace); err != nil {
		return fmt.Errorf("invalid namespace: %s", err)
	}

	return nil
}

// validateAndPromptRoleName validates and prompts for role name when explicitly provided
func (opts *DescribeIamServiceAccountUserOptions) validateAndPromptRoleName(cmd *cobra.Command) error {
	if interactive.Enabled() {
		var err error
		opts.RoleName, err = interactive.GetString(interactive.Input{
			Question: "IAM role name",
			Help:     cmd.Flags().Lookup("role-name").Usage,
			Default:  opts.RoleName,
			Required: true,
		})
		if err != nil {
			return fmt.Errorf("expected a valid role name: %s", err)
		}
	}

	return nil
}
