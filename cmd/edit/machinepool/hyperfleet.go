package machinepool

import (
	"context"
	"fmt"
	"math"
	"os"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

// clusterNamespacePrefix is the prefix the Platform API requires on the
// metadata.namespace field of NodePool resources ("cluster-<uuid>").
// TODO: this derivation ideally belongs in the SDK so consumers don't need
// to know the namespace format.
const clusterNamespacePrefix = "cluster-"

// hfNodePoolUpdateInput is the backing store for hyperfleet-specific edit machinepool flags.
var hfNodePoolUpdateInput hfpathbind.NodePoolUpdateInput

var (
	hfEnabled         = hyperfleet.Enabled
	exitFn            = func(code int) { os.Exit(code) }
	hfEditMachinePool = func(userOptions *EditMachinepoolUserOptions, cmd *cobra.Command, argv []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetEdit(r, userOptions, cmd, argv)
	}
)

// runHyperfleetEdit is a thin wrapper for direct test invocation.
func runHyperfleetEdit(r *rosa.Runtime, userOptions *EditMachinepoolUserOptions, cmd *cobra.Command, argv []string) {
	ctx := context.Background()

	nodePoolName := userOptions.machinepool
	if nodePoolName == "" && len(argv) > 0 {
		nodePoolName = argv[0]
	}
	if nodePoolName == "" {
		r.Reporter.Errorf("--machinepool is required")
		exitFn(1)
		return
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
		return
	}

	clusterUID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}

	nodePoolUID, err := hyperfleet.ResolveNodePoolUID(ctx, r.HyperFleetClient, clusterUID, nodePoolName)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}

	if err := hfpathbind.RunUpdateNodePool(ctx, r, cmd, nodePoolUID, &hfNodePoolUpdateInput,
		&hyperfleetNodePoolUpdate{
			clusterKey:  clusterKey,
			clusterUID:  clusterUID,
			nodePoolKey: nodePoolName,
			nodePoolUID: nodePoolUID,
			userOptions: userOptions,
			cmd:         cmd,
		},
		clusterNamespacePrefix+clusterUID,
	); err != nil {
		r.Reporter.Errorf("Failed to update node pool: %v", err)
		exitFn(1)
	}
}

// hyperfleetNodePoolUpdate implements hfpathbind.NodePoolUpdateHandler for rosa edit machinepool.
type hyperfleetNodePoolUpdate struct {
	hfpathbind.GeneratedNodePoolUpdatePrompt
	clusterKey  string
	clusterUID  string
	nodePoolKey string
	nodePoolUID string
	userOptions *EditMachinepoolUserOptions
	cmd         *cobra.Command
}

func (h *hyperfleetNodePoolUpdate) PreRequest(
	_ context.Context,
	r *rosa.Runtime,
	input *hfpathbind.NodePoolUpdateInput,
) error {
	if !h.cmd.Flags().Changed("replicas") {
		return fmt.Errorf("specify at least one supported flag: --replicas")
	}

	if h.userOptions.replicas < 0 || h.userOptions.replicas > math.MaxInt32 {
		return fmt.Errorf("--replicas must be between 0 and %d", math.MaxInt32)
	}

	// Bridge: --replicas shares its name with the OCM registration so registerIfNew
	// skips it; read from userOptions. Remove when OCM flag registration is dropped.
	replicas := int32(h.userOptions.replicas)
	input.Replicas = &replicas
	return nil
}

func (h *hyperfleetNodePoolUpdate) PostExpand(
	ctx context.Context,
	r *rosa.Runtime,
	_ *hfpathbind.NodePoolUpdateInput,
	obj *v1alpha1.NodePool,
) error {
	// Get current node pool to preserve fields not covered by this update.
	// TODO: this Get-then-merge could be eliminated if the Platform API
	// supports PATCH or the SDK bridge wrapper handles partial updates.
	np, err := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(clusterNamespacePrefix+h.clusterUID).
		Get(ctx, h.nodePoolUID, platform.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node pool %q: %w", h.nodePoolKey, err)
	}

	// Start from the current object — preserves UID and all existing spec fields.
	// The bridge wrapper routes the Update by obj.UID, which is carried over here.
	merged := np.DeepCopy()

	if h.cmd.Flags().Changed("replicas") {
		merged.Spec.NodePool.Replicas = obj.Spec.NodePool.Replicas
	}

	*obj = *merged
	return nil
}

func (h *hyperfleetNodePoolUpdate) PostResponse(_ context.Context, r *rosa.Runtime, obj *v1alpha1.NodePool) error {
	r.Reporter.Infof("Updated node pool '%s' in cluster '%s'", h.nodePoolKey, h.clusterKey)
	if obj.Spec.NodePool.Replicas != nil {
		fmt.Printf("  Replicas: %d\n", *obj.Spec.NodePool.Replicas)
	}
	return nil
}
