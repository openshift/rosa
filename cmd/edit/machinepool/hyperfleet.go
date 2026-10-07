package machinepool

import (
	"context"
	"fmt"
	"math"
	"os"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
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

	runtime := *r
	if cmd.Flags().Changed("labels") || cmd.Flags().Changed("taints") {
		runtime.HyperFleetClient = hyperfleet.WithNodePoolMetadataUpdates(r.HyperFleetClient,
			cmd.Flags().Changed("labels"), cmd.Flags().Changed("taints"))
	}
	if err := hfpathbind.RunUpdateNodePool(ctx, &runtime, cmd, nodePoolUID, &hfNodePoolUpdateInput,
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
	labels      map[string]string
	taints      []hypershiftv1beta1.Taint
}

func (h *hyperfleetNodePoolUpdate) PreRequest(
	_ context.Context,
	r *rosa.Runtime,
	input *hfpathbind.NodePoolUpdateInput,
) error {
	replicasChanged := h.cmd.Flags().Changed("replicas")
	spotMaxPriceChanged := h.cmd.Flags().Changed("spot-max-price")
	autoscalingChanged := h.cmd.Flags().Changed("enable-autoscaling") ||
		h.cmd.Flags().Changed("min-replicas") ||
		h.cmd.Flags().Changed("max-replicas")

	if !replicasChanged && !spotMaxPriceChanged && !autoscalingChanged &&
		!h.cmd.Flags().Changed("labels") && !h.cmd.Flags().Changed("taints") &&
		!h.cmd.Flags().Changed("autorepair") {
		return fmt.Errorf(
			"specify at least one supported flag: --replicas, --enable-autoscaling, " +
				"--min-replicas, --max-replicas, --spot-max-price, --labels, --taints, --autorepair",
		)
	}

	if h.cmd.Flags().Changed("labels") {
		labels, err := mpHelpers.ParseHyperfleetLabels(h.userOptions.labels)
		if err != nil {
			return err
		}
		h.labels = labels
	}
	if h.cmd.Flags().Changed("taints") {
		taints, err := mpHelpers.ParseHyperfleetTaints(h.userOptions.taints)
		if err != nil {
			return err
		}
		h.taints = taints
	}

	// Validate autoscaling and replicas are mutually exclusive
	if autoscalingChanged && replicasChanged {
		return fmt.Errorf("replicas cannot be set when autoscaling is enabled")
	}

	if replicasChanged {
		if h.userOptions.replicas < 0 || h.userOptions.replicas > math.MaxInt32 {
			return fmt.Errorf("--replicas must be between 0 and %d", math.MaxInt32)
		}

		// Bridge: --replicas shares its name with the OCM registration so registerIfNew
		// skips it; read from userOptions. Remove when OCM flag registration is dropped.
		replicas := int32(h.userOptions.replicas)
		input.Replicas = &replicas
	}

	if autoscalingChanged {
		if h.userOptions.minReplicas < 0 {
			return fmt.Errorf("min-replicas must be a non-negative number when autoscaling is enabled")
		}
		if h.userOptions.maxReplicas < 0 {
			return fmt.Errorf("max-replicas must be a non-negative number when autoscaling is enabled")
		}
		if h.userOptions.minReplicas > h.userOptions.maxReplicas {
			return fmt.Errorf("max-replicas must be greater than or equal to min-replicas")
		}
	}

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

	autoscalingChanged := h.cmd.Flags().Changed("enable-autoscaling") ||
		h.cmd.Flags().Changed("min-replicas") ||
		h.cmd.Flags().Changed("max-replicas")

	if h.cmd.Flags().Changed("replicas") {
		// Disable autoscaling when setting fixed replicas
		merged.Spec.NodePool.AutoScaling = nil
		merged.Spec.NodePool.Replicas = obj.Spec.NodePool.Replicas
	}

	if autoscalingChanged {
		if h.userOptions.autoscalingEnabled {
			// Enable autoscaling
			merged.Spec.NodePool.Replicas = nil
			minReplicas := int32(h.userOptions.minReplicas)
			merged.Spec.NodePool.AutoScaling = &hypershiftv1beta1.NodePoolAutoScaling{
				Min: &minReplicas,
				Max: int32(h.userOptions.maxReplicas),
			}
		} else {
			// Disable autoscaling - requires setting a fixed replica count
			merged.Spec.NodePool.AutoScaling = nil
			if merged.Spec.NodePool.Replicas == nil {
				// If no replicas set, default to min replicas value
				replicas := int32(h.userOptions.minReplicas)
				if replicas == 0 {
					replicas = 1
				}
				merged.Spec.NodePool.Replicas = &replicas
			}
		}
	}

	if h.cmd.Flags().Changed("spot-max-price") {
		// Ensure AWS platform and Placement are initialized
		if merged.Spec.NodePool.Platform.AWS == nil {
			return fmt.Errorf("cannot update spot-max-price on a non-AWS node pool")
		}
		if merged.Spec.NodePool.Platform.AWS.Placement == nil {
			return fmt.Errorf("cannot update spot-max-price on a node pool that is not using spot instances")
		}
		merged.Spec.NodePool.Platform.AWS.Placement.Spot.MaxPrice = h.userOptions.spotMaxPrice
	}

	if h.cmd.Flags().Changed("autorepair") {
		// Management is service-set; the operator renders it from spec.autoRepair.
		autorepair := h.userOptions.autorepair
		merged.Spec.AutoRepair = &autorepair
	}

	if h.cmd.Flags().Changed("labels") {
		merged.Spec.Labels = h.labels
	}
	if h.cmd.Flags().Changed("taints") {
		merged.Spec.NodePool.Taints = h.taints
	}

	// The Platform API rejects non-zero service-set management values on PUT,
	// including values returned by Get. The operator derives these from the
	// mutable top-level spec fields when rendering the HyperShift NodePool.
	merged.Spec.NodePool.Management = hypershiftv1beta1.NodePoolManagement{}
	merged.Spec.NodePool.NodeLabels = nil

	*obj = *merged
	return nil
}

func (h *hyperfleetNodePoolUpdate) PostResponse(_ context.Context, r *rosa.Runtime, obj *v1alpha1.NodePool) error {
	r.Reporter.Infof("Updated node pool '%s' in cluster '%s'", h.nodePoolKey, h.clusterKey)

	if obj.Spec.NodePool.AutoScaling != nil {
		minVal := int32(0)
		if obj.Spec.NodePool.AutoScaling.Min != nil {
			minVal = *obj.Spec.NodePool.AutoScaling.Min
		}
		maxVal := obj.Spec.NodePool.AutoScaling.Max
		fmt.Printf("  Autoscaling: enabled (min: %d, max: %d)\n", minVal, maxVal)
	} else if obj.Spec.NodePool.Replicas != nil {
		fmt.Printf("  Replicas: %d\n", *obj.Spec.NodePool.Replicas)
	}

	if obj.Spec.NodePool.Platform.AWS != nil &&
		obj.Spec.NodePool.Platform.AWS.Placement != nil &&
		obj.Spec.NodePool.Platform.AWS.Placement.Spot.MaxPrice != "" {
		fmt.Printf("  Spot Max Price: $%s\n", obj.Spec.NodePool.Platform.AWS.Placement.Spot.MaxPrice)
	}
	return nil
}
