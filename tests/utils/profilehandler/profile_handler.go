package profilehandler

import (
	"fmt"
	"os"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
)

var client rosacli.Client

func init() {
	client = *rosacli.NewClient()
}

func GetYAMLProfilesDir() string {
	return config.Test.YAMLProfilesDir
}
func LoadProfileYamlFile(profileName string) *Profile {
	p := GetProfile(profileName, GetYAMLProfilesDir())
	log.Logger.Infof("Loaded cluster profile configuration from origional profile %s : %v", profileName, *p)
	log.Logger.Infof("Loaded cluster profile configuration from origional cluster %s : %v", profileName, *p.ClusterConfig)
	log.Logger.Infof("Loaded cluster profile configuration from origional account-roles %s : %v", profileName, *p.AccountRoleConfig)
	return p
}

func LoadProfileYamlFileByENV() *Profile {
	if config.Test.TestProfile == "" {
		panic(fmt.Errorf("ENV Variable TEST_PROFILE is empty, please make sure you set the env value"))
	}
	profile := LoadProfileYamlFile(config.Test.TestProfile)

	// Supporting global env setting to overrite profile settings
	if os.Getenv("CHANNEL_GROUP") != "" {
		log.Logger.Infof("Got global env settings for CHANNEL_GROUP, overwritten the profile setting with value %s", os.Getenv("CHANNEL_GROUP"))
		profile.ChannelGroup = os.Getenv("CHANNEL_GROUP")
	}
	if os.Getenv("VERSION") != "" {
		log.Logger.Infof("Got global env settings for VERSION, overwritten the profile setting with value %s", os.Getenv("VERSION"))
		profile.Version = os.Getenv("VERSION")
	}
	if os.Getenv("REGION") != "" {
		log.Logger.Infof("Got global env settings for REGION, overwritten the profile setting with value %s", os.Getenv("REGION"))
		profile.Region = os.Getenv("REGION")
	}
	if os.Getenv("PROVISION_SHARD") != "" {
		log.Logger.Infof("Got global env settings for PROVISION_SHARD, overwritten the profile setting with value %s", os.Getenv("PROVISION_SHARD"))
		profile.ClusterConfig.ProvisionShard = os.Getenv("PROVISION_SHARD")
	}
	// Generate a name prefix for the profile CI run
	profile.NamePrefix = "xuelitest"

	return profile
}

