// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

var (
	exitFn                   = func(code int) { os.Exit(code) }
	seaplaneListMachinePools = func(_ *cobra.Command, _ []string) {
		r := rosa.NewRuntime().WithSeaplane()
		defer r.Cleanup()
		runSeaplaneList(r)
	}
)

func runSeaplaneList(r *rosa.Runtime) {
	ctx := context.Background()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	cluster, err := seaplane.ResolveCluster(ctx, r.SeaplaneClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	list, err := r.SeaplaneClient.SeaplaneV1alpha1().NodePools().List(ctx, platform.NodePoolListOptions{
		ClusterUID: string(cluster.UID),
	})
	if err != nil {
		r.Reporter.Errorf("Failed to list node pools: %v", err)
		exitFn(1)
	}

	if len(list.Items) == 0 {
		r.Reporter.Infof("No node pools found for cluster '%s'", clusterKey)
		return
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\tNAME\tREPLICAS\tINSTANCE TYPE\tSUBNET\tDISK SIZE\tSTATE\tLABELS\tTAINTS\t"+
		"AUTOSCALING\tAUTOREPAIR\tVERSION\n")
	for _, np := range list.Items {
		instanceType := ""
		subnetID := ""
		if np.Spec.NodePool.Platform.AWS != nil {
			instanceType = np.Spec.NodePool.Platform.AWS.InstanceType
			if np.Spec.NodePool.Platform.AWS.Subnet.ID != nil {
				subnetID = *np.Spec.NodePool.Platform.AWS.Subnet.ID
			}
		}
		diskSize := seaplane.FormatNodePoolDiskSize(np.Spec.NodePool.Platform.AWS)
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			string(np.UID),
			seaplane.NodePoolDisplayName(cluster.Name, np.Name),
			seaplane.FormatNodePoolReplicas(&np),
			instanceType,
			subnetID,
			diskSize,
			string(np.Status.Phase),
			seaplane.FormatNodePoolLabels(np.Spec.Labels),
			seaplane.FormatNodePoolTaints(np.Spec.NodePool.Taints),
			seaplane.FormatNodePoolAutoscaling(&np),
			seaplane.FormatNodePoolAutorepair(&np),
			np.Spec.NodePool.Release.Image,
		)
	}
	writer.Flush()
}
