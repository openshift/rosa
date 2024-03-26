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
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var externalAuthProvidersArgs *externalauthprovider.ExternalAuthProvidersArgs

const argsPrefix string = ""

var Cmd = &cobra.Command{
	Use:     "external-auth-provider",
	Aliases: []string{"externalauthproviders", "externalauthprovider", "external-auth-providers"},
	Short:   "Create an external authentication provider for a cluster.",
	Long:    "Configure a cluster to use an external authentication provider instead of an internal oidc provider.",
	Example: `  # Interactively create an external authentication provider to a cluster named "mycluster"
  rosa create external-auth-provider --cluster=mycluster --interactive`,
	Run:    run,
	Hidden: true,
	Args:   cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)
	interactive.AddFlag(flags)
	externalAuthProvidersArgs = externalauthprovider.AddExternalAuthProvidersFlags(Cmd, argsPrefix)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
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

	if !externalauthprovider.IsExternalAuthProviderSetViaCLI(cmd.Flags(), argsPrefix) && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}
	r.Reporter.Debugf("Creating an external authentication provider for cluster '%s'", clusterKey)

	externalAuthProvidersArgs, err := externalauthprovider.GetExternalAuthOptions(
		cmd.Flags(), "", false, externalAuthProvidersArgs)
	if err != nil {
		return fmt.Errorf("failed to create an external authentication provider for cluster '%s': %s",
			clusterKey, err)
	}

	err = externalAuthService.CreateExternalAuthProvider(cluster, clusterKey, externalAuthProvidersArgs, r)
	if err != nil {
		return err
	}

	r.Reporter.Infof("Successfully created an external authentication provider for cluster '%s'", cluster.ID())

	return nil
}
