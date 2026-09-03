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

// ClusterUpdateInput holds mutable inputs for Cluster update.
type ClusterUpdateInput struct {
	DeleteProtection               *bool  `hfsdk:"spec.deleteProtection"`
	DisplayName                    string `hfsdk:"spec.displayName"`
	ExpirationTimestamp            string `hfsdk:"spec.expirationTimestamp"`
	ContainerLogMaxFiles           *int32 `hfsdk:"spec.hostedCluster.configuration.kubelet.containerLogMaxFiles"`
	ContainerLogMaxSize            string `hfsdk:"spec.hostedCluster.configuration.kubelet.containerLogMaxSize"`
	ImageGCHighThresholdPercent    *int32 `hfsdk:"spec.hostedCluster.configuration.kubelet.imageGCHighThresholdPercent"`
	ImageGCLowThresholdPercent     *int32 `hfsdk:"spec.hostedCluster.configuration.kubelet.imageGCLowThresholdPercent"`
	ImageMinimumGCAge              string `hfsdk:"spec.hostedCluster.configuration.kubelet.imageMinimumGCAge"`
	MaxPods                        *int32 `hfsdk:"spec.hostedCluster.configuration.kubelet.maxPods"`
	PodPidsLimit                   *int64 `hfsdk:"spec.hostedCluster.configuration.kubelet.podPidsLimit"`
	RegistryBurst                  *int32 `hfsdk:"spec.hostedCluster.configuration.kubelet.registryBurst"`
	RegistryPullQPS                *int32 `hfsdk:"spec.hostedCluster.configuration.kubelet.registryPullQPS"`
	SerializeImagePulls            *bool  `hfsdk:"spec.hostedCluster.configuration.kubelet.serializeImagePulls"`
	StreamingConnectionIdleTimeout string `hfsdk:"spec.hostedCluster.configuration.kubelet.streamingConnectionIdleTimeout"`
	ImageContentSources            string `hfsdk:"spec.hostedCluster.imageContentSources"`
	IssuerURL                      string `hfsdk:"spec.hostedCluster.issuerURL"`
	AllocateNodeCIDRs              string `hfsdk:"spec.hostedCluster.networking.allocateNodeCIDRs"`
	AdvertiseAddress               string `hfsdk:"spec.hostedCluster.networking.apiServer.advertiseAddress"`
	AllowedCIDRBlocks              string `hfsdk:"spec.hostedCluster.networking.apiServer.allowedCIDRBlocks"`
	Port                           *int32 `hfsdk:"spec.hostedCluster.networking.apiServer.port"`
	ClusterNetwork                 string `hfsdk:"spec.hostedCluster.networking.clusterNetwork"`
	MachineNetwork                 string `hfsdk:"spec.hostedCluster.networking.machineNetwork"`
	NetworkType                    string `hfsdk:"spec.hostedCluster.networking.networkType"`
	ServiceNetwork                 string `hfsdk:"spec.hostedCluster.networking.serviceNetwork"`
	AdditionalAllowedPrincipals    string `hfsdk:"spec.hostedCluster.platform.aws.additionalAllowedPrincipals"`
	Filters                        string `hfsdk:"spec.hostedCluster.platform.aws.cloudProviderConfig.subnet.filters"`
	EndpointAccess                 string `hfsdk:"spec.hostedCluster.platform.aws.endpointAccess"`
	MultiArch                      *bool  `hfsdk:"spec.hostedCluster.platform.aws.multiArch"`
	ResourceTags                   string `hfsdk:"spec.hostedCluster.platform.aws.resourceTags"`
	ControlPlaneOperatorARN        string `hfsdk:"spec.hostedCluster.platform.aws.rolesRef.controlPlaneOperatorARN"`
	ImageRegistryARN               string `hfsdk:"spec.hostedCluster.platform.aws.rolesRef.imageRegistryARN"`
	AwsRolesRefIngressARN          string `hfsdk:"spec.hostedCluster.platform.aws.rolesRef.ingressARN"`
	KubeCloudControllerARN         string `hfsdk:"spec.hostedCluster.platform.aws.rolesRef.kubeCloudControllerARN"`
	NetworkARN                     string `hfsdk:"spec.hostedCluster.platform.aws.rolesRef.networkARN"`
	NodePoolManagementARN          string `hfsdk:"spec.hostedCluster.platform.aws.rolesRef.nodePoolManagementARN"`
	StorageARN                     string `hfsdk:"spec.hostedCluster.platform.aws.rolesRef.storageARN"`
	ServiceEndpoints               string `hfsdk:"spec.hostedCluster.platform.aws.serviceEndpoints"`
	LocalZoneID                    string `hfsdk:"spec.hostedCluster.platform.aws.sharedVPC.localZoneID"`
	ControlPlaneARN                string `hfsdk:"spec.hostedCluster.platform.aws.sharedVPC.rolesRef.controlPlaneARN"`
	SharedVPCRolesRefIngressARN    string `hfsdk:"spec.hostedCluster.platform.aws.sharedVPC.rolesRef.ingressARN"`
	TerminationHandlerQueueURL     string `hfsdk:"spec.hostedCluster.platform.aws.terminationHandlerQueueURL"`
	Properties                     string `hfsdk:"spec.properties"`
	Tags                           string `hfsdk:"spec.tags"`
}

