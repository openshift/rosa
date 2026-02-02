package handler

// Profile will map the profile settings from the profile yaml file
type Profile struct {
	ChannelGroup      string             `yaml:"channel_group,omitempty"`
	Name              string             `yaml:"as,omitempty"`
	NamePrefix        string             `yaml:"name_prefix,omitempty"`
	Region            string             `yaml:"region,omitempty"`
	Version           string             `yaml:"version,omitempty"`
	AccountRoleConfig *AccountRoleConfig `yaml:"account-role,omitempty"`
	ClusterConfig     *ClusterConfig     `yaml:"cluster,omitempty"`
	Day2Config        *Day2Config        `yaml:"day2-conf,omitempty"`
}

// Day2Config will map the configuration of day2-conf from profile settings
type Day2Config struct {
	LocalZoneMP       bool `yaml:"local-zone-mp,omitempty"`
	TuningConfig      bool `yaml:"tuning-config,omitempty"`
	TuningConfigMP    bool `yaml:"tuning-config-mp,omitempty"`
	ClusterAutoScaler bool `yaml:"cluster-autoscaler,omitempty"`
}

// AccountRoleConfig will map the configuration of account roles from profile settings
type AccountRoleConfig struct {
	Path               string `yaml:"path,omitempty"`
	PermissionBoundary string `yaml:"permission_boundary,omitempty"`
}

