/*
Copyright (c) 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm

import (
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func BuildLogForwader(logForwarderConfig *LogForwarderConfig) *cmv1.LogForwarderBuilder {
	logForwardbldr := cmv1.NewLogForwarder()
	if logForwarderConfig != nil {
		if len(logForwarderConfig.Applications) > 0 {
			logForwardbldr.Applications(logForwarderConfig.Applications...)
		}
		if logForwarderConfig.CloudWatchLogGroupName != "" || logForwarderConfig.CloudWatchLogRoleArn != "" {
			logForwardbldr.CloudWatch(cmv1.NewLogForwarderCloudWatchConfig().
				LogDistributionRoleArn(logForwarderConfig.CloudWatchLogRoleArn).
				LogGroupName(logForwarderConfig.CloudWatchLogGroupName))
		}
		if len(logForwarderConfig.GroupsLogVersion) > 0 {
			logForwarderGroupBlds := make([]*cmv1.LogForwarderGroupBuilder, 0)
			for _, version := range logForwarderConfig.GroupsLogVersion {
				logForwarderGroupBlds = append(logForwarderGroupBlds, cmv1.NewLogForwarderGroup().Version(version))
			}
			logForwardbldr.Groups(logForwarderGroupBlds...)
		}
		if logForwarderConfig.S3ConfigBucketName != "" || logForwarderConfig.S3ConfigBucketPrefix != "" {
			logForwardbldr.S3(cmv1.NewLogForwarderS3Config().BucketName(logForwarderConfig.S3ConfigBucketName).
				BucketPrefix(logForwarderConfig.S3ConfigBucketPrefix))
		}
	}

	return logForwardbldr
}

func GetLogForwardConfig(logForwarder *cmv1.LogForwarder) *LogForwarderConfig {
	if logForwarder != nil {
		logForwarderConfig := &LogForwarderConfig{}

		logForwarderConfig.Applications = logForwarder.Applications()
		if _, ok := logForwarder.GetCloudWatch(); ok {
			logForwarderConfig.CloudWatchLogRoleArn = logForwarder.CloudWatch().LogDistributionRoleArn()
			logForwarderConfig.CloudWatchLogGroupName = logForwarder.CloudWatch().LogGroupName()
		}
		if groups, ok := logForwarder.GetGroups(); ok && len(groups) > 0 {
			versions := make([]string, 0)
			for _, group := range groups {
				versions = append(versions, group.Version())
			}
			logForwarderConfig.GroupsLogVersion = versions
		}
		if _, ok := logForwarder.GetS3(); ok {
			logForwarderConfig.S3ConfigBucketName = logForwarder.S3().BucketName()
			logForwarderConfig.S3ConfigBucketPrefix = logForwarder.S3().BucketPrefix()
		}

		return logForwarderConfig
	}
	return nil
}

func (c *Client) GetLogForwarder(clusterID string) (*cmv1.LogForwarder, error) {
	LogForwarderList := []*cmv1.LogForwarder{}
	collection := c.ocm.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		ControlPlane().LogForwarders().List()

	page := 1
	size := 1
	for {
		response, err := collection.
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return nil, handleErr(response.Error(), err)
		}
		LogForwarderList = append(LogForwarderList, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}

	if len(LogForwarderList) > 0 {
		return LogForwarderList[0], nil
	}

	return nil, nil
}

func (c *Client) SetLogForwarder(clusterID string,
	logForwarder *cmv1.LogForwarder) (*cmv1.LogForwarder, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).ControlPlane().
		LogForwarders().Add().Body(logForwarder).Send()

	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}
