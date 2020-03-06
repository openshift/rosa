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

package create

import (
	"fmt"
	"os"
	"time"

	"github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	"gitlab.cee.redhat.com/service/moactl/pkg/properties"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create cluster",
	Long:  "Create cluster.",
	Run:   run,
}

func run(_ *cobra.Command, argv []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create reporter: %v\n", err)
		os.Exit(1)
	}

	// Create the logger:
	logger, err := logging.NewLogger().Build()
	if err != nil {
		reporter.Errorf("Can't create logger: %v", err)
		os.Exit(1)
	}

	// Check command line arguments:
	if len(argv) != 1 {
		reporter.Errorf(
			"Expected exactly one command line parameter containing the name " +
				"of the cluster",
		)
		os.Exit(1)
	}
	clusterName := argv[0]

	// Check that there is an OCM token in the environment. This will not be needed once we are
	// able to derive OCM credentials from AWS credentials.
	ocmToken := os.Getenv("OCM_TOKEN")
	if ocmToken == "" {
		reporter.Errorf("Environment variable 'OCM_TOKEN' is not set")
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Can't create AWS client: %v", err)
		os.Exit(1)
	}

	awsRegion := awsClient.GetRegion()
	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Can't get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the AWS user that will be used to create all the resources needed by the cluster:
	reporter.Infof("Creating cluster administrator user '%s'", awsAdminName)
	err = awsClient.CreateUser(awsAdminName, clusterName)
	if err != nil {
		reporter.Errorf("Can't create user '%s': %v", awsAdminName, err)
		os.Exit(1)
	}

	// Create the access key for the AWS user:
	awsAccessKey, err := awsClient.CreateAccessKey(awsAdminName)
	if err != nil {
		reporter.Errorf("Can't create access keys for user '%s'", awsAdminName)
		os.Exit(1)
	}
	reporter.Infof("Access key identifier is '%s'", awsAccessKey.AccessKeyID)
	reporter.Infof("Secret access key is '%s'", awsAccessKey.SecretAccessKey)

	// The created access key isn't immediately active, so we need to wait a bit till it is:
	reporter.Infof("Waiting for access key to be active")
	time.Sleep(10 * time.Second)

	// Create the client for the OCM API:
	ocmLogger, err := logging.NewOCMLogger().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Can't create OCM logger: %v", err)
		os.Exit(1)
	}
	ocmConnection, err := sdk.NewConnectionBuilder().
		Logger(ocmLogger).
		Tokens(ocmToken).
		URL("https://api.stage.openshift.com").
		Build()
	if err != nil {
		reporter.Errorf("Can't create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Can't close OCM connection: %v", err)
		}
	}()

	// Create the cluster:
	ocmCluster, err := cmv1.NewCluster().
		Name(clusterName).
		BYOC(true).
		CloudProvider(
			cmv1.NewCloudProvider().
				ID("aws"),
		).
		Region(
			cmv1.NewCloudRegion().
				ID(awsRegion),
		).
		AWS(
			cmv1.NewAWS().
				AccountID(awsCreator.AccountID).
				AccessKeyID(awsAccessKey.AccessKeyID).
				SecretAccessKey(awsAccessKey.SecretAccessKey),
		).
		Properties(map[string]string{
			properties.CreatorARN: awsCreator.ARN,
		}).
		Build()
	if err != nil {
		reporter.Errorf("Can't create description of cluster: %v", err)
		os.Exit(1)
	}
	createClusterResponse, err := ocmConnection.ClustersMgmt().V1().Clusters().Add().
		Body(ocmCluster).
		Send()
	if err != nil {
		reporter.Errorf("Can't create cluster: %v", err)
		os.Exit(1)
	}
	ocmCluster = createClusterResponse.Body()
	ocmClusterID := ocmCluster.ID()
	ocmClusterName := ocmCluster.Name()
	reporter.Infof(
		"Creating cluster with identifier '%s' and name '%s'",
		ocmClusterID, ocmClusterName,
	)

	// Add a tag to the AWS administrator user containing the identifier of the cluster:
	err = awsClient.TagUser(awsAdminName, ocmClusterID)
	if err != nil {
		reporter.Infof(
			"Can't add cluster identifier tag to user '%s'",
			awsAdminName,
		)
	}
}

// Name of the AWS user that will be used to create all the resources of the cluster:
const awsAdminName = "osdCcsAdmin"
