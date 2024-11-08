package profilehandler

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/tests/ci/config"
	ClusterConfigure "github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

func GetYAMLProfilesDir() string {
	return config.Test.YAMLProfilesDir
}
func LoadProfileYamlFile(profileName string) *Profile {
	p := GetProfile(profileName, GetYAMLProfilesDir())
	log.Logger.Infof("Loaded cluster profile configuration from original profile %s : %v", profileName, *p)
	log.Logger.Infof("Loaded cluster profile configuration from original cluster %s : %v", profileName, *p.ClusterConfig)
	if p.AccountRoleConfig != nil {
		log.Logger.Infof("Loaded cluster profile configuration from original account-roles %s : %v",
			profileName, *p.AccountRoleConfig)
	}
	return p
}

func LoadProfileYamlFileByENV() *Profile {
	if config.Test.TestProfile == "" {
		panic(fmt.Errorf("ENV Variable TEST_PROFILE is empty, please make sure you set the env value"))
	}
	profile := LoadProfileYamlFile(config.Test.TestProfile)

	// Supporting global env setting to overrite profile settings
	if config.Test.GlobalENV.ChannelGroup != "" {
		log.Logger.Infof("Got global env settings for CHANNEL_GROUP, overwritten the profile setting with value %s",
			config.Test.GlobalENV.ChannelGroup)
		profile.ChannelGroup = config.Test.GlobalENV.ChannelGroup
	}
	if config.Test.GlobalENV.Version != "" {
		log.Logger.Infof("Got global env settings for VERSION, overwritten the profile setting with value %s",
			config.Test.GlobalENV.Version)
		profile.Version = config.Test.GlobalENV.Version
	}
	if config.Test.GlobalENV.Region != "" {
		log.Logger.Infof("Got global env settings for REGION, overwritten the profile setting with value %s",
			config.Test.GlobalENV.Region)
		profile.Region = config.Test.GlobalENV.Region
	}
	if config.Test.GlobalENV.ProvisionShard != "" {
		log.Logger.Infof("Got global env settings for PROVISION_SHARD, overwritten the profile setting with value %s",
			config.Test.GlobalENV.ProvisionShard)
		profile.ClusterConfig.ProvisionShard = config.Test.GlobalENV.ProvisionShard
	}
	if config.Test.GlobalENV.NamePrefix != "" {
		log.Logger.Infof("Got global env settings for NAME_PREFIX, overwritten the profile setting with value %s",
			config.Test.GlobalENV.NamePrefix)
		profile.NamePrefix = config.Test.GlobalENV.NamePrefix
	}

	if config.Test.ClusterENV.ComputeMachineType != "" {
		log.Logger.Infof("Got global env settings for COMPUTE_MACHINE_TYPE, overwritten the profile setting with value %s",
			config.Test.ClusterENV.ComputeMachineType)
		profile.ClusterConfig.InstanceType = config.Test.ClusterENV.ComputeMachineType
	}
	if config.Test.ClusterENV.BYOVPC != "" {
		log.Logger.Infof("Got global env settings for BYOVPC, overwritten the profile setting with value %s",
			config.Test.ClusterENV.BYOVPC)

		profile.ClusterConfig.BYOVPC = helper.ParseBool(config.Test.ClusterENV.BYOVPC)
	}
	if config.Test.ClusterENV.Private != "" {
		log.Logger.Infof("Got global env settings for PRIVATE, overwritten the profile setting with value %s",
			config.Test.ClusterENV.Private)
		profile.ClusterConfig.Private = helper.ParseBool(config.Test.ClusterENV.Private)
	}
	if config.Test.ClusterENV.Autoscale != "" {
		log.Logger.Infof("Got global env settings for AUTOSCALE, overwritten the profile setting with value %s",
			config.Test.ClusterENV.Autoscale)
		profile.ClusterConfig.Autoscale = helper.ParseBool(config.Test.ClusterENV.Autoscale)
	}
	if config.Test.ClusterENV.ProxyEnabled != "" {
		log.Logger.Infof("Got global env settings for PROXY_ENABLED, overwritten the profile setting with value %s",
			config.Test.ClusterENV.ProxyEnabled)
		profile.ClusterConfig.ProxyEnabled = helper.ParseBool(config.Test.ClusterENV.ProxyEnabled)
	}
	if config.Test.ClusterENV.FipsEnabled != "" {
		log.Logger.Infof("Got global env settings for FIPS_ENABLED, overwritten the profile setting with value %s",
			config.Test.ClusterENV.FipsEnabled)
		profile.ClusterConfig.FIPS = helper.ParseBool(config.Test.ClusterENV.FipsEnabled)
	}
	if config.Test.ClusterENV.MultiAZ != "" {
		log.Logger.Infof("Got global env settings for MULTI_AZ, overwritten the profile setting with value %s",
			config.Test.ClusterENV.MultiAZ)
		profile.ClusterConfig.MultiAZ = helper.ParseBool(config.Test.ClusterENV.MultiAZ)
	}
	if config.Test.ClusterENV.VolumeSize != "" {
		log.Logger.Infof("Got global env settings for VOLUME_SIZE, overwritten the profile setting with value %s",
			config.Test.ClusterENV.VolumeSize)
		profile.ClusterConfig.VolumeSize = helper.ParseInt(config.Test.ClusterENV.VolumeSize)
	}
	if config.Test.ClusterENV.Replicas != "" {
		log.Logger.Infof("Got global env settings for REPLICAS, overwritten the profile setting with value %s",
			config.Test.ClusterENV.Replicas)
		profile.ClusterConfig.WorkerPoolReplicas = helper.ParseInt(config.Test.ClusterENV.Replicas)
	}
	if config.Test.ClusterENV.AllowRegistries != "" {
		log.Logger.Infof("Got global env settings for ALLOW_REGISTRIES, overwritten the profile setting with value %s",
			config.Test.ClusterENV.AllowRegistries)
		profile.ClusterConfig.AllowedRegistries = helper.ParseBool(config.Test.ClusterENV.AllowRegistries)
	}

	return profile
}

