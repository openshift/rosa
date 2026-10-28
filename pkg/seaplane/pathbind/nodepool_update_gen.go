package pathbind

import (
	"context"
	"encoding/json"
	"fmt"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	pathbind "github.com/openshift-online/rosa-seaplane-api/clientset/pathbind"
	platform "github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	interactive "github.com/openshift/rosa/pkg/interactive"
	rosa "github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

// NodePoolUpdateInput holds mutable inputs for NodePool update.
type NodePoolUpdateInput struct {
	AutoRepair                    *bool  `hfsdk:"spec.autoRepair"`
	Labels                        string `hfsdk:"spec.labels"`
	MaxSurge                      string `hfsdk:"spec.maxSurge"`
	MaxUnavailable                string `hfsdk:"spec.maxUnavailable"`
	Max                           *int32 `hfsdk:"spec.nodePool.autoScaling.max"`
	Min                           *int32 `hfsdk:"spec.nodePool.autoScaling.min"`
	Config                        string `hfsdk:"spec.nodePool.config"`
	NodeDrainTimeout              string `hfsdk:"spec.nodePool.nodeDrainTimeout"`
	NodeLabels                    string `hfsdk:"spec.nodePool.nodeLabels"`
	NodeVolumeDetachTimeout       string `hfsdk:"spec.nodePool.nodeVolumeDetachTimeout"`
	Ami                           string `hfsdk:"spec.nodePool.platform.aws.ami"`
	InstanceProfile               string `hfsdk:"spec.nodePool.platform.aws.instanceProfile"`
	CapacityReservationId         string `hfsdk:"spec.nodePool.platform.aws.placement.capacityReservation.id"`
	CapacityReservationMarketType string `hfsdk:"spec.nodePool.platform.aws.placement.capacityReservation.marketType"`
	Preference                    string `hfsdk:"spec.nodePool.platform.aws.placement.capacityReservation.preference"`
	PlacementMarketType           string `hfsdk:"spec.nodePool.platform.aws.placement.marketType"`
	MaxPrice                      string `hfsdk:"spec.nodePool.platform.aws.placement.spot.maxPrice"`
	Tenancy                       string `hfsdk:"spec.nodePool.platform.aws.placement.tenancy"`
	SecurityGroupIds              string `hfsdk:"spec.nodePool.platform.aws.securityGroups"`
	Filters                       string `hfsdk:"spec.nodePool.platform.aws.subnet.filters"`
	Image                         string `hfsdk:"spec.nodePool.release.image"`
	Replicas                      *int32 `hfsdk:"spec.nodePool.replicas"`
	Taints                        string `hfsdk:"spec.nodePool.taints"`
	TuningConfig                  string `hfsdk:"spec.nodePool.tuningConfig"`
	Tags                          string `hfsdk:"spec.tags"`
}

// NodePoolUpdatePlatformAPIFlags lists the cobra flag names registered by
// RegisterNodePoolUpdateFlags. Consumers use this to mark them in a help section.
var NodePoolUpdatePlatformAPIFlags = []string{
	"auto-repair",
	"labels",
	"max-surge",
	"max-unavailable",
	"node-labels",
	"ami",
	"instance-profile",
	"capacity-reservation-id",
	"capacity-reservation-market-type",
	"preference",
	"placement-market-type",
	"max-price",
	"tenancy",
	"additional-security-group-ids",
	"filters",
	"image",
	"replicas",
	"tags",
}

// RegisterNodePoolUpdateFlags registers cobra flags for mutable NodePool fields.
func RegisterNodePoolUpdateFlags(cmd *cobra.Command, input *NodePoolUpdateInput) {
	f := cmd.Flags()
	registerIfNew(f, "auto-repair", func() { input.AutoRepair = new(bool); f.BoolVar(input.AutoRepair, "auto-repair", false, "") })
	registerIfNew(f, "labels", func() { f.StringVar(&input.Labels, "labels", "", "") })
	registerIfNew(f, "max-surge", func() { f.StringVar(&input.MaxSurge, "max-surge", "", "") })
	registerIfNew(f, "max-unavailable", func() { f.StringVar(&input.MaxUnavailable, "max-unavailable", "", "") })
	registerIfNew(f, "node-labels", func() { f.StringVar(&input.NodeLabels, "node-labels", "", "") })
	registerIfNew(f, "ami", func() { f.StringVar(&input.Ami, "ami", "", "") })
	registerIfNew(f, "instance-profile", func() { f.StringVar(&input.InstanceProfile, "instance-profile", "", "") })
	registerIfNew(f, "capacity-reservation-id", func() { f.StringVar(&input.CapacityReservationId, "capacity-reservation-id", "", "") })
	registerIfNew(f, "capacity-reservation-market-type", func() { f.StringVar(&input.CapacityReservationMarketType, "capacity-reservation-market-type", "", "") })
	registerIfNew(f, "preference", func() { f.StringVar(&input.Preference, "preference", "", "") })
	registerIfNew(f, "placement-market-type", func() { f.StringVar(&input.PlacementMarketType, "placement-market-type", "", "") })
	registerIfNew(f, "max-price", func() { f.StringVar(&input.MaxPrice, "max-price", "", "") })
	registerIfNew(f, "tenancy", func() { f.StringVar(&input.Tenancy, "tenancy", "", "") })
	registerIfNew(f, "additional-security-group-ids", func() {
		f.StringVar(&input.SecurityGroupIds, "additional-security-group-ids", "", "Additional AWS Security Group IDs to attach to node pool instances. Format should be a comma-separated list.")
	})
	registerIfNew(f, "filters", func() { f.StringVar(&input.Filters, "filters", "", "") })
	registerIfNew(f, "image", func() { f.StringVar(&input.Image, "image", "", "") })
	registerIfNew(f, "replicas", func() {
		input.Replicas = new(int32)
		f.Int32Var(input.Replicas, "replicas", 0, "Number of worker nodes.")
	})
	registerIfNew(f, "tags", func() {
		f.StringVar(&input.Tags, "tags", "", "Apply user defined tags to all resources created by ROSA in AWS. Tags are comma separated, for example: 'key value, foo bar'. Keys or values with spaces use double quotes; quote the whole flag for your shell (bash/zsh/PowerShell: --tags '\"Cost Center\":\"Finance Team\"'; Windows cmd.exe: --tags \"\"\"Cost Center\"\":\"\"Finance Team\"\"\"). Empty values (key:) are supported for ROSA Classic only, not for Hosted Control Plane.")
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

// normalizeNodePoolUpdateInput clears pointer flag fields that were pre-allocated by
// cobra but not actually set by the user. This allows pathbind.Expand to distinguish
// between "not set" (nil) and "explicitly set to zero".
func normalizeNodePoolUpdateInput(cmd *cobra.Command, input *NodePoolUpdateInput) {
	if cmd == nil {
		return
	}

	if cmd.Flags().Lookup("auto-repair") != nil && !cmd.Flag("auto-repair").Changed &&
		input.AutoRepair != nil {
		input.AutoRepair = nil
	}
	if cmd.Flags().Lookup("replicas") != nil && !cmd.Flag("replicas").Changed &&
		input.Replicas != nil {
		input.Replicas = nil
	}
}

// overlayNodePoolUpdateInput starts from the current values and applies only
// fields explicitly supplied by the caller or computed by PostExpand. An
// explicitly empty value is allowed only for resettable fields.
func overlayNodePoolUpdateInput(cmd *cobra.Command, before, input, expanded *NodePoolUpdateInput) (NodePoolUpdateInput, error) {
	after := *before
	if flagChanged(cmd, "auto-repair") && pathbind.IsUnset(input.AutoRepair) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.autoRepair")
	} else if !pathbind.IsUnset(expanded.AutoRepair) {
		after.AutoRepair = expanded.AutoRepair
	} else if !pathbind.IsUnset(input.AutoRepair) {
		after.AutoRepair = input.AutoRepair
	}
	if flagChanged(cmd, "labels") && pathbind.IsUnset(input.Labels) {
		after.Labels = input.Labels
	} else if !pathbind.IsUnset(expanded.Labels) {
		after.Labels = expanded.Labels
	} else if !pathbind.IsUnset(input.Labels) {
		after.Labels = input.Labels
	}
	if flagChanged(cmd, "max-surge") && pathbind.IsUnset(input.MaxSurge) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.maxSurge")
	} else if !pathbind.IsUnset(expanded.MaxSurge) {
		after.MaxSurge = expanded.MaxSurge
	} else if !pathbind.IsUnset(input.MaxSurge) {
		after.MaxSurge = input.MaxSurge
	}
	if flagChanged(cmd, "max-unavailable") && pathbind.IsUnset(input.MaxUnavailable) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.maxUnavailable")
	} else if !pathbind.IsUnset(expanded.MaxUnavailable) {
		after.MaxUnavailable = expanded.MaxUnavailable
	} else if !pathbind.IsUnset(input.MaxUnavailable) {
		after.MaxUnavailable = input.MaxUnavailable
	}
	if flagChanged(cmd, "") && pathbind.IsUnset(input.Max) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.autoScaling.max")
	} else if !pathbind.IsUnset(expanded.Max) {
		after.Max = expanded.Max
	} else if !pathbind.IsUnset(input.Max) {
		after.Max = input.Max
	}
	if flagChanged(cmd, "") && pathbind.IsUnset(input.Min) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.autoScaling.min")
	} else if !pathbind.IsUnset(expanded.Min) {
		after.Min = expanded.Min
	} else if !pathbind.IsUnset(input.Min) {
		after.Min = input.Min
	}
	if flagChanged(cmd, "") && pathbind.IsUnset(input.Config) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.config")
	} else if !pathbind.IsUnset(expanded.Config) {
		after.Config = expanded.Config
	} else if !pathbind.IsUnset(input.Config) {
		after.Config = input.Config
	}
	if flagChanged(cmd, "") && pathbind.IsUnset(input.NodeDrainTimeout) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.nodeDrainTimeout")
	} else if !pathbind.IsUnset(expanded.NodeDrainTimeout) {
		after.NodeDrainTimeout = expanded.NodeDrainTimeout
	} else if !pathbind.IsUnset(input.NodeDrainTimeout) {
		after.NodeDrainTimeout = input.NodeDrainTimeout
	}
	if flagChanged(cmd, "node-labels") && pathbind.IsUnset(input.NodeLabels) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.nodeLabels")
	} else if !pathbind.IsUnset(expanded.NodeLabels) {
		after.NodeLabels = expanded.NodeLabels
	} else if !pathbind.IsUnset(input.NodeLabels) {
		after.NodeLabels = input.NodeLabels
	}
	if flagChanged(cmd, "") && pathbind.IsUnset(input.NodeVolumeDetachTimeout) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.nodeVolumeDetachTimeout")
	} else if !pathbind.IsUnset(expanded.NodeVolumeDetachTimeout) {
		after.NodeVolumeDetachTimeout = expanded.NodeVolumeDetachTimeout
	} else if !pathbind.IsUnset(input.NodeVolumeDetachTimeout) {
		after.NodeVolumeDetachTimeout = input.NodeVolumeDetachTimeout
	}
	if flagChanged(cmd, "ami") && pathbind.IsUnset(input.Ami) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.ami")
	} else if !pathbind.IsUnset(expanded.Ami) {
		after.Ami = expanded.Ami
	} else if !pathbind.IsUnset(input.Ami) {
		after.Ami = input.Ami
	}
	if flagChanged(cmd, "instance-profile") && pathbind.IsUnset(input.InstanceProfile) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.instanceProfile")
	} else if !pathbind.IsUnset(expanded.InstanceProfile) {
		after.InstanceProfile = expanded.InstanceProfile
	} else if !pathbind.IsUnset(input.InstanceProfile) {
		after.InstanceProfile = input.InstanceProfile
	}
	if flagChanged(cmd, "capacity-reservation-id") && pathbind.IsUnset(input.CapacityReservationId) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.placement.capacityReservation.id")
	} else if !pathbind.IsUnset(expanded.CapacityReservationId) {
		after.CapacityReservationId = expanded.CapacityReservationId
	} else if !pathbind.IsUnset(input.CapacityReservationId) {
		after.CapacityReservationId = input.CapacityReservationId
	}
	if flagChanged(cmd, "capacity-reservation-market-type") && pathbind.IsUnset(input.CapacityReservationMarketType) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.placement.capacityReservation.marketType")
	} else if !pathbind.IsUnset(expanded.CapacityReservationMarketType) {
		after.CapacityReservationMarketType = expanded.CapacityReservationMarketType
	} else if !pathbind.IsUnset(input.CapacityReservationMarketType) {
		after.CapacityReservationMarketType = input.CapacityReservationMarketType
	}
	if flagChanged(cmd, "preference") && pathbind.IsUnset(input.Preference) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.placement.capacityReservation.preference")
	} else if !pathbind.IsUnset(expanded.Preference) {
		after.Preference = expanded.Preference
	} else if !pathbind.IsUnset(input.Preference) {
		after.Preference = input.Preference
	}
	if flagChanged(cmd, "placement-market-type") && pathbind.IsUnset(input.PlacementMarketType) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.placement.marketType")
	} else if !pathbind.IsUnset(expanded.PlacementMarketType) {
		after.PlacementMarketType = expanded.PlacementMarketType
	} else if !pathbind.IsUnset(input.PlacementMarketType) {
		after.PlacementMarketType = input.PlacementMarketType
	}
	if flagChanged(cmd, "max-price") && pathbind.IsUnset(input.MaxPrice) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.placement.spot.maxPrice")
	} else if !pathbind.IsUnset(expanded.MaxPrice) {
		after.MaxPrice = expanded.MaxPrice
	} else if !pathbind.IsUnset(input.MaxPrice) {
		after.MaxPrice = input.MaxPrice
	}
	if flagChanged(cmd, "tenancy") && pathbind.IsUnset(input.Tenancy) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.placement.tenancy")
	} else if !pathbind.IsUnset(expanded.Tenancy) {
		after.Tenancy = expanded.Tenancy
	} else if !pathbind.IsUnset(input.Tenancy) {
		after.Tenancy = input.Tenancy
	}
	if flagChanged(cmd, "additional-security-group-ids") && pathbind.IsUnset(input.SecurityGroupIds) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.securityGroups")
	} else if !pathbind.IsUnset(expanded.SecurityGroupIds) {
		after.SecurityGroupIds = expanded.SecurityGroupIds
	} else if !pathbind.IsUnset(input.SecurityGroupIds) {
		after.SecurityGroupIds = input.SecurityGroupIds
	}
	if flagChanged(cmd, "filters") && pathbind.IsUnset(input.Filters) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.platform.aws.subnet.filters")
	} else if !pathbind.IsUnset(expanded.Filters) {
		after.Filters = expanded.Filters
	} else if !pathbind.IsUnset(input.Filters) {
		after.Filters = input.Filters
	}
	if flagChanged(cmd, "image") && pathbind.IsUnset(input.Image) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.release.image")
	} else if !pathbind.IsUnset(expanded.Image) {
		after.Image = expanded.Image
	} else if !pathbind.IsUnset(input.Image) {
		after.Image = input.Image
	}
	if flagChanged(cmd, "replicas") && pathbind.IsUnset(input.Replicas) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.replicas")
	} else if !pathbind.IsUnset(expanded.Replicas) {
		after.Replicas = expanded.Replicas
	} else if !pathbind.IsUnset(input.Replicas) {
		after.Replicas = input.Replicas
	}
	if flagChanged(cmd, "") && pathbind.IsUnset(input.Taints) {
		after.Taints = input.Taints
	} else if !pathbind.IsUnset(expanded.Taints) {
		after.Taints = expanded.Taints
	} else if !pathbind.IsUnset(input.Taints) {
		after.Taints = input.Taints
	}
	if flagChanged(cmd, "") && pathbind.IsUnset(input.TuningConfig) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.nodePool.tuningConfig")
	} else if !pathbind.IsUnset(expanded.TuningConfig) {
		after.TuningConfig = expanded.TuningConfig
	} else if !pathbind.IsUnset(input.TuningConfig) {
		after.TuningConfig = input.TuningConfig
	}
	if flagChanged(cmd, "tags") && pathbind.IsUnset(input.Tags) {
		after.Tags = input.Tags
	} else if !pathbind.IsUnset(expanded.Tags) {
		after.Tags = expanded.Tags
	} else if !pathbind.IsUnset(input.Tags) {
		after.Tags = input.Tags
	}
	return after, nil
}

