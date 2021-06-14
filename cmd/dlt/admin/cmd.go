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

package admin

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

const (
	idpName  = "Cluster-Admin"
	username = "cluster-admin"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:   "admin",
	Short: "Deletes the admin user",
	Long:  "Deletes the cluster-admin user used to login to the cluster",
	Example: `  # Delete the admin user
  rosa delete admin --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to delete the admin user from (required).",
	)
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, _ []string) {
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
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Try to find the htpasswd identity provider:
	reporter.Debugf("Loading '%s' identity provider", idpName)
	idps, err := ocmClient.GetIdentityProviders(cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get '%s' identity provider for cluster '%s': %v", idpName, clusterKey, err)
		os.Exit(1)
	}

	var idp *cmv1.IdentityProvider
	for _, item := range idps {
		if ocm.IdentityProviderType(item) == "htpasswd" {
			idp = item
		}
	}
	if idp == nil {
		reporter.Errorf("Failed to get '%s' identity provider for cluster '%s'", idpName, clusterKey)
		os.Exit(1)
	}

	if confirm.Confirm("delete %s user on cluster %s", username, clusterKey) {
		// Delete htpasswd IdP:
		reporter.Debugf("Deleting '%s' identity provider on cluster '%s'", idpName, clusterKey)
		err = ocmClient.DeleteIdentityProvider(cluster.ID(), idp.ID())
		if err != nil {
			reporter.Errorf("Failed to delete '%s' identity provider on cluster '%s': %s",
				idpName, clusterKey, err)
			os.Exit(1)
		}

		// Delete admin user from the cluster-admins group:
		reporter.Debugf("Deleting '%s' user from cluster-admins group on cluster '%s'", username, clusterKey)
		err = ocmClient.DeleteUser(cluster.ID(), "cluster-admins", username)
		if err != nil {
			reporter.Errorf("Failed to delete '%s' user from cluster '%s': %s",
				username, clusterKey, err)
			os.Exit(1)
		}

		reporter.Infof("Admin user '%s' has been deleted from cluster '%s'", username, clusterKey)
	}
}
