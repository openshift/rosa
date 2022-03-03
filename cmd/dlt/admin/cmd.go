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

	"github.com/openshift/rosa/cmd/create/idp"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "admin",
	Short: "Deletes the admin user",
	Long:  "Deletes the cluster-admin user used to login to the cluster",
	Example: `  # Delete the admin user
  rosa delete admin --cluster=mycluster`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
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

	// Try to find the htpasswd identity provider:
	reporter.Debugf("Loading HTPasswd identity provider")
	idps, err := ocmClient.GetIdentityProviders(cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get HTPasswd identity provider for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	var htpasswdIDP *cmv1.IdentityProvider
	for _, item := range idps {
		if ocm.IdentityProviderType(item) == "htpasswd" {
			htpasswdIDP = item
		}
	}
	if htpasswdIDP == nil {
		reporter.Errorf("Cluster '%s' does not have an admin user", clusterKey)
		os.Exit(1)
	}

	if confirm.Confirm("delete %s user on cluster %s", idp.ClusterAdminUsername, clusterKey) {
		// delete `cluster-admin` user from the HTPasswd IDP
		reporter.Debugf("Deleting user '%s' identity provider on cluster '%s'", idp.ClusterAdminUsername, clusterKey)
		err = ocmClient.DeleteHTPasswdUser(idp.ClusterAdminUsername, cluster.ID(), htpasswdIDP)
		if err != nil {
			reporter.Errorf("Failed to delete '%s' user from the HTPasswd IDP of cluster '%s': %s",
				idp.ClusterAdminUsername, clusterKey, err)
			os.Exit(1)
		}
		reporter.Infof("Admin user '%s' has been deleted from cluster '%s'", idp.ClusterAdminUsername, clusterKey)
	}
}
