package profilehandler

type Profile struct {
	Name              string             `yaml:"as,omitempty"`
	Version           string             `yaml:"version,omitempty"`
	ChannelGroup      string             `yaml:"channel_group,omitempty"`
	Region            string             `yaml:"region,omitempty"`
	NamePrefix        string             `yaml:"name_prefix,omitempty"`
	ClusterConfig     *ClusterConfig     `yaml:"cluster,omitempty"`
	AccountRoleConfig *AccountRoleConfig `yaml:"account-role,omitempty"`
}
type AccountRoleConfig struct {
	Path               string `yaml:"path,omitempty"`
	PermissionBoundary string `yaml:"permission_boundary,omitempty"`
}
type ClusterConfig struct {
	InstanceType                  string `yaml:"instance_type,omitempty" json:"instance_type,omitempty"`
	Zones                         string `yaml:"zones,omitempty" json:"zones,omitempty"`
	OIDCConfig                    string `yaml:"oidc_config,omitempty" json:"oidc_config,omitempty"`
	ProvisionShard                string `yaml:"provision_shard,omitempty" json:"provision_shard,omitempty"`
	Ec2MetadataHttpTokens         string `yaml:"imdsv2,omitempty" json:"imdsv2,omitempty"`
	BillingAccount                string `yaml:"billing_account,omitempty" json:"billing_account,omitempty"`
	ExpirationTime                int    `yaml:"expiration_time,omitempty" json:"expiration_time,omitempty"`
	VolumeSize                    int    `yaml:"volume_size,omitempty" json:"volume_size,omitempty"`
	AdditionalSGNumber            int    `yaml:"additional_sg_number,omitempty" json:"additional_sg_number,omitempty"`
	WorkerPoolReplicas            int    `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	TagEnabled                    bool   `yaml:"tag_enabled,omitempty" json:"tag_enabled,omitempty"`
	LabelEnabled                  bool   `yaml:"label_enabled,omitempty" json:"label_enabled,omitempty"`
	EtcdEncryption                bool   `yaml:"etcd_encryption,omitempty" json:"etcd_encryption,omitempty"`
	FIPS                          bool   `yaml:"fips,omitempty" json:"fips,omitempty"`
	STS                           bool   `yaml:"sts,omitempty" json:"sts,omitempty"`
	Autoscale                     bool   `yaml:"autoscale,omitempty" json:"autoscale,omitempty"`
	MultiAZ                       bool   `yaml:"multi_az,omitempty" json:"multi_az,omitempty"`
	BYOVPC                        bool   `yaml:"byo_vpc,omitempty" json:"byo_vpc,omitempty"`
	PrivateLink                   bool   `yaml:"private_link,omitempty" json:"private_link,omitempty"`
	Private                       bool   `yaml:"private,omitempty" json:"private,omitempty"`
	KMSKey                        bool   `yaml:"kms_key,omitempty" json:"kms_key,omitempty"`
	ETCDKMS                       bool   `yaml:"etcd_kms,omitempty" json:"etcd_kms,omitempty"`
	NetWorkingSet                 bool   `yaml:"networking,omitempty" json:"networking,omitempty"`
	ProxyEnabled                  bool   `yaml:"proxy_enabled,omitempty" json:"proxy_enabled,omitempty"`
	HCP                           bool   `yaml:"hcp,omitempty" json:"hypershift,omitempty"`
	AuditLogForward               bool   `yaml:"auditlog_forward,omitempty" json:"auditlog_forward,omitempty"`
	AdminEnabled                  bool   `yaml:"admin_enabled,omitempty" json:"admin_enabled,omitempty"`
	AutoscalerEnabled             bool   `yaml:"autoscaler_enabled,omitempty" json:"autoscaler_enabled,omitempty"`
	DisableUserWorKloadMonitoring bool   `yaml:"disable_uwm,omitempty" json:"disable_uwm,omitempty"`
	SharedVPC                     bool   `yaml:"shared_vpc,omitempty" json:"shared_vpc,omitempty"`
	DisableSCPChecks              bool   `yaml:"disable_scp_checks,omitempty" json:"disable_scp_checks,omitempty"`
	ExternalAuthConfig            bool   `yaml:"external_auth_config,omitempty" json:"external_auth_config,omitempty"`
	IngressCustomized             bool   `yaml:"ingress_customized,omitempty" json:"ingress_customized,omitempty"`
	LongName                      bool   `yaml:"long_name,omitempty" json:"long_name,omitempty"`
}

type UserData struct {
	VpcID               string `json:"vpc_id,omitempty"`
	AccountRolesPrefix  string `json:"account_roles_prefix,omitempty"`
	OperatorRolesPrefix string `json:"operator_roles_prefix,omitempty"`
	OIDCConfigID        string `json:"oidc_config_id,omitempty"`
	KMSKey              string `json:"kms_key,omitempty"`
	AuditLogArn         string `json:"audit_log,omitempty"`
}
