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
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/organizations/organizationsiface"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/aws/aws-sdk-go/service/servicequotas/servicequotasiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/sirupsen/logrus"

	"github.com/openshift/moactl/pkg/aws/tags"
	"github.com/openshift/moactl/pkg/logging"
)

// Name of the AWS user that will be used to create all the resources of the cluster:
const (
	AdminUserName        = "osdCcsAdmin"
	OsdCcsAdminStackName = "osdCcsAdminIAMUser"

	// Since CloudFormation stacks are region-dependent, we hard-code OCM's default region and
	// then use it to ensure that the user always gets the stack from the same region.
	DefaultRegion = "us-east-1"
)

// Client defines a client interface
type Client interface {
	GetRegion() string
	ValidateCredentials() (bool, error)
	ValidateCFUserCredentials() error
	EnsureOsdCcsAdminUser(stackName string) (bool, error)
	DeleteOsdCcsAdminUser(stackName string) error
	GetAccessKeyFromStack(stackName string) (*AccessKey, error)
	GetCreator() (*Creator, error)
	TagUser(username string, clusterID string, clusterName string) error
	ValidateSCP() (bool, error)
	ValidateQuota() (bool, error)
}

// ClientBuilder contains the information and logic needed to build a new AWS client.
type ClientBuilder struct {
	logger *logrus.Logger
	region *string
}

type awsClient struct {
	logger              *logrus.Logger
	iamClient           iamiface.IAMAPI
	orgClient           organizationsiface.OrganizationsAPI
	stsClient           stsiface.STSAPI
	cfClient            cloudformationiface.CloudFormationAPI
	servicequotasClient servicequotasiface.ServiceQuotasAPI
	awsSession          *session.Session
}

// NewClient creates a builder that can then be used to configure and build a new AWS client.
func NewClient() *ClientBuilder {
	return &ClientBuilder{}
}

func New(
	logger *logrus.Logger,
	iamClient iamiface.IAMAPI,
	orgClient organizationsiface.OrganizationsAPI,
	stsClient stsiface.STSAPI,
	cfClient cloudformationiface.CloudFormationAPI,
	servicequotasClient servicequotasiface.ServiceQuotasAPI,
	awsSession *session.Session,

) Client {
	return &awsClient{
		logger,
		iamClient,
		orgClient,
		stsClient,
		cfClient,
		servicequotasClient,
		awsSession,
	}
}

// Logger sets the logger that the AWS client will use to send messages to the log.
func (b *ClientBuilder) Logger(value *logrus.Logger) *ClientBuilder {
	b.logger = value
	return b
}

func (b *ClientBuilder) Region(value string) *ClientBuilder {
	b.region = aws.String(value)
	return b
}

