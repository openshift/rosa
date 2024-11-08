package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openshift/rosa/tests/ci/config"
	. "github.com/openshift/rosa/tests/utils/log"
)

type Version struct {
	ChannelGroup string `json:"channel_group,omitempty"`
	RawID        string `json:"raw_id,omitempty"`
}

type Encryption struct {
	KmsKeyArn            string `json:"kms_key_arn,omitempty"`
	EtcdEncryptionKmsArn string `json:"etcd_encryption_kms_arn,omitempty"`
}

type Properties struct {
	ProvisionShardID string `json:"provision_shard_id,omitempty"`
	ZeroEgress       bool   `json:"zero_egress,omitempty"`
}

type Sts struct {
	RoleArn             string `json:"role_arn,omitempty"`
	SupportRoleArn      string `json:"support_role_arn,omitempty"`
	WorkerRoleArn       string `json:"worker_role_arn,omitempty"`
	ControlPlaneRoleArn string `json:"control_plane_role_arn,omitempty"`
	OidcConfigID        string `json:"oidc_config_id,omitempty"`
	OperatorRolesPrefix string `json:"operator_roles_prefix,omitempty"`
}

type AWS struct {
	Sts Sts `json:"sts,omitempty"`
}

type Proxy struct {
	Enabled         bool   `json:"enabled,omitempty"`
	Http            string `json:"http,omitempty"`
	Https           string `json:"https,omitempty"`
	NoProxy         string `json:"no_proxy,omitempty"`
	TrustBundleFile string `json:"trust_bundle_file,omitempty"`
}

type Subnets struct {
	PrivateSubnetIds string `json:"private_subnet_ids,omitempty"`
	PublicSubnetIds  string `json:"public_subnet_ids,omitempty"`
}

type Nodes struct {
	Replicas            string `json:"replicas,omitempty"`
	MinReplicas         string `json:"min_replicas,omitempty"`
	MaxReplicas         string `json:"max_replicas,omitempty"`
	ComputeInstanceType string `json:"compute_instance_type,omitempty"`
}

type Autoscaling struct {
	Enabled bool `json:"enabled,omitempty"`
}
type Autoscaler struct {
	AutoscalerBalanceSimilarNodeGroups      bool   `json:"autoscaler_balance_similar_node_groups,omitempty"`
	AutoscalerSkipNodesWithLocalStorage     bool   `json:"autoscaler_skip_nodes_with_local_storage,omitempty"`
	AutoscalerLogVerbosity                  string `json:"autoscaler_log_verbosity,omitempty"`
	AutoscalerMaxPodGracePeriod             string `json:"autoscaler_max_pod_grace_period,omitempty"`
	AutoscalerPodPriorityThreshold          string `json:"autoscaler_pod_priority_threshold,omitempty"`
	AutoscalerIgnoreDaemonsetsUtilization   bool   `json:"autoscaler_ignore_daemonsets_utilization,omitempty"`
	AutoscalerMaxNodeProvisionTime          string `json:"autoscaler_max_node_provision_time,omitempty"`
	AutoscalerBalancingIgnoredLabels        string `json:"autoscaler_balancing_ignored_labels,omitempty"`
	AutoscalerMaxNodesTotal                 string `json:"autoscaler_max_nodes_total,omitempty"`
	AutoscalerMinCores                      string `json:"autoscaler_min_cores,omitempty"`
	AutoscalerMaxCores                      string `json:"autoscaler_max_cores,omitempty"`
	AutoscalerMinMemory                     string `json:"autoscaler_min_memory,omitempty"`
	AutoscalerMaxMemory                     string `json:"autoscaler_max_memory,omitempty"`
	AutoscalerGpuLimit                      string `json:"autoscaler_gpu_limit,omitempty"`
	AutoscalerScaleDownEnabled              bool   `json:"autoscaler_scale_down_enabled,omitempty"`
	AutoscalerScaleDownUnneededTime         string `json:"autoscaler_scale_down_unneeded_time,omitempty"`
	AutoscalerScaleDownUtilizationThreshold string `json:"autoscaler_scale_down_utilization_threshold,omitempty"`
	AutoscalerScaleDownDelayAfterAdd        string `json:"autoscaler_scale_down_delay_after_add,omitempty"`
	AutoscalerScaleDownDelayAfterDelete     string `json:"autoscaler_scale_down_delay_after_delete,omitempty"`
	AutoscalerScaleDownDelayAfterFailure    string `json:"autoscaler_scale_down_delay_after_failure,omitempty"`
}
type IngressConfig struct {
	DefaultIngressRouteSelector            string `json:"default_ingress_route_sector,omitempty"`
	DefaultIngressExcludedNamespaces       string `json:"default_ingress_excluded_namespaces,omitempty"`
	DefaultIngressWildcardPolicy           string `json:"default_ingress_wildcard_policy,omitempty"`
	DefaultIngressNamespaceOwnershipPolicy string `json:"default_ingress_namespace_ownership_policy,omitempty"`
}
type Networking struct {
	Type        string `json:"type,omitempty"`
	MachineCIDR string `json:"machine_cidr,omitempty"`
	ServiceCIDR string `json:"service_cidr,omitempty"`
	PodCIDR     string `json:"pod_cidr,omitempty"`
	HostPrefix  string `json:"host_prefix,omitempty"`
}
type AdditionalSecurityGroups struct {
	ControlPlaneSecurityGroups string `json:"control_plane_sgs,omitempty"`
	InfraSecurityGroups        string `json:"infra_sgs,omitempty"`
	WorkerSecurityGroups       string `json:"worker_sgs,omitempty"`
}
type RegistryConfig struct {
	AllowedRegistries          string   `json:"allowed_registries,omitempty"`
	RegistryAdditionalTrustCA  string   `json:"reristry_additional_trust_ca,omitempty"`
	AllowedRegistriesForImport []string `json:"allowed_registries_for_import,omitempty"`
}

