// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package autoscaler

import (
	"context"
	"fmt"

	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

func describeSeaplaneAutoscalerRunner() rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, _ *cobra.Command, _ []string) error {
		clusterKey := r.GetClusterKey()
		clusterName, err := seaplane.ResolveClusterName(ctx, r.SeaplaneClient, clusterKey)
		if err != nil {
			return err
		}
		cluster, err := r.SeaplaneClient.SeaplaneV1alpha1().Clusters().Get(
			ctx, clusterName, platform.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get cluster %q: %w", clusterKey, err)
		}

		spec := cluster.Spec.HostedCluster.Autoscaling
		config := &ocm.AutoscalerConfig{
			MaxNodeProvisionTime: spec.MaxNodeProvisionTime,
		}
		if spec.MaxNodesTotal != nil {
			config.ResourceLimits.MaxNodesTotal = int(*spec.MaxNodesTotal)
		}
		if spec.MaxPodGracePeriod != nil {
			config.MaxPodGracePeriod = int(*spec.MaxPodGracePeriod)
		}
		if spec.PodPriorityThreshold != nil {
			config.PodPriorityThreshold = int(*spec.PodPriorityThreshold)
		}
		autoscaler, err := ocm.BuildClusterAutoscalerHostedCp(config).Build()
		if err != nil {
			return fmt.Errorf("failed to build autoscaler output: %w", err)
		}

		if output.HasFlag() {
			return output.Print(autoscaler)
		}
		fmt.Print(clusterautoscaler.PrintHypershiftAutoscaler(autoscaler))
		return nil
	}
}
