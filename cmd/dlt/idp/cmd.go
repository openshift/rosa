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

package idp

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "idp [CLUSTER ID|NAME] [IDP NAME]",
	Short: "Delete cluster IDPs",
	Long:  "Delete a specific identity provider for a cluster.",
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
	if len(argv) != 2 {
		reporter.Errorf(
			"Expected exactly two command line parameters containing the name " +
				"or identifier of the cluster and the name of the Identity provider.",
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

	idpName := argv[1]
	if idpName == "" {
		reporter.Errorf("Identity provider name is required.")
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

	// Try to find the identity provider:
	reporter.Infof("Loading identity provider '%s'", idpName)
	idps, err := ocm.GetIdentityProviders(clustersCollection, cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get identity providers for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	var idp *cmv1.IdentityProvider
	for _, item := range idps {
		if item.Name() == idpName {
			idp = item
		}
	}
	if idp == nil {
		reporter.Errorf("Failed to get identity provider '%s' for cluster '%s'", idpName, clusterKey)
		os.Exit(1)
	}

	// Load any existing IDPs for this cluster
	reporter.Infof("Deleting identity provider '%s' on cluster '%s'", idpName, clusterKey)
	_, err = clustersCollection.
		Cluster(cluster.ID()).
		IdentityProviders().
		IdentityProvider(idp.ID()).
		Delete().
		Send()
	if err != nil {
		reporter.Errorf("Failed to delete identity provider '%s' on cluster '%s'", idpName, clusterKey)
		os.Exit(1)
	}
}
