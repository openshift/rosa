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
	"strings"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:     "users",
	Aliases: []string{"user"},
	Short:   "List cluster users",
	Long:    "List administrative cluster users.",
	Example: `  # List all users on a cluster named "mycluster"
  rosa list users --cluster=mycluster`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
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

	var clusterAdmins []*cmv1.User
	reporter.Debugf("Loading users for cluster '%s'", clusterKey)
	// Load cluster-admins for this cluster
	clusterAdmins, err = ocmClient.GetUsers(cluster.ID(), "cluster-admins")
	if err != nil {
		reporter.Errorf("Failed to get cluster-admins for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	// Remove cluster-admin user
	for i, user := range clusterAdmins {
		if user.ID() == "cluster-admin" {
			clusterAdmins = append(clusterAdmins[:i], clusterAdmins[i+1:]...)
		}
	}

	// Load dedicated-admins for this cluster
	dedicatedAdmins, err := ocmClient.GetUsers(cluster.ID(), "dedicated-admins")
	if err != nil {
		reporter.Errorf("Failed to get dedicated-admins for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if len(clusterAdmins) == 0 && len(dedicatedAdmins) == 0 {
		reporter.Warnf("There are no users configured for cluster '%s'", clusterKey)
		os.Exit(1)
	}

	groups := make(map[string][]string)
	for _, user := range clusterAdmins {
		groups[user.ID()] = []string{"cluster-admins"}
	}
	for _, user := range dedicatedAdmins {
		if _, ok := groups[user.ID()]; ok {
			groups[user.ID()] = []string{"cluster-admins", "dedicated-admins"}
		} else {
			groups[user.ID()] = []string{"dedicated-admins"}
		}
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\t\tGROUPS\n")

	for u, r := range groups {
		fmt.Fprintf(writer, "%s\t\t%s\n", u, strings.Join(r, ", "))
		writer.Flush()
	}
}
