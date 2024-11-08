package handler

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/helper"
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
		return nil
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
	if config.Test.GlobalENV.ZeroEgress {
		log.Logger.Infof("Got global env settings for ZERO_EGRESS, overwritten the profile setting with value %t",
			config.Test.GlobalENV.ZeroEgress)
		profile.ClusterConfig.ZeroEgress = config.Test.GlobalENV.ZeroEgress
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
