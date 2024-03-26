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

	"github.com/openshift/rosa/cmd/create/admin"
	cadmin "github.com/openshift/rosa/cmd/create/admin"
	"github.com/openshift/rosa/cmd/create/idp"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

type DeleteAdminUserStrategy interface {
	deleteAdmin(r *rosa.Runtime, identityProvider *cmv1.IdentityProvider)
}

type DeleteAdminIDP struct{}

func (d *DeleteAdminIDP) deleteAdmin(r *rosa.Runtime, identityProvider *cmv1.IdentityProvider) {
	err := r.OCMClient.DeleteIdentityProvider(r.Cluster.ID(), identityProvider.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to delete htpasswd idp '%s' of cluster '%s': %s",
			identityProvider.ID(), r.ClusterKey, err)
		os.Exit(1)
	}
}

type DeleteUserAdminFromIDP struct{}

func (d *DeleteUserAdminFromIDP) deleteAdmin(r *rosa.Runtime, identityProvider *cmv1.IdentityProvider) {
	clusterID := r.Cluster.ID()

	r.Reporter.Debugf("Deleting user '%s' from identity provider user list on cluster '%s'",
		cadmin.ClusterAdminUsername, r.ClusterKey)
	err := r.OCMClient.DeleteHTPasswdUser(cadmin.ClusterAdminUsername, clusterID, identityProvider)
	if err != nil {
		r.Reporter.Errorf("Failed to delete '%s' user from htpasswd idp users list of cluster '%s': %s",
			cadmin.ClusterAdminUsername, r.ClusterKey, err)
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
		r.Reporter.Debugf("Deleting '%s' identity provider on cluster '%s'", idp.HTPasswdIDPName, r.ClusterKey)
		err := r.OCMClient.DeleteIdentityProvider(clusterID, identityProvider.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to delete htpasswd idp '%s' of cluster '%s': %s",
				identityProvider.ID(), r.ClusterKey, err)
			os.Exit(1)
		}
	}
}

var Cmd = &cobra.Command{
	Use:   "admin",
	Short: "Deletes the 'cluster-admin' user",
	Long:  "Deletes the 'cluster-admin' user used to login to the cluster",
	Example: `  # Delete the 'cluster-admin' user
  rosa delete admin --cluster=mycluster`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", r.ClusterKey)
		os.Exit(1)
	}

	// Try to find the htpasswd identity provider:
	clusterID := cluster.ID()
	clusterAdminIDP, _, err := cadmin.FindIDPWithAdmin(cluster, r)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}

	if clusterAdminIDP == nil {
		r.Reporter.Errorf("Cluster '%s' does not have ‘%s’ user", r.ClusterKey, cadmin.ClusterAdminUsername)
		os.Exit(1)
	}

	if confirm.Confirm("delete %s user on cluster %s", cadmin.ClusterAdminUsername, r.ClusterKey) {
		// delete `cluster-admin` user from the HTPasswd IDP
		r.Reporter.Debugf("Deleting user '%s' from cluster-admins group on cluster '%s'",
			cadmin.ClusterAdminUsername, r.ClusterKey)
		err := r.OCMClient.DeleteUser(clusterID, admin.ClusterAdminGroupname, cadmin.ClusterAdminUsername)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		deletionStrategy := getAdminUserDeletionStrategy(r, clusterAdminIDP)
		deletionStrategy.deleteAdmin(r, clusterAdminIDP)

		r.Reporter.Infof("Admin user '%s' has been deleted from cluster '%s'", cadmin.ClusterAdminUsername, r.ClusterKey)
	}
}

func getAdminUserDeletionStrategy(r *rosa.Runtime, identityProvider *cmv1.IdentityProvider) DeleteAdminUserStrategy {
	if wasAdminCreatedUsingOldROSA(r, identityProvider) {
		return &DeleteAdminIDP{}
	}
	return &DeleteUserAdminFromIDP{}
}

func wasAdminCreatedUsingOldROSA(r *rosa.Runtime, identityProvider *cmv1.IdentityProvider) bool {
	htpasswdIdp, ok := identityProvider.GetHtpasswd()
	if !ok {
		r.Reporter.Errorf("Failed to get htpasswd idp for cluster '%s'", r.Cluster.ID())
		os.Exit(1)
	}
	return htpasswdIdp.Username() == cadmin.ClusterAdminUsername
}
