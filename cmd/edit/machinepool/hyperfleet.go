package machinepool

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled         = hyperfleet.Enabled
	exitFn            = func(code int) { os.Exit(code) }
	hfEditMachinePool = func(userOptions *EditMachinepoolUserOptions, cmd *cobra.Command, argv []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetEdit(r, userOptions, cmd, argv)
	}
)

// runHyperfleetEdit scales a node pool's replica count via the Platform API.
// Only --replicas is supported; autoscaling and other OCM-only flags are not applicable.
func runHyperfleetEdit(r *rosa.Runtime, userOptions *EditMachinepoolUserOptions, cmd *cobra.Command, argv []string) {
	ctx := context.Background()

	nodePoolName := userOptions.machinepool
	if nodePoolName == "" && len(argv) > 0 {
		nodePoolName = argv[0]
	}
	if nodePoolName == "" {
		r.Reporter.Errorf("--machinepool is required")
		exitFn(1)
	}

	if !cmd.Flags().Changed("replicas") {
		r.Reporter.Errorf("specify at least one supported flag: --replicas")
		exitFn(1)
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	clusterUID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	nodePoolID, err := hyperfleet.ResolveNodePoolUID(ctx, r.HyperFleetClient, clusterUID, nodePoolName)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	nodePools := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(clusterUID)
	np, err := nodePools.Get(ctx, nodePoolID, platform.GetOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to get node pool '%s': %v", nodePoolName, err)
		exitFn(1)
	}

	if userOptions.replicas < 0 || userOptions.replicas > math.MaxInt32 {
		r.Reporter.Errorf("--replicas must be between 0 and %d", math.MaxInt32)
		exitFn(1)
		return
	}

	updated := np.DeepCopy()
	replicas := int32(userOptions.replicas)
	updated.Spec.NodePool.Replicas = &replicas

	if _, err = nodePools.Update(ctx, updated, platform.UpdateOptions{}); err != nil {
		r.Reporter.Errorf("Failed to update node pool '%s': %v", nodePoolName, err)
		exitFn(1)
	}

	r.Reporter.Infof("Updated node pool '%s' in cluster '%s'", nodePoolName, clusterKey)
	fmt.Printf("  Replicas: %d\n", replicas)
}
