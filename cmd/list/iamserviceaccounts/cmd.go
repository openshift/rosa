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

package iamserviceaccounts

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/iamserviceaccount"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

type ListIamServiceAccountsUserOptions struct {
	ClusterKey string
	Namespace  string
}

func NewListIamServiceAccountsCommand() *cobra.Command {
	options := &ListIamServiceAccountsUserOptions{}

	cmd := &cobra.Command{
		Use:     "iamserviceaccounts",
		Aliases: []string{"iam-service-accounts", "iamserviceaccount", "iam-service-account"},
		Short:   "List IAM roles for Kubernetes service accounts",
		Long: "List IAM roles that were created for Kubernetes service accounts using " +
			"OpenID Connect (OIDC) identity federation.",
		Example: `  # List IAM roles for service accounts
  rosa list iamserviceaccounts --cluster my-cluster`,
		Args: cobra.NoArgs,
	}

	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCMAndAWS(), ListIamServiceAccountsRunner(options))

	flags := cmd.Flags()
	ocm.AddClusterFlag(cmd)

	flags.StringVar(
		&options.Namespace,
		"namespace",
		"",
		"Namespace to filter service account roles by.",
	)

	output.AddFlag(cmd)

	return cmd
}

var Cmd = NewListIamServiceAccountsCommand()

func ListIamServiceAccountsRunner(userOptions *ListIamServiceAccountsUserOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		clusterKey := r.GetClusterKey()

		// Validate cluster if specified
		var clusterName string
		if clusterKey != "" {
			cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
			if err != nil {
				return fmt.Errorf("failed to get cluster '%s': %s", clusterKey, err)
			}
			clusterName = cluster.Name()

			// Validate cluster has STS enabled
			if cluster.AWS().STS().RoleARN() == "" {
				return fmt.Errorf("cluster '%s' is not an STS cluster", cluster.Name())
			}
		}

		// List service account roles
		roles, err := r.AWSClient.ListServiceAccountRoles(clusterName)
		if err != nil {
			return fmt.Errorf("failed to list service account roles: %s", err)
		}

		// Filter by namespace if specified
		if userOptions.Namespace != "" {
			filteredRoles := []iamtypes.Role{}
			for _, role := range roles {
				for _, tag := range role.Tags {
					if aws.ToString(tag.Key) == iamserviceaccount.NamespaceTagKey && aws.ToString(tag.Value) == userOptions.Namespace {
						filteredRoles = append(filteredRoles, role)
						break
					}
				}
			}
			roles = filteredRoles
		}

		// Convert to output format
		serviceAccountRoles := make([]ServiceAccountRoleOutput, 0, len(roles))
		for _, role := range roles {
			serviceAccountRole := convertToOutput(role)
			serviceAccountRoles = append(serviceAccountRoles, serviceAccountRole)
		}

		// Table format
		if len(serviceAccountRoles) == 0 {
			r.Reporter.Infof("No IAM service account roles found")
			return nil
		}

		// Output results
		if output.HasFlag() {
			err = output.Print(serviceAccountRoles)
			if err != nil {
				return fmt.Errorf("failed to print output: %s", err)
			}
			return nil
		}

		// Print table
		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(writer, "NAME\tARN\tCLUSTER\tNAMESPACE\tSERVICE ACCOUNT\tCREATED")

		for _, role := range serviceAccountRoles {
			created := ""
			if role.CreatedDate != nil {
				created = role.CreatedDate.Format("2006-01-02 15:04:05")
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n",
				role.RoleName,
				role.ARN,
				role.Cluster,
				role.Namespace,
				role.ServiceAccount,
				created,
			)
		}

		return writer.Flush()
	}
}

type ServiceAccountRoleOutput struct {
	RoleName       string     `json:"roleName" yaml:"roleName"`
	ARN            string     `json:"arn" yaml:"arn"`
	Cluster        string     `json:"cluster" yaml:"cluster"`
	Namespace      string     `json:"namespace" yaml:"namespace"`
	ServiceAccount string     `json:"serviceAccount" yaml:"serviceAccount"`
	CreatedDate    *time.Time `json:"createdDate,omitempty" yaml:"createdDate,omitempty"`
	Path           string     `json:"path" yaml:"path"`
}

func convertToOutput(role iamtypes.Role) ServiceAccountRoleOutput {
	output := ServiceAccountRoleOutput{
		RoleName:    aws.ToString(role.RoleName),
		ARN:         aws.ToString(role.Arn),
		CreatedDate: role.CreateDate,
		Path:        aws.ToString(role.Path),
	}

	// Extract information from tags
	for _, tag := range role.Tags {
		switch aws.ToString(tag.Key) {
		case iamserviceaccount.ClusterTagKey:
			output.Cluster = aws.ToString(tag.Value)
		case iamserviceaccount.NamespaceTagKey:
			output.Namespace = aws.ToString(tag.Value)
		case iamserviceaccount.ServiceAccountTagKey:
			output.ServiceAccount = aws.ToString(tag.Value)
		}
	}

	return output
}
