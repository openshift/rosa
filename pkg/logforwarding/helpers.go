package logforwarding

import (
	"os"

	"gopkg.in/yaml.v3"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
)

// FlagName contains the common log forwarding config command flag name
const FlagName = "log-fwd-config"

func constructPodGroupsHelpMessage(options []*cmv1.LogForwarderGroupVersions) (s string) {
	s = ""
	for _, option := range options {
		apps := ""
		for i, application := range option.Versions()[len(option.Versions())-1].Applications() {
			if i != 0 {
				apps += ","
			}
			apps += application
		}
		s = s + option.ID() + ": " + apps + "\n"
	}
	return
}

func constructPodGroupsInteractiveOptions(options []*cmv1.LogForwarderGroupVersions) (l []string) {
	for _, option := range options {
		l = append(l, option.ID())
	}
	return
}

func bindCloudWatchLogForwarder(input, output *ocm.LogForwarderConfig) {
	output.CloudWatchLogRoleArn = input.CloudWatchLogRoleArn
	output.CloudWatchLogGroupName = input.CloudWatchLogGroupName
	output.Applications = input.Applications
	output.GroupsLogVersion = input.GroupsLogVersion
}

func bindS3LogForwarder(input, output *ocm.LogForwarderConfig) {
	output.S3ConfigBucketName = input.S3ConfigBucketName
	output.S3ConfigBucketPrefix = input.S3ConfigBucketPrefix
	output.Applications = input.Applications
	output.GroupsLogVersion = input.GroupsLogVersion
}

func UnmarshalLogForwarderConfigYaml(r reporter.Logger, yamlFile string) ([]*ocm.LogForwarderConfig, error) {
	logFwdConfigObjectCloudWatch := &ocm.LogForwarderConfig{}
	logFwdConfigObjectS3 := &ocm.LogForwarderConfig{}
	result := make([]*ocm.LogForwarderConfig, 0)
	fileContents, err := os.ReadFile(yamlFile)
	if err != nil {
		r.Errorf("Error reading log-fwd-config YAML file '%s': %s", yamlFile, err)
		return nil, err
	}
	tempFwdConfigObject := &ocm.LogForwarderConfig{}
	err = yaml.Unmarshal(fileContents, &tempFwdConfigObject)
	if err != nil {
		r.Errorf("Error parsing log forwarder config YAML file '%s': %s", yamlFile, err)
		return nil, err
	}
	if tempFwdConfigObject.CloudWatchLogRoleArn != "" {
		bindCloudWatchLogForwarder(tempFwdConfigObject, logFwdConfigObjectCloudWatch)
		result = append(result, logFwdConfigObjectCloudWatch)
	}
	if tempFwdConfigObject.S3ConfigBucketName != "" {
		bindS3LogForwarder(tempFwdConfigObject, logFwdConfigObjectS3)
		result = append(result, logFwdConfigObjectS3)
	}

	return result, nil
}