// Build uses the information stored in the builder to build a new AWS client.
func (b *ClientBuilder) Build() (Client, error) {
	// Check parameters:
	if b.logger == nil {
		return nil, fmt.Errorf("Logger is mandatory")
	}

	// Create the AWS logger:
	logger, err := logging.NewAWSLogger().
		Logger(b.logger).
		Build()
	if err != nil {
		return nil, err
	}

	// Create the AWS session:
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: b.region,
			// MaxRetries to limit the number of attempts on failed API calls
			MaxRetries: aws.Int(25),
			// Set MinThrottleDelay to 1 second
			Retryer: client.DefaultRetryer{
				NumMaxRetries:    5,
				MinThrottleDelay: 1 * time.Second,
			},
			Logger: logger,
			HTTPClient: &http.Client{
				Transport: http.DefaultTransport,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if b.logger.IsLevelEnabled(logrus.DebugLevel) {
		var dumper http.RoundTripper
		dumper, err = logging.NewRoundTripper().
			Logger(b.logger).
			Next(sess.Config.HTTPClient.Transport).
			Build()
		if err != nil {
			return nil, err
		}
		sess.Config.HTTPClient.Transport = dumper
	}

	// Check that the region is set:
	region := aws.StringValue(sess.Config.Region)
	if region == "" {
		return nil, fmt.Errorf("Region is not set")
	}

	// Check that the AWS credentials are available:
	_, err = sess.Config.Credentials.Get()
	if err != nil {
		return nil, fmt.Errorf("Failed to find credentials: %v", err)
	}

	// Create and populate the object:
	c := &awsClient{
		logger:              b.logger,
		iamClient:           iam.New(sess),
		orgClient:           organizations.New(sess),
		stsClient:           sts.New(sess),
		cfClient:            cloudformation.New(sess),
		servicequotasClient: servicequotas.New(sess),
		awsSession:          sess,
	}

	_, root, err := getClientDetails(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get client details: %v", err)
	}

	if root {
		return nil, errors.New("using a root account is not supported, please use an IAM user instead")
	}

	return c, err
}

func (c *awsClient) GetRegion() string {
	return aws.StringValue(c.awsSession.Config.Region)
}

type Creator struct {
	ARN       string
	AccountID string
}

func (c *awsClient) GetCreator() (*Creator, error) {
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
	return &Creator{
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

// ValidateCFUserCredentials checks if CF-IAM credentials are valid.
// it gets stack's key and actual key and compares them
// to get the stack credentials:
// aws cloudformation describe-stack-resource \
// --logical-resource-id osdCcsAdminAccessKeys --stack-name osdCcsAdminIAMUser
func (c *awsClient) ValidateCFUserCredentials() error {
	name := AdminUserName
	accessKeyInput := &iam.ListAccessKeysInput{
		UserName: &name,
	}
	accessKeyList, err := c.iamClient.ListAccessKeys(accessKeyInput)
	if err != nil {
		return err
	}

	OsdCcsAdminStackNamePtr := OsdCcsAdminStackName
	LogicalResourceIDPtr := "osdCcsAdminAccessKeys"
	stackResourceInput := &cloudformation.DescribeStackResourceInput{
		StackName:         &OsdCcsAdminStackNamePtr,
		LogicalResourceId: &LogicalResourceIDPtr,
	}
	resources, err := c.cfClient.DescribeStackResource(stackResourceInput)
	if err != nil {
		return err
	}
	cfAccessKey := resources.StackResourceDetail.PhysicalResourceId

	for _, key := range accessKeyList.AccessKeyMetadata {
		if *key.AccessKeyId == *cfAccessKey && *key.Status == "Active" {
			return nil
		}
	}

	return fmt.Errorf(
		"Invalid CloudFormation stack credentials: %s is not valid \n"+
			"you can recreate the CloudFormation stack with: \n"+
			"moactl init --delete-stack && moactl init \n", name)
}

// Ensure osdCcsAdmin IAM user is created
func (c *awsClient) EnsureOsdCcsAdminUser(stackName string) (bool, error) {
	// Check already existing cloudformation stack status
	stackReady, err := c.checkStackReadyOrNotExisting(stackName)
	if err != nil || stackReady {
		return false, err
	}

	// Read cloudformation template
	cfTemplateBody, err := readCFTemplate()
	if err != nil {
		return false, err
	}

	// Create cloudformation stack
	_, err = c.cfClient.CreateStack(buildStackInput(cfTemplateBody, stackName))
	if err != nil {
		return false, err
	}

	// Wait until cloudformation stack creates
	err = c.cfClient.WaitUntilStackCreateComplete(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		switch typed := err.(type) {
		case awserr.Error:
			// Waiter reached maximum attempts waiting for the resource to be ready
			if typed.Code() == request.WaiterResourceNotReadyErrorCode {
				c.logger.Errorf("Max retries reached waiting for stack to create")
				return false, err
			}
		}
		return false, err
	}

	return true, nil
}

func (c *awsClient) checkStackReadyOrNotExisting(stackName string) (stackReady bool, err error) {
	stackList, err := c.cfClient.ListStacks(&cloudformation.ListStacksInput{})
	if err != nil {
		return false, err
	}

	for _, summary := range stackList.StackSummaries {
		if *summary.StackName == stackName {
			if *summary.StackStatus == cloudformation.StackStatusCreateComplete {
				return true, nil
			}
			if *summary.StackStatus != cloudformation.StackStatusDeleteComplete {
				return false, fmt.Errorf("Error creating user: Cloudformation stack %s exists with status %s. "+
					"Expected status is %s.\n"+
					"Ensure user osdCcsAdmin does not exist, then retry with\n"+
					"moactl init --delete-stack; moactl init",
					*summary.StackName, *summary.StackStatus, cloudformation.StackStatusCreateComplete)
			}
		}
	}

	return false, nil
}

func (c *awsClient) DeleteOsdCcsAdminUser(stackName string) error {
	deleteStackInput := &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	}

	// Delete cloudformation stack
	_, err := c.cfClient.DeleteStack(deleteStackInput)
	if err != nil {
		switch typed := err.(type) {
		case awserr.Error:
			if typed.Code() == cloudformation.ErrCodeTokenAlreadyExistsException {
				return nil
			}
		}
		return err
	}

	// Wait until cloudformation stack deletes
	err = c.cfClient.WaitUntilStackDeleteComplete(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		switch typed := err.(type) {
		case awserr.Error:
			// Waiter reached maximum attempts waiting for the resource to be ready
			if typed.Code() == request.WaiterResourceNotReadyErrorCode {
				c.logger.Errorf("Max retries reached waiting for stack to delete")
				return err
			}
		}
		return err
	}

	return nil
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

type AccessKey struct {
	AccessKeyID     string
	SecretAccessKey string
}

func (c *awsClient) GetAccessKeyFromStack(stackName string) (*AccessKey, error) {
	outputKeySecretKey := "SecretKey"
	outputKeyAccessKey := "AccessKey"
	keys := AccessKey{}

	stackOutput, err := c.cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: &stackName})

	for _, stack := range stackOutput.Stacks {
		if *stack.StackName == stackName {
			for _, output := range stack.Outputs {
				if *output.OutputKey == outputKeyAccessKey {
					keys.AccessKeyID = aws.StringValue(output.OutputValue)
				}
				if *output.OutputKey == outputKeySecretKey {
					keys.SecretAccessKey = aws.StringValue(output.OutputValue)
				}
			}
		}
	}

	return &keys, err
}

// ValidateQuota
func (c *awsClient) ValidateQuota() (bool, error) {
	for _, quota := range serviceQuotaServices {
		serviceQuotas, err := ListServiceQuotas(c, quota.ServiceCode)
		if err != nil {
			return false, fmt.Errorf("Error listing AWS service quotas: %s %v", quota.ServiceCode, err)
		}

		serviceQuota, err := GetServiceQuota(serviceQuotas, quota.QuotaCode)
		if err != nil || serviceQuota == nil || (*serviceQuota).Value == nil {
			return false, fmt.Errorf("Error getting AWS service quota: %s %v", quota.ServiceCode, err)
		}

		if *serviceQuota.Value < *quota.DesiredValue {
			return false, fmt.Errorf(
				"Service %s quota code %s %s not valid, expected quota of at least %d, but got %d",
				quota.ServiceCode, quota.QuotaCode, quota.QuotaName,
				int(*quota.DesiredValue), int(*serviceQuota.Value))
		}

		c.logger.Debug(fmt.Sprintf("Service %s quota code %s is ok", quota.ServiceCode, quota.QuotaCode))
	}

	return true, nil
}

// ValidateSCP attempts to validate SCP policies by ensuring we have the correct permissions
func (c *awsClient) ValidateSCP() (bool, error) {
	scpPolicyPath := "templates/policies/osd_scp_policy.json"

	sParams := &SimulateParams{
		Region: *c.awsSession.Config.Region,
	}

	// Read installer permissions and OSD SCP Policy permissions
	osdPolicyDocument := readSCPPolicy(scpPolicyPath)
	policyDocuments := []PolicyDocument{osdPolicyDocument}

	// Validate permissions
	hasPermissions, err := validatePolicyDocuments(c, c, policyDocuments, sParams)
	if err != nil {
		return false, err
	}
	if !hasPermissions {
		return false, err
	}

	return true, nil
}
