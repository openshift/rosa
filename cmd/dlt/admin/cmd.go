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

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", r.ClusterKey)
		os.Exit(1)
	}

	// Try to find the htpasswd identity provider:
	r.Reporter.Debugf("Loading HTPasswd identity provider")
	clusterID := cluster.ID()
	idps, err := r.OCMClient.GetIdentityProviders(clusterID)
	if err != nil {
		r.Reporter.Errorf("Failed to get HTPasswd identity provider for cluster '%s': %v", r.ClusterKey, err)
		os.Exit(1)
	}

	var identityProvider *cmv1.IdentityProvider
	for _, item := range idps {
		if ocm.IdentityProviderType(item) == ocm.HTPasswdIDPType {
			identityProvider = item
		}
	}
	if identityProvider == nil {
		r.Reporter.Errorf("Cluster '%s' does not have an admin user", r.ClusterKey)
		os.Exit(1)
	}

	if confirm.Confirm("delete %s user on cluster %s", idp.ClusterAdminUsername, r.ClusterKey) {
		// delete `cluster-admin` user from the HTPasswd IDP
		r.Reporter.Debugf("Deleting user '%s' from cluster-admins group on cluster '%s'", idp.ClusterAdminUsername, r.ClusterKey)
		err = r.OCMClient.DeleteUser(clusterID, "cluster-admins", idp.ClusterAdminUsername)
		if err != nil {
			r.Reporter.Errorf("Failed to delete '%s' user from cluster-admins groups of cluster '%s': %s",
				idp.ClusterAdminUsername, r.ClusterKey, err)
			os.Exit(1)
		}

		htpasswdIdp, ok := identityProvider.GetHtpasswd()
		if !ok {
			r.Reporter.Errorf("Failed to get htpasswd idp for cluster '%s'", clusterID)
			os.Exit(1)
		}
		if htpasswdIdp.Username() == "cluster-admin" {
			//the admin was created with ROSA release less than 4.10
			//remove the entire idp
			err := r.OCMClient.DeleteIdentityProvider(clusterID, identityProvider.ID())
			if err != nil {
				r.Reporter.Errorf("Failed to delete htpasswd idp '%s' of cluster '%s': %s",
					identityProvider.ID(), r.ClusterKey, err)
				os.Exit(1)
			}
		} else {
			//delete now the cluster-admin user from the htpasswd idp
			r.Reporter.Debugf("Deleting user '%s' from identity provider user list on cluster '%s'", idp.ClusterAdminUsername, r.ClusterKey)
			err := r.OCMClient.DeleteHTPasswdUser(idp.ClusterAdminUsername, clusterID, identityProvider)
			if err != nil {
				r.Reporter.Errorf("Failed to delete '%s' user from htpasswd idp users list of cluster '%s': %s",
					idp.ClusterAdminUsername, r.ClusterKey, err)
				os.Exit(1)
			}

			users, err := r.OCMClient.GetHTPasswdUserList(clusterID, identityProvider.ID())
			if err != nil {
				r.Reporter.Errorf("Failed to list htpasswd idp users of cluster '%s': %s",
					r.ClusterKey, err)
				os.Exit(1)
			}

			htpasswdIdentityProvider, ok := identityProvider.GetHtpasswd()
			if !ok {
				r.Reporter.Errorf("Failed to get htpasswd idp of cluster '%s': %s",
					r.ClusterKey, err)
				os.Exit(1)
			}

			if users.Len() == 0 && htpasswdIdentityProvider.Username() == "" {
				//delete the idp as users list is empty
				r.Reporter.Debugf("Deleting '%s' identity provider on cluster '%s'", idp.HTPasswdIDPName, r.ClusterKey)
				err := r.OCMClient.DeleteIdentityProvider(clusterID, identityProvider.ID())
				if err != nil {
					r.Reporter.Errorf("Failed to delete htpasswd idp '%s' of cluster '%s': %s",
						identityProvider.ID(), r.ClusterKey, err)
					os.Exit(1)
				}
			}
		}
		r.Reporter.Infof("Admin user '%s' has been deleted from cluster '%s'", idp.ClusterAdminUsername, r.ClusterKey)
	}
}
