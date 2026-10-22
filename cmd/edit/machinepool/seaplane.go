// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	rosaaws "github.com/openshift/rosa/pkg/aws"
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
	clusterKey               string
	nodePoolKey              string
	nodePoolName             string
	userOptions              *EditMachinepoolUserOptions
	cmd                      *cobra.Command
	labels                   map[string]string
	taints                   []hypershiftv1beta1.Taint
	imageType                string
	tags                     map[string]string
	nodeDrainGracePeriod     *metav1.Duration
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
	tagsChanged := h.cmd.Flags().Changed("tags")
	autoscalingChanged := h.cmd.Flags().Changed("enable-autoscaling") ||
		minReplicasChanged || maxReplicasChanged
	imageTypeChanged := h.cmd.Flags().Changed("image-type")
	nodeDrainChanged := h.cmd.Flags().Changed("node-drain-grace-period")

	if !replicasChanged && !spotMaxPriceChanged && !autoscalingChanged &&
		!h.cmd.Flags().Changed("labels") && !h.cmd.Flags().Changed("taints") &&
		!h.cmd.Flags().Changed("autorepair") && !h.cmd.Flags().Changed("additional-security-group-ids") &&
		!imageTypeChanged && !tagsChanged && !nodeDrainChanged {
		return fmt.Errorf(
			"specify at least one supported flag: --replicas, --enable-autoscaling, " +
				"--min-replicas, --max-replicas, --spot-max-price, --labels, --taints, --autorepair, " +
				"--additional-security-group-ids, --image-type, --tags, --node-drain-grace-period",
		)
	}
	if tagsChanged {
		tags, err := rosaaws.ParseUserTags([]string{input.Tags}, rosaaws.UserTagOptions{})
		if err != nil {
			return fmt.Errorf("invalid machine pool AWS tags: %w", err)
		}
		h.tags = tags
		encodedTags, err := json.Marshal(tags)
		if err != nil {
			return fmt.Errorf("failed to encode --tags: %w", err)
		}
		input.Tags = string(encodedTags)
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

	if imageTypeChanged {
		// Read --image-type from the command flag and validate before request.
		// The API enum has no empty member, so clearing the flag is rejected
		// rather than silently leaving the node pool as it is.
		imageTypeVal, err := h.cmd.Flags().GetString("image-type")
		if err != nil {
			return fmt.Errorf("error reading --image-type: %w", err)
		}
		if imageTypeVal == "" {
			return fmt.Errorf("--image-type requires a value: %s or %s",
				hypershiftv1beta1.ImageTypeLinux, hypershiftv1beta1.ImageTypeWindows)
		}
		imageType, err := mpHelpers.ParseSeaplaneImageType(imageTypeVal)
		if err != nil {
			return err
		}
		h.imageType = imageType
	}

	if nodeDrainChanged {
		nodeDrainVal := h.userOptions.nodeDrainGracePeriod
		nodeDrainGracePeriod, err := mpHelpers.ParseSeaplaneNodeDrainGracePeriod(nodeDrainVal)
		if err != nil {
			return fmt.Errorf("invalid --node-drain-grace-period: %w", err)
		}
		h.nodeDrainGracePeriod = nodeDrainGracePeriod
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
	if h.cmd.Flags().Changed("tags") {
		merged.Spec.Tags = h.tags
	}

	// The merge starts from the current node pool, so carry over the updated image type.
	if h.cmd.Flags().Changed("image-type") {
		if merged.Spec.NodePool.Platform.AWS == nil {
			return fmt.Errorf("cannot update the image type on a non-AWS node pool")
		}
		merged.Spec.NodePool.Platform.AWS.ImageType = hypershiftv1beta1.ImageType(h.imageType)
	}

	if h.cmd.Flags().Changed("node-drain-grace-period") {
		merged.Spec.NodePool.NodeDrainTimeout = h.nodeDrainGracePeriod
	}

	// Security groups - convert comma-separated string to []AWSResourceReference
	if h.cmd.Flags().Changed("additional-security-group-ids") {
		if merged.Spec.NodePool.Platform.AWS == nil {
			return fmt.Errorf("cannot update security groups on a non-AWS node pool")
		}

		securityGroupIDsStr, err := h.cmd.Flags().GetString("additional-security-group-ids")
		if err != nil {
			return fmt.Errorf("failed to get additional-security-group-ids flag: %w", err)
		}

		if securityGroupIDsStr == "" {
			// User explicitly set to empty - clear security groups
			merged.Spec.NodePool.Platform.AWS.SecurityGroups = nil
		} else {
			// Parse comma-separated list, trim and filter empty values
			rawIDs := strings.Split(securityGroupIDsStr, ",")
			validIDs := make([]string, 0, len(rawIDs))
			for _, sgID := range rawIDs {
				trimmed := strings.TrimSpace(sgID)
				if trimmed != "" {
					validIDs = append(validIDs, trimmed)
				}
			}

			// Only create SecurityGroups if we have valid IDs, otherwise clear
			if len(validIDs) > 0 {
				merged.Spec.NodePool.Platform.AWS.SecurityGroups = make(
					[]hypershiftv1beta1.AWSResourceReference,
					len(validIDs),
				)
				for i, sgID := range validIDs {
					id := sgID
					merged.Spec.NodePool.Platform.AWS.SecurityGroups[i] = hypershiftv1beta1.AWSResourceReference{
						ID: &id,
					}
				}
			} else {
				merged.Spec.NodePool.Platform.AWS.SecurityGroups = nil
			}
		}
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

	if obj.Spec.NodePool.Platform.AWS != nil && len(obj.Spec.NodePool.Platform.AWS.SecurityGroups) > 0 {
		sgIDs := make([]string, 0, len(obj.Spec.NodePool.Platform.AWS.SecurityGroups))
		for _, sg := range obj.Spec.NodePool.Platform.AWS.SecurityGroups {
			if sg.ID != nil {
				sgIDs = append(sgIDs, *sg.ID)
			}
		}
		if len(sgIDs) > 0 {
			fmt.Printf("  Additional Security Groups: %s\n", strings.Join(sgIDs, ", "))
		}
	}

	return nil
}
