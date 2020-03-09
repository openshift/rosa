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
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/organizations/organizationsiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/sirupsen/logrus"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws/tags"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
)

// Name of the AWS user that will be used to create all the resources of the cluster:
const AdminUserName = "osdCcsAdmin"

type Client interface {
	GetRegion() string
	ValidateCredentials() (bool, error)
	EnsureUser(username string) (bool, error)
	CreateAccessKey(username string) (*AWSAccessKey, error)
	GetCreator() (*AWSCreator, error)
	TagUser(username string, clusterID string, clusterName string) error
	ValidateSCP() (bool, error)
}

// ClientBuilder contains the information and logic needed to build a new AWS client.
type ClientBuilder struct {
	logger *logrus.Logger
}

type awsClient struct {
	logger *logrus.Logger
	// cloudformationClient cloudformationiface.CloudFormationAPI
	iamClient  iamiface.IAMAPI
	orgClient  organizationsiface.OrganizationsAPI
	stsClient  stsiface.STSAPI
	awsSession *session.Session
}

// NewClient creates a builder that can then be used to configure and build a new AWS client.
func NewClient() *ClientBuilder {
	return &ClientBuilder{}
}

// Logger sets the logger that the AWS client will use to send messages to the log.
func (b *ClientBuilder) Logger(value *logrus.Logger) *ClientBuilder {
	b.logger = value
	return b
}

// Build uses the information stored in the builder to build a new AWS client.
func (b *ClientBuilder) Build() (result Client, err error) {
	// Check parameters:
	if b.logger == nil {
		err = fmt.Errorf("logger is mandatory")
		return
	}

	// Create the AWS logger:
	logger, err := logging.NewAWSLogger().
		Logger(b.logger).
		Build()
	if err != nil {
		return
	}

	// Create the AWS session:
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	sess.Config.Logger = logger
	sess.Config.HTTPClient = &http.Client{
		Transport: http.DefaultTransport,
	}
	if b.logger.IsLevelEnabled(logrus.DebugLevel) {
		var dumper http.RoundTripper
		dumper, err = logging.NewRoundTripper().
			Logger(b.logger).
			Next(sess.Config.HTTPClient.Transport).
			Build()
		if err != nil {
			return
		}
		sess.Config.HTTPClient.Transport = dumper
	}
	if err != nil {
		return
	}

	// Check that the region is set:
	region := aws.StringValue(sess.Config.Region)
	if region == "" {
		err = fmt.Errorf("region is not set")
		return
	}

	// Check that the AWS credentials are available:
	_, err = sess.Config.Credentials.Get()
	if err != nil {
		err = fmt.Errorf("can't find credentials: %v", err)
		return
	}

	// Create and populate the object:
	result = &awsClient{
		logger:     b.logger,
		iamClient:  iam.New(sess),
		orgClient:  organizations.New(sess),
		stsClient:  sts.New(sess),
		awsSession: sess,
	}

	return
}

func (c *awsClient) GetRegion() string {
	return aws.StringValue(c.awsSession.Config.Region)
}

type AWSCreator struct {
	ARN       string
	AccountID string
}

func (c *awsClient) GetCreator() (*AWSCreator, error) {
	getCallerIdentityOutput, err := c.stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	creatorARN := aws.StringValue(getCallerIdentityOutput.Arn)

	// Extract the account identifier from the ARN of the user:
	creatorParsedARN, err := arn.Parse(creatorARN)
	if err != nil {
		return nil, err
	}
	return &AWSCreator{
		ARN:       creatorARN,
		AccountID: creatorParsedARN.AccountID,
	}, nil
}

// Checks if given credentials are valid.
func (c *awsClient) ValidateCredentials() (bool, error) {
	_, err := c.stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// Ensure osdCcsAdmin account
func (c *awsClient) EnsureUser(username string) (bool, error) {
	_, err := c.iamClient.CreateUser(&iam.CreateUserInput{
		UserName: aws.String(username),
	})
	if err != nil {
		switch typed := err.(type) {
		case awserr.Error:
			if typed.Code() == iam.ErrCodeEntityAlreadyExistsException {
				return false, nil
			}
		}
		return false, err
	}

	_, err = c.iamClient.AttachUserPolicy(&iam.AttachUserPolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AdministratorAccess"),
		UserName:  aws.String(username),
	})
	if err != nil {
		return true, err
	}

	return true, nil
}

// FIXME: Since we support multiple clusters per user, we need to find a better way to
// tag the user so that the tags don't overwrite each other with each new cluster.
func (c *awsClient) TagUser(username string, clusterID string, clusterName string) error {
	_, err := c.iamClient.TagUser(&iam.TagUserInput{
		UserName: aws.String(username),
		Tags: []*iam.Tag{
			{
				Key:   aws.String(tags.ClusterID),
				Value: aws.String(clusterID),
			},
			{
				Key:   aws.String(tags.ClusterName),
				Value: aws.String(clusterName),
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

type AWSAccessKey struct {
	AccessKeyID     string
	SecretAccessKey string
}

func (c *awsClient) CreateAccessKey(username string) (*AWSAccessKey, error) {
	createAccessKeyOutput, err := c.iamClient.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	accessKey := createAccessKeyOutput.AccessKey
	return &AWSAccessKey{
		AccessKeyID:     aws.StringValue(accessKey.AccessKeyId),
		SecretAccessKey: aws.StringValue(accessKey.SecretAccessKey),
	}, nil
}

// Validate SCP...
func (c *awsClient) ValidateSCP() (bool, error) {
	hasScpAccess := true
	policyType := aws.String("SERVICE_CONTROL_POLICY")

	_, err := c.orgClient.ListPolicies(&organizations.ListPoliciesInput{
		Filter: policyType,
	})
	if err != nil {
		switch typed := err.(type) {
		case awserr.Error:
			// Current user does not have access to SCP policies. This is normal for most
			// users, so we should find other ways of validating proper account permissions
			if typed.Code() == organizations.ErrCodeAccessDeniedException {
				hasScpAccess = false
				err = nil
			} else {
				return false, err
			}
		default:
			return false, err
		}
	}

	// TODO: Find another way to verify permissions
	if !hasScpAccess {
		return false, nil
	}

	return true, nil
}
