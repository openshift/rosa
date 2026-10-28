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

// ClusterUpdateInput holds mutable inputs for Cluster update.
type ClusterUpdateInput struct {
	AdditionalTrustBundle          string `hfsdk:"spec.additionalTrustBundle"`
	DeleteProtection               *bool  `hfsdk:"spec.deleteProtection"`
	ExpirationTimestamp            string `hfsdk:"spec.expirationTimestamp"`
	RoleARN                        string `hfsdk:"spec.hostedCluster.autoNode.provisionerConfig.karpenter.aws.roleARN"`
	Platform                       string `hfsdk:"spec.hostedCluster.autoNode.provisionerConfig.karpenter.platform"`
	ProvisionerConfigName          string `hfsdk:"spec.hostedCluster.autoNode.provisionerConfig.name"`
	Channel                        string `hfsdk:"spec.hostedCluster.channel"`
	OidcProviders                  string `hfsdk:"spec.hostedCluster.configuration.authentication.oidcProviders"`
	AuthenticationType             string `hfsdk:"spec.hostedCluster.configuration.authentication.type"`
	ComponentRoutes                string `hfsdk:"spec.hostedCluster.configuration.ingress.componentRoutes"`
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
	HttpProxy                      string `hfsdk:"spec.hostedCluster.configuration.proxy.httpProxy"`
	HttpsProxy                     string `hfsdk:"spec.hostedCluster.configuration.proxy.httpsProxy"`
	NoProxy                        string `hfsdk:"spec.hostedCluster.configuration.proxy.noProxy"`
	SchedulerProfile               string `hfsdk:"spec.hostedCluster.configuration.scheduler.profile"`
	ImageContentSources            string `hfsdk:"spec.hostedCluster.imageContentSources"`
	AdditionalAllowedPrincipals    string `hfsdk:"spec.hostedCluster.platform.aws.additionalAllowedPrincipals"`
	Filters                        string `hfsdk:"spec.hostedCluster.platform.aws.cloudProviderConfig.subnet.filters"`
	EndpointAccess                 string `hfsdk:"spec.hostedCluster.platform.aws.endpointAccess"`
	MultiArch                      *bool  `hfsdk:"spec.hostedCluster.platform.aws.multiArch"`
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
	"additional-trust-bundle",
	"delete-protection",
	"expiration-time",
	"role-arn",
	"platform",
	"provisioner-config-name",
	"channel",
	"authentication-type",
	"component-routes",
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
	"http-proxy",
	"https-proxy",
	"no-proxy",
	"scheduler-profile",
	"image-content-sources",
	"additional-allowed-principals",
	"filters",
	"endpoint-access",
	"multi-arch",
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
	registerIfNew(f, "additional-trust-bundle", func() { f.StringVar(&input.AdditionalTrustBundle, "additional-trust-bundle", "", "") })
	registerIfNew(f, "delete-protection", func() {
		input.DeleteProtection = new(bool)
		f.BoolVar(input.DeleteProtection, "delete-protection", false, "Enable delete protection on the cluster.")
	})
	registerIfNew(f, "expiration-time", func() {
		f.StringVar(&input.ExpirationTimestamp, "expiration-time", "", "Cluster expiration time (RFC3339).")
	})
	registerIfNew(f, "role-arn", func() { f.StringVar(&input.RoleARN, "role-arn", "", "") })
	registerIfNew(f, "platform", func() { f.StringVar(&input.Platform, "platform", "", "") })
	registerIfNew(f, "provisioner-config-name", func() { f.StringVar(&input.ProvisionerConfigName, "provisioner-config-name", "", "") })
	registerIfNew(f, "channel", func() { f.StringVar(&input.Channel, "channel", "", "") })
	registerIfNew(f, "authentication-type", func() { f.StringVar(&input.AuthenticationType, "authentication-type", "", "") })
	registerIfNew(f, "component-routes", func() { f.StringVar(&input.ComponentRoutes, "component-routes", "", "") })
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
	registerIfNew(f, "http-proxy", func() { f.StringVar(&input.HttpProxy, "http-proxy", "", "") })
	registerIfNew(f, "https-proxy", func() { f.StringVar(&input.HttpsProxy, "https-proxy", "", "") })
	registerIfNew(f, "no-proxy", func() { f.StringVar(&input.NoProxy, "no-proxy", "", "") })
	registerIfNew(f, "scheduler-profile", func() {
		f.StringVar(&input.SchedulerProfile, "scheduler-profile", "", "Scheduler profile. Valid values are LowNodeUtilization, HighNodeUtilization, and NoScoring.")
	})
	registerIfNew(f, "image-content-sources", func() { f.StringVar(&input.ImageContentSources, "image-content-sources", "", "") })
	registerIfNew(f, "additional-allowed-principals", func() { f.StringVar(&input.AdditionalAllowedPrincipals, "additional-allowed-principals", "", "") })
	registerIfNew(f, "filters", func() { f.StringVar(&input.Filters, "filters", "", "") })
	registerIfNew(f, "endpoint-access", func() { f.StringVar(&input.EndpointAccess, "endpoint-access", "", "") })
	registerIfNew(f, "multi-arch", func() { input.MultiArch = new(bool); f.BoolVar(input.MultiArch, "multi-arch", false, "") })
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
	registerIfNew(f, "tags", func() {
		f.StringVar(&input.Tags, "tags", "", "Apply user defined tags to all resources created by ROSA in AWS. Tags are comma separated, for example: 'key value, foo bar'. Keys or values with spaces use double quotes; quote the whole flag for your shell (bash/zsh/PowerShell: --tags '\"Cost Center\":\"Finance Team\"'; Windows cmd.exe: --tags \"\"\"Cost Center\"\":\"\"Finance Team\"\"\"). Empty values (key:) are supported for ROSA Classic only, not for Hosted Control Plane.")
	})
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

// normalizeClusterUpdateInput clears pointer flag fields that were pre-allocated by
// cobra but not actually set by the user. This allows pathbind.Expand to distinguish
// between "not set" (nil) and "explicitly set to zero".
func normalizeClusterUpdateInput(cmd *cobra.Command, input *ClusterUpdateInput) {
	if cmd == nil {
		return
	}

	if cmd.Flags().Lookup("delete-protection") != nil && !cmd.Flag("delete-protection").Changed &&
		input.DeleteProtection != nil {
		input.DeleteProtection = nil
	}
	if cmd.Flags().Lookup("container-log-max-files") != nil && !cmd.Flag("container-log-max-files").Changed &&
		input.ContainerLogMaxFiles != nil {
		input.ContainerLogMaxFiles = nil
	}
	if cmd.Flags().Lookup("image-gc-high-threshold-percent") != nil && !cmd.Flag("image-gc-high-threshold-percent").Changed &&
		input.ImageGCHighThresholdPercent != nil {
		input.ImageGCHighThresholdPercent = nil
	}
	if cmd.Flags().Lookup("image-gc-low-threshold-percent") != nil && !cmd.Flag("image-gc-low-threshold-percent").Changed &&
		input.ImageGCLowThresholdPercent != nil {
		input.ImageGCLowThresholdPercent = nil
	}
	if cmd.Flags().Lookup("max-pods") != nil && !cmd.Flag("max-pods").Changed &&
		input.MaxPods != nil {
		input.MaxPods = nil
	}
	if cmd.Flags().Lookup("pod-pids-limit") != nil && !cmd.Flag("pod-pids-limit").Changed &&
		input.PodPidsLimit != nil {
		input.PodPidsLimit = nil
	}
	if cmd.Flags().Lookup("registry-burst") != nil && !cmd.Flag("registry-burst").Changed &&
		input.RegistryBurst != nil {
		input.RegistryBurst = nil
	}
	if cmd.Flags().Lookup("registry-pull-qps") != nil && !cmd.Flag("registry-pull-qps").Changed &&
		input.RegistryPullQPS != nil {
		input.RegistryPullQPS = nil
	}
	if cmd.Flags().Lookup("serialize-image-pulls") != nil && !cmd.Flag("serialize-image-pulls").Changed &&
		input.SerializeImagePulls != nil {
		input.SerializeImagePulls = nil
	}
	if cmd.Flags().Lookup("multi-arch") != nil && !cmd.Flag("multi-arch").Changed &&
		input.MultiArch != nil {
		input.MultiArch = nil
	}
}

// overlayClusterUpdateInput starts from the current values and applies only
// fields explicitly supplied by the caller or computed by PostExpand. An
// explicitly empty value is allowed only for resettable fields.
func overlayClusterUpdateInput(cmd *cobra.Command, before, input, expanded *ClusterUpdateInput) (ClusterUpdateInput, error) {
	after := *before
	if flagChanged(cmd, "additional-trust-bundle") && pathbind.IsUnset(input.AdditionalTrustBundle) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.additionalTrustBundle")
	} else if !pathbind.IsUnset(expanded.AdditionalTrustBundle) {
		after.AdditionalTrustBundle = expanded.AdditionalTrustBundle
	} else if !pathbind.IsUnset(input.AdditionalTrustBundle) {
		after.AdditionalTrustBundle = input.AdditionalTrustBundle
	}
	if flagChanged(cmd, "delete-protection") && pathbind.IsUnset(input.DeleteProtection) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.deleteProtection")
	} else if !pathbind.IsUnset(expanded.DeleteProtection) {
		after.DeleteProtection = expanded.DeleteProtection
	} else if !pathbind.IsUnset(input.DeleteProtection) {
		after.DeleteProtection = input.DeleteProtection
	}
	if flagChanged(cmd, "expiration-time") && pathbind.IsUnset(input.ExpirationTimestamp) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.expirationTimestamp")
	} else if !pathbind.IsUnset(expanded.ExpirationTimestamp) {
		after.ExpirationTimestamp = expanded.ExpirationTimestamp
	} else if !pathbind.IsUnset(input.ExpirationTimestamp) {
		after.ExpirationTimestamp = input.ExpirationTimestamp
	}
	if flagChanged(cmd, "role-arn") && pathbind.IsUnset(input.RoleARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.autoNode.provisionerConfig.karpenter.aws.roleARN")
	} else if !pathbind.IsUnset(expanded.RoleARN) {
		after.RoleARN = expanded.RoleARN
	} else if !pathbind.IsUnset(input.RoleARN) {
		after.RoleARN = input.RoleARN
	}
	if flagChanged(cmd, "platform") && pathbind.IsUnset(input.Platform) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.autoNode.provisionerConfig.karpenter.platform")
	} else if !pathbind.IsUnset(expanded.Platform) {
		after.Platform = expanded.Platform
	} else if !pathbind.IsUnset(input.Platform) {
		after.Platform = input.Platform
	}
	if flagChanged(cmd, "provisioner-config-name") && pathbind.IsUnset(input.ProvisionerConfigName) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.autoNode.provisionerConfig.name")
	} else if !pathbind.IsUnset(expanded.ProvisionerConfigName) {
		after.ProvisionerConfigName = expanded.ProvisionerConfigName
	} else if !pathbind.IsUnset(input.ProvisionerConfigName) {
		after.ProvisionerConfigName = input.ProvisionerConfigName
	}
	if flagChanged(cmd, "channel") && pathbind.IsUnset(input.Channel) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.channel")
	} else if !pathbind.IsUnset(expanded.Channel) {
		after.Channel = expanded.Channel
	} else if !pathbind.IsUnset(input.Channel) {
		after.Channel = input.Channel
	}
	if flagChanged(cmd, "") && pathbind.IsUnset(input.OidcProviders) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.authentication.oidcProviders")
	} else if !pathbind.IsUnset(expanded.OidcProviders) {
		after.OidcProviders = expanded.OidcProviders
	} else if !pathbind.IsUnset(input.OidcProviders) {
		after.OidcProviders = input.OidcProviders
	}
	if flagChanged(cmd, "authentication-type") && pathbind.IsUnset(input.AuthenticationType) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.authentication.type")
	} else if !pathbind.IsUnset(expanded.AuthenticationType) {
		after.AuthenticationType = expanded.AuthenticationType
	} else if !pathbind.IsUnset(input.AuthenticationType) {
		after.AuthenticationType = input.AuthenticationType
	}
	if flagChanged(cmd, "component-routes") && pathbind.IsUnset(input.ComponentRoutes) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.ingress.componentRoutes")
	} else if !pathbind.IsUnset(expanded.ComponentRoutes) {
		after.ComponentRoutes = expanded.ComponentRoutes
	} else if !pathbind.IsUnset(input.ComponentRoutes) {
		after.ComponentRoutes = input.ComponentRoutes
	}
	if flagChanged(cmd, "container-log-max-files") && pathbind.IsUnset(input.ContainerLogMaxFiles) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.containerLogMaxFiles")
	} else if !pathbind.IsUnset(expanded.ContainerLogMaxFiles) {
		after.ContainerLogMaxFiles = expanded.ContainerLogMaxFiles
	} else if !pathbind.IsUnset(input.ContainerLogMaxFiles) {
		after.ContainerLogMaxFiles = input.ContainerLogMaxFiles
	}
	if flagChanged(cmd, "container-log-max-size") && pathbind.IsUnset(input.ContainerLogMaxSize) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.containerLogMaxSize")
	} else if !pathbind.IsUnset(expanded.ContainerLogMaxSize) {
		after.ContainerLogMaxSize = expanded.ContainerLogMaxSize
	} else if !pathbind.IsUnset(input.ContainerLogMaxSize) {
		after.ContainerLogMaxSize = input.ContainerLogMaxSize
	}
	if flagChanged(cmd, "image-gc-high-threshold-percent") && pathbind.IsUnset(input.ImageGCHighThresholdPercent) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.imageGCHighThresholdPercent")
	} else if !pathbind.IsUnset(expanded.ImageGCHighThresholdPercent) {
		after.ImageGCHighThresholdPercent = expanded.ImageGCHighThresholdPercent
	} else if !pathbind.IsUnset(input.ImageGCHighThresholdPercent) {
		after.ImageGCHighThresholdPercent = input.ImageGCHighThresholdPercent
	}
	if flagChanged(cmd, "image-gc-low-threshold-percent") && pathbind.IsUnset(input.ImageGCLowThresholdPercent) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.imageGCLowThresholdPercent")
	} else if !pathbind.IsUnset(expanded.ImageGCLowThresholdPercent) {
		after.ImageGCLowThresholdPercent = expanded.ImageGCLowThresholdPercent
	} else if !pathbind.IsUnset(input.ImageGCLowThresholdPercent) {
		after.ImageGCLowThresholdPercent = input.ImageGCLowThresholdPercent
	}
	if flagChanged(cmd, "image-minimum-gc-age") && pathbind.IsUnset(input.ImageMinimumGCAge) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.imageMinimumGCAge")
	} else if !pathbind.IsUnset(expanded.ImageMinimumGCAge) {
		after.ImageMinimumGCAge = expanded.ImageMinimumGCAge
	} else if !pathbind.IsUnset(input.ImageMinimumGCAge) {
		after.ImageMinimumGCAge = input.ImageMinimumGCAge
	}
	if flagChanged(cmd, "max-pods") && pathbind.IsUnset(input.MaxPods) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.maxPods")
	} else if !pathbind.IsUnset(expanded.MaxPods) {
		after.MaxPods = expanded.MaxPods
	} else if !pathbind.IsUnset(input.MaxPods) {
		after.MaxPods = input.MaxPods
	}
	if flagChanged(cmd, "pod-pids-limit") && pathbind.IsUnset(input.PodPidsLimit) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.podPidsLimit")
	} else if !pathbind.IsUnset(expanded.PodPidsLimit) {
		after.PodPidsLimit = expanded.PodPidsLimit
	} else if !pathbind.IsUnset(input.PodPidsLimit) {
		after.PodPidsLimit = input.PodPidsLimit
	}
	if flagChanged(cmd, "registry-burst") && pathbind.IsUnset(input.RegistryBurst) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.registryBurst")
	} else if !pathbind.IsUnset(expanded.RegistryBurst) {
		after.RegistryBurst = expanded.RegistryBurst
	} else if !pathbind.IsUnset(input.RegistryBurst) {
		after.RegistryBurst = input.RegistryBurst
	}
	if flagChanged(cmd, "registry-pull-qps") && pathbind.IsUnset(input.RegistryPullQPS) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.registryPullQPS")
	} else if !pathbind.IsUnset(expanded.RegistryPullQPS) {
		after.RegistryPullQPS = expanded.RegistryPullQPS
	} else if !pathbind.IsUnset(input.RegistryPullQPS) {
		after.RegistryPullQPS = input.RegistryPullQPS
	}
	if flagChanged(cmd, "serialize-image-pulls") && pathbind.IsUnset(input.SerializeImagePulls) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.serializeImagePulls")
	} else if !pathbind.IsUnset(expanded.SerializeImagePulls) {
		after.SerializeImagePulls = expanded.SerializeImagePulls
	} else if !pathbind.IsUnset(input.SerializeImagePulls) {
		after.SerializeImagePulls = input.SerializeImagePulls
	}
	if flagChanged(cmd, "streaming-connection-idle-timeout") && pathbind.IsUnset(input.StreamingConnectionIdleTimeout) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.kubelet.streamingConnectionIdleTimeout")
	} else if !pathbind.IsUnset(expanded.StreamingConnectionIdleTimeout) {
		after.StreamingConnectionIdleTimeout = expanded.StreamingConnectionIdleTimeout
	} else if !pathbind.IsUnset(input.StreamingConnectionIdleTimeout) {
		after.StreamingConnectionIdleTimeout = input.StreamingConnectionIdleTimeout
	}
	if flagChanged(cmd, "http-proxy") && pathbind.IsUnset(input.HttpProxy) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.proxy.httpProxy")
	} else if !pathbind.IsUnset(expanded.HttpProxy) {
		after.HttpProxy = expanded.HttpProxy
	} else if !pathbind.IsUnset(input.HttpProxy) {
		after.HttpProxy = input.HttpProxy
	}
	if flagChanged(cmd, "https-proxy") && pathbind.IsUnset(input.HttpsProxy) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.proxy.httpsProxy")
	} else if !pathbind.IsUnset(expanded.HttpsProxy) {
		after.HttpsProxy = expanded.HttpsProxy
	} else if !pathbind.IsUnset(input.HttpsProxy) {
		after.HttpsProxy = input.HttpsProxy
	}
	if flagChanged(cmd, "no-proxy") && pathbind.IsUnset(input.NoProxy) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.proxy.noProxy")
	} else if !pathbind.IsUnset(expanded.NoProxy) {
		after.NoProxy = expanded.NoProxy
	} else if !pathbind.IsUnset(input.NoProxy) {
		after.NoProxy = input.NoProxy
	}
	if flagChanged(cmd, "scheduler-profile") && pathbind.IsUnset(input.SchedulerProfile) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.configuration.scheduler.profile")
	} else if !pathbind.IsUnset(expanded.SchedulerProfile) {
		after.SchedulerProfile = expanded.SchedulerProfile
	} else if !pathbind.IsUnset(input.SchedulerProfile) {
		after.SchedulerProfile = input.SchedulerProfile
	}
	if flagChanged(cmd, "image-content-sources") && pathbind.IsUnset(input.ImageContentSources) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.imageContentSources")
	} else if !pathbind.IsUnset(expanded.ImageContentSources) {
		after.ImageContentSources = expanded.ImageContentSources
	} else if !pathbind.IsUnset(input.ImageContentSources) {
		after.ImageContentSources = input.ImageContentSources
	}
	if flagChanged(cmd, "additional-allowed-principals") && pathbind.IsUnset(input.AdditionalAllowedPrincipals) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.additionalAllowedPrincipals")
	} else if !pathbind.IsUnset(expanded.AdditionalAllowedPrincipals) {
		after.AdditionalAllowedPrincipals = expanded.AdditionalAllowedPrincipals
	} else if !pathbind.IsUnset(input.AdditionalAllowedPrincipals) {
		after.AdditionalAllowedPrincipals = input.AdditionalAllowedPrincipals
	}
	if flagChanged(cmd, "filters") && pathbind.IsUnset(input.Filters) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.cloudProviderConfig.subnet.filters")
	} else if !pathbind.IsUnset(expanded.Filters) {
		after.Filters = expanded.Filters
	} else if !pathbind.IsUnset(input.Filters) {
		after.Filters = input.Filters
	}
	if flagChanged(cmd, "endpoint-access") && pathbind.IsUnset(input.EndpointAccess) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.endpointAccess")
	} else if !pathbind.IsUnset(expanded.EndpointAccess) {
		after.EndpointAccess = expanded.EndpointAccess
	} else if !pathbind.IsUnset(input.EndpointAccess) {
		after.EndpointAccess = input.EndpointAccess
	}
	if flagChanged(cmd, "multi-arch") && pathbind.IsUnset(input.MultiArch) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.multiArch")
	} else if !pathbind.IsUnset(expanded.MultiArch) {
		after.MultiArch = expanded.MultiArch
	} else if !pathbind.IsUnset(input.MultiArch) {
		after.MultiArch = input.MultiArch
	}
	if flagChanged(cmd, "control-plane-operator-arn") && pathbind.IsUnset(input.ControlPlaneOperatorARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.rolesRef.controlPlaneOperatorARN")
	} else if !pathbind.IsUnset(expanded.ControlPlaneOperatorARN) {
		after.ControlPlaneOperatorARN = expanded.ControlPlaneOperatorARN
	} else if !pathbind.IsUnset(input.ControlPlaneOperatorARN) {
		after.ControlPlaneOperatorARN = input.ControlPlaneOperatorARN
	}
	if flagChanged(cmd, "image-registry-arn") && pathbind.IsUnset(input.ImageRegistryARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.rolesRef.imageRegistryARN")
	} else if !pathbind.IsUnset(expanded.ImageRegistryARN) {
		after.ImageRegistryARN = expanded.ImageRegistryARN
	} else if !pathbind.IsUnset(input.ImageRegistryARN) {
		after.ImageRegistryARN = input.ImageRegistryARN
	}
	if flagChanged(cmd, "aws-roles-ref-ingress-arn") && pathbind.IsUnset(input.AwsRolesRefIngressARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.rolesRef.ingressARN")
	} else if !pathbind.IsUnset(expanded.AwsRolesRefIngressARN) {
		after.AwsRolesRefIngressARN = expanded.AwsRolesRefIngressARN
	} else if !pathbind.IsUnset(input.AwsRolesRefIngressARN) {
		after.AwsRolesRefIngressARN = input.AwsRolesRefIngressARN
	}
	if flagChanged(cmd, "kube-cloud-controller-arn") && pathbind.IsUnset(input.KubeCloudControllerARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.rolesRef.kubeCloudControllerARN")
	} else if !pathbind.IsUnset(expanded.KubeCloudControllerARN) {
		after.KubeCloudControllerARN = expanded.KubeCloudControllerARN
	} else if !pathbind.IsUnset(input.KubeCloudControllerARN) {
		after.KubeCloudControllerARN = input.KubeCloudControllerARN
	}
	if flagChanged(cmd, "network-arn") && pathbind.IsUnset(input.NetworkARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.rolesRef.networkARN")
	} else if !pathbind.IsUnset(expanded.NetworkARN) {
		after.NetworkARN = expanded.NetworkARN
	} else if !pathbind.IsUnset(input.NetworkARN) {
		after.NetworkARN = input.NetworkARN
	}
	if flagChanged(cmd, "node-pool-management-arn") && pathbind.IsUnset(input.NodePoolManagementARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.rolesRef.nodePoolManagementARN")
	} else if !pathbind.IsUnset(expanded.NodePoolManagementARN) {
		after.NodePoolManagementARN = expanded.NodePoolManagementARN
	} else if !pathbind.IsUnset(input.NodePoolManagementARN) {
		after.NodePoolManagementARN = input.NodePoolManagementARN
	}
	if flagChanged(cmd, "storage-arn") && pathbind.IsUnset(input.StorageARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.rolesRef.storageARN")
	} else if !pathbind.IsUnset(expanded.StorageARN) {
		after.StorageARN = expanded.StorageARN
	} else if !pathbind.IsUnset(input.StorageARN) {
		after.StorageARN = input.StorageARN
	}
	if flagChanged(cmd, "service-endpoints") && pathbind.IsUnset(input.ServiceEndpoints) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.serviceEndpoints")
	} else if !pathbind.IsUnset(expanded.ServiceEndpoints) {
		after.ServiceEndpoints = expanded.ServiceEndpoints
	} else if !pathbind.IsUnset(input.ServiceEndpoints) {
		after.ServiceEndpoints = input.ServiceEndpoints
	}
	if flagChanged(cmd, "local-zone-id") && pathbind.IsUnset(input.LocalZoneID) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.sharedVPC.localZoneID")
	} else if !pathbind.IsUnset(expanded.LocalZoneID) {
		after.LocalZoneID = expanded.LocalZoneID
	} else if !pathbind.IsUnset(input.LocalZoneID) {
		after.LocalZoneID = input.LocalZoneID
	}
	if flagChanged(cmd, "control-plane-arn") && pathbind.IsUnset(input.ControlPlaneARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.sharedVPC.rolesRef.controlPlaneARN")
	} else if !pathbind.IsUnset(expanded.ControlPlaneARN) {
		after.ControlPlaneARN = expanded.ControlPlaneARN
	} else if !pathbind.IsUnset(input.ControlPlaneARN) {
		after.ControlPlaneARN = input.ControlPlaneARN
	}
	if flagChanged(cmd, "shared-vpc-roles-ref-ingress-arn") && pathbind.IsUnset(input.SharedVPCRolesRefIngressARN) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.sharedVPC.rolesRef.ingressARN")
	} else if !pathbind.IsUnset(expanded.SharedVPCRolesRefIngressARN) {
		after.SharedVPCRolesRefIngressARN = expanded.SharedVPCRolesRefIngressARN
	} else if !pathbind.IsUnset(input.SharedVPCRolesRefIngressARN) {
		after.SharedVPCRolesRefIngressARN = input.SharedVPCRolesRefIngressARN
	}
	if flagChanged(cmd, "termination-handler-queue-url") && pathbind.IsUnset(input.TerminationHandlerQueueURL) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.hostedCluster.platform.aws.terminationHandlerQueueURL")
	} else if !pathbind.IsUnset(expanded.TerminationHandlerQueueURL) {
		after.TerminationHandlerQueueURL = expanded.TerminationHandlerQueueURL
	} else if !pathbind.IsUnset(input.TerminationHandlerQueueURL) {
		after.TerminationHandlerQueueURL = input.TerminationHandlerQueueURL
	}
	if flagChanged(cmd, "properties") && pathbind.IsUnset(input.Properties) {
		return after, fmt.Errorf("field %s cannot be reset", "spec.properties")
	} else if !pathbind.IsUnset(expanded.Properties) {
		after.Properties = expanded.Properties
	} else if !pathbind.IsUnset(input.Properties) {
		after.Properties = input.Properties
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

func buildClusterUpdateDocument(ctx context.Context, cmd *cobra.Command, current *v1alpha1.Cluster, input *ClusterUpdateInput, obj *v1alpha1.Cluster) (map[string]any, error) {
	var before, expanded ClusterUpdateInput
	if err := pathbind.Flatten(ctx, current, &before); err != nil {
		return nil, err
	}
	if err := pathbind.Flatten(ctx, obj, &expanded); err != nil {
		return nil, err
	}
	after, err := overlayClusterUpdateInput(cmd, &before, input, &expanded)
	if err != nil {
		return nil, err
	}
	return pathbind.UpdateDocument(ctx, before, after, obj)
}

// RunUpdateCluster executes the update dispatch. Generated; do not edit.
// name is the account-scoped metadata.name used to address the resource in the Platform API.
func RunUpdateCluster(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, name string, input *ClusterUpdateInput, h ClusterUpdateHandler) error {
	if interactive.Enabled() {
		if err := h.Prompt(ctx, r, cmd, input); err != nil {
			return err
		}
	}
	if err := h.PreRequest(ctx, r, input); err != nil {
		return err
	}
	normalizeClusterUpdateInput(cmd, input)
	current, err := r.SeaplaneClient.SeaplaneV1alpha1().Clusters().Get(ctx, name, platform.GetOptions{})
	if err != nil {
		return err
	}
	obj := &v1alpha1.Cluster{}
	if err := pathbind.Expand(ctx, *input, obj); err != nil {
		return err
	}
	obj.Name = name
	if err := h.PostExpand(ctx, r, input, obj); err != nil {
		return err
	}
	document, err := buildClusterUpdateDocument(ctx, cmd, current, input, obj)
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
	updated, err := r.SeaplaneClient.SeaplaneV1alpha1().Clusters().Patch(ctx, name, types.MergePatchType, payload, platform.PatchOptions{})
	if err != nil {
		return err
	}
	return h.PostResponse(ctx, r, updated)
}
