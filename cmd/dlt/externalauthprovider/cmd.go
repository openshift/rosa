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

package externalauthprovider

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/externalauthprovider"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "external-auth-provider",
	Aliases: []string{"externalauthproviders", "externalauthprovider", "external-auth-providers"},
	Short:   "Delete external authentication provider",
	Long:    "Delete an external authentication provider from a cluster.",
	Example: `  # Delete an external authentication provider named exauth-1
  rosa delete external-auth-provider exauth-1  --cluster=mycluster`,
	Run:    run,
	Hidden: true,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"expected exactly one command line parameter containing the name of the external authentication provider",
			)
		}
		return nil
	},
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd, argv)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
	externalAuthName := argv[0]

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	externalAuthService := externalauthprovider.NewExternalAuthService(r.OCMClient)
	err := externalAuthService.IsExternalAuthProviderSupported(cluster, clusterKey)
	if err != nil {
		return err
	}

	if confirm.Confirm("delete external authentication provider %s on cluster %s", externalAuthName, clusterKey) {
		r.Reporter.Debugf("Deleting external authentication provider '%s' on cluster '%s'", externalAuthName, clusterKey)
		err := r.OCMClient.DeleteExternalAuth(cluster.ID(), externalAuthName)
		if err != nil {
			return fmt.Errorf("failed to delete external authentication provider '%s' on cluster '%s': %s",
				externalAuthName, clusterKey, err)
		}
		r.Reporter.Infof("Successfully deleted external authentication provider '%s' from cluster '%s'",
			externalAuthName, clusterKey)
	}
	return nil
}
