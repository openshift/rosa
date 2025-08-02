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
	example = `  # Create an IAM role for a service account with S3 access
  rosa create iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace default \
    --attach-policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess

  # Create with custom role name and inline policy
  rosa create iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace my-namespace \
    --role-name my-custom-role \
    --inline-policy file://my-policy.json

  # Create with permissions boundary and approval
  rosa create iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace default \
    --attach-policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess \
    --permissions-boundary arn:aws:iam::123456789012:policy/boundary \
    --approve

  # Create for multiple service accounts (e.g., AWS Load Balancer Controller)
  rosa create iamserviceaccount --cluster my-cluster \
    --name aws-load-balancer-operator-controller-manager \
    --name aws-load-balancer-controller-cluster \
    --namespace aws-load-balancer-operator \
    --role-name my-cluster-alb-controller-role \
    --attach-policy-arn arn:aws:iam::aws:policy/ElasticLoadBalancingFullAccess`
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
		"/",
		"IAM path for the role.",
	)

	interactive.AddModeFlag(cmd)
	interactive.AddFlag(flags)
	return cmd, options
}
