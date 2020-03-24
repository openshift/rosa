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

package dlt

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
	"gitlab.cee.redhat.com/service/moactl/pkg/interactive"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var args struct {
	dedicatedAdmins string
	// clusterAdmins   string
}

var Cmd = &cobra.Command{
	Use:   "delete [CLUSTER ID|NAME] [--dedicated-admins=USER1,USER2]",
	Short: "Delete cluster users",
	Long:  "Delete administrative cluster users.",
	Run:   run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.dedicatedAdmins,
		"dedicated-admins",
		"",
		"Delete dedicated-admin users.",
	)
	// flags.StringVar(
	// 	&args.clusterAdmins,
	// 	"cluster-admins",
	// 	"",
	// 	"Grant cluster-admin permission to these users.",
	// )
}

func run(_ *cobra.Command, argv []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create reporter: %v\n", err)
		os.Exit(1)
	}

	// Create the logger:
	logger, err := logging.NewLogger().Build()
	if err != nil {
		reporter.Errorf("Can't create logger: %v", err)
		os.Exit(1)
	}

	// Check command line arguments:
	if len(argv) < 1 {
		reporter.Errorf(
			"Expected exactly one command line parameter containing the name " +
				"or identifier of the cluster",
		)
		os.Exit(1)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := argv[0]
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
		reporter.Errorf("Can't create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Can't get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Can't create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Can't close OCM connection: %v", err)
		}
	}()

	// Get the client for the OCM collection of clusters:
	clustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()

	// Try to find the cluster:
	reporter.Infof("Loading cluster '%s'", clusterKey)
	cluster, err := ocm.GetCluster(clustersCollection, clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	dedicatedAdmins := args.dedicatedAdmins
	if dedicatedAdmins == "" {
		dedicatedAdmins, err = interactive.GetInput("Enter a comma-separated list of dedicated-admin usernames to delete")
		if err != nil {
			reporter.Errorf("Expected a commad-separated list of usernames")
			os.Exit(1)
		}
	}

	// clusterAdmins := args.clusterAdmins
	// if clusterAdmins == "" {
	// 	clusterAdmins, err = interactive.GetInput("Enter a comma-separated list of cluster-admin usernames to delete")
	// 	if err != nil {
	// 		reporter.Errorf("Expected a commad-separated list of usernames")
	// 		os.Exit(1)
	// 	}
	// }

	reporter.Infof("Deleting dedicated-admin users from cluster '%s'", clusterKey)
	for _, username := range strings.Split(dedicatedAdmins, ",") {
		_, err = clustersCollection.Cluster(cluster.ID()).
			Groups().
			Group("dedicated-admins").
			Users().
			User(username).
			Delete().
			Send()
		if err != nil {
			reporter.Errorf("Failed to delete dedicated-admin user '%s' from cluster '%s': %v", username, clusterKey, err)
			continue
		}
	}

	// reporter.Infof("Deleting cluster-admin users from cluster '%s'", clusterKey)
	// for _, username := range strings.Split(clusterAdmins, ",") {
	// 	_, err = clustersCollection.Cluster(cluster.ID()).
	// 		Groups().
	// 		Group("cluster-admins").
	// 		Users().
	// 		User(username).
	// 		Delete().
	// 		Send()
	// 	if err != nil {
	// 		reporter.Errorf("Failed to delete cluster-admin user '%s' from cluster '%s': %v", username, clusterKey, err)
	// 		continue
	// 	}
	// }
}
