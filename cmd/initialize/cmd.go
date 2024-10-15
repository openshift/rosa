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

package initialize

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/login"
	"github.com/openshift/rosa/cmd/verify/oc"
	"github.com/openshift/rosa/cmd/verify/permissions"
	"github.com/openshift/rosa/cmd/verify/quota"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/region"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	dlt              bool
	disableSCPChecks bool
	sts              bool
	region           string
	// Use local AWS credentials instead of the 'osdCcsAdmin' user
	useLocalCredentials bool
}

var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Applies templates to support Red Hat OpenShift Service on AWS",
	Long: "Applies templates to support Red Hat OpenShift Service on AWS. If you are not\n" +
		"yet logged in to OCM, it will prompt you for credentials.",
	Example: `  # Configure your AWS account to allow IAM (non-STS) ROSA clusters
  rosa init

  # Configure a new AWS account using pre-existing OCM credentials
  rosa init --token=$OFFLINE_ACCESS_TOKEN`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	flags.BoolVar(
		&args.dlt,
		"delete-stack",
		false,
		"Deletes stack template applied to your AWS account during the 'init' command.",
	)
	flags.MarkDeprecated("delete-stack", "use --delete instead")

	flags.BoolVar(
		&args.dlt,
		"delete",
		false,
		"Deletes stack template applied to your AWS account during the 'init' command.",
	)

	flags.BoolVar(
		&args.disableSCPChecks,
		"disable-scp-checks",
		false,
		"Indicates if cloud permission checks are disabled when attempting installation of the cluster.",
	)

	flags.BoolVar(
		&args.useLocalCredentials,
		"use-local-credentials",
		false,
		"Use local AWS credentials instead of the 'osdCcsAdmin' user. This is not supported.",
	)
	flags.MarkHidden("use-local-credentials")

	// Force-load all flags from `login` into `init`
	flags.AddFlagSet(login.Cmd.Flags())

	arguments.AddProfileFlag(flags)

	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	// If necessary, call `login` as part of `init`. We do this before
	// other validations to get the prompt out of the way before performing
	// longer checks.
	err := login.Call(cmd, argv, r.Reporter)
	if err != nil {
		r.Reporter.Errorf("Failed to login to OCM: %v", err)
		os.Exit(1)
	}

	// Get AWS region
	awsRegion, err := aws.GetRegion(arguments.GetRegion())
	if err != nil {
		r.Reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}
	supportedRegions, err := r.OCMClient.GetDatabaseRegionList()
	if err != nil {
		r.Reporter.Errorf("Unable to retrieve supported regions: %v", err)
	}
	if !helper.Contains(supportedRegions, awsRegion) {
		r.Reporter.Errorf("Unsupported region '%s', available regions: %s",
			awsRegion, helper.SliceToSortedString(supportedRegions))
		os.Exit(1)
	}
	// Create the AWS client:
	client, err := aws.NewClient().
		Logger(r.Logger).
		Region(awsRegion).
		UseLocalCredentials(args.useLocalCredentials).
		Build()

	if err != nil {
		// FIXME Hack to capture errors due to using STS accounts
		if strings.Contains(fmt.Sprintf("%s", err), "STS") {
			r.OCMClient.LogEvent("ROSAInitCredentialsSTS", nil)
		}
		r.Reporter.Errorf("Error creating AWS client: %v", err)
		os.Exit(1)
	}

	// Validate AWS credentials for current user
	r.Reporter.Infof("Validating AWS credentials...")
	ok, err := client.ValidateCredentials()
	if err != nil {
		r.OCMClient.LogEvent("ROSAInitCredentialsFailed", nil)
		r.Reporter.Errorf("Error validating AWS credentials: %v", err)
		os.Exit(1)
	}
	if !ok {
		r.OCMClient.LogEvent("ROSAInitCredentialsInvalid", nil)
		r.Reporter.Errorf("AWS credentials are invalid")
		os.Exit(1)
	}
	r.Reporter.Infof("AWS credentials are valid!")

	cfClient := aws.GetAWSClientForUserRegion(r.Reporter, r.Logger, supportedRegions, args.useLocalCredentials)

	// Delete CloudFormation stack and exit
	if args.dlt {
		if !confirm.Confirm("delete cluster administrator user '%s'", aws.AdminUserName) {
			os.Exit(0)
		}
		r.Reporter.Infof("Deleting cluster administrator user '%s'...", aws.AdminUserName)
		err = deleteStack(cfClient, r.OCMClient)
		if err != nil {
			r.Reporter.Errorf("%v", err)
			os.Exit(1)
		}

		r.Reporter.Infof("Admin user '%s' deleted successfully!", aws.AdminUserName)
		os.Exit(0)
	}

	// Validate AWS SCP/IAM Permissions
	// Call `verify permissions` as part of init
	// Skip this check if --disable-scp-checks is true
	// If SCP policies conditions restrict an AWS accounts permissions by region
	// AWS will fail all policy simulation checks
	// Validate AWS credentials for current user
	if !args.disableSCPChecks {
		permissions.Cmd.Run(cmd, argv)
	} else {
		r.Reporter.Infof("Skipping AWS SCP policies check")
	}

	// Validate AWS quota
	// Call `verify quota` as part of init
	quota.Cmd.Run(cmd, argv)

	// Ensure that there is an AWS user to create all the resources needed by the cluster:
	r.Reporter.Infof("Ensuring cluster administrator user '%s'...", aws.AdminUserName)
	created, err := cfClient.EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName, aws.AdminUserName, awsRegion)
	if err != nil {
		r.OCMClient.LogEvent("ROSAInitCreateStackFailed", nil)
		r.Reporter.Errorf("Failed to create user '%s': %v", aws.AdminUserName, err)
		os.Exit(1)
	}
	if created {
		r.Reporter.Infof("Admin user '%s' created successfully!", aws.AdminUserName)
	} else {
		r.Reporter.Infof("Admin user '%s' already exists!", aws.AdminUserName)
	}

	// Check if osdCcsAdmin has right permissions
	// Skip this check if --disable-scp-checks is true
	if !args.disableSCPChecks {
		r.Reporter.Infof("Validating SCP policies for '%s'...", aws.AdminUserName)
		target := aws.AdminUserName

		policies, err := r.OCMClient.GetPolicies("OSDSCPPolicy")
		if err != nil {
			r.Reporter.Errorf("Failed to get 'osdscppolicy' for '%s': %v", aws.AdminUserName, err)
			os.Exit(1)
		}
		isValid, err := client.ValidateSCP(&target, policies)
		if !isValid {
			r.OCMClient.LogEvent("ROSAInitSCPPoliciesFailed", nil)
			r.Reporter.Errorf("Failed to verify permissions for user '%s': %v", target, err)
			os.Exit(1)
		}
		r.Reporter.Infof("AWS SCP policies ok")
	} else {
		r.Reporter.Infof("Skipping AWS SCP policies check for '%s'...", aws.AdminUserName)
	}

	// Check whether the user can create a basic cluster
	r.Reporter.Infof("Validating cluster creation...")
	err = simulateCluster(cfClient, r.OCMClient, region.Region())
	if err != nil {
		r.OCMClient.LogEvent("ROSAInitDryRunFailed", nil)
		r.Reporter.Warnf("Cluster creation failed. "+
			"If you create a cluster, it should fail with the following error:\n%s", err)
	} else {
		r.Reporter.Infof("Cluster creation valid")
	}

	// Verify version of `oc`
	oc.Cmd.Run(cmd, argv)
}