// ClusterConfig will map the clsuter configuration from profile settings
type ClusterConfig struct {
	BillingAccount                string `yaml:"billing_account,omitempty" json:"billing_account,omitempty"`
	Ec2MetadataHttpTokens         string `yaml:"imdsv2,omitempty" json:"imdsv2,omitempty"`
	InstanceType                  string `yaml:"instance_type,omitempty" json:"instance_type,omitempty"`
	Name                          string `yaml:"name,omitempty" json:"name,omitempty"`
	OIDCConfig                    string `yaml:"oidc_config,omitempty" json:"oidc_config,omitempty"`
	ProvisionShard                string `yaml:"provision_shard,omitempty" json:"provision_shard,omitempty"`
	Zones                         string `yaml:"zones,omitempty" json:"zones,omitempty"`
	AdditionalSGNumber            int    `yaml:"additional_sg_number,omitempty" json:"additional_sg_number,omitempty"`
	ExpirationTime                int    `yaml:"expiration_time,omitempty" json:"expiration_time,omitempty"`
	NameLength                    int    `default:"15" yaml:"name_length,omitempty" json:"name_length,omitempty"`
	VolumeSize                    int    `yaml:"volume_size,omitempty" json:"volume_size,omitempty"`
	WorkerPoolReplicas            int    `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	AdditionalPrincipals          bool   `yaml:"additional_principals,omitempty" json:"additional_principals,omitempty"`
	AdminEnabled                  bool   `yaml:"admin_enabled,omitempty" json:"admin_enabled,omitempty"`
	AuditLogForward               bool   `yaml:"auditlog_forward,omitempty" json:"auditlog_forward,omitempty"`
	Autoscale                     bool   `yaml:"autoscale,omitempty" json:"autoscale,omitempty"`
	AutoscalerEnabled             bool   `yaml:"autoscaler_enabled,omitempty" json:"autoscaler_enabled,omitempty"`
	BYOVPC                        bool   `yaml:"byo_vpc,omitempty" json:"byo_vpc,omitempty"`
	DomainPrefixEnabled           bool   `yaml:"domain_prefix_enabled,omitempty" json:"domain_prefix_enabled,omitempty"`
	DisableUserWorKloadMonitoring bool   `yaml:"disable_uwm,omitempty" json:"disable_uwm,omitempty"`
	DisableSCPChecks              bool   `yaml:"disable_scp_checks,omitempty" json:"disable_scp_checks,omitempty"`
	ExternalAuthConfig            bool   `yaml:"external_auth_config,omitempty" json:"external_auth_config,omitempty"`
	EtcdEncryption                bool   `yaml:"etcd_encryption,omitempty" json:"etcd_encryption,omitempty"`
	EtcdKMS                       bool   `yaml:"etcd_kms,omitempty" json:"etcd_kms,omitempty"`
	FIPS                          bool   `yaml:"fips,omitempty" json:"fips,omitempty"`
	HCP                           bool   `yaml:"hcp,omitempty" json:"hypershift,omitempty"`
	IngressCustomized             bool   `yaml:"ingress_customized,omitempty" json:"ingress_customized,omitempty"`
	KMSKey                        bool   `yaml:"kms_key,omitempty" json:"kms_key,omitempty"`
	LabelEnabled                  bool   `yaml:"label_enabled,omitempty" json:"label_enabled,omitempty"`
	MultiAZ                       bool   `yaml:"multi_az,omitempty" json:"multi_az,omitempty"`
	NetworkingSet                 bool   `yaml:"networking,omitempty" json:"networking,omitempty"`
	PrivateLink                   bool   `yaml:"private_link,omitempty" json:"private_link,omitempty"`
	DefaultIngressPrivate         bool   `yaml:"default_ingress_private,omitempty"`
	Private                       bool   `yaml:"private,omitempty" json:"private,omitempty"`
	ProxyEnabled                  bool   `yaml:"proxy_enabled,omitempty" json:"proxy_enabled,omitempty"`
	ProxyType                     string `yaml:"proxy_type,omitempty" json:"proxy_type,omitempty"`
	STS                           bool   `yaml:"sts,omitempty" json:"sts,omitempty"`
	SharedVPC                     bool   `yaml:"shared_vpc,omitempty" json:"shared_vpc,omitempty"`
	TagEnabled                    bool   `yaml:"tag_enabled,omitempty" json:"tag_enabled,omitempty"`
	NetworkType                   string `yaml:"network_type,omitempty" json:"network_type,omitempty"`
	RegistriesConfig              bool   `yaml:"registries_config" json:"registries_config,omitempty"`
	AllowedRegistries             bool   `yaml:"allowed_registries" json:"allowed_registries,omitempty"`
	BlockedRegistries             bool   `yaml:"blocked_registries" json:"blocked_registries,omitempty"`
	ManualCreationMode            bool   `yaml:"manual_creation_mode" json:"manual_creation_mode,omitempty"`
	FedRAMP                       bool   `yaml:"fedramp" json:"fedramp,omitempty"`
	ZeroEgress                    bool   `yaml:"zero_egress" json:"zero_egress,omitempty"`
	UseLocalCredentials           bool   `yaml:"use_local_credentials,omitempty" json:"use_local_credentials,omitempty"`
	Add_UnManaged_Tag             bool   `yaml:"add_unmanaged_tag" json:"add_unmanaged_tag,omitempty"`
	LogForward                    bool   `yaml:"log_forward,omitempty" json:"log_forward,omitempty"`
}

// Resources will record the resources prepared
type Resources struct {
	ClusterID                    string                `json:"cluster_id,omitempty"`
	AccountRolesPrefix           string                `json:"account_roles_prefix,omitempty"`
	AdditionalPrincipals         string                `json:"additional_principals,omitempty"`
	AuditLogArn                  string                `json:"audit_log,omitempty"`
	DNSDomain                    string                `json:"dns_domain,omitempty"`
	EtcdKMSKey                   string                `json:"etcd_kms_key,omitempty"`
	FromSharedAWSAccount         *FromSharedAWSAccount `json:"from_shared_aws_account,omitempty"`
	IngressHostedZoneID          string                `json:"ingress_hosted_zone_id,omitempty"`
	HostedCPInternalHostedZoneID string                `json:"hostedcp_internal_hosted_zone_id,omitempty"`
	KMSKey                       string                `json:"kms_key,omitempty"`
	OIDCConfigID                 string                `json:"oidc_config_id,omitempty"`
	OIDCProviderID               string                `json:"oidc_provider_id,omitempty"`
	OperatorRolesPrefix          string                `json:"operator_roles_prefix,omitempty"`
	Region                       string                `json:"region,omitempty"`
	ResourceShareArn             string                `json:"resource_share,omitempty"`
	SharedVPCRole                string                `json:"shared_vpc_role,omitempty"`
	VpcID                        string                `json:"vpc_id,omitempty"`
	HCPRoute53ShareRole          string                `json:"hcp_route53_share_role,omitempty"`
	HCPVPCEndpointShareRole      string                `json:"hcp_vpc_endpoint_share_role,omitempty"`
	ProxyInstanceID              string                `json:"proxy_instance_id,omitempty"`
	LogForwardConigs             *LogForwardConigs     `json:"lfw_configs,omitempty" yaml:"lfw_configs,omitempty"`
	OCMRoleArn                   string                `json:"ocm_role_arn,omitempty"`
	UserRoleArn                  string                `json:"user_role_arn,omitempty"`
}

type FromSharedAWSAccount struct {
	VPC                 bool `json:"vpc,omitempty"`
	AdditionalPrincipls bool `json:"additional_principals,omitempty"`
}

type LogForwardConigs struct {
	Cloudwatch *CloudWatchLogForward `json:"cloudwatch,omitempty" yaml:"cloudwatch,omitempty"`
	S3         *S3LogForward         `json:"s3,omitempty" yaml:"s3,omitempty"`
}
type S3LogForward struct {
	Applications         []string `json:"applications,omitempty" yaml:"applications,omitempty"`
	Groups               []string `json:"groups,omitempty" yaml:"groups,omitempty"`
	S3ConfigBucketName   string   `json:"s3_config_bucket_name,omitempty" yaml:"s3_config_bucket_name,omitempty"`
	S3ConfigBucketPrefix string   `json:"s3_config_bucket_prefix,omitempty" yaml:"s3_config_bucket_prefix,omitempty"`
}

type CloudWatchLogForward struct {
	CloudwatchLogRoleArn   string   `json:"cloudwatch_log_role_arn,omitempty" yaml:"cloudwatch_log_role_arn,omitempty"`
	CloudwatchLogGroupName string   `json:"cloudwatch_log_group_name,omitempty" yaml:"cloudwatch_log_group_name,omitempty"`
	Applications           []string `json:"applications,omitempty" yaml:"applications,omitempty"`
	Groups                 []string `json:"groups,omitempty" yaml:"groups,omitempty"`
}

// ClusterDetail will record basic cluster info to support other team's testing
type ClusterDetail struct {
	APIURL           string   `json:"api_url,omitempty"`
	ClusterID        string   `json:"cluster_id,omitempty"`
	ClusterName      string   `json:"cluster_name,omitempty"`
	ClusterType      string   `json:"cluster_type,omitempty"`
	ConsoleURL       string   `json:"console_url,omitempty"`
	InfraID          string   `json:"infra_id,omitempty"`
	OIDCEndpointURL  string   `json:"oidc_endpoint_url,omitempty"`
	OperatorRoleArns []string `json:"operator_role_arn,omitempty"`
}

type ProxyDetail struct {
	HTTPsProxy       string
	HTTPProxy        string
	CABundleFilePath string
	NoProxy          string
	Username         string
	Password         string
	InstanceID       string
}
