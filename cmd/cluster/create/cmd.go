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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm/properties"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var env string

var Cmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create cluster",
	Long:  "Create cluster.",
	PreRun: func(cmd *cobra.Command, argv[] string) {
		env = cmd.Flags().Lookup("env").Value.String()
	},
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

	// Create the access key for the AWS user:
	awsAccessKey, err := awsClient.CreateAccessKey(aws.AdminUserName)
	if err != nil {
		reporter.Errorf("Can't create access keys for user '%s'", aws.AdminUserName)
		os.Exit(1)
	}
	reporter.Infof("Access key identifier is '%s'", awsAccessKey.AccessKeyID)
	reporter.Infof("Secret access key is '%s'", awsAccessKey.SecretAccessKey)

	// The created access key isn't immediately active, so we need to wait a bit till it is:
	reporter.Infof("Waiting for access key to be active")
	time.Sleep(10 * time.Second)

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().
		SetEnv(env).
		Logger(logger).
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

	// Add tags to the AWS administrator user containing the identifier and name of the cluster:
	err = awsClient.TagUser(aws.AdminUserName, ocmClusterID, ocmClusterName)
	if err != nil {
		reporter.Infof(
			"Can't add cluster tags to user '%s'",
			aws.AdminUserName,
		)
	}
}
