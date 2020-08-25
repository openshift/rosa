/*
Copyright (c) 2020 Red Hat, Inc.

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

package regions

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
)

func GetRegions(client *cmv1.Client) (regions []*cmv1.CloudRegion, err error) {
	// Create logger:
	logger, err := logging.NewLogger().Build()
	if err != nil {
		return nil, fmt.Errorf("Unable to create AWS logger: %v", err)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().Logger(logger).Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create AWS client: %v", err)
	}

	// Get AWS credentials from the cloudformation stack:
	awsAccessKey, err := awsClient.GetAccessKeyFromStack(aws.OsdCcsAdminStackName)
	if err != nil {
		return nil, fmt.Errorf("Failed to get access keys for user '%s': %v", aws.AdminUserName, err)
	}

	// Build cmv1.AWS object to get list of available regions:
	awsCredentials, err := cmv1.NewAWS().
		AccessKeyID(awsAccessKey.AccessKeyID).
		SecretAccessKey(awsAccessKey.SecretAccessKey).
		Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to build AWS credentials for user '%s': %v", aws.AdminUserName, err)
	}

	collection := client.CloudProviders().CloudProvider("aws").AvailableRegions()
	page := 1
	size := 100
	for {
		var response *cmv1.AvailableRegionsSearchResponse
		response, err = collection.Search().
			Page(page).
			Size(size).
			Body(awsCredentials).
			Send()
		if err != nil {
			return
		}
		regions = append(regions, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}
	return
}
