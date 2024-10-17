package handler

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/log"
)

type profiles struct {
	Profiles []*Profile `yaml:"profiles,omitempty"`
}

func GetYAMLProfilesDir() string {
	return config.Test.YAMLProfilesDir
}

func ParseProfiles(profilesDir string) map[string]*Profile {
	files, err := os.ReadDir(profilesDir)
	if err != nil {
		log.Logger.Fatal(err)
	}

	profileMap := make(map[string]*Profile)
	for _, file := range files {
		yfile, err := os.ReadFile(path.Join(profilesDir, file.Name()))
		if err != nil {
			log.Logger.Fatal(err)
		}

		p := new(profiles)
		err = yaml.Unmarshal(yfile, &p)
		if err != nil {
			log.Logger.Fatal(err)
		}

		for _, theProfile := range p.Profiles {
			profileMap[theProfile.Name] = theProfile
		}

	}

	return profileMap
}

func ParseProfilesByFile(profileLocation string) map[string]*Profile {
	profileMap := make(map[string]*Profile)

	yfile, err := os.ReadFile(profileLocation)
	if err != nil {
		log.Logger.Fatal(err)
	}

	p := new(profiles)
	err = yaml.Unmarshal(yfile, &p)
	if err != nil {
		log.Logger.Fatal(err)
	}

	for _, theProfile := range p.Profiles {
		profileMap[theProfile.Name] = theProfile
	}

	return profileMap
}

func GetProfile(profileName string, profilesDir string) *Profile {
	profileMap := ParseProfiles(profilesDir)

	if _, exist := profileMap[profileName]; !exist {
		log.Logger.Fatalf("Can not find the profile %s in %s\n", profileName, profilesDir)
	}

	return profileMap[profileName]
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

	if config.Test.GlobalENV.ComputeMachineType != "" {
		log.Logger.Infof("Got global env settings for INSTANCE_TYPE, overwritten the profile setting with value %s",
			config.Test.GlobalENV.ComputeMachineType)
		profile.ClusterConfig.InstanceType = config.Test.GlobalENV.ComputeMachineType
	}

	return profile
}