// GenerateClusterCreateFlags will generate flags
func GenerateClusterCreateFlags(profile *Profile, client *rosacli.Client) (flags []string, err error) {
	flags = []string{
		"--cluster-name", profile.NamePrefix,
	}
	if profile.Version != "" {
		version, err := PrepareVersion(client, profile.Version, profile.ChannelGroup, profile.ClusterConfig.HCP)
		if err != nil {
			return flags, err
		}
		profile.Version = version.Version
		flags = append(flags, "--version", version.Version)
	}
	if profile.ChannelGroup != "" {
		flags = append(flags, "--channel-group", profile.ChannelGroup)
	}
	if profile.Region != "" {
		flags = append(flags, "--region", profile.Region)
	}
	if profile.ClusterConfig.STS {
		var accRoles *rosacli.AccountRolesUnit
		accRoles, err = PrepareAccountRoles(
			client, profile.NamePrefix,
			profile.ClusterConfig.HCP,
			profile.Version,
			profile.ChannelGroup,
			profile.AccountRoleConfig.Path,
			profile.AccountRoleConfig.PermissionBoundary,
		)
		if err != nil {
			return flags, err
		}
		flags = append(flags,
			"--role-arn", accRoles.InstallerRole,
			"--support-role-arn", accRoles.SupportRole,
			"--worker-iam-role", accRoles.WorkerRole,
		)
		if !profile.ClusterConfig.HCP {
			flags = append(flags,
				"--controlplane-iam-role", accRoles.ControlPlaneRole,
			)
		}
		if profile.ClusterConfig.OIDCConfig != "" {
			var oidcConfigID string
			oidcConfigID, err = PrepareOIDCConfig(client, profile.ClusterConfig.OIDCConfig,
				profile.Region, accRoles.InstallerRole, profile.NamePrefix)
			if err != nil {
				return flags, err
			}
			err = PrepareOIDCProvider(client, oidcConfigID)
			if err != nil {
				return
			}
			err = PrepareOperatorRolesByOIDCConfig(client, profile.NamePrefix,
				oidcConfigID, accRoles.InstallerRole, "", profile.ClusterConfig.HCP)
			if err != nil {
				return flags, err
			}
			flags = append(flags, "--oidc-config-id", oidcConfigID)
		}
		flags = append(flags, "--operator-roles-prefix", profile.NamePrefix)
	}
	if profile.ClusterConfig.AdditionalSGNumber != 0 {
		PrepareSecurityGroupsDummy("", profile.Region, profile.ClusterConfig.AdditionalSGNumber)
	}
	if profile.ClusterConfig.AdminEnabled {
		PrepareAdminUserDummy()
	}
	if profile.ClusterConfig.AuditLogForward {
		PrepareAuditlogDummy()
	}
	if profile.ClusterConfig.Autoscale {
		minReplicas := "3"
		maxRelicas := "6"
		flags = append(flags,
			"--enable-autoscaling",
			"--min-replicas", minReplicas,
			"--max-replicas", maxRelicas,
		)
	}

	if profile.ClusterConfig.AutoscalerEnabled {
		PrepareAutoscalerDummy()
	}
	if profile.ClusterConfig.BYOVPC {
		PrepareSubnetsDummy("", profile.Region, "")
	}
	if profile.ClusterConfig.BillingAccount != "" {
		flags = append(flags, " --billing-account", profile.ClusterConfig.BillingAccount)
	}
	if profile.ClusterConfig.DisableSCPChecks {
		flags = append(flags, "--disable-scp-checks")
	}
	if profile.ClusterConfig.DisableUserWorKloadMonitoring {
		flags = append(flags, "--disable-workload-monitoring")
	}
	if profile.ClusterConfig.ETCDKMS {
		PrepareKMSKeyDummy(profile.Region)
	}
	if profile.ClusterConfig.Ec2MetadataHttpTokens != "" {
		flags = append(flags, "--ec2-metadata-http-tokens", profile.ClusterConfig.Ec2MetadataHttpTokens)
	}
	if profile.ClusterConfig.EtcdEncryption {
		flags = append(flags, "--etcd-encryption")
	}
	if profile.ClusterConfig.ExternalAuthConfig {
		PrepareExternalAuthConfigDummy()
	}

	if profile.ClusterConfig.FIPS {
		flags = append(flags, "--fips")
	}
	if profile.ClusterConfig.HCP {
		flags = append(flags, "--hosted-cp")
	}
	if profile.ClusterConfig.InstanceType != "" {
		flags = append(flags, "--compute-machine-type", profile.ClusterConfig.InstanceType)
	}
	if profile.ClusterConfig.KMSKey {
		PrepareKMSKeyDummy(profile.Region)
	}
	if profile.ClusterConfig.LabelEnabled {
		flags = append(flags, "--worker-mp-labels", "test-label/openshift.io=,test-label=testvalue")
	}
	if profile.ClusterConfig.MultiAZ {
		flags = append(flags, "--multi-az")
	}
	if profile.ClusterConfig.NetWorkingSet {
		flags = append(flags,
			"--machine-cidr", "$PLACEHOLDER_VPC_CIDR",
			"--service-cidr", "172.31.0.0/24",
			"--pod-cidr", "192.168.0.0/18",
			"--host-prefix", "25",
		)
	}
	if profile.ClusterConfig.Private {
		flags = append(flags, "--private")
	}
	if profile.ClusterConfig.PrivateLink {
		flags = append(flags, "--private-link")
	}
	if profile.ClusterConfig.ProvisionShard != "" {
		flags = append(flags, "--properties", fmt.Sprintf("provision_shard_id:%s", profile.ClusterConfig.ProvisionShard))
	}
	if profile.ClusterConfig.ProxyEnabled {
		PrepareProxysDummy("", profile.Region, "")
	}
	if profile.ClusterConfig.SharedVPC {
		//Placeholder for shared vpc, need to research what to be set here
	}
	if profile.ClusterConfig.TagEnabled {
		flags = append(flags, "--tags", "test-tag:tagvalue,qe-managed:true")
	}
	if profile.ClusterConfig.VolumeSize != 0 {
		flags = append(flags, "--worker-disk-size", fmt.Sprintf("%dGiB", profile.ClusterConfig.VolumeSize))
	}
	if profile.ClusterConfig.Zones != "" && !profile.ClusterConfig.BYOVPC {
		flags = append(flags, " --availability-zones", profile.ClusterConfig.Zones)
	}

	return flags, nil
}

func CreateClusterByProfile(profile *Profile, client *rosacli.Client) (*rosacli.ClusterDescription, error) {
	flags, err := GenerateClusterCreateFlags(profile, client)
	if err != nil {
		return nil, err
	}
	_, err = client.Cluster.Create(profile.NamePrefix, flags...)
	if err != nil {
		return nil, err
	}
	output, err := client.Cluster.DescribeCluster(profile.NamePrefix)
	if err != nil {
		return nil, err
	}
	description, err := client.Cluster.ReflectClusterDescription(output)
	// Need to do the post step when cluster has no oidcconfig enabled
	if profile.ClusterConfig.OIDCConfig == "" {
		err = PrepareOIDCProviderByCluster(client, description.ID)
		if err != nil {
			return description, err
		}
		err = PrepareOperatorRolesByCluster(client, description.ID)
		if err != nil {
			return description, err
		}

	}
	return description, err
}
