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
	"context"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	argsPrefix = ""
	use        = "autoscaler"
	short      = "Edit the autoscaler of a cluster"
	long       = "Configuring cluster-wide autoscaling behavior. At least one machine-pool should " +
		"have autoscaling enabled for the configuration to be active"
	example = `  # Interactively edit an autoscaler to a cluster named "mycluster"
  rosa edit autoscaler --cluster=mycluster --interactive

  # Edit a cluster-autoscaler to skip nodes with local storage
  rosa edit autoscaler --cluster=mycluster --skip-nodes-with-local-storage

  # Edit a cluster-autoscaler with log verbosity of '3'
  rosa edit autoscaler --cluster=mycluster --log-verbosity 3

  # Edit a cluster-autoscaler with total CPU constraints
  rosa edit autoscaler --cluster=mycluster --min-cores 10 --max-cores 100`
)

var aliases = []string{"cluster-autoscaler"}

func NewEditAutoscalerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(cmd)
	interactive.AddFlag(flags)
	autoscalerArgs := clusterautoscaler.AddClusterAutoscalerFlags(cmd, argsPrefix)
	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCM(), EditAutoscalerRunner(autoscalerArgs))
	return cmd
}

func EditAutoscalerRunner(autoscalerArgs *clusterautoscaler.AutoscalerArgs) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, command *cobra.Command, _ []string) error {
		clusterKey := r.GetClusterKey()
		cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
		if err != nil {
			return err
		}

		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Hosted Control Plane clusters do not support cluster-autoscaler configuration")
		}

		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("Cluster '%s' is not yet ready. Current state is '%s'", clusterKey, cluster.State())
		}

		autoscaler, err := r.OCMClient.GetClusterAutoscaler(cluster.ID())
		if err != nil {
			return fmt.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
				cluster.ID(), err)
		}

		if autoscaler == nil {
			return fmt.Errorf("No autoscaler for cluster '%s' has been found. "+
				"You should first create it via 'rosa create autoscaler'", clusterKey)
		}

		if !clusterautoscaler.IsAutoscalerSetViaCLI(command.Flags(), argsPrefix) && !interactive.Enabled() {
			interactive.Enable()
			r.Reporter.Infof("Enabling interactive mode")
		}

		r.Reporter.Debugf("Updating autoscaler for cluster '%s'", clusterKey)

		autoscalerArgs, err := clusterautoscaler.PrefillAutoscalerArgs(command, autoscalerArgs, autoscaler)
		if err != nil {
			return fmt.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
				cluster.ID(), err)
		}

		autoscalerValidationArgs := &clusterautoscaler.AutoscalerValidationArgs{
			ClusterVersion: cluster.OpenshiftVersion(),
			MultiAz:        cluster.MultiAZ(),
			IsHostedCp:     cluster.Hypershift().Enabled(),
		}

		autoscalerArgs, err = clusterautoscaler.GetAutoscalerOptions(
			command.Flags(), "", false, autoscalerArgs, autoscalerValidationArgs)
		if err != nil {
			return fmt.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
				cluster.ID(), err)
		}

		autoscalerConfig, err := clusterautoscaler.CreateAutoscalerConfig(autoscalerArgs)
		if err != nil {
			return fmt.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
				cluster.ID(), err)
		}

		_, err = r.OCMClient.UpdateClusterAutoscaler(cluster.ID(), autoscalerConfig)
		if err != nil {
			return fmt.Errorf("Failed updating autoscaler configuration for cluster '%s': %s",
				cluster.ID(), err)
		}

		r.Reporter.Infof("Successfully updated autoscaler configuration for cluster '%s'", cluster.ID())
		return nil
	}
}
