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
	"regexp"

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm/properties"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "delete [ID|NAME]",
	Short: "Delete cluster",
	Long:  "Delete cluster.",
	Run:   run,
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
	if len(argv) != 1 {
		reporter.Errorf(
			"Expected exactly one command line parameter containing the name " +
				"or identifier of the cluster",
		)
		os.Exit(1)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := argv[0]
	if !clusterKeyRE.MatchString(clusterKey) {
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
	ocmClustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()

	// Try to find the cluster:
	reporter.Infof("Loading cluster '%s'", clusterKey)
	ocmQuery := fmt.Sprintf(
		"(id = '%s' or name = '%s') and properties.%s = '%s'",
		clusterKey, clusterKey, properties.CreatorARN, awsCreator.ARN,
	)
	ocmListResponse, err := ocmClustersCollection.List().
		Search(ocmQuery).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		reporter.Errorf("Can't locate cluster '%s': %v", err)
		os.Exit(1)
	}
	switch ocmListResponse.Total() {
	case 0:
		reporter.Errorf("There is no cluster with identifier or name '%s'", clusterKey)
		os.Exit(1)
	case 1:
		ocmCluster := ocmListResponse.Items().Slice()[0]
		ocmClusterID := ocmCluster.ID()
		ocmClusterName := ocmCluster.ID()
		reporter.Infof(
			"Deleting cluster with identifier '%s' and name '%s'",
			ocmClusterID, ocmClusterName,
		)
		_, err = ocmClustersCollection.Cluster(ocmClusterID).Delete().Send()
		if err != nil {
			reporter.Errorf(
				"Can't delete cluster with identifier '%s' and name '%s'",
				ocmClusterID, ocmClusterName,
			)
		}
	default:
		reporter.Errorf("There are %d clusters with identifier or name '%s'", clusterKey)
		os.Exit(1)
	}
}

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)
