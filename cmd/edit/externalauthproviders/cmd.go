/*
Copyright (c) 2024 Red Hat, Inc.

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

package externalauthproviders

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/externalauthproviders"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var externalAuthProvidersArgs *externalauthproviders.ExternalAuthProvidersArgs

const argsPrefix string = ""

var Cmd = &cobra.Command{
	Use:     "external-auth-providers",
	Aliases: []string{"externalauthproviders", "externalauthprovider", "external-auth-provider"},
	Short:   "Edit external authentication providers",
	Long:    "Edit external authentication providers on a cluster.",
	Example: `  # Set issuer name 'issuer1' on cluster 'mycluster'
  rosa edit external-auth-providers --issuer-name=issuer1 --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)
	interactive.AddFlag(flags)
	externalAuthProvidersArgs = externalauthproviders.AddExternalAuthProvidersFlags(Cmd, argsPrefix)

}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	err := externalauthproviders.ValidateHCPCluster(cluster)

	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready. Current state is '%s'", clusterKey, cluster.State())
		os.Exit(1)
	}

	externalAuthProviders, err := r.OCMClient.GetExternalAuthProviders(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to fetch existing  external authentication providers for cluster '%s': %s",
			clusterKey, err)
		os.Exit(1)
	}

	if externalAuthProviders == nil {
		r.Reporter.Errorf("No external authentication providers for cluster '%s' has been found. "+
			"You should first create it via 'rosa create external-auth-providers'", clusterKey)
		os.Exit(1)
	}

	r.Reporter.Debugf("Updating external authentication providers for cluster '%s'", clusterKey)

	_, err = r.OCMClient.UpdateExternalAuthProviders(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed creating custom external authentication providers for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}

	r.Reporter.Infof("Successfully updated custom external authentication providers for cluster '%s'", clusterKey)
	os.Exit(0)

}
