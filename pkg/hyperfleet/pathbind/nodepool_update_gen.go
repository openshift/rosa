package pathbind

import (
	"context"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	pathbind "github.com/openshift-online/rosa-hyperfleet-api/clientset/pathbind"
	platform "github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	interactive "github.com/openshift/rosa/pkg/interactive"
	rosa "github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

// NodePoolUpdateInput holds mutable inputs for NodePool update.
type NodePoolUpdateInput struct {
	AutoRepair                    *bool  `hfsdk:"spec.autoRepair"`
	DisplayName                   string `hfsdk:"spec.displayName"`
	Labels                        string `hfsdk:"spec.labels"`
	Ami                           string `hfsdk:"spec.nodePool.platform.aws.ami"`
	ImageType                     string `hfsdk:"spec.nodePool.platform.aws.imageType"`
	InstanceProfile               string `hfsdk:"spec.nodePool.platform.aws.instanceProfile"`
	CapacityReservationId         string `hfsdk:"spec.nodePool.platform.aws.placement.capacityReservation.id"`
	CapacityReservationMarketType string `hfsdk:"spec.nodePool.platform.aws.placement.capacityReservation.marketType"`
	Preference                    string `hfsdk:"spec.nodePool.platform.aws.placement.capacityReservation.preference"`
	PlacementMarketType           string `hfsdk:"spec.nodePool.platform.aws.placement.marketType"`
	MaxPrice                      string `hfsdk:"spec.nodePool.platform.aws.placement.spot.maxPrice"`
	Tenancy                       string `hfsdk:"spec.nodePool.platform.aws.placement.tenancy"`
	ResourceTags                  string `hfsdk:"spec.nodePool.platform.aws.resourceTags"`
	Encrypted                     *bool  `hfsdk:"spec.nodePool.platform.aws.rootVolume.encrypted"`
	EncryptionKey                 string `hfsdk:"spec.nodePool.platform.aws.rootVolume.encryptionKey"`
	Iops                          *int64 `hfsdk:"spec.nodePool.platform.aws.rootVolume.iops"`
	Size                          *int64 `hfsdk:"spec.nodePool.platform.aws.rootVolume.size"`
	RootVolumeType                string `hfsdk:"spec.nodePool.platform.aws.rootVolume.type"`
	SecurityGroups                string `hfsdk:"spec.nodePool.platform.aws.securityGroups"`
	Filters                       string `hfsdk:"spec.nodePool.platform.aws.subnet.filters"`
	Image                         string `hfsdk:"spec.nodePool.release.image"`
	Replicas                      *int32 `hfsdk:"spec.nodePool.replicas"`
}

// NodePoolUpdatePlatformAPIFlags lists the cobra flag names registered by
// RegisterNodePoolUpdateFlags. Consumers use this to mark them in a help section.
var NodePoolUpdatePlatformAPIFlags = []string{
	"auto-repair",
	"display-name",
	"labels",
	"ami",
	"image-type",
	"instance-profile",
	"capacity-reservation-id",
	"capacity-reservation-market-type",
	"preference",
	"placement-market-type",
	"max-price",
	"tenancy",
	"resource-tags",
	"encrypted",
	"encryption-key",
	"iops",
	"size",
	"root-volume-type",
	"security-groups",
	"filters",
	"image",
	"replicas",
}

// RegisterNodePoolUpdateFlags registers cobra flags for mutable NodePool fields.
func RegisterNodePoolUpdateFlags(cmd *cobra.Command, input *NodePoolUpdateInput) {
	f := cmd.Flags()
	registerIfNew(f, "auto-repair", func() { input.AutoRepair = new(bool); f.BoolVar(input.AutoRepair, "auto-repair", false, "") })
	registerIfNew(f, "display-name", func() { f.StringVar(&input.DisplayName, "display-name", "", "") })
	registerIfNew(f, "labels", func() { f.StringVar(&input.Labels, "labels", "", "") })
	registerIfNew(f, "ami", func() { f.StringVar(&input.Ami, "ami", "", "") })
	registerIfNew(f, "image-type", func() { f.StringVar(&input.ImageType, "image-type", "", "") })
	registerIfNew(f, "instance-profile", func() { f.StringVar(&input.InstanceProfile, "instance-profile", "", "") })
	registerIfNew(f, "capacity-reservation-id", func() { f.StringVar(&input.CapacityReservationId, "capacity-reservation-id", "", "") })
	registerIfNew(f, "capacity-reservation-market-type", func() { f.StringVar(&input.CapacityReservationMarketType, "capacity-reservation-market-type", "", "") })
	registerIfNew(f, "preference", func() { f.StringVar(&input.Preference, "preference", "", "") })
	registerIfNew(f, "placement-market-type", func() { f.StringVar(&input.PlacementMarketType, "placement-market-type", "", "") })
	registerIfNew(f, "max-price", func() { f.StringVar(&input.MaxPrice, "max-price", "", "") })
	registerIfNew(f, "tenancy", func() { f.StringVar(&input.Tenancy, "tenancy", "", "") })
	registerIfNew(f, "resource-tags", func() { f.StringVar(&input.ResourceTags, "resource-tags", "", "") })
	registerIfNew(f, "encrypted", func() { input.Encrypted = new(bool); f.BoolVar(input.Encrypted, "encrypted", false, "") })
	registerIfNew(f, "encryption-key", func() { f.StringVar(&input.EncryptionKey, "encryption-key", "", "") })
	registerIfNew(f, "iops", func() { input.Iops = new(int64); f.Int64Var(input.Iops, "iops", 0, "") })
	registerIfNew(f, "size", func() { input.Size = new(int64); f.Int64Var(input.Size, "size", 0, "") })
	registerIfNew(f, "root-volume-type", func() { f.StringVar(&input.RootVolumeType, "root-volume-type", "", "") })
	registerIfNew(f, "security-groups", func() { f.StringVar(&input.SecurityGroups, "security-groups", "", "") })
	registerIfNew(f, "filters", func() { f.StringVar(&input.Filters, "filters", "", "") })
	registerIfNew(f, "image", func() { f.StringVar(&input.Image, "image", "", "") })
	registerIfNew(f, "replicas", func() {
		input.Replicas = new(int32)
		f.Int32Var(input.Replicas, "replicas", 0, "Number of worker nodes.")
	})
}

// NodePoolUpdateHandler is implemented by consumers to provide NodePool update logic.
type NodePoolUpdateHandler interface {
	// Prompt fills missing inputs interactively (called only when interactive.Enabled()).
	// Embed GeneratedNodePoolUpdatePrompt to inherit auto-generated prompting for required fields.
	Prompt(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, input *NodePoolUpdateInput) error
	// PreRequest validates and transforms inputs before Expand.
	PreRequest(ctx context.Context, r *rosa.Runtime, input *NodePoolUpdateInput) error
	// PostExpand sets SDK fields pathbind cannot express.
	PostExpand(ctx context.Context, r *rosa.Runtime, input *NodePoolUpdateInput, obj *v1alpha1.NodePool) error
	// PostResponse handles the SDK response.
	PostResponse(ctx context.Context, r *rosa.Runtime, obj *v1alpha1.NodePool) error
}

// GeneratedNodePoolUpdatePrompt provides auto-generated interactive prompting for required update fields.
// Embed in your NodePoolUpdateHandler to inherit it; override Prompt() to extend.
type GeneratedNodePoolUpdatePrompt struct{}

func (GeneratedNodePoolUpdatePrompt) Prompt(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, input *NodePoolUpdateInput) error {
	return nil
}

// normalizeNodePoolUpdateInput clears numeric pointer fields that were pre-allocated by
// cobra but not actually set by the user. This allows pathbind.Expand to distinguish
// between "not set" (nil) and "explicitly set to zero".
func normalizeNodePoolUpdateInput(cmd *cobra.Command, input *NodePoolUpdateInput) {

	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("iops") != nil && !cmd.Flag("iops").Changed &&
		input.Iops != nil {
		input.Iops = nil
	}
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("size") != nil && !cmd.Flag("size").Changed &&
		input.Size != nil {
		input.Size = nil
	}
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("replicas") != nil && !cmd.Flag("replicas").Changed &&
		input.Replicas != nil {
		input.Replicas = nil
	}
}

// RunUpdateNodePool executes the update dispatch. Generated; do not edit.
// uid is the server-assigned UID of the resource to update.
func RunUpdateNodePool(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, uid string, input *NodePoolUpdateInput, h NodePoolUpdateHandler, namespace string) error {
	if interactive.Enabled() {
		if err := h.Prompt(ctx, r, cmd, input); err != nil {
			return err
		}
	}
	if err := h.PreRequest(ctx, r, input); err != nil {
		return err
	}
	normalizeNodePoolUpdateInput(cmd, input)
	obj := &v1alpha1.NodePool{}
	if err := pathbind.Expand(ctx, *input, obj); err != nil {
		return err
	}
	obj.Name = uid
	if err := h.PostExpand(ctx, r, input, obj); err != nil {
		return err
	}
	updated, err := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(namespace).Update(ctx, obj, platform.UpdateOptions{})
	if err != nil {
		return err
	}
	return h.PostResponse(ctx, r, updated)
}