// GenerateAccountRoleCreationFlag will generate account role creation flags
func GenerateAccountRoleCreationFlag(client *rosacli.Client,
	namePrefix string,
	hcp bool,
	openshiftVersion string,
	channelGroup string,
	path string,
	permissionsBoundary string) []string {
	flags := []string{
		"--prefix", namePrefix,
		"--mode", "auto",
		"-y",
	}
	if openshiftVersion != "" {
		majorVersion := helper.SplitMajorVersion(openshiftVersion)
		flags = append(flags, "--version", majorVersion)
	}
	if channelGroup != "" {
		flags = append(flags, "--channel-group", channelGroup)
	}
	if hcp {
		flags = append(flags, "--hosted-cp")
	} else {
		flags = append(flags, "--classic")
	}
	if path != "" {
		flags = append(flags, "--path", path)
	}
	if permissionsBoundary != "" {
		flags = append(flags, "--permissions-boundary", permissionsBoundary)
	}
	return flags

}

// GenerateClusterCreateFlags will generate cluster creation flags
func GenerateClusterCreateFlags(profile *Profile, client *rosacli.Client) ([]string, error) {
	if profile.ClusterConfig.NameLegnth == 0 {
		profile.ClusterConfig.NameLegnth = constants.DefaultNameLength //Set to a default value when it is not set
	}
	if profile.NamePrefix == "" {
		panic("The profile name prefix is empty. Please set with env variable NAME_PREFIX")
	}
	clusterName := PreparePrefix(profile.NamePrefix, profile.ClusterConfig.NameLegnth)
	profile.ClusterConfig.Name = clusterName
	var clusterConfiguration = new(ClusterConfigure.ClusterConfig)
	var userData = new(UserData)
	sharedVPCRoleArn := ""
	sharedVPCRolePrefix := ""
	awsSharedCredentialFile := ""
	envVariableErrMsg := "'SHARED_VPC_AWS_SHARED_CREDENTIALS_FILE' env is not set or empty, it is: %s"
	defer func() {

		// Record userdata
		_, err := helper.CreateFileWithContent(config.Test.UserDataFile, userData)
		if err != nil {
			log.Logger.Errorf("Cannot record user data: %s", err.Error())
			panic(fmt.Errorf("cannot record user data: %s", err.Error()))
		}

		// Record cluster configuration

		_, err = helper.CreateFileWithContent(config.Test.ClusterConfigFile, clusterConfiguration)
		if err != nil {
			log.Logger.Errorf("Cannot record cluster configuration: %s", err.Error())
			panic(fmt.Errorf("cannot record cluster configuration: %s", err.Error()))
		}
	}()
	flags := []string{"-y"}
	clusterConfiguration.Name = clusterName

	if profile.Version != "" {
		// Force set the hcp parameter to false since hcp cannot filter the upgrade versions
		version, err := PrepareVersion(client, profile.Version, profile.ChannelGroup, false)

		if err != nil {
			return flags, err
		}
		if version == nil {
			err = fmt.Errorf("cannot find a version match the condition %s", profile.Version)
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
	if profile.ClusterConfig.DomainPrefixEnabled {
		flags = append(flags,
			"--domain-prefix", helper.TrimNameByLength(clusterName, ocm.MaxClusterDomainPrefixLength),
		)
	}
	if profile.ClusterConfig.STS {
		var accRoles *rosacli.AccountRolesUnit
		var oidcConfigID string
		accountRolePrefix := helper.TrimNameByLength(clusterName, constants.MaxRolePrefixLength)
		log.Logger.Infof("Got sts set to true. Going to prepare Account roles with prefix %s", accountRolePrefix)
		accRoles, err := PrepareAccountRoles(
			client, accountRolePrefix,
			profile.ClusterConfig.HCP,
			profile.Version,
			profile.ChannelGroup,
			profile.AccountRoleConfig.Path,
			profile.AccountRoleConfig.PermissionBoundary,
		)
		if err != nil {
			log.Logger.Errorf("Got error happens when prepare account roles: %s", err.Error())
			return flags, err
		}
		userData.AccountRolesPrefix = accountRolePrefix
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

		if profile.ClusterConfig.SharedVPC {
			awsSharedCredentialFile = config.Test.GlobalENV.SVPC_CREDENTIALS_FILE
			if awsSharedCredentialFile == "" {
				log.Logger.Errorf(envVariableErrMsg, awsSharedCredentialFile)
				panic(fmt.Errorf(envVariableErrMsg, awsSharedCredentialFile))
			}

			sharedVPCRolePrefix = accountRolePrefix
			awsClient, err := aws_client.CreateAWSClient("", profile.Region, awsSharedCredentialFile)
			if err != nil {
				return flags, err
			}
			sharedVPCAccountID := awsClient.AccountID
			sharedVPCRoleArn = fmt.Sprintf("arn:aws:iam::%s:role/%s-shared-vpc-role", sharedVPCAccountID, sharedVPCRolePrefix)
		}

		operatorRolePrefix := accountRolePrefix
		if profile.ClusterConfig.OIDCConfig != "" {
			oidcConfigPrefix := helper.TrimNameByLength(clusterName, constants.MaxOIDCConfigPrefixLength)
			log.Logger.Infof("Got  oidc config setting, going to prepare the %s oidc with prefix %s",
				profile.ClusterConfig.OIDCConfig, oidcConfigPrefix)
			oidcConfigID, err = PrepareOIDCConfig(client, profile.ClusterConfig.OIDCConfig,
				profile.Region, accRoles.InstallerRole, oidcConfigPrefix)
			if err != nil {
				return flags, err
			}
			flags = append(flags, "--oidc-config-id", oidcConfigID)
			clusterConfiguration.Aws.Sts.OidcConfigID = oidcConfigID
			userData.OIDCConfigID = oidcConfigID

			if !profile.ClusterConfig.ManualCreationMode {
				err = PrepareOIDCProvider(client, oidcConfigID)
				if err != nil {
					return flags, err
				}
				err = PrepareOperatorRolesByOIDCConfig(client, operatorRolePrefix,
					oidcConfigID, accRoles.InstallerRole, sharedVPCRoleArn, profile.ClusterConfig.HCP, profile.ChannelGroup)
				if err != nil {
					return flags, err
				}
			}
		}

		flags = append(flags, "--operator-roles-prefix", operatorRolePrefix)
		clusterConfiguration.Aws.Sts.OperatorRolesPrefix = operatorRolePrefix
		userData.OperatorRolesPrefix = operatorRolePrefix

		if profile.ClusterConfig.SharedVPC {
			log.Logger.Info("Got shared vpc settings. Going to sleep 30s to wait for the operator roles prepared")
			time.Sleep(30 * time.Second)
			installRoleArn := accRoles.InstallerRole
			ingressOperatorRoleArn := fmt.Sprintf("%s/%s-%s", strings.Split(installRoleArn, "/")[0],
				sharedVPCRolePrefix, "openshift-ingress-operator-cloud-credentials")
			sharedVPCRoleName, sharedVPCRoleArn, err := PrepareSharedVPCRole(sharedVPCRolePrefix, installRoleArn,
				ingressOperatorRoleArn, profile.Region, awsSharedCredentialFile)
			if err != nil {
				return flags, err
			}
			flags = append(flags, "--shared-vpc-role-arn", sharedVPCRoleArn)
			userData.SharedVPCRole = sharedVPCRoleName
		}

		if profile.ClusterConfig.AuditLogForward {
			auditLogRoleName := accountRolePrefix
			auditRoleArn, err := PrepareAuditlogRoleArnByOIDCConfig(client, auditLogRoleName, oidcConfigID, profile.Region)
			clusterConfiguration.AuditLogArn = auditRoleArn
			userData.AuditLogArn = auditRoleArn
			if err != nil {
				return flags, err
			}
			flags = append(flags,
				"--audit-log-arn", auditRoleArn)
		}

		if profile.ClusterConfig.AdditionalPrincipals {
			awsSharedCredentialFile = config.Test.GlobalENV.SVPC_CREDENTIALS_FILE
			if awsSharedCredentialFile == "" {
				log.Logger.Errorf(envVariableErrMsg, awsSharedCredentialFile)
				panic(fmt.Errorf(envVariableErrMsg, awsSharedCredentialFile))
			}
			installRoleArn := accRoles.InstallerRole
			additionalPrincipalRolePrefix := accountRolePrefix
			additionalPrincipalRoleName := fmt.Sprintf("%s-%s", additionalPrincipalRolePrefix, "additional-principal-role")
			additionalPrincipalRoleArn, err := PrepareAdditionalPrincipalsRole(additionalPrincipalRoleName, installRoleArn,
				profile.Region, awsSharedCredentialFile)
			if err != nil {
				return flags, err
			}
			flags = append(flags, "--additional-allowed-principals", additionalPrincipalRoleArn)
			clusterConfiguration.AdditionalPrincipals = additionalPrincipalRoleArn
			userData.AdditionalPrincipals = additionalPrincipalRoleName
		}
	}

	// Put this part before the BYOVPC preparation so the subnets is prepared based on PrivateLink
	if profile.ClusterConfig.Private {
		flags = append(flags, "--private")
		clusterConfiguration.Private = profile.ClusterConfig.Private
		if profile.ClusterConfig.HCP {
			profile.ClusterConfig.PrivateLink = true
		}
	}

	if profile.ClusterConfig.AdminEnabled {
		// Comment below part due to OCM-7112
		log.Logger.Infof("Day1 admin is enabled. Going to generate the admin user and password and record in %s",
			config.Test.ClusterAdminFile)
		_, password := PrepareAdminUser() // Unuse cluster-admin right now
		userName := "cluster-admin"

		flags = append(flags,
			"--create-admin-user",
			"--cluster-admin-password", password,
			// "--cluster-admin-user", userName,
		)
		helper.CreateFileWithContent(config.Test.ClusterAdminFile, fmt.Sprintf("%s:%s", userName, password))
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
			"--default-ingress-route-selector",
			clusterConfiguration.IngressConfig.DefaultIngressRouteSelector,
			"--default-ingress-excluded-namespaces",
			clusterConfiguration.IngressConfig.DefaultIngressExcludedNamespaces,
			"--default-ingress-wildcard-policy",
			clusterConfiguration.IngressConfig.DefaultIngressWildcardPolicy,
			"--default-ingress-namespace-ownership-policy",
			clusterConfiguration.IngressConfig.DefaultIngressNamespaceOwnershipPolicy,
		)
	}
	if profile.ClusterConfig.AutoscalerEnabled {
		if !profile.ClusterConfig.Autoscale {
			return nil, errors.New("Autoscaler is enabled without having enabled the autoscale field") // nolint
		}
		autoscaler := &ClusterConfigure.Autoscaler{
			AutoscalerBalanceSimilarNodeGroups:    true,
			AutoscalerSkipNodesWithLocalStorage:   true,
			AutoscalerLogVerbosity:                "4",
			AutoscalerMaxPodGracePeriod:           "0",
			AutoscalerPodPriorityThreshold:        "0",
			AutoscalerIgnoreDaemonsetsUtilization: true,
			AutoscalerMaxNodeProvisionTime:        "10m",
			AutoscalerBalancingIgnoredLabels:      "aaa",
			AutoscalerMaxNodesTotal:               "100",
			AutoscalerMinCores:                    "0",
			AutoscalerMaxCores:                    "1000",
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
	if profile.ClusterConfig.NetworkingSet {
		networking := &ClusterConfigure.Networking{
			MachineCIDR: "10.0.0.0/16",
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
	if profile.ClusterConfig.BYOVPC {
		var vpc *vpc_client.VPC
		var err error
		vpcPrefix := helper.TrimNameByLength(clusterName, 20)
		log.Logger.Info("Got BYOVPC set to true. Going to prepare subnets")
		cidrValue := constants.DefaultVPCCIDRValue
		if profile.ClusterConfig.NetworkingSet {
			cidrValue = clusterConfiguration.Networking.MachineCIDR
		}

		if profile.ClusterConfig.SharedVPC {
			vpc, err = PrepareVPC(profile.Region, vpcPrefix, cidrValue, awsSharedCredentialFile)
		} else {
			vpc, err = PrepareVPC(profile.Region, vpcPrefix, cidrValue, "")
		}
		if err != nil {
			return flags, err
		}

		userData.VpcID = vpc.VpcID
		zones := strings.Split(profile.ClusterConfig.Zones, ",")
		zones = helper.RemoveFromStringSlice(zones, "")
		subnets, err := PrepareSubnets(vpc, profile.Region, zones, profile.ClusterConfig.MultiAZ)
		if err != nil {
			return flags, err
		}
		subnetsFlagValue := strings.Join(append(subnets["private"], subnets["public"]...), ",")
		clusterConfiguration.Subnets = &ClusterConfigure.Subnets{
			PrivateSubnetIds: strings.Join(subnets["private"], ","),
			PublicSubnetIds:  strings.Join(subnets["public"], ","),
		}
		if profile.ClusterConfig.PrivateLink {
			log.Logger.Info("Got private link set to true. Only set private subnets to cluster flags")
			subnetsFlagValue = strings.Join(subnets["private"], ",")
			clusterConfiguration.Subnets = &ClusterConfigure.Subnets{
				PrivateSubnetIds: strings.Join(subnets["private"], ","),
			}
		}
		flags = append(flags,
			"--subnet-ids", subnetsFlagValue)

		if profile.ClusterConfig.AdditionalSGNumber != 0 {
			securityGroups, err := PrepareAdditionalSecurityGroups(vpc, profile.ClusterConfig.AdditionalSGNumber, vpcPrefix)
			if err != nil {
				return flags, err
			}
			computeSGs := strings.Join(securityGroups, ",")
			infraSGs := strings.Join(securityGroups, ",")
			controlPlaneSGs := strings.Join(securityGroups, ",")
			if profile.ClusterConfig.HCP {
				flags = append(flags,
					"--additional-compute-security-group-ids", computeSGs,
				)
				clusterConfiguration.AdditionalSecurityGroups = &ClusterConfigure.AdditionalSecurityGroups{
					WorkerSecurityGroups: computeSGs,
				}
			} else {
				flags = append(flags,
					"--additional-infra-security-group-ids", infraSGs,
					"--additional-control-plane-security-group-ids", controlPlaneSGs,
					"--additional-compute-security-group-ids", computeSGs,
				)
				clusterConfiguration.AdditionalSecurityGroups = &ClusterConfigure.AdditionalSecurityGroups{
					ControlPlaneSecurityGroups: controlPlaneSGs,
					InfraSecurityGroups:        infraSGs,
					WorkerSecurityGroups:       computeSGs,
				}
			}
		}
		if profile.ClusterConfig.ProxyEnabled {
			proxyName := vpc.VPCName
			if proxyName == "" {
				proxyName = clusterName
			}
			proxy, err := PrepareProxy(vpc, profile.Region, proxyName, config.Test.OutputDir, config.Test.ProxyCABundleFile)
			if err != nil {
				return flags, err
			}

			clusterConfiguration.Proxy = &ClusterConfigure.Proxy{
				Enabled:         profile.ClusterConfig.ProxyEnabled,
				Http:            proxy.HTTPProxy,
				Https:           proxy.HTTPsProxy,
				NoProxy:         proxy.NoProxy,
				TrustBundleFile: proxy.CABundleFilePath,
			}
			flags = append(flags,
				"--http-proxy", proxy.HTTPProxy,
				"--https-proxy", proxy.HTTPsProxy,
				"--no-proxy", proxy.NoProxy,
				"--additional-trust-bundle-file", proxy.CABundleFilePath,
			)

		}
		if profile.ClusterConfig.SharedVPC {
			subnetArns, err := PrepareSubnetArns(subnetsFlagValue, profile.Region, awsSharedCredentialFile)
			if err != nil {
				return flags, err
			}

			resourceShareName := fmt.Sprintf("%s-%s", sharedVPCRolePrefix, "resource-share")
			resourceShareArn, err := PrepareResourceShare(resourceShareName, subnetArns, profile.Region, awsSharedCredentialFile)
			if err != nil {
				return flags, err
			}
			userData.ResourceShareArn = resourceShareArn

			dnsDomain, err := PrepareDNSDomain(client)
			if err != nil {
				return flags, err
			}
			flags = append(flags, "--base-domain", dnsDomain)
			userData.DNSDomain = dnsDomain

			hostedZoneID, err := PrepareHostedZone(clusterName, dnsDomain, vpc.VpcID, profile.Region, true,
				awsSharedCredentialFile)
			if err != nil {
				return flags, err
			}
			flags = append(flags, "--private-hosted-zone-id", hostedZoneID)
			userData.HostedZoneID = hostedZoneID

			clusterConfiguration.SharedVPC = profile.ClusterConfig.SharedVPC
		}
	}
	if profile.ClusterConfig.BillingAccount != "" {
		flags = append(flags, "--billing-account", profile.ClusterConfig.BillingAccount)
		clusterConfiguration.BillingAccount = profile.ClusterConfig.BillingAccount
	}
	if profile.ClusterConfig.DisableSCPChecks {
		flags = append(flags, "--disable-scp-checks")
		clusterConfiguration.DisableScpChecks = true
	}
	if profile.ClusterConfig.DisableUserWorKloadMonitoring {
		flags = append(flags, "--disable-workload-monitoring")
		clusterConfiguration.DisableWorkloadMonitoring = true
	}
	if profile.ClusterConfig.EtcdKMS {
		keyArn, err := PrepareKMSKey(profile.Region, false, "rosacli", profile.ClusterConfig.HCP, true)
		userData.EtcdKMSKey = keyArn
		if err != nil {
			return flags, err
		}
		flags = append(flags,
			"--etcd-encryption-kms-arn", keyArn,
		)
		if clusterConfiguration.Encryption == nil {
			clusterConfiguration.Encryption = &ClusterConfigure.Encryption{}
		}
		clusterConfiguration.Encryption.EtcdEncryptionKmsArn = keyArn
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
		flags = append(flags, "--external-auth-providers-enabled")
		clusterConfiguration.ExternalAuthentication = profile.ClusterConfig.ExternalAuthConfig
	}

	if profile.ClusterConfig.FIPS {
		flags = append(flags, "--fips")
	}
	if profile.ClusterConfig.HCP {
		flags = append(flags, "--hosted-cp")
	}
	clusterConfiguration.Nodes = &ClusterConfigure.Nodes{}
	if profile.ClusterConfig.InstanceType != "" {
		flags = append(flags, "--compute-machine-type", profile.ClusterConfig.InstanceType)
		clusterConfiguration.Nodes.ComputeInstanceType = profile.ClusterConfig.InstanceType
	} else {
		clusterConfiguration.Nodes.ComputeInstanceType = constants.DefaultInstanceType
	}
	if profile.ClusterConfig.KMSKey {
		kmsKeyArn, err := PrepareKMSKey(profile.Region, false, "rosacli", profile.ClusterConfig.HCP, false)
		userData.KMSKey = kmsKeyArn
		if err != nil {
			return flags, err
		}
		flags = append(flags,
			"--kms-key-arn", kmsKeyArn,
			"--enable-customer-managed-key",
		)
		if clusterConfiguration.Encryption == nil {
			clusterConfiguration.Encryption = &ClusterConfigure.Encryption{}
		}
		clusterConfiguration.EnableCustomerManagedKey = profile.ClusterConfig.KMSKey
		clusterConfiguration.Encryption.KmsKeyArn = kmsKeyArn
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

	if profile.ClusterConfig.ZeroEgress {
		flags = append(flags, "--properties", fmt.Sprintf("zero_egress:%t", profile.ClusterConfig.ZeroEgress))
		clusterConfiguration.Properties = &ClusterConfigure.Properties{
			ZeroEgress: profile.ClusterConfig.ZeroEgress,
		}
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
		flags = append(flags, "--availability-zones", profile.ClusterConfig.Zones)
		clusterConfiguration.AvailabilityZones = profile.ClusterConfig.Zones
	}
	if profile.ClusterConfig.ExternalAuthConfig {
		flags = append(flags, "--external-auth-providers-enabled")
	}
	if profile.ClusterConfig.NetworkType == "other" {
		flags = append(flags, "--no-cni")
		clusterConfiguration.Networking.Type = profile.ClusterConfig.NetworkType
	}

	if profile.ClusterConfig.RegistriesConfig && profile.ClusterConfig.HCP {
		caContent, err := helper.CreatePEMCertificate()

		if err != nil {
			return flags, err
		}
		registryConfigCa := map[string]string{
			"test.io": caContent,
		}
		caFile := path.Join(config.Test.OutputDir, "registryConfig")
		_, err = helper.CreateFileWithContent(caFile, registryConfigCa)
		if err != nil {
			return flags, err
		}
		flags = append(flags,
			"--registry-config-additional-trusted-ca", caFile,
			"--registry-config-insecure-registries", "test.com,*.example",
			"--registry-config-allowed-registries-for-import",
			"docker.io:false,registry.redhat.com:false,registry.access.redhat.com:false,quay.io:false",
		)
		if profile.ClusterConfig.AllowedRegistries {
			flags = append(flags,
				"--registry-config-allowed-registries", "allowed.example.com,*.test",
			)
		} else {
			flags = append(flags,
				"--registry-config-blocked-registries", "blocked.example.com,*.test",
			)
		}
	}
	return flags, nil
}
func WaitForClusterPassWaiting(client *rosacli.Client, cluster string, timeoutMin int) error {
	endTime := time.Now().Add(time.Duration(timeoutMin) * time.Minute)
	for time.Now().Before(endTime) {
		output, err := client.Cluster.DescribeClusterAndReflect(cluster)
		if err != nil {
			return err
		}
		if !strings.Contains(output.State, constants.Waiting) {
			log.Logger.Infof("Cluster %s is not in waiting state anymore", cluster)
			return nil
		}
		time.Sleep(time.Minute)
	}
	return fmt.Errorf("timeout for cluster stuck waiting after %d mins", timeoutMin)
}

func WaitForClusterReady(client *rosacli.Client, cluster string, timeoutMin int) error {
	var description *rosacli.ClusterDescription
	var clusterDetail *ClusterDetail
	var err error
	clusterDetail, err = ParserClusterDetail()
	if err != nil {
		return err
	}
	defer func() {
		log.Logger.Info("Going to record the necessary information")
		helper.CreateFileWithContent(config.Test.ClusterDetailFile, clusterDetail)
		// Temporary recoding file to make it compatible to existing jobs
		helper.CreateFileWithContent(config.Test.APIURLFile, description.APIURL)
		helper.CreateFileWithContent(config.Test.ConsoleUrlFile, description.ConsoleURL)
		helper.CreateFileWithContent(config.Test.InfraIDFile, description.InfraID)
		// End of temporary
	}()
	err = WaitForClusterPassWaiting(client, cluster, 2)
	if err != nil {
		return err
	}
	endTime := time.Now().Add(time.Duration(timeoutMin) * time.Minute)
	sleepTime := 0
	for time.Now().Before(endTime) {
		description, err = client.Cluster.DescribeClusterAndReflect(cluster)
		if err != nil {
			return err
		}
		clusterDetail.APIURL = description.APIURL
		clusterDetail.ConsoleURL = description.ConsoleURL
		clusterDetail.InfraID = description.InfraID
		switch description.State {
		case constants.Ready:
			log.Logger.Infof("Cluster %s is ready now.", cluster)
			return nil
		case constants.Uninstalling:
			return fmt.Errorf("cluster %s is %s now. Cannot wait for it ready",
				cluster, constants.Uninstalling)
		default:
			if strings.Contains(description.State, constants.Error) {
				log.Logger.Errorf("Cluster is in %s status now. Recording the installation log", constants.Error)
				RecordClusterInstallationLog(client, cluster)
				return fmt.Errorf("cluster %s is in %s state with reason: %s",
					cluster, constants.Error, description.State)
			}
			if strings.Contains(description.State, constants.Pending) ||
				strings.Contains(description.State, constants.Installing) ||
				strings.Contains(description.State, constants.Validating) {
				time.Sleep(2 * time.Minute)
				continue
			}
			if strings.Contains(description.State, constants.Waiting) {
				log.Logger.Infof("Cluster is in status of %v, wait for ready", constants.Waiting)
				if sleepTime >= 6 {
					return fmt.Errorf("cluster stuck to %s status for more than 6 mins. "+
						"Check the user data preparation for roles", description.State)
				}
				sleepTime += 2
				time.Sleep(2 * time.Minute)
				continue
			}
			return fmt.Errorf("unknown cluster state %s", description.State)
		}

	}

	return fmt.Errorf("timeout for cluster ready waiting after %d mins", timeoutMin)
}

func ReverifyClusterNetwork(client *rosacli.Client, clusterID string) error {
	log.Logger.Infof("verify network of cluster %s ", clusterID)
	_, err := client.NetworkVerifier.CreateNetworkVerifierWithCluster(clusterID)
	return err
}

func RecordClusterInstallationLog(client *rosacli.Client, cluster string) error {
	output, err := client.Cluster.InstallLog(cluster)
	if err != nil {
		return err
	}
	_, err = helper.CreateFileWithContent(config.Test.ClusterInstallLogArtifactFile, output.String())
	return err
}

func CreateClusterByProfileWithoutWaiting(
	profile *Profile,
	client *rosacli.Client) (*rosacli.ClusterDescription, error) {

	clusterDetail := new(ClusterDetail)

	flags, err := GenerateClusterCreateFlags(profile, client)
	if err != nil {
		log.Logger.Errorf("Error happened when generate flags: %s", err.Error())
		return nil, err
	}
	log.Logger.Infof("User data and flags preparation finished")
	_, err, createCMD := client.Cluster.Create(profile.ClusterConfig.Name, flags...)
	if err != nil {
		return nil, err
	}
	helper.CreateFileWithContent(config.Test.CreateCommandFile, createCMD)
	log.Logger.Info("Cluster created successfully")
	description, err := client.Cluster.DescribeClusterAndReflect(profile.ClusterConfig.Name)
	if err != nil {
		return description, err
	}
	defer func() {
		log.Logger.Info("Going to record the necessary information")
		helper.CreateFileWithContent(config.Test.ClusterDetailFile, clusterDetail)
		// Temporary recoding file to make it compatible to existing jobs
		helper.CreateFileWithContent(config.Test.ClusterIDFile, description.ID)
		helper.CreateFileWithContent(config.Test.ClusterNameFile, description.Name)
		helper.CreateFileWithContent(config.Test.ClusterTypeFile, "rosa")
		// End of temporary
	}()
	clusterDetail.ClusterID = description.ID
	clusterDetail.ClusterName = description.Name
	clusterDetail.ClusterType = "rosa"
	clusterDetail.OIDCEndpointURL = description.OIDCEndpointURL
	clusterDetail.OperatorRoleArns = description.OperatorIAMRoles

	// Need to do the post step when cluster has no oidcconfig enabled
	if profile.ClusterConfig.OIDCConfig == "" && profile.ClusterConfig.STS {
		err = PrepareOIDCProviderByCluster(client, description.ID)
		if err != nil {
			return description, err
		}
		err = PrepareOperatorRolesByCluster(client, description.ID)
		if err != nil {
			return description, err
		}
	}
	// Need to decorate the KMS key
	if profile.ClusterConfig.KMSKey && profile.ClusterConfig.STS {
		err = ElaborateKMSKeyForSTSCluster(client, description.ID, false)
		if err != nil {
			return description, err
		}
	}
	if profile.ClusterConfig.EtcdKMS && profile.ClusterConfig.STS {
		err = ElaborateKMSKeyForSTSCluster(client, description.ID, true)
		if err != nil {
			return description, err
		}
	}
	return description, err
}
func CreateClusterByProfile(profile *Profile,
	client *rosacli.Client,
	waitForClusterReady bool) (*rosacli.ClusterDescription, error) {

	description, err := CreateClusterByProfileWithoutWaiting(profile, client)
	if err != nil {
		return description, err
	}
	if profile.ClusterConfig.BYOVPC {
		log.Logger.Infof("Reverify the network for the cluster %s to make sure it can be parsed", description.ID)
		ReverifyClusterNetwork(client, description.ID)
	}
	if waitForClusterReady {
		log.Logger.Infof("Waiting for the cluster %s to ready", description.ID)
		err = WaitForClusterReady(client, description.ID, config.Test.GlobalENV.ClusterWaitingTime)
		if err != nil {
			return description, err
		}
		description, err = client.Cluster.DescribeClusterAndReflect(profile.ClusterConfig.Name)
	}
	return description, err
}

func WaitForClusterUninstalled(client *rosacli.Client, cluster string, timeoutMin int) error {
	endTime := time.Now().Add(time.Duration(timeoutMin) * time.Minute)
	for time.Now().Before(endTime) {
		output, err := client.Cluster.DescribeCluster(cluster)
		if err != nil &&
			strings.Contains(output.String(),
				fmt.Sprintf("There is no cluster with identifier or name '%s'", cluster)) {
			log.Logger.Infof("Cluster %s has been deleted.", cluster)
			return nil
		}
		desc, err := client.Cluster.ReflectClusterDescription(output)

		if err != nil {
			return err
		}
		if strings.Contains(desc.State, constants.Uninstalling) {
			time.Sleep(2 * time.Minute)
			continue
		}
		return fmt.Errorf("cluster %s is in status of %s which won't be deleted, stop waiting", cluster, desc.State)
	}
	return fmt.Errorf("timeout for waiting for cluster deletion finished after %d mins", timeoutMin)
}

func DestroyCluster(client *rosacli.Client) (*ClusterDetail, []error) {
	var (
		cd             *ClusterDetail
		clusterService rosacli.ClusterService
		errors         []error
	)
	// get cluster info from cluster detail file
	cd, err := ParserClusterDetail()
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	// delete cluster
	if cd != nil && cd.ClusterID != "" {
		clusterService = client.Cluster
		output, errDeleteCluster := clusterService.DeleteCluster(cd.ClusterID, "-y")
		if errDeleteCluster != nil {
			if strings.Contains(output.String(), fmt.Sprintf("There is no cluster with identifier or name '%s'", cd.ClusterID)) {
				log.Logger.Infof("Cluster %s not exists.", cd.ClusterID)
			} else {
				log.Logger.Errorf("Error happened when delete cluster: %s", output.String())
				errors = append(errors, errDeleteCluster)
				return nil, errors
			}
		} else {
			log.Logger.Infof("Waiting for the cluster %s to be uninstalled", cd.ClusterID)

			err = WaitForClusterUninstalled(client, cd.ClusterID, config.Test.GlobalENV.ClusterWaitingTime)
			if err != nil {
				log.Logger.Errorf("Error happened when waiting cluster uninstall: %s", err.Error())
				errors = append(errors, err)
				return nil, errors
			} else {
				log.Logger.Infof("Delete cluster %s successfully.", cd.ClusterID)
			}
		}
	}
	return cd, errors
}

func DestroyPreparedUserData(client *rosacli.Client, clusterID string, region string, isSTS bool,
	isSharedVPC bool, isAdditionalPrincipalAllowed bool) []error {

	var (
		ud                 *UserData
		ocmResourceService rosacli.OCMResourceService
		errors             []error
	)
	ocmResourceService = client.OCMResource

	awsSharedCredentialFile := ""
	if isSharedVPC || isAdditionalPrincipalAllowed {
		awsSharedCredentialFile = config.Test.GlobalENV.SVPC_CREDENTIALS_FILE
	}

	// get user data from resource file
	ud, err := ParseUserData()
	if err != nil {
		errors = append(errors, err)
		return errors
	}
	defer func() {
		log.Logger.Info("Rewrite User data file")
		// rewrite user data
		_, err = helper.CreateFileWithContent(config.Test.UserDataFile, ud)
	}()

	destroyLog := func(err error, resource string) bool {
		if err != nil {
			log.Logger.Errorf("Error happened when delete %s: %s", resource, err.Error())
			errors = append(errors, err)
			return false
		}
		log.Logger.Infof("Delete %s successfully", resource)
		return true
	}

	if ud != nil {
		// schedule KMS key
		if ud.KMSKey != "" {
			log.Logger.Infof("Find prepared kms key: %s. Going to schedule the deletion.", ud.KMSKey)
			err = ScheduleKMSDesiable(ud.KMSKey, region)
			success := destroyLog(err, "kms key")
			if success {
				ud.KMSKey = ""
			}
		}
		// schedule Etcd KMS key
		if ud.EtcdKMSKey != "" {
			log.Logger.Infof("Find prepared etcd kms key: %s. Going to schedule the deletion", ud.EtcdKMSKey)
			err = ScheduleKMSDesiable(ud.EtcdKMSKey, region)
			success := destroyLog(err, "etcd kms key")
			if success {
				ud.EtcdKMSKey = ""
			}
		}
		// delete audit log arn
		if ud.AuditLogArn != "" {
			log.Logger.Infof("Find prepared audit log arn: %s", ud.AuditLogArn)
			roleName := strings.Split(ud.AuditLogArn, "/")[1]
			err = DeleteAuditLogRoleArn(roleName, region)
			success := destroyLog(err, "audit log arn")
			if success {
				ud.AuditLogArn = ""
			}
		}
		//delete hosted zone
		if ud.HostedZoneID != "" {
			log.Logger.Infof("Find prepared hosted zone: %s", ud.HostedZoneID)
			err = DeleteHostedZone(ud.HostedZoneID, region, awsSharedCredentialFile)
			success := destroyLog(err, "hosted zone")
			if success {
				ud.HostedZoneID = ""
			}
		}
		//delete dns domain
		if ud.DNSDomain != "" {
			log.Logger.Infof("Find prepared DNS Domain: %s", ud.DNSDomain)
			_, err = ocmResourceService.DeleteDNSDomain(ud.DNSDomain)
			success := destroyLog(err, "dns domain")
			if success {
				ud.DNSDomain = ""
			}
		}
		// delete resource share
		if ud.ResourceShareArn != "" {
			log.Logger.Infof("Find prepared resource share: %s", ud.ResourceShareArn)
			err = DeleteResourceShare(ud.ResourceShareArn, region, awsSharedCredentialFile)
			success := destroyLog(err, "resource share")
			if success {
				ud.ResourceShareArn = ""
			}
		}
		// delete vpc chain
		if ud.VpcID != "" {
			log.Logger.Infof("Find prepared vpc id: %s", ud.VpcID)
			if isSharedVPC {
				err = DeleteSharedVPCChain(ud.VpcID, region, awsSharedCredentialFile)
			} else {
				err = DeleteVPCChain(ud.VpcID, region)
			}
			success := destroyLog(err, "vpc chain")
			if success {
				ud.VpcID = ""
			}
		}
		// delete shared vpc role
		if ud.SharedVPCRole != "" {
			log.Logger.Infof("Find prepared shared vpc role: %s", ud.SharedVPCRole)
			err = DeleteSharedVPCRole(ud.SharedVPCRole, false, region, awsSharedCredentialFile)
			success := destroyLog(err, "shared vpc role")
			if success {
				ud.SharedVPCRole = ""
			}
		}
		// delete additional principal role
		if ud.AdditionalPrincipals != "" {
			log.Logger.Infof("Find prepared additional principal role: %s", ud.AdditionalPrincipals)
			err = DeleteAdditionalPrincipalsRole(ud.AdditionalPrincipals, true, region, awsSharedCredentialFile)
			success := destroyLog(err, "additional principal role")
			if success {
				ud.AdditionalPrincipals = ""
			}
		}
		// delete operator roles
		if ud.OperatorRolesPrefix != "" {
			log.Logger.Infof("Find prepared operator roles with prefix: %s", ud.OperatorRolesPrefix)
			_, err = ocmResourceService.DeleteOperatorRoles("--prefix", ud.OperatorRolesPrefix, "--mode", "auto", "-y")
			success := destroyLog(err, "operator roles")
			if success {
				ud.OperatorRolesPrefix = ""
			}
		}
		// delete oidc config
		if ud.OIDCConfigID != "" {
			log.Logger.Infof("Find prepared oidc config id: %s", ud.OIDCConfigID)
			_, err = ocmResourceService.DeleteOIDCConfig(
				"--oidc-config-id",
				ud.OIDCConfigID,
				"--region",
				region,
				"--mode",
				"auto",
				"-y")
			success := destroyLog(err, "oidc config")
			if success {
				ud.OIDCConfigID = ""
			}
		} else {
			if clusterID != "" && isSTS {
				_, err = ocmResourceService.DeleteOIDCProvider("-c", clusterID, "-y", "--mode", "auto")
				success := destroyLog(err, "oidc provider")
				if success {
					ud.OIDCConfigID = ""
				}
			}
		}
		// delete account roles
		if ud.AccountRolesPrefix != "" {
			log.Logger.Infof("Find prepared account roles with prefix: %s", ud.AccountRolesPrefix)
			_, err = ocmResourceService.DeleteAccountRole("--mode", "auto", "--prefix", ud.AccountRolesPrefix, "-y")
			success := destroyLog(err, "account roles")
			if success {
				ud.AccountRolesPrefix = ""
			}
		}
	}
	return errors
}

func DestroyResourceByProfile(profile *Profile, client *rosacli.Client) (errors [][]error) {
	// destroy cluster
	cd, errDestroyCluster := DestroyCluster(client)
	if len(errDestroyCluster) > 0 {
		errors = append(errors, errDestroyCluster)
		return errors
	}

	// destroy prepared user data
	clusterId := ""
	if cd != nil {
		clusterId = cd.ClusterID
	}
	region := profile.Region
	isSTS := profile.ClusterConfig.STS
	isSharedVPC := profile.ClusterConfig.SharedVPC
	isAdditionalPrincipalAllowed := profile.ClusterConfig.AdditionalPrincipals
	errDestroyUserData := DestroyPreparedUserData(client, clusterId, region, isSTS, isSharedVPC,
		isAdditionalPrincipalAllowed)
	if len(errDestroyUserData) > 0 {
		errors = append(errors, errDestroyUserData)
	}
	return errors
}
