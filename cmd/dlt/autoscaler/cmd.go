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

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use   = "autoscaler"
	short = "Delete autoscaler for cluster"
	long  = "Delete autoscaler configuration for a given cluster. " +
		"Supported only on ROSA clusters with self-hosted Control Plane (Classic)"
	example = `  # Delete the autoscaler config for cluster named "mycluster"
  rosa delete autoscaler --cluster=mycluster`
)

var aliases = []string{"cluster-autoscaler"}

func NewDeleteAutoscalerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DeleteAutoscalerRunner()),
	}

	flags := cmd.Flags()
	ocm.AddClusterFlag(cmd)
	confirm.AddFlag(flags)
	return cmd
}

func DeleteAutoscalerRunner() rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, _ *cobra.Command, _ []string) error {
		clusterKey := r.GetClusterKey()
		cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
		if err != nil {
			return err
		}

		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Hosted Control Plane clusters do not support cluster-autoscaler configuration")
		}

		if !confirm.Confirm("delete cluster autoscaler?") {
			return nil
		}

		r.Reporter.Debugf("Deleting autoscaler for cluster '%s''", clusterKey)

		err = r.OCMClient.DeleteClusterAutoscaler(cluster.ID())
		if err != nil {
			return fmt.Errorf("Failed to delete autoscaler configuration for cluster '%s': %s",
				cluster.ID(), err)
		}
		r.Reporter.Infof("Successfully deleted autoscaler configuration for cluster '%s'", cluster.ID())
		return nil
	}
}
