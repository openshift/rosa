package logforwarding

import (
	"fmt"
	"strings"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
)

// Options for initial interactive prompt
const skip = "Skip"
const cloudWatch = "CloudWatch"
const s3 = "S3"
const both = "Both"

func InteractiveLogForwardingConfig(ocmClient *ocm.Client, mainHelpMsg string) ([]*ocm.LogForwarderConfig, error) {
	con, err := interactive.GetOption(interactive.Input{
		Question: "Enabled log forwarding",
		Help:     mainHelpMsg,
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

	result := make([]*ocm.LogForwarderConfig, 0)

	if con == cloudWatch || con == both {
		result = append(result, &ocm.LogForwarderConfig{})
		err = interactiveCloudWatch(ocmClient, result[0])
		if err != nil {
			return nil, err
		}
	}
	if con == s3 || con == both {
		result = append(result, &ocm.LogForwarderConfig{})
		err = interactiveS3(ocmClient, result[len(result)-1])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func interactiveCloudWatch(ocmClient *ocm.Client, config *ocm.LogForwarderConfig) error {
	roleArn, err := interactive.GetString(interactive.Input{
		Question: "CloudWatch Log forwarding role ARN",
		Help:     "The role ARN which forwards logs to CloudWatch",
		Required: true,
	})
	if err != nil {
		return err
	}
	config.CloudWatchLogRoleArn = roleArn

	groupName, err := interactive.GetString(interactive.Input{
		Question: "CloudWatch log group name",
		Help:     "The name of the group on CloudWatch which will contain the logs",
		Required: true,
	})
	if err != nil {
		return err
	}
	config.CloudWatchLogGroupName = groupName

	podGroups, err := promptForPodGroups(ocmClient, "CloudWatch")
	if err != nil {
		return err
	}
	config.GroupsLogVersion = podGroups

	applications, err := promptForApplications("CloudWatch")
	if err != nil {
		return err
	}
	config.Applications = strings.Split(applications, ",")

	return nil
}

func interactiveS3(ocmClient *ocm.Client, config *ocm.LogForwarderConfig) error {
	bucketPrefix, err := interactive.GetString(interactive.Input{
		Question: "S3 Bucket prefix",
		Help:     "The identifiable prefix to prepend to the S3 Bucket used for log forwarding",
		Required: true,
	})
	if err != nil {
		return err
	}
	config.S3ConfigBucketPrefix = bucketPrefix

	bucketName, err := interactive.GetString(interactive.Input{
		Question: "S3 Bucket name",
		Help:     "The identifiable name to append to the S3 Bucket used for log forwarding",
		Required: true,
	})
	if err != nil {
		return err
	}
	config.S3ConfigBucketName = bucketName

	podGroups, err := promptForPodGroups(ocmClient, "S3")
	if err != nil {
		return err
	}
	config.GroupsLogVersion = podGroups

	applications, err := promptForApplications("S3")
	if err != nil {
		return err
	}
	config.Applications = strings.Split(applications, ",")

	return nil
}

func promptForApplications(t string) (applications string, err error) {
	applications, err = interactive.GetString(interactive.Input{
		Question: fmt.Sprintf("%s Log forwarding applications", t),
		Help: fmt.Sprintf("Which applications to forward to %s, please use a comma-separated list "+
			"(example: \"audit-webhook,cluster-api\")", t),
		Default:  "",
		Required: true,
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
			constructPodGroupsHelpMessage(availableOptions), t),
		Options:  constructPodGroupsInteractiveOptions(availableOptions),
		Required: true,
	})
	return
}
