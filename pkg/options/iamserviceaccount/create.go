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
)

type CreateIamServiceAccountUserOptions struct {
	ServiceAccountNames []string
	Namespace           string
	RoleName            string
	PolicyArns          []string
	InlinePolicy        string
	PermissionsBoundary string
	Path                string
}

const (
	use   = "iamserviceaccount"
	short = "Create IAM role for Kubernetes service account"
	long  = "Create an IAM role that can be assumed by a Kubernetes service account using " +
		"OpenID Connect (OIDC) identity federation. This allows pods running in the service " +
		"account to assume the IAM role and access AWS resources."
	example = `  # Create an IAM role for a service account
  rosa create iamserviceaccount --cluster my-cluster --name my-app --namespace default`
)

func NewCreateIamServiceAccountUserOptions() *CreateIamServiceAccountUserOptions {
	return &CreateIamServiceAccountUserOptions{
		Namespace: "default",
		Path:      "/",
	}
}

func BuildIamServiceAccountCreateCommandWithOptions() (*cobra.Command, *CreateIamServiceAccountUserOptions) {
	options := NewCreateIamServiceAccountUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Aliases: []string{"iam-service-account"},
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
	}

	flags := cmd.Flags()
	ocm.AddClusterFlag(cmd)

	flags.StringSliceVar(
		&options.ServiceAccountNames,
		"name",
		[]string{},
		"Name of the Kubernetes service account (can be used multiple times).",
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
		"Name of the IAM role (auto-generated if not specified).",
	)

	flags.StringSliceVar(
		&options.PolicyArns,
		"attach-policy-arn",
		[]string{},
		"ARN of the IAM policy to attach to the role (can be used multiple times).",
	)

	flags.StringVar(
		&options.InlinePolicy,
		"inline-policy",
		"",
		"Inline policy document (JSON) or path to policy file (use file://path/to/policy.json).",
	)

	flags.StringVar(
		&options.PermissionsBoundary,
		"permissions-boundary",
		"",
		"ARN of the IAM policy to use as permissions boundary.",
	)

	flags.StringVar(
		&options.Path,
		"path",
		"",
		"IAM path for the role.",
	)

	interactive.AddModeFlag(cmd)
	interactive.AddFlag(flags)
	return cmd, options
}
