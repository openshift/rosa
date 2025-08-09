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
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
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
