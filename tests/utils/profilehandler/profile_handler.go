package profilehandler

import (
	"bytes"
	"fmt"
	"os"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/common"
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

// PrepareAccountRoles will prepare account roles according to the parameters
// openshiftVersion must follow 4.15.2-x format
func PrepareAccountRoles(client *rosacli.Client, namePrefix string, hcp bool, openshiftVersion string, channelGroup string) (
	accRoles *rosacli.AccountRolesUnit, err error) {
	flags := []string{
		"--prefix", namePrefix,
		"--mode", "auto",
		"-y",
	}
	if openshiftVersion != "" {
		majorVersion := common.SplitMajorVersion(openshiftVersion)
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
	_, err = client.OCMResource.CreateAccountRole(
		flags...,
	)
	if err != nil {
		return
	}
	accRoleList, _, err := client.OCMResource.ListAccountRole()
	if err != nil {
		return
	}
	roleDig := accRoleList.DigAccountRoles(namePrefix, hcp)

	return roleDig, nil

}

// PrepareOperatorRolesByOIDCConfig will prepare operator roles with OIDC config ID
// When sharedVPCRoleArn is not empty it will be set to the flag
func PrepareOperatorRolesByOIDCConfig(client *rosacli.Client,
	namePrefix string,
	oidcConfigID string,
	roleArn string,
	sharedVPCRoleArn string,
	hcp bool) error {
	flags := []string{
		"-y",
		"--mode", "auto",
		"--prefix", namePrefix,
		"--role-arn", roleArn,
	}
	if hcp {
		flags = append(flags, "--hosted-cp")
	}
	if sharedVPCRoleArn != "" {
		flags = append(flags, "--shared-vpc-role-arn", sharedVPCRoleArn)
	}
	_, err := client.OCMResource.CreateOperatorRoles(
		flags...,
	)
	return err
}

func PrepareOperatorRolesByCluster(client *rosacli.Client,
	namePrefix string, cluster string) error {
	flags := []string{
		"-y",
		"--mode", "auto",
		"--cluster", cluster,
	}
	_, err := client.OCMResource.CreateOperatorRoles(
		flags...,
	)
	return err
}

// PrepareOIDCConfig will prepare the oidc config for the cluster, if the oidcConfigType="managed", roleArn and prefix won't be set
func PrepareOIDCConfig(client *rosacli.Client,
	oidcConfigType string,
	region string,
	roleArn string,
	prefix string) (string, error) {
	var oidcConfigID string
	var output bytes.Buffer
	var err error
	switch oidcConfigType {
	case "managed":
		output, err = client.OCMResource.CreateOIDCConfig(
			"-o", "json",
			"--mode", "auto",
			"--region", region,
			"--managed",
			"-y",
		)
	case "unmanaged":
		output, err = client.OCMResource.CreateOIDCConfig(
			"-o", "json",
			"--mode", "auto",
			"--prefix", prefix,
			"--region", region,
			"--role-arn", roleArn,
			"--managed=false",
			"-y",
		)

	default:
		return "", fmt.Errorf("only 'managed' or 'unmanaged' oidc config is allowed")
	}
	if err != nil {
		return oidcConfigID, err
	}
	parser := rosacli.NewParser()
	oidcConfigID = parser.JsonData.Input(output).Parse().DigString("id")
	return oidcConfigID, nil
}

func PrepareOIDCProvider(client *rosacli.Client, oidcConfigID string) error {
	_, err := client.OCMResource.CreateOIDCProvider(
		"--mode", "auto",
		"-y",
		"--oidc-config-id", oidcConfigID,
	)
	return err
}

// GenerateClusterCreateFlags will generate flags
func GenerateClusterCreateFlags(profile *Profile, client *rosacli.Client) (flags []string, err error) {
	flags = []string{
		"--cluster-name", profile.NamePrefix,
	}
	if profile.Version != "" {
		version := PrepareVersionDummy(profile.Version)
		flags = append(flags, "--version", version)
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
			profile.ChannelGroup)
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
			flags = append(flags, "--oidc-config-id", oidcConfigID)
		}
		flags = append(flags, "--operator-roles-prefix", profile.NamePrefix)
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
	return description, err
}
