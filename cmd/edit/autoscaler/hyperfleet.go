// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package autoscaler

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

func editHyperfleetAutoscalerRunner(args *clusterautoscaler.AutoscalerArgs) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		config, err := clusterautoscaler.BuildHyperfleetAutoscalingConfig(cmd, argsPrefix, args, true)
		if err != nil {
			return err
		}

		clusterKey := r.GetClusterKey()
		clusterUID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, clusterKey)
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
		if _, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().Patch(
			ctx, clusterUID, types.MergePatchType, patch, platform.PatchOptions{}); err != nil {
			return fmt.Errorf("failed updating autoscaler configuration for cluster %q: %w", clusterKey, err)
		}

		r.Reporter.Infof("Successfully updated autoscaler configuration for cluster '%s'", clusterKey)
		return nil
	}
}
