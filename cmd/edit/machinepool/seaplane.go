package machinepool

import (
	"context"
	"fmt"
	"math"
	"os"

	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
	seaplanepathbind "github.com/openshift/rosa/pkg/seaplane/pathbind"
)

// seaplaneNodePoolUpdateInput is the backing store for seaplane-specific edit machinepool flags.
var seaplaneNodePoolUpdateInput seaplanepathbind.NodePoolUpdateInput

var (
	exitFn                  = func(code int) { os.Exit(code) }
	seaplaneEditMachinePool = func(userOptions *EditMachinepoolUserOptions, cmd *cobra.Command, argv []string) {
		r := rosa.NewRuntime().WithSeaplane()
		defer r.Cleanup()
		runSeaplaneEdit(r, userOptions, cmd, argv)
	}
)

// runSeaplaneEdit is a thin wrapper for direct test invocation.
func runSeaplaneEdit(r *rosa.Runtime, userOptions *EditMachinepoolUserOptions, cmd *cobra.Command, argv []string) {
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

	cluster, err := seaplane.ResolveCluster(ctx, r.SeaplaneClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}

	resourceName, err := seaplane.ResolveNodePoolName(ctx, r.SeaplaneClient, string(cluster.UID), nodePoolName)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}

	runtime := *r
	runtime.SeaplaneClient = seaplane.WithNodePoolMetadataUpdates(r.SeaplaneClient,
		cmd.Flags().Changed("labels"), cmd.Flags().Changed("taints"))
	if err := seaplanepathbind.RunUpdateNodePool(ctx, &runtime, cmd, resourceName, &seaplaneNodePoolUpdateInput,
		&seaplaneNodePoolUpdate{
			clusterKey:   clusterKey,
			nodePoolKey:  nodePoolName,
			nodePoolName: resourceName,
			userOptions:  userOptions,
			cmd:          cmd,
		},
	); err != nil {
		r.Reporter.Errorf("Failed to update node pool: %v", err)
		exitFn(1)
	}
}

// seaplaneNodePoolUpdate implements seaplanepathbind.NodePoolUpdateHandler for rosa edit machinepool.
type seaplaneNodePoolUpdate struct {
	seaplanepathbind.GeneratedNodePoolUpdatePrompt
	clusterKey   string
	nodePoolKey  string
	nodePoolName string
	userOptions  *EditMachinepoolUserOptions
	cmd          *cobra.Command
	labels       map[string]string
	taints       []hypershiftv1beta1.Taint
}

func (h *seaplaneNodePoolUpdate) PreRequest(
	_ context.Context,
	r *rosa.Runtime,
	input *seaplanepathbind.NodePoolUpdateInput,
) error {
	replicasChanged := h.cmd.Flags().Changed("replicas")
	spotMaxPriceChanged := h.cmd.Flags().Changed("spot-max-price")
	minReplicasChanged := h.cmd.Flags().Changed("min-replicas")
	maxReplicasChanged := h.cmd.Flags().Changed("max-replicas")
	autoscalingChanged := h.cmd.Flags().Changed("enable-autoscaling") ||
		minReplicasChanged || maxReplicasChanged

	if !replicasChanged && !spotMaxPriceChanged && !autoscalingChanged &&
		!h.cmd.Flags().Changed("labels") && !h.cmd.Flags().Changed("taints") &&
		!h.cmd.Flags().Changed("autorepair") {
		return fmt.Errorf(
			"specify at least one supported flag: --replicas, --enable-autoscaling, " +
				"--min-replicas, --max-replicas, --spot-max-price, --labels, --taints, --autorepair",
		)
	}

	if h.cmd.Flags().Changed("labels") {
		labels, err := mpHelpers.ParseSeaplaneLabels(h.userOptions.labels)
		if err != nil {
			return err
		}
		h.labels = labels
	}
	if h.cmd.Flags().Changed("taints") {
		taints, err := mpHelpers.ParseSeaplaneTaints(h.userOptions.taints)
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

	return nil
}

func (h *seaplaneNodePoolUpdate) PostExpand(
	ctx context.Context,
	r *rosa.Runtime,
	_ *seaplanepathbind.NodePoolUpdateInput,
	obj *v1alpha1.NodePool,
) error {
	// Get current node pool to preserve fields not covered by this update.
	// TODO: this Get-then-merge could be eliminated if the Platform API
	// supports PATCH or the SDK bridge wrapper handles partial updates.
	np, err := r.SeaplaneClient.SeaplaneV1alpha1().NodePools().
		Get(ctx, h.nodePoolName, platform.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node pool %q: %w", h.nodePoolKey, err)
	}

	// Start from the current object — preserves UID and all existing spec fields.
	// The bridge wrapper routes the Update by obj.Name, which is carried over here.
	merged := np.DeepCopy()

	enableAutoscalingChanged := h.cmd.Flags().Changed("enable-autoscaling")
	minReplicasChanged := h.cmd.Flags().Changed("min-replicas")
	maxReplicasChanged := h.cmd.Flags().Changed("max-replicas")
	autoscalingChanged := enableAutoscalingChanged ||
		minReplicasChanged ||
		maxReplicasChanged

	autoscalingEnabled := h.userOptions.autoscalingEnabled
	if !enableAutoscalingChanged {
		autoscalingEnabled = merged.Spec.NodePool.AutoScaling != nil
	}

	if (minReplicasChanged || maxReplicasChanged) && !autoscalingEnabled {
		return fmt.Errorf(
			"autoscaling is not enabled on node pool '%s'. can't set min or max replicas",
			h.nodePoolKey,
		)
	}

	if h.cmd.Flags().Changed("replicas") {
		// Disable autoscaling when setting fixed replicas
		merged.Spec.NodePool.AutoScaling = nil
		merged.Spec.NodePool.Replicas = obj.Spec.NodePool.Replicas
	}

	if autoscalingChanged {
		if autoscalingEnabled {
			minReplicas := 0
			maxReplicas := 0
			if merged.Spec.NodePool.AutoScaling != nil {
				if merged.Spec.NodePool.AutoScaling.Min != nil {
					minReplicas = int(*merged.Spec.NodePool.AutoScaling.Min)
				}
				maxReplicas = int(merged.Spec.NodePool.AutoScaling.Max)
			} else if merged.Spec.NodePool.Replicas != nil {
				minReplicas = int(*merged.Spec.NodePool.Replicas)
				maxReplicas = int(*merged.Spec.NodePool.Replicas)
			}

			if minReplicasChanged {
				minReplicas = h.userOptions.minReplicas
			}
			if maxReplicasChanged {
				maxReplicas = h.userOptions.maxReplicas
			}
			if minReplicas < 0 {
				return fmt.Errorf("min-replicas must be a non-negative number when autoscaling is enabled")
			}
			if maxReplicas < 1 {
				return fmt.Errorf("max-replicas must be greater than zero")
			}
			if minReplicas > maxReplicas {
				return fmt.Errorf("max-replicas must be greater than or equal to min-replicas")
			}

			merged.Spec.NodePool.Replicas = nil
			minimumReplicas := int32(minReplicas)
			merged.Spec.NodePool.AutoScaling = &hypershiftv1beta1.NodePoolAutoScaling{
				Min: &minimumReplicas,
				Max: int32(maxReplicas),
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

	// Management is service-set: the API field validator accepts it unchanged
	// from Get, but rejects clearing or changing it during an update.
	merged.Spec.NodePool.NodeLabels = nil

	*obj = *merged
	return nil
}

func (h *seaplaneNodePoolUpdate) PostResponse(_ context.Context, r *rosa.Runtime, obj *v1alpha1.NodePool) error {
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
