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

func BuildLogForwarder(logForwarderConfig *cmv1.LogForwarder) *cmv1.LogForwarderBuilder {
	logForwardbldr := cmv1.NewLogForwarder()
	if logForwarderConfig != nil {
		if len(logForwarderConfig.Applications()) > 0 {
			logForwardbldr.Applications(logForwarderConfig.Applications()...)
		}
		if logForwarderConfig.Cloudwatch() != nil && (logForwarderConfig.Cloudwatch().LogGroupName() != "" ||
			logForwarderConfig.Cloudwatch().LogDistributionRoleArn() != "") {
			logForwardbldr.Cloudwatch(cmv1.NewLogForwarderCloudWatchConfig().
				LogDistributionRoleArn(logForwarderConfig.Cloudwatch().LogDistributionRoleArn()).
				LogGroupName(logForwarderConfig.Cloudwatch().LogGroupName()))
		}
		if len(logForwarderConfig.Groups()) > 0 {
			logForwarderGroupBlds := make([]*cmv1.LogForwarderGroupBuilder, 0)
			for _, group := range logForwarderConfig.Groups() {
				logForwarderGroupBlds = append(logForwarderGroupBlds, cmv1.NewLogForwarderGroup().
					Version(group.Version()).ID(group.ID()))
			}
			logForwardbldr.Groups(logForwarderGroupBlds...)
		}
		if logForwarderConfig.S3() != nil && (logForwarderConfig.S3().BucketName() != "" ||
			logForwarderConfig.S3().BucketPrefix() != "") {
			logForwardbldr.S3(cmv1.NewLogForwarderS3Config().BucketName(logForwarderConfig.S3().BucketName()).
				BucketPrefix(logForwarderConfig.S3().BucketPrefix()))
		}
	}

	return logForwardbldr
}

func (c *Client) GetLogForwarder(clusterID string) (*cmv1.LogForwarder, error) {
	var LogForwarderList []*cmv1.LogForwarder
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

func (c *Client) GetLogForwarderGroupVersions() ([]*cmv1.LogForwarderGroupVersions, error) {
	response, err := c.ocm.ClustersMgmt().V1().LogForwarding().Groups().List().Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Items().Slice(), nil
}
