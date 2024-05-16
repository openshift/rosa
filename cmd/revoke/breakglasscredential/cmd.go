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

package breakglasscredential

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
	Use:     "break-glass-credentials",
	Aliases: []string{"break-glass-credential", "breakglasscredential", "breakglasscredentials"},
	Short:   "Revoke break glass credentials",
	Long:    "Revoke all the break glass credentials from a cluster.",
	Example: `  # Revoke all break glass credentials
  rosa revoke break-glass-credentials --cluster=mycluster`,
	Run:  run,
	Args: cobra.NoArgs,
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

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()

	externalAuthService := externalauthprovider.NewExternalAuthService(r.OCMClient)
	err := externalAuthService.IsExternalAuthProviderSupported(cluster, clusterKey)
	if err != nil {
		return err
	}

	breakGlassCredentials, err := r.OCMClient.GetBreakGlassCredentials(cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to get break glass credentials for cluster '%s': %v", clusterKey, err)
	}

	if len(breakGlassCredentials) == 0 {
		r.Reporter.Infof("There are no break glass credentials for cluster '%s'", clusterKey)
		return nil
	}

	if confirm.Confirm("revoke all the break glass credentials on cluster '%s'", clusterKey) {
		r.Reporter.Debugf("Revoking break glass credentials on cluster '%s'", clusterKey)
		err := r.OCMClient.DeleteBreakGlassCredentials(cluster.ID())
		if err != nil {
			return fmt.Errorf("failed to revoke break glass credentials on cluster '%s': %s",
				clusterKey, err)
		}
		r.Reporter.Infof("Successfully requested revocation for all break glass credentials from cluster '%s'",
			clusterKey)
	}
	return nil
}
