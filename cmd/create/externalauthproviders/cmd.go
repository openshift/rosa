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
	"github.com/openshift/rosa/pkg/input"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var externalAuthProvidersArgs *externalauthproviders.ExternalAuthProvidersArgs

const argsPrefix string = ""

var Cmd = &cobra.Command{
	Use:     "external-auth-providers",
	Aliases: []string{"externalauthproviders", "externalauthprovider", "external-auth-provider"},
	Short:   "Create external authentication providers for a cluster.",
	Long:    "Configuring the cluster with external authentication configuration instead of internal oidc provider. ",
	Example: `  # Interactively create external authentication providers to a cluster named "mycluster"
  rosa create external-auth-providers --cluster=mycluster --interactive`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)
	interactive.AddFlag(flags)
	externalAuthProvidersArgs = externalauthproviders.AddExternalAuthProvidersFlags(Cmd, argsPrefix)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	input.CheckIfHypershiftClusterOrExit(r, cluster)

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	if cluster.ExternalAuthConfig().Enabled() {
		// continue on the creation of the external auth config

	} else {
		r.Reporter.Errorf("External authentication configuration is not enabled for cluster '%s'\n"+
			"Create a hosted control plane with '--external-auth-providers-enabled' parameter to enabled the configuration",
			clusterKey)
		os.Exit(1)
	}

	// name := externalAuthProvidersArgs.name
	// if name == "" && !interactive.Enabled() {
	// 	interactive.Enable()
	// 	r.Reporter.Infof("Enabling interactive mode")
	// }

}
