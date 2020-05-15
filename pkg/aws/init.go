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

package aws

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/moactl/pkg/ocm"
	"github.com/openshift/moactl/pkg/utils"
)

// DeleteStack will delete the Cloud Formation stack applied by init.
func DeleteStack(clusterClient *cmv1.ClustersClient) error {
	// Create the reporter/logger
	reporter, logger, err := utils.CreateReporterAndLogger()
	if err != nil {
		return fmt.Errorf("unable to create reporter/logger: %v", err)
	}

	// Create the AWS client:
	client, err := NewClient().
		Logger(logger).
		Build()
	if err != nil {
		return fmt.Errorf("error creating AWS client: %v", err)
	}

	reporter.Infof("Deleting cluster administrator user '%s'...", AdminUserName)

	// Get creator ARN to determine existing clusters:
	awsCreator, err := client.GetCreator()
	if err != nil {
		return fmt.Errorf("failed to get AWS creator: %v", err)
	}

	// Check whether the account has clusters:
	hasClusters, err := ocm.HasClusters(clusterClient, awsCreator.ARN)
	if err != nil {
		return fmt.Errorf("failed to check for clusters: %v", err)
	}

	if hasClusters {
		return fmt.Errorf(
			"failed to delete '%s': user still has clusters",
			AdminUserName)
	}

	// Delete the CloudFormation stack
	err = client.DeleteOsdCcsAdminUser(OsdCcsAdminStackName)
	if err != nil {
		return fmt.Errorf("Failed to delete user '%s': %v", AdminUserName, err)
	}

	reporter.Infof("Admin user '%s' deleted successfuly!", AdminUserName)

	return nil
}

// CreateStack will create the Cloud Formation stack for init.
func CreateStack() error {
	// Create the reporter/logger
	reporter, logger, err := utils.CreateReporterAndLogger()
	if err != nil {
		return fmt.Errorf("unable to create reporter/logger: %v", err)
	}

	// Create the AWS client:
	client, err := NewClient().
		Logger(logger).
		Build()
	if err != nil {
		return fmt.Errorf("error creating AWS client: %v", err)
	}

	// Validate SCP policies for current user's account
	reporter.Infof("Validating SCP policies...")
	ok, err := client.ValidateSCP()
	if err != nil {
		return fmt.Errorf("error validating SCP policies: %v", err)
	}
	if !ok {
		reporter.Warnf("Failed to validate SCP policies. Will try to continue anyway...")
	}
	reporter.Infof("SCP/IAM permissions validated...")

	// Ensure that there is an AWS user to create all the resources needed by the cluster:
	reporter.Infof("Ensuring cluster administrator user '%s'...", AdminUserName)
	created, err := client.EnsureOsdCcsAdminUser(OsdCcsAdminStackName)
	if err != nil {
		return fmt.Errorf("failed to create user '%s': %v", AdminUserName, err)
	}
	if created {
		reporter.Infof("Admin user '%s' created successfuly!", AdminUserName)
	} else {
		reporter.Infof("Admin user '%s' already exists!", AdminUserName)
	}

	return nil
}
