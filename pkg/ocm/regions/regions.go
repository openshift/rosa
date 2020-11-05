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
	"errors"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

func GetRegions(client *cmv1.Client) (regions []*cmv1.CloudRegion, err error) {
	// Retrieve AWS credentials from the local AWS user
	// pass these to OCM to validate what regions are available
	// in this AWS account

	// Build AWS client and retrieve credentials
	// This ensures we use the profile flag if passed to rosa
	// Create the AWS client:
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	awsClient, err := aws.NewClient().
		Logger(logger).
		Region(aws.DefaultRegion).
		Build()
	if err != nil {
		return nil, fmt.Errorf("Error creating AWS client: %v", err)
	}

	// Get AWS region
	currentAWSCreds, err := awsClient.GetIAMCredentials()

	if err != nil {
		return nil, fmt.Errorf("Failed to get local AWS credentials: %v", err)
	}

	// Build cmv1.AWS object to get list of available regions:
	awsCredentials, err := cmv1.NewAWS().
		AccessKeyID(currentAWSCreds.AccessKeyID).
		SecretAccessKey(currentAWSCreds.SecretAccessKey).
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
			errMsg := response.Error().Reason()
			if errMsg == "" {
				errMsg = err.Error()
			}
			return nil, errors.New(errMsg)
		}
		regions = append(regions, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}
	return
}

func GetRegionList(client *cmv1.Client, multiAZ bool) (regionList []string, regionAZ map[string]bool, err error) {
	regions, err := GetRegions(client)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve AWS regions: %s", err)
		return
	}

	regionAZ = make(map[string]bool, len(regions))

	for _, v := range regions {
		if !v.Enabled() {
			continue
		}
		if !multiAZ || v.SupportsMultiAZ() {
			regionList = append(regionList, v.ID())
		}
		regionAZ[v.ID()] = v.SupportsMultiAZ()
	}

	return
}
