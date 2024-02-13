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
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "external-auth-providers",
	Aliases: []string{"externalauthproviders", "externalauthprovider", "external-auth-provider"},
	Short:   "Delete External Authentication Providers",
	Long:    "Cleans up external authentication providers for the selected cluster.",
	Example: `  # Delete External Authentication Providers"
  rosa delete external-auth-providers with TBD param`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	confirm.AddFlag(Cmd.Flags())
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	err := handleExternalAuthProvidersDelete(r, cluster, clusterKey)

	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
}

func handleExternalAuthProvidersDelete(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string) error {
	err := externalauthproviders.ValidateHCPCluster(cluster)

	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	r.Reporter.Infof("Deleting External Authentication Providers for cluster '%s'", clusterKey)
	return r.OCMClient.DeleteExternalAuthProviders(clusterKey, "test")

}