func buildNodePoolUpdateDocument(ctx context.Context, cmd *cobra.Command, current *v1alpha1.NodePool, input *NodePoolUpdateInput, obj *v1alpha1.NodePool) (map[string]any, error) {
	var before, expanded NodePoolUpdateInput
	if err := pathbind.Flatten(ctx, current, &before); err != nil {
		return nil, err
	}
	if err := pathbind.Flatten(ctx, obj, &expanded); err != nil {
		return nil, err
	}
	after, err := overlayNodePoolUpdateInput(cmd, &before, input, &expanded)
	if err != nil {
		return nil, err
	}
	return pathbind.UpdateDocument(ctx, before, after, obj)
}

// RunUpdateNodePool executes the update dispatch. Generated; do not edit.
// name is the account-scoped metadata.name used to address the resource in the Platform API.
func RunUpdateNodePool(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, name string, input *NodePoolUpdateInput, h NodePoolUpdateHandler) error {
	if interactive.Enabled() {
		if err := h.Prompt(ctx, r, cmd, input); err != nil {
			return err
		}
	}
	if err := h.PreRequest(ctx, r, input); err != nil {
		return err
	}
	normalizeNodePoolUpdateInput(cmd, input)
	current, err := r.SeaplaneClient.SeaplaneV1alpha1().NodePools().Get(ctx, name, platform.GetOptions{})
	if err != nil {
		return err
	}
	obj := &v1alpha1.NodePool{}
	if err := pathbind.Expand(ctx, *input, obj); err != nil {
		return err
	}
	obj.Name = name
	if err := h.PostExpand(ctx, r, input, obj); err != nil {
		return err
	}
	document, err := buildNodePoolUpdateDocument(ctx, cmd, current, input, obj)
	if err != nil {
		return err
	}
	if len(document) == 0 {
		return h.PostResponse(ctx, r, current)
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return err
	}
	updated, err := r.SeaplaneClient.SeaplaneV1alpha1().NodePools().Patch(ctx, name, types.MergePatchType, payload, platform.PatchOptions{})
	if err != nil {
		return err
	}
	return h.PostResponse(ctx, r, updated)
}
