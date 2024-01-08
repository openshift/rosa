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
	"strconv"

	commonUtils "github.com/openshift-online/ocm-common/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const argsPrefix string = ""

var Cmd = &cobra.Command{
	Use:     "autoscaler",
	Aliases: []string{"cluster-autoscaler"},
	Short:   "Edit the autoscaler of a cluster",
	Long: "Configuring cluster-wide autoscaling behavior. At least one machine-pool should " +
		"have autoscaling enabled for the configuration to be active",
	Example: `  # Interactively edit an autoscaler to a cluster named "mycluster"
  rosa edit autoscaler --cluster=mycluster --interactive

  # Edit a cluster-autoscaler to skip nodes with local storage
  rosa edit autoscaler --cluster=mycluster --skip-nodes-with-local-storage

  # Edit a cluster-autoscaler with log verbosity of '3'
  rosa edit autoscaler --cluster=mycluster --log-verbosity 3

  # Edit a cluster-autoscaler with total CPU constraints
  rosa edit autoscaler --cluster=mycluster --min-cores 10 --max-cores 100`,
	Run: run,
}

var autoscalerArgs *clusterautoscaler.AutoscalerArgs

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)
	interactive.AddFlag(flags)
	autoscalerArgs = clusterautoscaler.AddClusterAutoscalerFlags(Cmd, argsPrefix)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()

	if cluster.Hypershift().Enabled() {
		r.Reporter.Errorf("Hosted Control Plane clusters do not support cluster-autoscaler configuration")
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready. Current state is '%s'", clusterKey, cluster.State())
		os.Exit(1)
	}

	autoscaler, err := r.OCMClient.GetClusterAutoscaler(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}

	if autoscaler == nil {
		r.Reporter.Errorf("No autoscaler for cluster '%s' has been found. "+
			"You should first create it via 'rosa create autoscaler'", clusterKey)
		os.Exit(1)
	}

	if !clusterautoscaler.IsAutoscalerSetViaCLI(cmd.Flags(), argsPrefix) && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}

	r.Reporter.Debugf("Updating autoscaler for cluster '%s'", clusterKey)

	if interactive.Enabled() {
		// pre-filling the parameters from the existing resource if running in interactive mode

		autoscalerArgs.BalanceSimilarNodeGroups = autoscaler.BalanceSimilarNodeGroups()
		autoscalerArgs.SkipNodesWithLocalStorage = autoscaler.SkipNodesWithLocalStorage()
		autoscalerArgs.LogVerbosity = autoscaler.LogVerbosity()
		autoscalerArgs.MaxPodGracePeriod = autoscaler.MaxPodGracePeriod()
		autoscalerArgs.PodPriorityThreshold = autoscaler.PodPriorityThreshold()
		autoscalerArgs.IgnoreDaemonsetsUtilization = autoscaler.IgnoreDaemonsetsUtilization()
		autoscalerArgs.MaxNodeProvisionTime = autoscaler.MaxNodeProvisionTime()
		autoscalerArgs.BalancingIgnoredLabels = autoscaler.BalancingIgnoredLabels()
		autoscalerArgs.ResourceLimits.MaxNodesTotal = autoscaler.ResourceLimits().MaxNodesTotal()
		autoscalerArgs.ResourceLimits.Cores.Min = autoscaler.ResourceLimits().Cores().Min()
		autoscalerArgs.ResourceLimits.Cores.Max = autoscaler.ResourceLimits().Cores().Max()
		autoscalerArgs.ResourceLimits.Memory.Min = autoscaler.ResourceLimits().Memory().Min()
		autoscalerArgs.ResourceLimits.Memory.Max = autoscaler.ResourceLimits().Memory().Max()

		// be aware we cannot easily pre-load GPU limits from existing configuration, so we'll have to
		// request it from scratch when interactive mode is on

		autoscalerArgs.ScaleDown.Enabled = autoscaler.ScaleDown().Enabled()
		autoscalerArgs.ScaleDown.UnneededTime = autoscaler.ScaleDown().UnneededTime()
		autoscalerArgs.ScaleDown.DelayAfterAdd = autoscaler.ScaleDown().DelayAfterAdd()
		autoscalerArgs.ScaleDown.DelayAfterDelete = autoscaler.ScaleDown().DelayAfterDelete()
		autoscalerArgs.ScaleDown.DelayAfterFailure = autoscaler.ScaleDown().DelayAfterFailure()

		utilizationThreshold, err := strconv.ParseFloat(
			autoscaler.ScaleDown().UtilizationThreshold(),
			commonUtils.MaxByteSize,
		)
		if err != nil {
			r.Reporter.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
				cluster.ID(), err)
			os.Exit(1)
		}
		autoscalerArgs.ScaleDown.UtilizationThreshold = utilizationThreshold
	}

	autoscalerArgs, err := clusterautoscaler.GetAutoscalerOptions(cmd.Flags(), "", false, autoscalerArgs)
	if err != nil {
		r.Reporter.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}

	autoscalerConfig, err := clusterautoscaler.CreateAutoscalerConfig(autoscalerArgs)
	if err != nil {
		r.Reporter.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}

	_, err = r.OCMClient.UpdateClusterAutoscaler(cluster.ID(), autoscalerConfig)
	if err != nil {
		r.Reporter.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}

	r.Reporter.Infof("Successfully updated autoscaler configuration for cluster '%s'", cluster.ID())
}
