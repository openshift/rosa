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
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/moactl/cmd/login"
	"github.com/openshift/moactl/cmd/verify/oc"
	"github.com/openshift/moactl/cmd/verify/permissions"
	"github.com/openshift/moactl/cmd/verify/quota"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	"github.com/openshift/moactl/pkg/ocm/config"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var args struct {
	region      string
	deleteStack bool
}

var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Applies templates to support Managed OpenShift on AWS clusters",
	Long: "Applies templates to support Managed OpenShift on AWS clusters. If you are not\n" +
		"yet logged in to OCM, it will prompt you for credentials.",
	Example: `  # Configure your AWS account to allow MOA clusters
  moactl init

  # Configure a new AWS account using pre-existing OCM credentials
  moactl init --token=$OFFLINE_ACCESS_TOKEN`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(
		&args.region,
		"region",
		"r",
		"",
		"AWS region in which verify quota and permissions (overrides the AWS_REGION environment variable)",
	)

	flags.BoolVar(
		&args.deleteStack,
		"delete-stack",
		false,
		"Deletes stack template applied to your AWS account during the 'init' command.\n",
	)

	// Force-load all flags from `login` into `init`
	flags.AddFlagSet(login.Cmd.Flags())
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Get AWS region
	region, err := aws.GetRegion(args.region)
	if err != nil {
		reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}

	// Create the AWS client:
	client, err := aws.NewClient().
		Logger(logger).
		Region(region).
		Build()
	if err != nil {
		reporter.Errorf("Error creating AWS client: %v", err)
		os.Exit(1)
	}

	// If necessary, call `login` as part of `init`. We do this before
	// other validations to get the prompt out of the way before performing
	// longer checks.
	if cmd.Flags().NFlag() == 0 || (args.deleteStack && cmd.Flags().NFlag() == 1) {
		// Verify if user is already logged in:
		isLoggedIn := false
		cfg, err := config.Load()
		if err != nil {
			reporter.Errorf("Failed to load config file: %v", err)
			os.Exit(1)
		}
		if cfg != nil {
			// Check that credentials in the config file are valid
			isLoggedIn, err = cfg.Armed()
			if err != nil {
				reporter.Errorf("Failed to determine if user is logged in: %v", err)
				os.Exit(1)
			}
		}

		if isLoggedIn {
			username, err := cfg.GetData("username")
			if err != nil {
				reporter.Errorf("Failed to get username: %v", err)
				os.Exit(1)
			}

			reporter.Infof("Logged in as '%s' on '%s'", username, cfg.URL)
		} else {
			login.Cmd.Run(cmd, argv)
		}
	} else {
		// Always force login if user sets login flags
		login.Cmd.Run(cmd, argv)
	}

	// Validate AWS credentials for current user
	reporter.Infof("Validating AWS credentials...")
	ok, err := client.ValidateCredentials()
	if err != nil {
		reporter.Errorf("Error validating AWS credentials: %v", err)
		os.Exit(1)
	}
	if !ok {
		reporter.Errorf("AWS credentials are invalid")
		os.Exit(1)
	}
	reporter.Infof("AWS credentials are valid!")

	// Delete CloudFormation stack and exit
	if args.deleteStack {
		reporter.Infof("Deleting cluster administrator user '%s'...", aws.AdminUserName)

		// Create the client for the OCM API:
		ocmConnection, err := ocm.NewConnection().
			Logger(logger).
			Build()
		if err != nil {
			reporter.Errorf("Failed to create OCM connection: %v", err)
			os.Exit(1)
		}
		defer func() {
			err = ocmConnection.Close()
			if err != nil {
				reporter.Errorf("Failed to close OCM connection: %v", err)
			}
		}()

		// Get creator ARN to determine existing clusters:
		awsCreator, err := client.GetCreator()
		if err != nil {
			reporter.Errorf("Failed to get AWS creator: %v", err)
			os.Exit(1)
		}

		// Check whether the account has clusters:
		clustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()
		hasClusters, err := ocm.HasClusters(clustersCollection, awsCreator.ARN)
		if err != nil {
			reporter.Errorf("Failed to check for clusters: %v", err)
			os.Exit(1)
		}

		if hasClusters {
			reporter.Errorf(
				"Failed to delete '%s': User still has clusters.",
				aws.AdminUserName)
			os.Exit(1)
		}

		// Delete the CloudFormation stack
		err = client.DeleteOsdCcsAdminUser(aws.OsdCcsAdminStackName)
		if err != nil {
			reporter.Errorf("Failed to delete user '%s': %v", aws.AdminUserName, err)
			os.Exit(1)
		}

		reporter.Infof("Admin user '%s' deleted successfully!", aws.AdminUserName)
		os.Exit(0)
	}

	// Validate AWS SCP/IAM Permissions
	// Call `verify permissions` as part of init
	permissions.Cmd.Run(cmd, argv)

	// Validate AWS quota
	// Call `verify quota` as part of init
	quota.Cmd.Run(cmd, argv)

	// Ensure that there is an AWS user to create all the resources needed by the cluster:
	reporter.Infof("Ensuring cluster administrator user '%s'...", aws.AdminUserName)
	created, err := client.EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName)
	if err != nil {
		reporter.Errorf("Failed to create user '%s': %v", aws.AdminUserName, err)
		os.Exit(1)
	}
	if created {
		reporter.Infof("Admin user '%s' created successfully!", aws.AdminUserName)
	} else {
		reporter.Infof("Admin user '%s' already exists!", aws.AdminUserName)
	}

	oc.Cmd.Run(cmd, argv)
}
