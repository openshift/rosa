package profilehandler

import (
	"fmt"
	"os"

	"github.com/openshift/rosa/tests/ci/config"
	. "github.com/openshift/rosa/tests/utils/log"
)

func GetYAMLProfilesDir() string {
	return config.Test.YAMLProfilesDir
}
func LoadProfileYamlFile(profileName string) *Profile {
	p := GetProfile(profileName, GetYAMLProfilesDir())
	Logger.Infof("Loaded cluster profile configuration from origional profile %s : %v", profileName, *p)
	Logger.Infof("Loaded cluster profile configuration from origional cluster %s : %v", profileName, *p.ClusterConfig)
	Logger.Infof("Loaded cluster profile configuration from origional account-roles %s : %v", profileName, *p.AccountRoleConfig)
	return p
}

func LoadProfileYamlFileByENV() *Profile {
	if config.Test.TestProfile == "" {
		panic(fmt.Errorf("ENV Variable TEST_PROFILE is empty, please make sure you set the env value"))
	}
	profile := LoadProfileYamlFile(config.Test.TestProfile)

	// Supporting global env setting to overrite profile settings
	if os.Getenv("CHANNEL_GROUP") != "" {
		Logger.Infof("Got global env settings for CHANNEL_GROUP, overwritten the profile setting with value %s", os.Getenv("CHANNEL_GROUP"))
		profile.ChannelGroup = os.Getenv("CHANNEL_GROUP")
	}
	if os.Getenv("VERSION") != "" {
		Logger.Infof("Got global env settings for VERSION, overwritten the profile setting with value %s", os.Getenv("VERSION"))
		profile.Version = os.Getenv("VERSION")
	}
	if os.Getenv("REGION") != "" {
		Logger.Infof("Got global env settings for REGION, overwritten the profile setting with value %s", os.Getenv("REGION"))
		profile.Region = os.Getenv("REGION")
	}
	if os.Getenv("PROVISION_SHARD") != "" {
		Logger.Infof("Got global env settings for PROVISION_SHARD, overwritten the profile setting with value %s", os.Getenv("PROVISION_SHARD"))
		profile.ClusterConfig.ProvisionShard = os.Getenv("PROVISION_SHARD")
	}
	return profile
}
