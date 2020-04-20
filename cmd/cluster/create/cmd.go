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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm/properties"
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
	awsAccessKey, err := awsClient.GetAccessKeyFromStack(aws.OsdCcsAdminStackName)
	if err != nil {
		reporter.Errorf("Can't create access keys for user '%s'", aws.AdminUserName)
		os.Exit(1)
	}
	reporter.Debugf("Access key identifier is '%s'", awsAccessKey.AccessKeyID)
	reporter.Debugf("Secret access key is '%s'", awsAccessKey.SecretAccessKey)

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().
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
		Product(
			cmv1.NewProduct().
				ID("moa"),
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
	reporter.Infof("To view list of clusters and their status, run `moactl cluster list`")

	// Add tags to the AWS administrator user containing the identifier and name of the cluster:
	err = awsClient.TagUser(aws.AdminUserName, ocmClusterID, ocmClusterName)
	if err != nil {
		reporter.Infof(
			"Can't add cluster tags to user '%s'",
			aws.AdminUserName,
		)
	}

	reporter.Infof(
		"Cluster '%s' has been created. Once the cluster is 'Ready' you will need to add "+
			"an Identity Provider and define the list of cluster administrators. See `moactl "+
			"idp add --help` and `moactl user add --help` for more information.",
		ocmClusterName,
	)
}
