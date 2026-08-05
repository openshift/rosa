package machinepool

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled          = hyperfleet.Enabled
	exitFn             = func(code int) { os.Exit(code) }
	hfListMachinePools = func(_ *cobra.Command, _ []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetList(r)
	}
)

func runHyperfleetList(r *rosa.Runtime) {
	ctx := context.Background()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	clusterUID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, r.Creator.AccountID, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	list, err := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(clusterUID).List(ctx, wrappers.ListOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to list node pools: %v", err)
		exitFn(1)
	}

	if len(list.Items) == 0 {
		r.Reporter.Infof("No node pools found for cluster '%s'", clusterKey)
		return
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\tNAME\tREPLICAS\tINSTANCE TYPE\tSUBNET\tSTATE\n")
	for _, np := range list.Items {
		replicas := int32(0)
		if np.Spec.NodePool.Replicas != nil {
			replicas = *np.Spec.NodePool.Replicas
		}
		instanceType := ""
		subnetID := ""
		if np.Spec.NodePool.Platform.AWS != nil {
			instanceType = np.Spec.NodePool.Platform.AWS.InstanceType
			if np.Spec.NodePool.Platform.AWS.Subnet.ID != nil {
				subnetID = *np.Spec.NodePool.Platform.AWS.Subnet.ID
			}
		}
		fmt.Fprintf(writer, "%s\t%s\t%d\t%s\t%s\t%s\n",
			string(np.UID),
			np.Name,
			replicas,
			instanceType,
			subnetID,
			string(np.Status.Phase),
		)
	}
	writer.Flush()
}
