package autoscaler

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "autoscaler"
	short   = "Show details of the autoscaler for a cluster"
	long    = short
	example = ` # Describe the autoscaler for cluster 'foo'
rosa describe autoscaler --cluster foo`
)

var aliases = []string{"cluster-autoscaler"}

func NewDescribeAutoscalerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeAutoscalerRunner()),
	}

	output.AddFlag(cmd)
	ocm.AddClusterFlag(cmd)
	return cmd
}

func DescribeAutoscalerRunner() rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, _ *cobra.Command, _ []string) error {
		cluster, err := runtime.OCMClient.GetCluster(runtime.GetClusterKey(), runtime.Creator)
		if err != nil {
			return err
		}

		err = clusterautoscaler.IsAutoscalerSupported(runtime, cluster)
		if err != nil {
			return err
		}

		autoscaler, err := runtime.OCMClient.GetClusterAutoscaler(cluster.ID())
		if err != nil {
			return err
		}

		if autoscaler == nil {
			return fmt.Errorf("No autoscaler exists for cluster '%s'", runtime.ClusterKey)
		}

		if output.HasFlag() {
			output.Print(autoscaler)
		} else {
			if cluster.Hypershift().Enabled() {
				fmt.Print(clusterautoscaler.PrintHypershiftAutoscaler(autoscaler))
			} else {
				fmt.Print(clusterautoscaler.PrintAutoscaler(autoscaler))
			}
		}
		return nil
	}
}
