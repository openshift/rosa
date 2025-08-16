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
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
)

type DeleteIamServiceAccountUserOptions struct {
	ServiceAccountName string
	Namespace          string
	RoleName           string
}

const (
	deleteUse   = "iamserviceaccount"
	deleteShort = "Delete IAM role for Kubernetes service account"
	deleteLong  = "Delete an IAM role that was created for a Kubernetes service account. " +
		"This will remove the role and all attached policies."
	deleteExample = `  # Delete IAM role for service account
  rosa delete iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace default`
)

func NewDeleteIamServiceAccountUserOptions() *DeleteIamServiceAccountUserOptions {
	return &DeleteIamServiceAccountUserOptions{
		Namespace: "default",
	}
}

func BuildIamServiceAccountDeleteCommandWithOptions() (*cobra.Command, *DeleteIamServiceAccountUserOptions) {
	options := NewDeleteIamServiceAccountUserOptions()
	cmd := &cobra.Command{
		Use:     deleteUse,
		Aliases: []string{"iam-service-account"},
		Short:   deleteShort,
		Long:    deleteLong,
		Example: deleteExample,
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
		"Name of the IAM role to delete (auto-detected if not specified).",
	)

	interactive.AddModeFlag(cmd)
	interactive.AddFlag(flags)
	confirm.AddFlag(flags)
	return cmd, options
}
