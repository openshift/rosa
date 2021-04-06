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
	"github.com/openshift/rosa/pkg/arguments"

	"github.com/openshift/rosa/pkg/aws"
	clusterprovider "github.com/openshift/rosa/pkg/cluster"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/ocm/config"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	deleteStack      bool
	disableSCPChecks bool
	region           string
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

	flags.BoolVar(
		&args.deleteStack,
		"delete-stack",
		false,
		"Deletes stack template applied to your AWS account during the 'init' command.\n",
	)

	flags.BoolVar(
		&args.disableSCPChecks,
		"disable-scp-checks",
		false,
		"Indicates if cloud permission checks are disabled when attempting installation of the cluster.",
	)

	// Force-load all flags from `login` into `init`
	flags.AddFlagSet(login.Cmd.Flags())

	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
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

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().Logger(logger).Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer ocmConnection.Close()
	ocmClient := ocmConnection.ClustersMgmt().V1()

	// Validate AWS credentials for current user
	reporter.Infof("Validating AWS credentials...")
	ok, isSTS, err := client.ValidateCredentials()
	if err != nil {
		ocm.LogEvent(ocmClient, "ROSAInitCredentialsFailed")
		if isSTS {
			ocm.LogEvent(ocmClient, "ROSAInitCredentialsSTS")
		}
		reporter.Errorf("Error validating AWS credentials: %v", err)
		os.Exit(1)
	}
	if !ok {
		ocm.LogEvent(ocmClient, "ROSAInitCredentialsInvalid")
		reporter.Errorf("AWS credentials are invalid")
		os.Exit(1)
	}
	reporter.Infof("AWS credentials are valid!")
	clustersCollection := ocmClient.Clusters()

	// Delete CloudFormation stack and exit
	if args.deleteStack {
		reporter.Infof("Deleting cluster administrator user '%s'...", aws.AdminUserName)

		// Get creator ARN to determine existing clusters:
		awsCreator, err := client.GetCreator()
		if err != nil {
			ocm.LogEvent(ocmClient, "ROSAInitGetCreatorFailed")
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
			ocm.LogEvent(ocmClient, "ROSAInitDeleteStackFailed")
			reporter.Errorf("Failed to delete user '%s': %v", aws.AdminUserName, err)
			os.Exit(1)
		}

		reporter.Infof("Admin user '%s' deleted successfully!", aws.AdminUserName)
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
		reporter.Infof("Skipping AWS SCP policies check")
	}

	// Validate AWS quota
	// Call `verify quota` as part of init
	quota.Cmd.Run(cmd, argv)

	// Ensure that there is an AWS user to create all the resources needed by the cluster:
	reporter.Infof("Ensuring cluster administrator user '%s'...", aws.AdminUserName)
	created, err := client.EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName, aws.AdminUserName)
	if err != nil {
		ocm.LogEvent(ocmClient, "ROSAInitCreateStackFailed")
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
		ocm.LogEvent(ocmClient, "ROSAInitSCPPoliciesFailed")
		reporter.Errorf("Failed to verify permissions for user '%s': %v", target, err)
		os.Exit(1)
	}
	reporter.Infof("AWS SCP policies ok")

	// Check whether the user can create a basic cluster
	reporter.Infof("Validating cluster creation...")
	err = simulateCluster(clustersCollection, args.region)
	if err != nil {
		ocm.LogEvent(ocmClient, "ROSAInitDryRunFailed")
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