// ClusterUpdatePlatformAPIFlags lists the cobra flag names registered by
// RegisterClusterUpdateFlags. Consumers use this to mark them in a help section.
var ClusterUpdatePlatformAPIFlags = []string{
	"delete-protection",
	"display-name",
	"expiration-time",
	"container-log-max-files",
	"container-log-max-size",
	"image-gc-high-threshold-percent",
	"image-gc-low-threshold-percent",
	"image-minimum-gc-age",
	"max-pods",
	"pod-pids-limit",
	"registry-burst",
	"registry-pull-qps",
	"serialize-image-pulls",
	"streaming-connection-idle-timeout",
	"image-content-sources",
	"issuer-url",
	"allocate-node-cidrs",
	"advertise-address",
	"allowed-cidr-blocks",
	"port",
	"cluster-network",
	"machine-network",
	"network-type",
	"service-network",
	"additional-allowed-principals",
	"filters",
	"endpoint-access",
	"multi-arch",
	"resource-tags",
	"control-plane-operator-arn",
	"image-registry-arn",
	"aws-roles-ref-ingress-arn",
	"kube-cloud-controller-arn",
	"network-arn",
	"node-pool-management-arn",
	"storage-arn",
	"service-endpoints",
	"local-zone-id",
	"control-plane-arn",
	"shared-vpc-roles-ref-ingress-arn",
	"termination-handler-queue-url",
	"properties",
	"tags",
}

