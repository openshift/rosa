package config

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/openshift/rosa/tests/utils/common"
	. "github.com/openshift/rosa/tests/utils/log"
)

var Test *TestConfig

// TestConfig contains platforms info for the rosacli testing
type TestConfig struct {
	// Env is the OpenShift Cluster Management environment used to provision clusters.
	ENV                           string `env:"OCM_LOGIN_ENV" default:""`
	TestProfile                   string `env:"TEST_PROFILE" default:""`
	OutputDir                     string `env:"OUTPUT_DIR" default:""`
	YAMLProfilesDir               string `env:"TEST_PROFILE_DIR" default:""`
	RootDir                       string `env:"WORKSPACE" default:""`
	ClusterConfigFile             string
	ArtifactDir                   string `env:"ARTIFACT_DIR" default:""`
	UserDataFile                  string
	ClusterIDFile                 string // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	CreateCommandFile             string
	APIURLFile                    string // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	ClusterNameFile               string // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	ClusterTypeFile               string // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	ConsoleUrlFile                string // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	InfraIDFile                   string // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	ClusterDetailFile             string
	ClusterInstallLogArtifactFile string
	ClusterAdminFile              string
	TestFocusFile                 string
	GlobalENV                     *GlobalENVVariables
}
type GlobalENVVariables struct {
	ChannelGroup       string `env:"CHANNEL_GROUP" default:""`
	Version            string `env:"VERSION" default:""`
	Region             string `env:"REGION" default:""`
	ProvisionShard     string `env:"PROVISION_SHARD" default:""`
	NamePrefix         string `env:"NAME_PREFIX"`
	ClusterWaitingTime int    `env:"CLUSTER_TIMEOUT" default:"60"`
}

func init() {
	Test = new(TestConfig)
	currentDir, _ := os.Getwd()
	project := "rosa"

	Test.TestProfile = common.ReadENVWithDefaultValue("TEST_PROFILE", "")
	Test.RootDir = common.ReadENVWithDefaultValue("WORKSPACE", strings.SplitAfter(currentDir, project)[0])
	Test.YAMLProfilesDir = common.ReadENVWithDefaultValue("TEST_PROFILE_DIR", path.Join(Test.RootDir, "tests", "ci", "data", "profiles"))
	Test.OutputDir = common.ReadENVWithDefaultValue("SHARED_DIR", path.Join(Test.RootDir, "tests", "output", Test.TestProfile))
	Test.ArtifactDir = common.ReadENVWithDefaultValue("ARTIFACT_DIR", Test.OutputDir)
	err := os.MkdirAll(Test.OutputDir, 0777)
	if err != nil {
		Logger.Errorf("Meet error %s when create output dirs", err.Error())
	}
	Test.ClusterConfigFile = path.Join(Test.OutputDir, "cluster-config")
	Test.UserDataFile = path.Join(Test.OutputDir, "resources.json")
	Test.APIURLFile = path.Join(Test.OutputDir, "api.url")
	Test.ClusterIDFile = path.Join(Test.OutputDir, "cluster-id")     // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	Test.ClusterNameFile = path.Join(Test.OutputDir, "cluster-name") // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	Test.ClusterTypeFile = path.Join(Test.OutputDir, "cluster-type") // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	Test.ConsoleUrlFile = path.Join(Test.OutputDir, "console.url")   // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	Test.InfraIDFile = path.Join(Test.OutputDir, "infra_id")         // Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	Test.ClusterDetailFile = path.Join(Test.OutputDir, "cluster-detail.json")
	Test.ClusterInstallLogArtifactFile = path.Join(Test.ArtifactDir, ".install.log")
	Test.ClusterAdminFile = path.Join(Test.ArtifactDir, ".admin")
	Test.TestFocusFile = path.Join(Test.RootDir, "tests", "ci", "data", "commit-focus")

	waitingTime, err := strconv.Atoi(common.ReadENVWithDefaultValue("CLUSTER_TIMEOUT", "60"))
	if err != nil {
		panic(fmt.Errorf("env variable CLUSTER_TIMEOUT must be set to an integer"))
	}
	Test.GlobalENV = &GlobalENVVariables{
		ChannelGroup:       os.Getenv("CHANNEL_GROUP"),
		Version:            os.Getenv("VERSION"),
		Region:             os.Getenv("REGION"),
		ProvisionShard:     os.Getenv("PROVISION_SHARD"),
		NamePrefix:         os.Getenv("NAME_PREFIX"),
		ClusterWaitingTime: waitingTime,
	}

}
