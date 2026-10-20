// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package autoscaler

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

func editSeaplaneAutoscalerRunner(args *clusterautoscaler.AutoscalerArgs) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		if _, err := clusterautoscaler.ValidateAutoscalerFlagsForHostedCp(argsPrefix, cmd); err != nil {
			return err
		}
		changes := clusterautoscaler.SeaplaneAutoscalingChanges{
			MaxNodeProvisionTime: cmd.Flags().Changed(argsPrefix + "max-node-provision-time"),
			MaxNodesTotal:        cmd.Flags().Changed(argsPrefix + "max-nodes-total"),
			MaxPodGracePeriod:    cmd.Flags().Changed(argsPrefix + "max-pod-grace-period"),
			PodPriorityThreshold: cmd.Flags().Changed(argsPrefix + "pod-priority-threshold"),
		}
		config, err := clusterautoscaler.BuildSeaplaneAutoscalingConfig(args, argsPrefix, changes)
		if err != nil {
			return err
		}

		clusterKey := r.GetClusterKey()
		clusterName, err := seaplane.ResolveClusterName(ctx, r.SeaplaneClient, clusterKey)
		if err != nil {
			return err
		}

		patch, err := json.Marshal(map[string]any{
			"spec": map[string]any{
				"hostedCluster": map[string]any{"autoscaling": config},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to build autoscaler update: %w", err)
		}
		if _, err := r.SeaplaneClient.SeaplaneV1alpha1().Clusters().Patch(
			ctx, clusterName, types.MergePatchType, patch, platform.PatchOptions{}); err != nil {
			return fmt.Errorf("failed updating autoscaler configuration for cluster %q: %w", clusterKey, err)
		}

		r.Reporter.Infof("Successfully updated autoscaler configuration for cluster '%s'", clusterKey)
		return nil
	}
}