// RegisterClusterUpdateFlags registers cobra flags for mutable Cluster fields.
func RegisterClusterUpdateFlags(cmd *cobra.Command, input *ClusterUpdateInput) {
	f := cmd.Flags()
	registerIfNew(f, "delete-protection", func() {
		input.DeleteProtection = new(bool)
		f.BoolVar(input.DeleteProtection, "delete-protection", false, "Enable delete protection on the cluster.")
	})
	registerIfNew(f, "display-name", func() {
		f.StringVar(&input.DisplayName, "display-name", "", "Human-readable display name for the cluster.")
	})
	registerIfNew(f, "expiration-time", func() {
		f.StringVar(&input.ExpirationTimestamp, "expiration-time", "", "Cluster expiration time (RFC3339).")
	})
	registerIfNew(f, "container-log-max-files", func() {
		input.ContainerLogMaxFiles = new(int32)
		f.Int32Var(input.ContainerLogMaxFiles, "container-log-max-files", 0, "")
	})
	registerIfNew(f, "container-log-max-size", func() { f.StringVar(&input.ContainerLogMaxSize, "container-log-max-size", "", "") })
	registerIfNew(f, "image-gc-high-threshold-percent", func() {
		input.ImageGCHighThresholdPercent = new(int32)
		f.Int32Var(input.ImageGCHighThresholdPercent, "image-gc-high-threshold-percent", 0, "")
	})
	registerIfNew(f, "image-gc-low-threshold-percent", func() {
		input.ImageGCLowThresholdPercent = new(int32)
		f.Int32Var(input.ImageGCLowThresholdPercent, "image-gc-low-threshold-percent", 0, "")
	})
	registerIfNew(f, "image-minimum-gc-age", func() { f.StringVar(&input.ImageMinimumGCAge, "image-minimum-gc-age", "", "") })
	registerIfNew(f, "max-pods", func() { input.MaxPods = new(int32); f.Int32Var(input.MaxPods, "max-pods", 0, "") })
	registerIfNew(f, "pod-pids-limit", func() { input.PodPidsLimit = new(int64); f.Int64Var(input.PodPidsLimit, "pod-pids-limit", 0, "") })
	registerIfNew(f, "registry-burst", func() { input.RegistryBurst = new(int32); f.Int32Var(input.RegistryBurst, "registry-burst", 0, "") })
	registerIfNew(f, "registry-pull-qps", func() {
		input.RegistryPullQPS = new(int32)
		f.Int32Var(input.RegistryPullQPS, "registry-pull-qps", 0, "")
	})
	registerIfNew(f, "serialize-image-pulls", func() {
		input.SerializeImagePulls = new(bool)
		f.BoolVar(input.SerializeImagePulls, "serialize-image-pulls", false, "")
	})
	registerIfNew(f, "streaming-connection-idle-timeout", func() {
		f.StringVar(&input.StreamingConnectionIdleTimeout, "streaming-connection-idle-timeout", "", "")
	})
	registerIfNew(f, "image-content-sources", func() { f.StringVar(&input.ImageContentSources, "image-content-sources", "", "") })
	registerIfNew(f, "issuer-url", func() { f.StringVar(&input.IssuerURL, "issuer-url", "", "") })
	registerIfNew(f, "allocate-node-cidrs", func() { f.StringVar(&input.AllocateNodeCIDRs, "allocate-node-cidrs", "", "") })
	registerIfNew(f, "advertise-address", func() { f.StringVar(&input.AdvertiseAddress, "advertise-address", "", "") })
	registerIfNew(f, "allowed-cidr-blocks", func() { f.StringVar(&input.AllowedCIDRBlocks, "allowed-cidr-blocks", "", "") })
	registerIfNew(f, "port", func() { input.Port = new(int32); f.Int32Var(input.Port, "port", 0, "") })
	registerIfNew(f, "cluster-network", func() { f.StringVar(&input.ClusterNetwork, "cluster-network", "", "") })
	registerIfNew(f, "machine-network", func() { f.StringVar(&input.MachineNetwork, "machine-network", "", "") })
	registerIfNew(f, "network-type", func() { f.StringVar(&input.NetworkType, "network-type", "", "") })
	registerIfNew(f, "service-network", func() { f.StringVar(&input.ServiceNetwork, "service-network", "", "") })
	registerIfNew(f, "additional-allowed-principals", func() { f.StringVar(&input.AdditionalAllowedPrincipals, "additional-allowed-principals", "", "") })
	registerIfNew(f, "filters", func() { f.StringVar(&input.Filters, "filters", "", "") })
	registerIfNew(f, "endpoint-access", func() { f.StringVar(&input.EndpointAccess, "endpoint-access", "", "") })
	registerIfNew(f, "multi-arch", func() { input.MultiArch = new(bool); f.BoolVar(input.MultiArch, "multi-arch", false, "") })
	registerIfNew(f, "resource-tags", func() { f.StringVar(&input.ResourceTags, "resource-tags", "", "") })
	registerIfNew(f, "control-plane-operator-arn", func() { f.StringVar(&input.ControlPlaneOperatorARN, "control-plane-operator-arn", "", "") })
	registerIfNew(f, "image-registry-arn", func() { f.StringVar(&input.ImageRegistryARN, "image-registry-arn", "", "") })
	registerIfNew(f, "aws-roles-ref-ingress-arn", func() { f.StringVar(&input.AwsRolesRefIngressARN, "aws-roles-ref-ingress-arn", "", "") })
	registerIfNew(f, "kube-cloud-controller-arn", func() { f.StringVar(&input.KubeCloudControllerARN, "kube-cloud-controller-arn", "", "") })
	registerIfNew(f, "network-arn", func() { f.StringVar(&input.NetworkARN, "network-arn", "", "") })
	registerIfNew(f, "node-pool-management-arn", func() { f.StringVar(&input.NodePoolManagementARN, "node-pool-management-arn", "", "") })
	registerIfNew(f, "storage-arn", func() { f.StringVar(&input.StorageARN, "storage-arn", "", "") })
	registerIfNew(f, "service-endpoints", func() { f.StringVar(&input.ServiceEndpoints, "service-endpoints", "", "") })
	registerIfNew(f, "local-zone-id", func() { f.StringVar(&input.LocalZoneID, "local-zone-id", "", "") })
	registerIfNew(f, "control-plane-arn", func() { f.StringVar(&input.ControlPlaneARN, "control-plane-arn", "", "") })
	registerIfNew(f, "shared-vpc-roles-ref-ingress-arn", func() { f.StringVar(&input.SharedVPCRolesRefIngressARN, "shared-vpc-roles-ref-ingress-arn", "", "") })
	registerIfNew(f, "termination-handler-queue-url", func() { f.StringVar(&input.TerminationHandlerQueueURL, "termination-handler-queue-url", "", "") })
	registerIfNew(f, "properties", func() { f.StringVar(&input.Properties, "properties", "", "") })
	registerIfNew(f, "tags", func() { f.StringVar(&input.Tags, "tags", "", "") })
}

