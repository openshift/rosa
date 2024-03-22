package profilehandler

import (
	"fmt"
	"os"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
)

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
func PrepareVersionDummy(versionRequirement string) string {
	return ""
}

func PrepareSubnetsDummy(vpcID string, region string, zones string) map[string][]string {
	return map[string][]string{}
}

func PrepareProxysDummy(vpcID string, region string, zones string) map[string]string {
	return map[string]string{
		"https_proxy":  "",
		"http_proxy":   "",
		"no_proxy":     "",
		"ca_file_path": "",
	}
}

func PrepareKMSKeyDummy(region string) string {
	return ""
}

func PrepareSecurityGroupsDummy(vpcID string, region string, securityGroupCount int) []string {
	return []string{}
}

func PrepareAccountRolesDummy(namePrefix string) *rosacli.AccountRoles {
	return nil
}

func PrepareOperatorRolesDummy(namePrefix string) error {
	return nil
}

func PrepareOIDCConfigDummy(providerRequired bool) string {
	var oidcConfigID string
	return oidcConfigID
}

func PrepareOIDCProviderDummy(oidcConfigID string) error {
	return nil
}

// GenerateClusterCreateCMD will generate
func GenerateClusterCreateCMD(profile *Profile) {
	flags := []string{}
	if profile.Version != "" {
		version := PrepareVersionDummy(profile.Version)
		flags = append(flags, "--version", version)
	}
	if profile.Region != "" {
		flags = append(flags, "--region", profile.Region)
	}
	if profile.ClusterConfig.STS {
		accRoles := PrepareAccountRolesDummy(profile.NamePrefix)
		flags = append(flags,
			"--role-arn", accRoles.InstallerRole,
			"--support-role-arn", accRoles.SupportRole,
			"--worker-iam-role", accRoles.WorkRole,
		)
		if !profile.ClusterConfig.HCP {
			flags = append(flags,
				"--controlplane-iam-role", accRoles.ControPlaneRole,
			)
		}
	}
	client := rosacli.NewClient()
	clusterService := client.Cluster
	clusterService.CreateDryRun("", flags...)
}
