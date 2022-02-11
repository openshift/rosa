/*
Copyright (c) 2022 Red Hat, Inc.

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

package roleBinding

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	username string
}

var Cmd = &cobra.Command{
	Use:     "role-binding",
	Aliases: []string{"role-bindings,rolebinding,rolebindings"},
	Short:   "Revoke user access to cluster specification in OCM",
	Long:    "Revoke user access to view/edit cluster specification in OCM",
	Example: `  # Revoke role-bindings to a user
  rosa revoke role-binding --user=myusername --cluster=mycluster

  # Revoke ClusterEditor role to a user
  rosa revoke role-binding ClusterEditor --user=myusername --cluster=mycluster`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line argument containing the role-binding" +
					" to grant the user.",
			)
		}
		return nil
	},
}

var validRoleBindingAliases = []string{"ClusterEditor", "ClusterViewer"}

func init() {
	flags := Cmd.Flags()
	ocm.AddClusterFlag(Cmd)
	flags.StringVarP(
		&args.username,
		"user",
		"u",
		"",
		"Username to grant the role-binding to (required).",
	)
	Cmd.MarkFlagRequired("user")
}

func run(_ *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)
	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}
	username := args.username
	if !ocm.IsValidUsername(username) {
		reporter.Errorf(
			"Username '%s' isn't valid: it must contain only letters, digits, dashes and underscores",
			username,
		)
		os.Exit(1)
	}
	roleID := argv[0]
	isRoleIDValid := false
	// Determine if role is valid
	for _, validRole := range validRoleBindingAliases {
		if roleID == validRole {
			isRoleIDValid = true
		}
	}
	if !isRoleIDValid {
		reporter.Errorf("Expected at least one of %s", validRoleBindingAliases)
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() == cmv1.ClusterStateUninstalling {
		reporter.Errorf("Cluster '%s' is in uninstalling state", clusterKey)
		os.Exit(1)
	}
	if !confirm.Confirm("revoke role %s from user %s in cluster %s", roleID, username, clusterKey) {
		os.Exit(0)
	}

	reporter.Debugf("Removing role binding '%s' to user  '%s' in cluster '%s'", roleID, username, clusterKey)
	err = ocmClient.DeleteRoleBinding(cluster.Subscription().ID(), username, roleID)
	if err != nil {
		reporter.Errorf("Failed to revoke '%s' to user '%s' to cluster '%s': %s",
			roleID, username, clusterKey, err)
		os.Exit(1)
	}

	reporter.Infof("Revoked role binding '%s' to user '%s' on cluster '%s'", roleID, username, clusterKey)
}
