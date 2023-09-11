/*
Copyright (c) 2023 Red Hat, Inc.

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

package autoscaler

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "autoscaler",
	Aliases: []string{"cluster-autoscaler"},
	Short:   "Create an autoscaler for a cluster",
	Long: "Configuring cluster-wide autoscaling behavior. At least one machine-pool should " +
		"have autoscaling enabled for the configuration to be active",
	Example: `  # Interactively create an autoscaler to a cluster named "mycluster"
  rosa create autoscaler --cluster=mycluster --interactive

  # Create a cluster-autoscaler where it should skip nodes with local storage
  rosa create autoscaler --cluster=mycluster --skip-nodes-with-local-storage

  # Create a cluster-autoscaler with log verbosity of '3'
  rosa create autoscaler --cluster=mycluster --log-verbosity 3

  # Create a cluster-autoscaler with total CPU constraints
  rosa create autoscaler --cluster=mycluster --min-cores 10 --max-cores 100`,
	Run: run,
}

var autoscalerArgs *clusterautoscaler.AutoscalerArgs

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)
	interactive.AddFlag(flags)
	autoscalerArgs = clusterautoscaler.AddClusterAutoscalerFlags(flags, "")
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready. Current state is '%s'", clusterKey, cluster.State())
		os.Exit(1)
	}

	autoscaler, err := r.OCMClient.GetClusterAutoscaler(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed getting autoscaler configuration for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}

	if autoscaler != nil {
		r.Reporter.Errorf("Autoscaler for cluster '%s' already exists. "+
			"You should edit it via 'rosa edit autoscaler'", clusterKey)
		os.Exit(1)
	}

	if !clusterautoscaler.IsAutoscalerSetViaCLI(cmd.Flags()) && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}

	r.Reporter.Debugf("Creating autoscaler for cluster '%s'", clusterKey)

	autoscalerArgs, err := clusterautoscaler.GetAutoscalerOptions(cmd.Flags(), "", false, autoscalerArgs)
	if err != nil {
		r.Reporter.Errorf("Failed creating autoscaler configuration for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}

	autoscalerConfig, err := clusterautoscaler.CreateAutoscalerConfig(autoscalerArgs)
	if err != nil {
		r.Reporter.Errorf("Failed creating autoscaler configuration for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}

	_, err = r.OCMClient.CreateClusterAutoscaler(cluster.ID(), autoscalerConfig)
	if err != nil {
		r.Reporter.Errorf("Failed creating autoscaler configuration for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}

	r.Reporter.Infof("Successfully created autoscaler configuration for cluster '%s'", cluster.ID())
}
