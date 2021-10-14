/*
Copyright (c) 2020 Red Hat, Inc.

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

package user

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	username string
}

var Cmd = &cobra.Command{
	Use:     "user ROLE",
	Aliases: []string{"role"},
	Short:   "Grant user access to cluster",
	Long:    "Grant user access to cluster under a specific role",
	Example: `  # Add cluster-admin role to a user
  rosa grant user cluster-admin --user=myusername --cluster=mycluster

  # Grant dedicated-admins role to a user
  rosa grant user dedicated-admin --user=myusername --cluster=mycluster`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line argument containing the name " +
					"of the group or role to grant the user.",
			)
		}
		return nil
	},
}

var validRoles = []string{"cluster-admins", "dedicated-admins"}
var validRolesAliases = []string{"cluster-admin", "dedicated-admin"}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVarP(
		&args.username,
		"user",
		"u",
		"",
		"Username to grant the role to (required).",
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
	if username == "cluster-admin" {
		reporter.Errorf("Username 'cluster-admin' is not allowed")
		os.Exit(1)
	}

	role := argv[0]
	// Allow role aliases
	for _, validAlias := range validRolesAliases {
		if role == validAlias {
			role = fmt.Sprintf("%ss", role)
		}
	}
	isRoleValid := false
	// Determine if role is valid
	for _, validRole := range validRoles {
		if role == validRole {
			isRoleValid = true
		}
	}
	if !isRoleValid {
		reporter.Errorf("Expected at least one of %s", validRoles)
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

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	user, err := cmv1.NewUser().ID(username).Build()
	if err != nil {
		reporter.Errorf("Failed to create user '%s' for cluster '%s'", username, clusterKey)
		os.Exit(1)
	}

	reporter.Debugf("Adding user '%s' to group '%s' in cluster '%s'", username, role, clusterKey)
	_, err = ocmClient.CreateUser(cluster.ID(), role, user)
	if err != nil {
		reporter.Errorf("Failed to grant '%s' to user '%s' to cluster '%s': %s",
			role, username, clusterKey, err)
		os.Exit(1)
	}

	reporter.Infof("Granted role '%s' to user '%s' on cluster '%s'", role, username, clusterKey)
}
