package profilehandler

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/common"
	ClusterConfigure "github.com/openshift/rosa/tests/utils/config"
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
	var clusterConfiguration = new(ClusterConfigure.ClusterConfig)
	var userData = new(UserData)
	defer func() {

		// Record userdata
		marshaledUserData, err := json.Marshal(userData)
		if err != nil {
			log.Logger.Errorf("Cannot marshal user data : %s", err.Error())
			panic(fmt.Errorf("cannot marshal user data : %s", err.Error()))
		}
		_, err = common.CreateFileWithContent(config.Test.UserDataFile, string(marshaledUserData))
		if err != nil {
			log.Logger.Errorf("Cannot record user data: %s", err.Error())
			panic(fmt.Errorf("cannot record user data: %s", err.Error()))
		}

		// Record cluster configuration
		marshaledClusterConfig, err := json.Marshal(clusterConfiguration)
		if err != nil {
			log.Logger.Errorf("Cannot marshal cluster configuration: %s", err.Error())
			panic(fmt.Errorf("cannot marshal cluster configuration: %s", err.Error()))
		}
		_, err = common.CreateFileWithContent(config.Test.ClusterConfigFile, string(marshaledClusterConfig))
		if err != nil {
			log.Logger.Errorf("Cannot record cluster configuration: %s", err.Error())
			panic(fmt.Errorf("cannot record cluster configuration: %s", err.Error()))
		}
	}()
	flags = []string{
		"--cluster-name", profile.NamePrefix,
	}
	clusterConfiguration.Name = profile.NamePrefix

	if profile.Version != "" {
		version, err := PrepareVersion(client, profile.Version, profile.ChannelGroup, profile.ClusterConfig.HCP)
		if err != nil {
			return flags, err
		}
		profile.Version = version.Version
		flags = append(flags, "--version", version.Version)

		clusterConfiguration.Version = &ClusterConfigure.Version{
			ChannelGroup: profile.ChannelGroup,
			RawID:        version.Version,
		}
	}
	if profile.ChannelGroup != "" {
		flags = append(flags, "--channel-group", profile.ChannelGroup)
		if clusterConfiguration.Version == nil {
			clusterConfiguration.Version = &ClusterConfigure.Version{}
		}
		clusterConfiguration.Version.ChannelGroup = profile.ChannelGroup
	}
	if profile.Region != "" {
		flags = append(flags, "--region", profile.Region)
		clusterConfiguration.Region = profile.Region
	}
	if profile.ClusterConfig.LongName {
		clusterConfiguration.DomainPrefix = "long-name"
		flags = append(flags,
			"--domain-prefix", clusterConfiguration.DomainPrefix,
		)

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
		userData.AccountRolesPrefix = profile.NamePrefix
		flags = append(flags,
			"--role-arn", accRoles.InstallerRole,
			"--support-role-arn", accRoles.SupportRole,
			"--worker-iam-role", accRoles.WorkerRole,
		)
		clusterConfiguration.Sts = true
		clusterConfiguration.Aws = &ClusterConfigure.AWS{
			Sts: ClusterConfigure.Sts{
				RoleArn:        accRoles.InstallerRole,
				SupportRoleArn: accRoles.SupportRole,
				WorkerRoleArn:  accRoles.WorkerRole,
			},
		}
		if !profile.ClusterConfig.HCP {
			flags = append(flags,
				"--controlplane-iam-role", accRoles.ControlPlaneRole,
			)
			clusterConfiguration.Aws.Sts.ControlPlaneRoleArn = accRoles.ControlPlaneRole
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
			clusterConfiguration.Aws.Sts.OidcConfigID = oidcConfigID
			userData.OIDCConfigID = oidcConfigID
		}
		flags = append(flags, "--operator-roles-prefix", profile.NamePrefix)
		clusterConfiguration.Aws.Sts.OperatorRolesPrefix = profile.NamePrefix
		userData.OperatorRolesPrefix = profile.NamePrefix
	}
	if profile.ClusterConfig.AdditionalSGNumber != 0 {
		PrepareSecurityGroupsDummy("", profile.Region, profile.ClusterConfig.AdditionalSGNumber)
	}
	if profile.ClusterConfig.AdminEnabled {
		PrepareAdminUserDummy()
	}
	if profile.ClusterConfig.AuditLogForward {
		PrepareAuditlogDummy()
		clusterConfiguration.AuditLogArn = ""
	}
	if profile.ClusterConfig.Autoscale {
		minReplicas := "3"
		maxRelicas := "6"
		flags = append(flags,
			"--enable-autoscaling",
			"--min-replicas", minReplicas,
			"--max-replicas", maxRelicas,
		)
		clusterConfiguration.Autoscaling = &ClusterConfigure.Autoscaling{
			Enabled: true,
		}
		clusterConfiguration.Nodes = &ClusterConfigure.Nodes{
			MinReplicas: minReplicas,
			MaxReplicas: maxRelicas,
		}
	}
	if profile.ClusterConfig.WorkerPoolReplicas != 0 {
		flags = append(flags, "--replicas", fmt.Sprintf("%v", profile.ClusterConfig.WorkerPoolReplicas))
		clusterConfiguration.Nodes = &ClusterConfigure.Nodes{
			Replicas: fmt.Sprintf("%v", profile.ClusterConfig.WorkerPoolReplicas),
		}
	}

	if profile.ClusterConfig.IngressCustomized {
		clusterConfiguration.IngressConfig = &ClusterConfigure.IngressConfig{
			DefaultIngressRouteSelector:            "app1=test1,app2=test2",
			DefaultIngressExcludedNamespaces:       "test-ns1,test-ns2",
			DefaultIngressWildcardPolicy:           "WildcardsDisallowed",
			DefaultIngressNamespaceOwnershipPolicy: "Strict",
		}
		flags = append(flags,
			"--default-ingress-route-selector", clusterConfiguration.IngressConfig.DefaultIngressRouteSelector,
			"--default-ingress-excluded-namespaces", clusterConfiguration.IngressConfig.DefaultIngressExcludedNamespaces,
			"--default-ingress-wildcard-policy", clusterConfiguration.IngressConfig.DefaultIngressWildcardPolicy,
			"--default-ingress-namespace-ownership-policy", clusterConfiguration.IngressConfig.DefaultIngressNamespaceOwnershipPolicy,
		)
	}
	if profile.ClusterConfig.AutoscalerEnabled {
		autoscaler := &ClusterConfigure.Autoscaler{
			AutoscalerBalanceSimilarNodeGroups:    true,
			AutoscalerSkipNodesWithLocalStorage:   true,
			AutoscalerLogVerbosity:                "4",
			AutoscalerMaxPodGracePeriod:           "0",
			AutoscalerPodPriorityThreshold:        "0",
			AutoscalerIgnoreDaemonsetsUtilization: true,
			AutoscalerMaxNodeProvisionTime:        "10m",
			AutoscalerBalancingIgnoredLabels:      "aaa",
			AutoscalerMaxNodesTotal:               "1000",
			AutoscalerMinCores:                    "0",
			AutoscalerMaxCores:                    "100",
			AutoscalerMinMemory:                   "0",
			AutoscalerMaxMemory:                   "4096",
			// AutoscalerGpuLimit:                      "1",
			AutoscalerScaleDownEnabled:              true,
			AutoscalerScaleDownUtilizationThreshold: "0.5",
			AutoscalerScaleDownDelayAfterAdd:        "10s",
			AutoscalerScaleDownDelayAfterDelete:     "10s",
			AutoscalerScaleDownDelayAfterFailure:    "10s",
			// AutoscalerScaleDownUnneededTime:         "3m",
		}
		flags = append(flags,
			"--autoscaler-balance-similar-node-groups",
			"--autoscaler-skip-nodes-with-local-storage",
			"--autoscaler-log-verbosity", autoscaler.AutoscalerLogVerbosity,
			"--autoscaler-max-pod-grace-period", autoscaler.AutoscalerMaxPodGracePeriod,
			"--autoscaler-pod-priority-threshold", autoscaler.AutoscalerPodPriorityThreshold,
			"--autoscaler-ignore-daemonsets-utilization",
			"--autoscaler-max-node-provision-time", autoscaler.AutoscalerMaxNodeProvisionTime,
			"--autoscaler-balancing-ignored-labels", autoscaler.AutoscalerBalancingIgnoredLabels,
			"--autoscaler-max-nodes-total", autoscaler.AutoscalerMaxNodesTotal,
			"--autoscaler-min-cores", autoscaler.AutoscalerMinCores,
			"--autoscaler-max-cores", autoscaler.AutoscalerMaxCores,
			"--autoscaler-min-memory", autoscaler.AutoscalerMinMemory,
			"--autoscaler-max-memory", autoscaler.AutoscalerMaxMemory,
			// "--autoscaler-gpu-limit", autoscaler.AutoscalerGpuLimit,
			"--autoscaler-scale-down-enabled",
			// "--autoscaler-scale-down-unneeded-time", autoscaler.AutoscalerScaleDownUnneededTime,
			"--autoscaler-scale-down-utilization-threshold", autoscaler.AutoscalerScaleDownUtilizationThreshold,
			"--autoscaler-scale-down-delay-after-add", autoscaler.AutoscalerScaleDownDelayAfterAdd,
			"--autoscaler-scale-down-delay-after-delete", autoscaler.AutoscalerScaleDownDelayAfterDelete,
			"--autoscaler-scale-down-delay-after-failure", autoscaler.AutoscalerScaleDownDelayAfterFailure,
		)

		clusterConfiguration.Autoscaler = autoscaler
	}
	if profile.ClusterConfig.BYOVPC {
		PrepareSubnetsDummy("", profile.Region, "")
	}
	if profile.ClusterConfig.BillingAccount != "" {
		flags = append(flags, " --billing-account", profile.ClusterConfig.BillingAccount)
		clusterConfiguration.BillingAccount = profile.ClusterConfig.BillingAccount
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
		clusterConfiguration.Ec2MetadataHttpTokens = profile.ClusterConfig.Ec2MetadataHttpTokens
	}
	if profile.ClusterConfig.EtcdEncryption {
		flags = append(flags, "--etcd-encryption")
		clusterConfiguration.EtcdEncryption = profile.ClusterConfig.EtcdEncryption

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
		clusterConfiguration.Encryption = &ClusterConfigure.Encryption{
			KmsKeyArn: "", // placeHolder
		}
	}
	if profile.ClusterConfig.LabelEnabled {
		dmpLabel := "test-label/openshift.io=,test-label=testvalue"
		flags = append(flags, "--worker-mp-labels", dmpLabel)
		clusterConfiguration.DefaultMpLabels = dmpLabel
	}
	if profile.ClusterConfig.MultiAZ {
		flags = append(flags, "--multi-az")
		clusterConfiguration.MultiAZ = profile.ClusterConfig.MultiAZ
	}
	if profile.ClusterConfig.NetWorkingSet {
		networking := &ClusterConfigure.Networking{
			MachineCIDR: "10.2.0.0/16",
			PodCIDR:     "192.168.0.0/18",
			ServiceCIDR: "172.31.0.0/24",
			HostPrefix:  "25",
		}
		flags = append(flags,
			"--machine-cidr", networking.MachineCIDR, // Placeholder, it should be vpc CIDR
			"--service-cidr", networking.ServiceCIDR,
			"--pod-cidr", networking.PodCIDR,
			"--host-prefix", networking.HostPrefix,
		)
		clusterConfiguration.Networking = networking
	}
	if profile.ClusterConfig.Private {
		flags = append(flags, "--private")
		clusterConfiguration.Private = profile.ClusterConfig.Private
	}
	if profile.ClusterConfig.PrivateLink {
		flags = append(flags, "--private-link")
		clusterConfiguration.PrivateLink = profile.ClusterConfig.PrivateLink
	}
	if profile.ClusterConfig.ProvisionShard != "" {
		flags = append(flags, "--properties", fmt.Sprintf("provision_shard_id:%s", profile.ClusterConfig.ProvisionShard))
		clusterConfiguration.Properties = &ClusterConfigure.Properties{
			ProvisionShardID: profile.ClusterConfig.ProvisionShard,
		}
	}
	if profile.ClusterConfig.ProxyEnabled {
		PrepareProxysDummy("", profile.Region, "")

		clusterConfiguration.Proxy = &ClusterConfigure.Proxy{
			Enabled: profile.ClusterConfig.ProxyEnabled,
		}

	}
	if profile.ClusterConfig.SharedVPC {
		//Placeholder for shared vpc, need to research what to be set here
	}
	if profile.ClusterConfig.TagEnabled {
		tags := "test-tag:tagvalue,qe-managed:true"
		flags = append(flags, "--tags", tags)
		clusterConfiguration.Tags = tags
	}
	if profile.ClusterConfig.VolumeSize != 0 {
		diskSize := fmt.Sprintf("%dGiB", profile.ClusterConfig.VolumeSize)
		flags = append(flags, "--worker-disk-size", diskSize)
		clusterConfiguration.WorkerDiskSize = diskSize
	}
	if profile.ClusterConfig.Zones != "" && !profile.ClusterConfig.BYOVPC {
		flags = append(flags, " --availability-zones", profile.ClusterConfig.Zones)
		clusterConfiguration.AvailabilityZones = profile.ClusterConfig.Zones
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
