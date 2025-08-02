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
	describeExample = `  # Describe by service account details
  rosa describe iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace default

  # Describe by explicit role name
  rosa describe iamserviceaccount --cluster my-cluster \
    --role-name my-custom-role

  # Output as JSON
  rosa describe iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace default \
    --output json`
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
