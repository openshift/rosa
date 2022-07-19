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
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
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
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Try to find the htpasswd identity provider:
	r.Reporter.Debugf("Loading HTPasswd identity provider")
	idps, err := r.OCMClient.GetIdentityProviders(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get HTPasswd identity provider for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	var htpasswdIDP *cmv1.IdentityProvider
	for _, item := range idps {
		if ocm.IdentityProviderType(item) == ocm.HTPasswdIDPType {
			htpasswdIDP = item
		}
	}
	if htpasswdIDP == nil {
		r.Reporter.Errorf("Cluster '%s' does not have an admin user", clusterKey)
		os.Exit(1)
	}

	if confirm.Confirm("delete %s user on cluster %s", idp.ClusterAdminUsername, clusterKey) {
		// delete `cluster-admin` user from the HTPasswd IDP
		r.Reporter.Debugf("Deleting user '%s' identity provider on cluster '%s'", idp.ClusterAdminUsername, clusterKey)
		err = r.OCMClient.DeleteUser(cluster.ID(), "cluster-admins", idp.ClusterAdminUsername)
		if err != nil {
			r.Reporter.Errorf("Failed to delete '%s' user from cluster-admins groups of cluster '%s': %s",
				idp.ClusterAdminUsername, clusterKey, err)
			os.Exit(1)
		}

		//delete now the cluster-admin user from the htpasswd idp
		users, err := r.OCMClient.GetHTPasswdUserList(cluster.ID(), htpasswdIDP.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to list htpasswd idp users of cluster '%s': %s",
				clusterKey, err)
			os.Exit(1)
		}
		for _, user := range users.Slice() {
			if user.Username() == idp.ClusterAdminUsername {
				err := r.OCMClient.DeleteHTPasswdUser(idp.ClusterAdminUsername, cluster.ID(), htpasswdIDP)
				if err != nil {
					r.Reporter.Errorf("Failed to delete '%s' user from htpasswd idp users list of cluster '%s': %s",
						idp.ClusterAdminUsername, clusterKey, err)
					os.Exit(1)
				}

			}
		}

		users, err = r.OCMClient.GetHTPasswdUserList(cluster.ID(), htpasswdIDP.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to list htpasswd idp users of cluster '%s': %s",
				clusterKey, err)
			os.Exit(1)
		}

		htpasswdIdentityProvider, ok := htpasswdIDP.GetHtpasswd()
		if !ok {
			r.Reporter.Errorf("Failed to get htpasswd idp of cluster '%s': %s",
				clusterKey, err)
			os.Exit(1)
		}

		if users.Len() == 0 && htpasswdIdentityProvider.Username() == "" {
			//delete the idp as users list is empty
			err := r.OCMClient.DeleteIdentityProvider(cluster.ID(), htpasswdIDP.ID())
			if err != nil {
				r.Reporter.Errorf("Failed to delete htpasswd idp '%s' of cluster '%s': %s",
					htpasswdIDP.ID(), clusterKey, err)
				os.Exit(1)
			}
		}
	}
}
