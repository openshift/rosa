package kubeletconfig

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "kubeletconfigs"
	short   = "List kubeletconfigs"
	long    = short
	example = ` # List the kubeletconfigs for cluster 'foo'
rosa list kubeletconfig --cluster foo`
)

var aliases = []string{"kubelet-configs", "kubeletconfig", "kubelet-config"}

func NewListKubeletConfigsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Aliases: aliases,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), ListKubeletConfigRunner()),
	}

	output.AddFlag(cmd)
	ocm.AddClusterFlag(cmd)
	return cmd
}

func ListKubeletConfigRunner() rosa.CommandRunner {
	return func(ctx context.Context, runtime *rosa.Runtime, command *cobra.Command, args []string) error {

		cluster, err := runtime.OCMClient.GetCluster(runtime.GetClusterKey(), runtime.Creator)
		if err != nil {
			return err
		}
		kubeletConfigs, err := runtime.OCMClient.ListKubeletConfigs(ctx, cluster.ID())
		if err != nil {
			return err
		}

		if output.HasFlag() {
			output.Print(kubeletConfigs)
		} else {
			if len(kubeletConfigs) == 0 {
				runtime.Reporter.Infof("There are no KubeletConfigs for cluster '%s'.", runtime.ClusterKey)
				return nil
			}

			// Create the writer that will be used to print the tabulated results:
			tb := output.NewTableBuilder()
			tb.SetHeaders("ID", "NAME", "POD PIDS LIMIT")

			for _, kc := range kubeletConfigs {
				name := kc.ID()
				if kc.Name() != "" {
					name = kc.Name()
				}
				tb.AddRow(
					kc.ID(),
					name,
					fmt.Sprintf("%d", kc.PodPidsLimit()),
				)
			}
			tb.Render()
			return nil
		}

		return nil
	}
}
