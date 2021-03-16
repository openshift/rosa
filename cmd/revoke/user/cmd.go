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

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	clusterKey string
	username   string
}

var Cmd = &cobra.Command{
	Use:     "user ROLE",
	Aliases: []string{"role"},
	Short:   "Revoke role from users",
	Long:    "Revoke role from cluster user",
	Example: `  # Revoke cluster-admin role from a user
  rosa revoke user cluster-admins --user=myusername --cluster=mycluster

  # Revoke dedicated-admin role from a user
  rosa revoke user dedicate-admins --user=myusername --cluster=mycluster`,
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

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to delete the users from (required).",
	)
	Cmd.MarkFlagRequired("cluster")

	flags.StringVarP(
		&args.username,
		"user",
		"u",
		"",
		"Username to revoke the role from (required).",
	)
	Cmd.MarkFlagRequired("user")
}

func run(_ *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !ocm.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
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
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Get the client for the OCM collection of clusters:
	clustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocm.GetCluster(clustersCollection, clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Try to find the user:
	reporter.Debugf("Loading '%s' users for cluster '%s'", role, clusterKey)
	user, err := ocm.GetUser(clustersCollection, cluster.ID(), role, username)
	if err != nil {
		reporter.Errorf(err.Error())
		os.Exit(1)
	}

	if user == nil {
		reporter.Warnf("Cannot find user '%s' with role '%s' on cluster '%s'", username, role, clusterKey)
		os.Exit(0)
	}

	if !confirm.Confirm("revoke role %s from user %s in cluster %s", role, username, clusterKey) {
		os.Exit(0)
	}

	reporter.Debugf("Removing user '%s' from group '%s' in cluster '%s'", username, role, clusterKey)
	res, err := clustersCollection.Cluster(cluster.ID()).Groups().Group(role).Users().User(username).Delete().Send()
	if err != nil {
		reporter.Debugf(err.Error())
		reporter.Errorf("Failed to revoke '%s' from user '%s' in cluster '%s': %s",
			role, username, clusterKey, res.Error().Reason())
		os.Exit(1)
	}
	reporter.Infof("Revoked role '%s' from user '%s' on cluster '%s'", role, username, clusterKey)
}
