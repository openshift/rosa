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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/debug"
	"gitlab.cee.redhat.com/service/moactl/pkg/properties"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
	"gitlab.cee.redhat.com/service/moactl/pkg/tags"
)

var Cmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create cluster",
	Long:  "Create cluster.",
	Run:   run,
}

func run(cmd *cobra.Command, argv []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Errorf("Can't create reporter: %v\n", err)
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

	// Create the logger that will be used by the OCM connection:
	ocmLogger, err := sdk.NewStdLoggerBuilder().
		Debug(debug.Enabled()).
		Build()
	if err != nil {
		reporter.Errorf("Can't create OCM logger: %v", err)
		os.Exit(1)
	}

	// Create the AWS session:
	awsSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		reporter.Errorf("Can't create AWS session: %v", err)
		os.Exit(1)
	}

	// Check that the session is set:
	awsRegion := aws.StringValue(awsSession.Config.Region)
	if awsRegion == "" {
		reporter.Errorf("Region is not set")
		os.Exit(1)
	}

	// Check that the AWS credentials are available:
	_, err = awsSession.Config.Credentials.Get()
	if err != nil {
		reporter.Errorf("Can't find AWS credentials: %v", err)
		os.Exit(1)
	}

	// Get the clients for the AWS services that we will be using:
	awsSts := sts.New(awsSession)
	awsIam := iam.New(awsSession)

	// Get the details of the current user:
	getCallerIdentityOutput, err := awsSts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		reporter.Errorf("Can't get caller identity: %v", err)
		os.Exit(1)
	}
	awsCreatorARN := aws.StringValue(getCallerIdentityOutput.Arn)
	reporter.Infof("ARN of current user is '%s'", awsCreatorARN)

	// Extract the account identifier from the ARN of the user:
	awsCreatorParsedARN, err := arn.Parse(awsCreatorARN)
	if err != nil {
		reporter.Infof("Can't parse user ARN '%s': %v", awsCreatorARN, err)
		os.Exit(1)
	}
	awsCreatorAccountID := awsCreatorParsedARN.AccountID
	reporter.Infof("Account identifier of current user is '%s'", awsCreatorAccountID)

	// Create the AWS user that will be used to create all the resources needed by the cluster:
	reporter.Infof("Creating cluster administrator user '%s'", awsAdminName)
	createUserOutput, err := awsIam.CreateUser(&iam.CreateUserInput{
		UserName: aws.String(awsAdminName),
		Tags: []*iam.Tag{
			{
				Key:   aws.String(tags.ClusterName),
				Value: aws.String(clusterName),
			},
		},
	})
	if err != nil {
		switch typed := err.(type) {
		case awserr.Error:
			if typed.Code() == iam.ErrCodeEntityAlreadyExistsException {
				reporter.Errorf(
					"User '%s' already exists, which means that there is "+
						"already a cluster created in the account",
					awsAdminName,
				)
			} else {
				reporter.Errorf("Can't create user '%s': %v", awsAdminName, err)
			}
		default:
			reporter.Errorf("Can't create user '%s': %v", awsAdminName, err)
		}
		os.Exit(1)
	}
	awsAdmin := createUserOutput.User
	awsAdminID := aws.StringValue(awsAdmin.UserId)
	awsAdminARN := aws.StringValue(awsAdmin.Arn)
	reporter.Infof("User identifier is '%s'", awsAdminID)
	reporter.Infof("User ARN is '%s'", awsAdminARN)

	// Make the AWS user an administrator:
	reporter.Infof("Attaching administrator policy to user '%s'", awsAdminName)
	_, err = awsIam.AttachUserPolicy(&iam.AttachUserPolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AdministratorAccess"),
		UserName:  aws.String(awsAdminName),
	})
	if err != nil {
		reporter.Errorf("Can't attach administrator policy to user '%s'", awsAdminName)
		os.Exit(1)
	}

	// Create the access key for the AWS user:
	reporter.Infof("Creating access key for user '%s'", awsAdminName)
	createAccessKeyOutput, err := awsIam.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(awsAdminName),
	})
	if err != nil {
		reporter.Errorf("Can't create access key for user '%s': %v", awsAdminName, err)
		os.Exit(1)
	}
	awsAccessKey := createAccessKeyOutput.AccessKey
	awsAccessKeyID := aws.StringValue(awsAccessKey.AccessKeyId)
	awsSecretAccessKey := aws.StringValue(awsAccessKey.SecretAccessKey)
	reporter.Infof("Access key identifier is '%s'", awsAccessKeyID)
	reporter.Infof("Secret access key is '%s'", awsSecretAccessKey)

	// The created access key isn't immediately active, so we need to wait a bit till it is:
	reporter.Infof("Waiting for access key to be active")
	time.Sleep(10 * time.Second)

	// Create the client for the OCM API:
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
				AccountID(awsCreatorAccountID).
				AccessKeyID(awsAccessKeyID).
				SecretAccessKey(awsSecretAccessKey),
		).
		Properties(map[string]string{
			properties.CreatorARN: awsCreatorARN,
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
	_, err = awsIam.TagUser(&iam.TagUserInput{
		UserName: aws.String(awsAdminName),
		Tags: []*iam.Tag{
			{
				Key:   aws.String(tags.ClusterID),
				Value: aws.String(ocmClusterID),
			},
		},
	})
	if err != nil {
		reporter.Infof(
			"Can't add cluster identifier tag to user '%s'",
			awsAdminName,
		)
	}
}

// Name of the AWS user that will be used to create all the resources of the cluster:
const awsAdminName = "osdCcsAdmin"