// ClusterUpdateHandler is implemented by consumers to provide Cluster update logic.
type ClusterUpdateHandler interface {
	// Prompt fills missing inputs interactively (called only when interactive.Enabled()).
	// Embed GeneratedClusterUpdatePrompt to inherit auto-generated prompting for required fields.
	Prompt(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, input *ClusterUpdateInput) error
	// PreRequest validates and transforms inputs before Expand.
	PreRequest(ctx context.Context, r *rosa.Runtime, input *ClusterUpdateInput) error
	// PostExpand sets SDK fields pathbind cannot express.
	PostExpand(ctx context.Context, r *rosa.Runtime, input *ClusterUpdateInput, obj *v1alpha1.Cluster) error
	// PostResponse handles the SDK response.
	PostResponse(ctx context.Context, r *rosa.Runtime, obj *v1alpha1.Cluster) error
}

// GeneratedClusterUpdatePrompt provides auto-generated interactive prompting for required update fields.
// Embed in your ClusterUpdateHandler to inherit it; override Prompt() to extend.
type GeneratedClusterUpdatePrompt struct{}

func (GeneratedClusterUpdatePrompt) Prompt(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, input *ClusterUpdateInput) error {
	return nil
}

// normalizeClusterUpdateInput clears numeric pointer fields that were pre-allocated by
// cobra but not actually set by the user. This allows pathbind.Expand to distinguish
// between "not set" (nil) and "explicitly set to zero".
func normalizeClusterUpdateInput(cmd *cobra.Command, input *ClusterUpdateInput) {

	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("container-log-max-files") != nil && !cmd.Flag("container-log-max-files").Changed &&
		input.ContainerLogMaxFiles != nil {
		input.ContainerLogMaxFiles = nil
	}
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("image-gc-high-threshold-percent") != nil && !cmd.Flag("image-gc-high-threshold-percent").Changed &&
		input.ImageGCHighThresholdPercent != nil {
		input.ImageGCHighThresholdPercent = nil
	}
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("image-gc-low-threshold-percent") != nil && !cmd.Flag("image-gc-low-threshold-percent").Changed &&
		input.ImageGCLowThresholdPercent != nil {
		input.ImageGCLowThresholdPercent = nil
	}
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("max-pods") != nil && !cmd.Flag("max-pods").Changed &&
		input.MaxPods != nil {
		input.MaxPods = nil
	}
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("pod-pids-limit") != nil && !cmd.Flag("pod-pids-limit").Changed &&
		input.PodPidsLimit != nil {
		input.PodPidsLimit = nil
	}
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("registry-burst") != nil && !cmd.Flag("registry-burst").Changed &&
		input.RegistryBurst != nil {
		input.RegistryBurst = nil
	}
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("registry-pull-qps") != nil && !cmd.Flag("registry-pull-qps").Changed &&
		input.RegistryPullQPS != nil {
		input.RegistryPullQPS = nil
	}
	if cmd == nil {
		return
	}
	if cmd.Flags().Lookup("port") != nil && !cmd.Flag("port").Changed &&
		input.Port != nil {
		input.Port = nil
	}
}

// RunUpdateCluster executes the update dispatch. Generated; do not edit.
// uid is the server-assigned UID of the resource to update.
func RunUpdateCluster(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, uid string, input *ClusterUpdateInput, h ClusterUpdateHandler) error {
	if interactive.Enabled() {
		if err := h.Prompt(ctx, r, cmd, input); err != nil {
			return err
		}
	}
	if err := h.PreRequest(ctx, r, input); err != nil {
		return err
	}
	normalizeClusterUpdateInput(cmd, input)
	obj := &v1alpha1.Cluster{}
	if err := pathbind.Expand(ctx, *input, obj); err != nil {
		return err
	}
	obj.Name = uid
	if err := h.PostExpand(ctx, r, input, obj); err != nil {
		return err
	}
	updated, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().Update(ctx, obj, platform.UpdateOptions{})
	if err != nil {
		return err
	}
	return h.PostResponse(ctx, r, updated)
}
