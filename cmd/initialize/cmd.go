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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/login"
	"github.com/openshift/rosa/cmd/verify/oc"
	"github.com/openshift/rosa/cmd/verify/permissions"
	"github.com/openshift/rosa/cmd/verify/quota"

	"github.com/openshift/rosa/pkg/aws"
	clusterprovider "github.com/openshift/rosa/pkg/cluster"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/ocm/config"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	region      string
	deleteStack bool
}

var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Applies templates to support Red Hat OpenShift Service on AWS",
	Long: "Applies templates to support Red Hat OpenShift Service on AWS. If you are not\n" +
		"yet logged in to OCM, it will prompt you for credentials.",
	Example: `  # Configure your AWS account to allow ROSA clusters
  rosa init

  # Configure a new AWS account using pre-existing OCM credentials
  rosa init --token=$OFFLINE_ACCESS_TOKEN`,
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

	// Create the AWS client:
	client, err := aws.NewClient().
		Logger(logger).
		Region(aws.DefaultRegion).
		Build()
	if err != nil {
		reporter.Errorf("Error creating AWS client: %v", err)
		os.Exit(1)
	}

	// If necessary, call `login` as part of `init`. We do this before
	// other validations to get the prompt out of the way before performing
	// longer checks.
	loginFlags := []string{"token-url", "client-id", "client-secret", "scope", "env", "token", "insecure"}
	hasLoginFlags := false
	// Check if the user set login flags
	for _, loginFlag := range loginFlags {
		if cmd.Flags().Changed(loginFlag) {
			hasLoginFlags = true
			break
		}
	}
	if hasLoginFlags {
		// Always force login if user sets login flags
		login.Cmd.Run(cmd, argv)
	} else {
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

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().Logger(logger).Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer ocmConnection.Close()
	clustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()

	// Delete CloudFormation stack and exit
	if args.deleteStack {
		reporter.Infof("Deleting cluster administrator user '%s'...", aws.AdminUserName)

		// Get creator ARN to determine existing clusters:
		awsCreator, err := client.GetCreator()
		if err != nil {
			reporter.Errorf("Failed to get AWS creator: %v", err)
			os.Exit(1)
		}

		// Check whether the account has clusters:
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
	created, err := client.EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName, aws.AdminUserName)
	if err != nil {
		reporter.Errorf("Failed to create user '%s': %v", aws.AdminUserName, err)
		os.Exit(1)
	}
	if created {
		reporter.Infof("Admin user '%s' created successfully!", aws.AdminUserName)
	} else {
		reporter.Infof("Admin user '%s' already exists!", aws.AdminUserName)
	}

	// Check if osdCcsAdmin has right permissions
	reporter.Infof("Validating SCP policies for '%s'...", aws.AdminUserName)
	target := aws.AdminUserName
	isValid, err := client.ValidateSCP(&target)
	if !isValid {
		reporter.Errorf("Failed to verify permissions for user '%s': %v", target, err)
		os.Exit(1)
	}
	reporter.Infof("AWS SCP policies ok")

	// Check whether the user can create a basic cluster
	reporter.Infof("Validating cluster creation...")
	err = simulateCluster(clustersCollection, args.region)
	if err != nil {
		reporter.Warnf("Cluster creation failed. "+
			"If you create a cluster, it should fail with the following error:\n%s", err)
	} else {
		reporter.Infof("Cluster creation valid")
	}

	oc.Cmd.Run(cmd, argv)
}

func simulateCluster(client *cmv1.ClustersClient, region string) error {
	dryRun := true
	if region == "" {
		region = aws.DefaultRegion
	}
	spec := clusterprovider.Spec{
		Name:   "rosa-init",
		Region: region,
		DryRun: &dryRun,
	}

	_, err := clusterprovider.CreateCluster(client, spec)
	if err != nil {
		return err
	}

	return nil
}