type ClusterConfig struct {
	DisableScpChecks          bool                      `json:"disable_scp_checks,omitempty"`
	DisableWorkloadMonitoring bool                      `json:"disable_workload_monitoring,omitempty"`
	EnableCustomerManagedKey  bool                      `json:"enable_customer_managed_key,omitempty"`
	EtcdEncryption            bool                      `json:"etcd_encryption,omitempty"`
	Fips                      bool                      `json:"fips,omitempty"`
	Hypershift                bool                      `json:"hypershift,omitempty"`
	MultiAZ                   bool                      `json:"multi_az,omitempty"`
	Private                   bool                      `json:"private,omitempty"`
	PrivateLink               bool                      `json:"private_link,omitempty"`
	Sts                       bool                      `json:"sts,omitempty"`
	AuditLogArn               string                    `json:"audit_log_arn,omitempty"`
	AvailabilityZones         string                    `json:"availability_zones,omitempty"`
	DefaultMpLabels           string                    `json:"default_mp_labels,omitempty"`
	Ec2MetadataHttpTokens     string                    `json:"ec2_metadata_http_tokens,omitempty"`
	Name                      string                    `json:"name,omitempty"`
	Region                    string                    `json:"region,omitempty"`
	Tags                      string                    `json:"tags,omitempty"`
	WorkerDiskSize            string                    `json:"worker_disk_size,omitempty"`
	DomainPrefix              string                    `json:"domain_prefix,omitempty"`
	BillingAccount            string                    `json:"billing_account,omitempty"`
	AdditionalPrincipals      string                    `json:"additional_principals,omitempty"`
	AdditionalSecurityGroups  *AdditionalSecurityGroups `json:"additional_sgs,omitempty"`
	Autoscaling               *Autoscaling              `json:"autoscaling,omitempty"`
	Aws                       *AWS                      `json:"aws,omitempty"`
	Autoscaler                *Autoscaler               `json:"autoscaler,omitempty"`
	Encryption                *Encryption               `json:"encryption,omitempty"`
	IngressConfig             *IngressConfig            `json:"ingress_config,omitempty"`
	Networking                *Networking               `json:"networking,omitempty"`
	Nodes                     *Nodes                    `json:"nodes,omitempty"`
	Properties                *Properties               `json:"properties,omitempty"`
	Proxy                     *Proxy                    `json:"proxy,omitempty"`
	Subnets                   *Subnets                  `json:"subnets,omitempty"`
	Version                   *Version                  `json:"version,omitempty"`
	ExternalAuthentication    bool                      `json:"external_authentication,omitempty"`
	SharedVPC                 bool                      `json:"shared_vpc,omitempty"`
	RegistryConfig            bool                      `json:"registry_config,omitempty"`
}

func ParseClusterProfile() (*ClusterConfig, error) {
	filePath := config.Test.ClusterConfigFile
	// Load the JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file: %v", err)
	}

	// Parse the JSON data into the ClusterConfig struct
	var config ClusterConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON data: %v", err)
	}

	return &config, nil
}

func GetClusterID() (clusterID string) {
	clusterID = getClusterIDENVExisted()
	if clusterID != "" {
		return
	}

	if _, err := os.Stat(config.Test.ClusterIDFile); err != nil {
		Logger.Errorf("Cluster detail file not existing")
		return ""
	}
	fileCont, _ := os.ReadFile(config.Test.ClusterIDFile)
	clusterID = string(fileCont)
	return
}

// Get the clusterID env.
func getClusterIDENVExisted() string {
	return os.Getenv("CLUSTER_ID")
}

// IsNodePoolGlobalCheck Get the nodepool global check flag
func IsNodePoolGlobalCheck() bool {
	nodePoolGlobalCheck := os.Getenv("CLUSTER_NODE_POOL_GLOBAL_CHECK")
	if nodePoolGlobalCheck == "true" {
		return true
	} else {
		return false
	}
}
