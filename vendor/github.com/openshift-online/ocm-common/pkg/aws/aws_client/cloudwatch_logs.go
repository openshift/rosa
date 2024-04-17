package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) DescribeLogGroupsByName(logGroupName string) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	output, err := client.CloudWatchLogsClient.DescribeLogGroups(context.TODO(), &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: &logGroupName,
	})
	if err != nil {
		log.LogError("Got error describe log group:%s ", err)
	}
	return output, err
}

func (client *AWSClient) DescribeLogStreamByName(logGroupName string) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	output, err := client.CloudWatchLogsClient.DescribeLogStreams(context.TODO(), &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: &logGroupName,
	})
	if err != nil {
		log.LogError("Got error describe log stream: %s", err)
	}
	return output, err
}

func (client *AWSClient) DeleteLogGroupByName(logGroupName string) (*cloudwatchlogs.DeleteLogGroupOutput, error) {
	output, err := client.CloudWatchLogsClient.DeleteLogGroup(context.TODO(), &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: &logGroupName,
	})
	if err != nil {
		log.LogError("Got error delete log group: %s", err)
	}
	return output, err
}
