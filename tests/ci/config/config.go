package config

import (
	"os"
	"path"
	"strings"

	"github.com/openshift/rosa/tests/utils/common"
	. "github.com/openshift/rosa/tests/utils/log"
)

var Test *TestConfig

// TestConfig contains platforms info for the rosacli testing
type TestConfig struct {
	// Env is the OpenShift Cluster Management environment used to provision clusters.
	ENV               string `env:"OCM_LOGIN_ENV" default:""`
	TestProfile       string `env:"TEST_PROFILE" default:""`
	OutputDir         string `env:"OUTPUT_DIR" default:""`
	YAMLProfilesDir   string `env:"TEST_PROFILE_DIR" default:""`
	RootDir           string `env:"WORKSPACE" default:""`
	ClusterConfigFile string
	UserDataFile      string
}

func init() {
	Test = new(TestConfig)
	currentDir, _ := os.Getwd()
	project := "rosa"

	Test.TestProfile = common.ReadENVWithDefaultValue("TEST_PROFILE", "")
	Test.RootDir = common.ReadENVWithDefaultValue("WORKSPACE", strings.SplitAfter(currentDir, project)[0])
	Test.YAMLProfilesDir = common.ReadENVWithDefaultValue("TEST_PROFILE_DIR", path.Join(Test.RootDir, "tests", "ci", "data", "profiles"))
	Test.OutputDir = common.ReadENVWithDefaultValue("SHARED_DIR", path.Join(Test.RootDir, "tests", "output", Test.TestProfile))
	err := os.MkdirAll(Test.OutputDir, 0777)
	if err != nil {
		Logger.Errorf("Meet error %s when create output dirs", err.Error())
	}
	Test.ClusterConfigFile = path.Join(Test.OutputDir, "cluster-config")
	Test.UserDataFile = path.Join(Test.OutputDir, "user-data")
}
