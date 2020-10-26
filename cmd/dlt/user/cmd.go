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
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/confirm"
	"github.com/openshift/moactl/pkg/interactive"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var args struct {
	clusterKey      string
	clusterAdmins   string
	dedicatedAdmins string
}

var Cmd = &cobra.Command{
	Use:     "user",
	Aliases: []string{"users"},
	Short:   "Delete cluster users",
	Long:    "Delete administrative cluster users.",
	Example: `  # Delete a user from the cluster-admins group
  rosa delete user --cluster=mycluster --cluster-admins=myusername

  # Delete a user from the dedicated-admins group
  rosa delete user --cluster=mycluster --dedicated-admins=myusername

  # Delete a user following interactive prompts
  rosa delete user --cluster=mycluster`,
	Run: run,
}

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

	flags.StringVar(
		&args.clusterAdmins,
		"cluster-admins",
		"",
		"Grant cluster-admin permission to these users.",
	)

	flags.StringVar(
		&args.dedicatedAdmins,
		"dedicated-admins",
		"",
		"Delete dedicated-admin users.",
	)
}

func run(_ *cobra.Command, _ []string) {
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

	clusterAdmins := args.clusterAdmins
	dedicatedAdmins := args.dedicatedAdmins

	if clusterAdmins == "" && dedicatedAdmins == "" {
		clusterAdmins, err = interactive.GetInput("Comma-separated list of cluster-admins to delete")
		if err != nil {
			reporter.Errorf("Expected a commad-separated list of usernames")
			os.Exit(1)
		}

		dedicatedAdmins, err = interactive.GetInput("Comma-separated list of dedicated-admins to delete")
		if err != nil {
			reporter.Errorf("Expected a commad-separated list of usernames")
			os.Exit(1)
		}
	}

	if clusterAdmins == "" && dedicatedAdmins == "" {
		reporter.Errorf("Expected at least one of 'cluster-admins' or 'dedicated-admins'")
		os.Exit(1)
	}

	allUsers := strings.Join([]string{clusterAdmins, dedicatedAdmins}, ",")
	if !confirm.Confirm("delete users %s from cluster %s", strings.Trim(allUsers, ","), clusterKey) {
		os.Exit(0)
	}

	if clusterAdmins != "" {
		reporter.Debugf("Deleting cluster-admin users from cluster '%s'", clusterKey)
		for _, username := range strings.Split(clusterAdmins, ",") {
			res, err := clustersCollection.Cluster(cluster.ID()).
				Groups().
				Group("cluster-admins").
				Users().
				User(username).
				Delete().
				Send()
			if err != nil {
				reporter.Debugf(err.Error())
				reporter.Errorf("Failed to delete cluster-admin user '%s' from cluster '%s': %s",
					username, clusterKey, res.Error().Reason())
				continue
			}
		}
	}

	if dedicatedAdmins != "" {
		reporter.Debugf("Deleting dedicated-admin users from cluster '%s'", clusterKey)
		for _, username := range strings.Split(dedicatedAdmins, ",") {
			res, err := clustersCollection.Cluster(cluster.ID()).
				Groups().
				Group("dedicated-admins").
				Users().
				User(username).
				Delete().
				Send()
			if err != nil {
				reporter.Debugf(err.Error())
				reporter.Errorf("Failed to delete dedicated-admin user '%s' from cluster '%s': %s",
					username, clusterKey, res.Error().Reason())
				continue
			}
		}
	}
}
