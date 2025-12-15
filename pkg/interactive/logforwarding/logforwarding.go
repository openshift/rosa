package interactive

import (
	"fmt"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/logforwarding"
	"github.com/openshift/rosa/pkg/ocm"
)

// Options for initial interactive prompt
const skip = "Skip"
const cloudWatch = "CloudWatch"
const s3 = "S3"
const both = "Both"

func InteractiveLogForwardingConfig(ocmClient *ocm.Client) (
	*logforwarding.LogForwarderYaml, error) {
	con, err := interactive.GetOption(interactive.Input{
		Question: "Enabled log forwarding",
		Help:     "Whether log forwarding is enabled, and if so, which types of log forwarder(s) to create",
		Required: false,
		Default:  false,
		Options:  []string{cloudWatch, s3, both},
	})
	if err != nil {
		return nil, err
	}

	if con == skip {
		return nil, nil
	}

	s3Result := &logforwarding.S3LogForwarderConfig{}
	cloudWatchResult := &logforwarding.CloudWatchLogForwarderConfig{}

	if con == cloudWatch || con == both {
		cloudWatchResult, err = interactiveCloudWatch(ocmClient, "", "")
		if err != nil {
			return nil, err
		}
	}
	if con == s3 || con == both {
		s3Result, err = interactiveS3(ocmClient, "", "")
		if err != nil {
			return nil, err
		}
	}

	result := logforwarding.LogForwarderYaml{}
	result.S3 = s3Result
	result.CloudWatch = cloudWatchResult

	return &result, nil
}

func InteractiveLogForwardingConfigWithDefaults(ocmClient *ocm.Client, logForwarder *cmv1.LogForwarder) (
	*logforwarding.LogForwarderYaml, error) {

	s3Result := &logforwarding.S3LogForwarderConfig{}
	cloudWatchResult := &logforwarding.CloudWatchLogForwarderConfig{}

	var err error

	if logForwarder.Cloudwatch() != nil && logForwarder.Cloudwatch().LogDistributionRoleArn() != "" {
		cloudWatchResult, err = interactiveCloudWatch(ocmClient,
			logForwarder.Cloudwatch().LogGroupName(), logForwarder.Cloudwatch().LogDistributionRoleArn())
		if err != nil {
			return nil, err
		}
	}
	if logForwarder.S3() != nil && logForwarder.S3().BucketName() != "" {
		s3Result, err = interactiveS3(ocmClient, logForwarder.S3().BucketName(), logForwarder.S3().BucketPrefix())
		if err != nil {
			return nil, err
		}
	}

	result := logforwarding.LogForwarderYaml{}
	result.S3 = s3Result
	result.CloudWatch = cloudWatchResult

	return &result, nil
}

func interactiveCloudWatch(ocmClient *ocm.Client, defaultLogGroupName string, defaultLogRoleArn string) (
	*logforwarding.CloudWatchLogForwarderConfig, error) {

	cloudWatchConfig := logforwarding.CloudWatchLogForwarderConfig{}
	roleArn, err := interactive.GetString(interactive.Input{
		Question: "CloudWatch Log forwarding role ARN",
		Help:     "The role ARN which forwards logs to CloudWatch",
		Default:  defaultLogRoleArn,
		Required: true,
	})
	if err != nil {
		return nil, err
	}
	cloudWatchConfig.CloudWatchLogRoleArn = roleArn

	groupName, err := interactive.GetString(interactive.Input{
		Question: "CloudWatch log group name",
		Help:     "The name of the group on CloudWatch which will contain the logs",
		Default:  defaultLogGroupName,
		Required: true,
	})
	if err != nil {
		return nil, err
	}
	cloudWatchConfig.CloudWatchLogGroupName = groupName

	podGroups, err := promptForPodGroups(ocmClient, "CloudWatch")
	if err != nil {
		return nil, err
	}
	cloudWatchConfig.GroupsLogVersions = podGroups

	applications, err := promptForApplications("CloudWatch", podGroups)
	if err != nil {
		return nil, err
	}
	if applications == "" {
		cloudWatchConfig.Applications = []string{}
	} else {
		cloudWatchConfig.Applications = strings.Split(applications, ",")
	}
	return &cloudWatchConfig, nil
}

func interactiveS3(ocmClient *ocm.Client, defaultBucketName string, defaultBucketPrefix string) (
	*logforwarding.S3LogForwarderConfig, error) {

	s3Config := logforwarding.S3LogForwarderConfig{}
	bucketPrefix, err := interactive.GetString(interactive.Input{
		Question: "S3 Bucket prefix",
		Help:     "The identifiable prefix to prepend to the S3 Bucket used for log forwarding",
		Default:  defaultBucketPrefix,
		Required: false,
	})
	if err != nil {
		return nil, err
	}
	s3Config.S3ConfigBucketPrefix = bucketPrefix

	bucketName, err := interactive.GetString(interactive.Input{
		Question: "S3 Bucket name",
		Help:     "The identifiable name to append to the S3 Bucket used for log forwarding",
		Default:  defaultBucketName,
		Required: true,
	})
	if err != nil {
		return nil, err
	}
	s3Config.S3ConfigBucketName = bucketName

	podGroups, err := promptForPodGroups(ocmClient, "S3")
	if err != nil {
		return nil, err
	}
	s3Config.GroupsLogVersions = podGroups

	applications, err := promptForApplications("S3", podGroups)
	if err != nil {
		return nil, err
	}
	if applications == "" {
		s3Config.Applications = []string{}
	} else {
		s3Config.Applications = strings.Split(applications, ",")
	}

	return &s3Config, nil
}

func promptForApplications(t string, podGroups []string) (applications string, err error) {
	applications, err = interactive.GetString(interactive.Input{
		Question: fmt.Sprintf("%s Log forwarding applications", t),
		Help: fmt.Sprintf("Which applications to forward to %s, please use a comma-separated list "+
			"(example: \"audit-webhook,cluster-api\")", t),
		Default:  "",
		Required: len(podGroups) == 0,
	})

	return
}

func promptForPodGroups(ocmClient *ocm.Client, t string) (podGroups []string, err error) {
	availableOptions, err := ocmClient.GetLogForwarderGroupVersions()
	if err != nil {
		return
	}
	podGroups, err = interactive.GetMultipleOptions(interactive.Input{
		Question: fmt.Sprintf("%s Log forwarding pod groups", t),
		Help: fmt.Sprintf("Which preset pod group of logs to forward to '%s'. Available options:\n"+
			logforwarding.ConstructPodGroupsHelpMessage(availableOptions), t),
		Options:  logforwarding.ConstructPodGroupsInteractiveOptions(availableOptions),
		Required: false,
	})
	return
}