func deleteStack(awsClient aws.Client, ocmClient *ocm.Client) error {
	// Get creator ARN to determine existing clusters:
	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		ocmClient.LogEvent("ROSAInitGetCreatorFailed", nil)
		return fmt.Errorf("Failed to get AWS creator: %v", err)
	}

	// Check whether the account has clusters:
	hasClusters, err := ocmClient.HasClusters(awsCreator)
	if err != nil {
		return fmt.Errorf("Failed to check for clusters: %v", err)
	}

	if hasClusters {
		return fmt.Errorf("Failed to delete '%s': User still has clusters", aws.AdminUserName)
	}

	// Delete the CloudFormation stack
	err = awsClient.DeleteOsdCcsAdminUser(aws.OsdCcsAdminStackName)
	if err != nil {
		ocmClient.LogEvent("ROSAInitDeleteStackFailed", nil)
		return fmt.Errorf("Failed to delete user '%s': %v", aws.AdminUserName, err)
	}

	return nil
}

func simulateCluster(awsClient aws.Client, ocmClient *ocm.Client, region string) error {
	dryRun := true
	if region == "" {
		region = aws.DefaultRegion
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		ocmClient.LogEvent("ROSAInitGetCreatorFailed", nil)
		return fmt.Errorf("Failed to get AWS creator: %v", err)
	}

	awsAccessKey, err := awsClient.GetLocalAWSAccessKeys()
	if err != nil {
		ocmClient.LogEvent("ROSAInitGetLocalAWSAccessKeysFailed", nil)
		return fmt.Errorf("Failed to get AWS Access Key: %v", err)
	}
	spec := ocm.Spec{
		Name:           "rosa-init",
		Region:         region,
		DryRun:         &dryRun,
		DefaultIngress: ocm.NewDefaultIngressSpec(),
		AWSCreator:     awsCreator,
		AWSAccessKey:   awsAccessKey,
	}

	_, err = ocmClient.CreateCluster(spec)
	if err != nil {
		return err
	}

	return nil
}
