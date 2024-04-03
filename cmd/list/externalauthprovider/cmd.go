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
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/externalauthprovider"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "external-auth-providers",
	Aliases: []string{"externalauthproviders", "externalauthprovider", "external-auth-provider"},
	Short:   "List external authentication provider",
	Long:    "List external authentication provider for a cluster.",
	Example: `  # List all external authentication providers for a cluster named 'mycluster'"
  rosa list external-auth-provider -c mycluster`,
	Run:    run,
	Args:   cobra.NoArgs,
	Hidden: true,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command) error {
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	externalAuthService := externalauthprovider.NewExternalAuthService(r.OCMClient)
	err := externalAuthService.IsExternalAuthProviderSupported(cluster, clusterKey)
	if err != nil {
		return err
	}

	// Load any existing external auth providers for this cluster
	r.Reporter.Debugf("Loading external authentication providers for cluster '%s'", clusterKey)
	externalAuthProviders, err := r.OCMClient.GetExternalAuths(cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to get external authentication providers for cluster '%s': %v", cluster.ID(), err)
	}

	if output.HasFlag() {
		err = output.Print(externalAuthProviders)
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		return nil
	}

	if len(externalAuthProviders) == 0 {
		return fmt.Errorf("there are no external authentication providers for this cluster")
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprintf(writer, "NAME\tISSUER URL\n")
	for _, externalAuthProvider := range externalAuthProviders {
		fmt.Fprintf(writer, "%s\t%s\n",
			externalAuthProvider.ID(),
			externalAuthProvider.Issuer().URL(),
		)
	}
	writer.Flush()

	return nil
}
