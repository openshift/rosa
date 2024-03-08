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

	cadmin "github.com/openshift/rosa/cmd/create/admin"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "idp ID",
	Aliases: []string{"idps"},
	Short:   "Delete cluster IDPs",
	Long:    "Delete a specific identity provider for a cluster.",
	Example: `  # Delete an identity provider named github-1
  rosa delete idp github-1 --cluster=mycluster`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line parameter containing the name of the identity provider",
			)
		}
		return nil
	},
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	idpName := argv[0]

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	// Try to find the identity provider:
	r.Reporter.Debugf("Loading identity provider '%s'", idpName)
	idps, err := r.OCMClient.GetIdentityProviders(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get identity providers for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	var idp *cmv1.IdentityProvider
	for _, item := range idps {
		if item.Name() == idpName {
			idp = item
			break
		}
	}
	if idp == nil {
		r.Reporter.Errorf("Failed to get identity provider '%s' for cluster '%s'", idpName, clusterKey)
		os.Exit(1)
	}
	if ocm.IdentityProviderType(idp) == ocm.HTPasswdIDPType {
		clusterAdminIDP, _, err := cadmin.FindIDPWithAdmin(cluster, r)
		if err != nil {
			r.Reporter.Errorf(err.Error())
			os.Exit(1)
		}
		if clusterAdminIDP != nil && clusterAdminIDP.Name() == idp.Name() {
			r.Reporter.Warnf("The cluster-admin user is contained in the HTPasswd IDP. Deleting the IDP will " +
				"also delete the admin user.")
		}
	}
	if confirm.Confirm("delete identity provider %s on cluster %s", idpName, clusterKey) {
		r.Reporter.Debugf("Deleting identity provider '%s' on cluster '%s'", idpName, clusterKey)
		err = r.OCMClient.DeleteIdentityProvider(cluster.ID(), idp.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to delete identity provider '%s' on cluster '%s': %s",
				idpName, clusterKey, err)
			os.Exit(1)
		}
		r.Reporter.Infof("Successfully deleted identity provider '%s' from cluster '%s'", idpName, clusterKey)
	}
}
