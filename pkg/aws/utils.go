package aws

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/openshift/rosa/assets"
	"github.com/openshift/rosa/pkg/arguments"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/sirupsen/logrus"
)

// GetRegion will return a region selected by the user or given as a default to the AWS client.
// If the region given is empty, it will first attempt to use the default, and, failing that, will
// prompt for user input.
func GetRegion(region string) (string, error) {
	if region == "" {
		defaultSession, err := session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		})

		if err != nil {
			return "", fmt.Errorf("Error creating default session for AWS client: %v", err)
		}

		region = *defaultSession.Config.Region
	}
	return region, nil
}

// getClientDetails will return the *iam.User associated with the provided client's credentials,
// a boolean indicating whether the user is the 'root' account, and any error encountered
// while trying to gather the info.
func getClientDetails(awsClient *awsClient) (*iam.User, bool, error) {
	rootUser := false

	_, _, err := awsClient.ValidateCredentials()
	if err != nil {
		return nil, rootUser, err
	}

	user, err := awsClient.iamClient.GetUser(nil)
	if err != nil {
		return nil, rootUser, err
	}

	// Detect whether the AWS account's root user is being used
	parsed, err := arn.Parse(*user.User.Arn)
	if err != nil {
		return nil, rootUser, err
	}
	if parsed.AccountID == *user.User.UserId {
		rootUser = true
	}

	return user.User, rootUser, nil
}

// Build cloudformation create stack input
func buildCreateStackInput(cfTemplateBody, stackName string) *cloudformation.CreateStackInput {
	// Special cloudformation capabilities are required to create IAM resources in AWS
	cfCapabilityIAM := "CAPABILITY_IAM"
	cfCapabilityNamedIAM := "CAPABILITY_NAMED_IAM"
	cfTemplateCapabilities := []*string{&cfCapabilityIAM, &cfCapabilityNamedIAM}

	return &cloudformation.CreateStackInput{
		Capabilities: cfTemplateCapabilities,
		StackName:    aws.String(stackName),
		TemplateBody: aws.String(cfTemplateBody),
	}
}

// Build cloudformation update stack input
func buildUpdateStackInput(cfTemplateBody, stackName string) *cloudformation.UpdateStackInput {
	// Special cloudformation capabilities are required to update IAM resources in AWS
	cfCapabilityIAM := "CAPABILITY_IAM"
	cfCapabilityNamedIAM := "CAPABILITY_NAMED_IAM"
	cfTemplateCapabilities := []*string{&cfCapabilityIAM, &cfCapabilityNamedIAM}

	return &cloudformation.UpdateStackInput{
		Capabilities: cfTemplateCapabilities,
		StackName:    aws.String(stackName),
		TemplateBody: aws.String(cfTemplateBody),
	}
}

// Read cloudformation template
func readCFTemplate() (string, error) {
	cfTemplateBodyPath := "templates/cloudformation/iam_user_osdCcsAdmin.json"

	cfTemplate, err := assets.Asset(cfTemplateBodyPath)
	if err != nil {
		return "", fmt.Errorf("Unable to read cloudformation template: %s", err)
	}

	return string(cfTemplate), nil
}

/**
Currently user can rosa init using the region from their config or using --region
When checking for cloud formation we need to check in the region used by the user
*/
func GetAWSClientForUserRegion(reporter *rprtr.Object, logger *logrus.Logger) Client {
	// Get AWS region from env
	awsRegionInUserConfig, err := GetRegion(arguments.GetRegion())
	if err != nil {
		reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}
	if awsRegionInUserConfig == "" {
		reporter.Errorf("AWS Region not set")
		os.Exit(1)
	}

	// Create the AWS client:
	client, err := NewClient().
		Logger(logger).
		Region(awsRegionInUserConfig).
		Build()
	if err != nil {
		reporter.Errorf("Error creating aws client for stack validation: %v", err)
		os.Exit(1)
	}
	regionUsedForInit, err := client.GetClusterRegionTagForUser(AdminUserName)
	if err != nil {
		return client
	}

	if regionUsedForInit != awsRegionInUserConfig {
		// Create the AWS client with the region used in the init
		//So we can check for the stack in that region
		awsClient, err := NewClient().
			Logger(logger).
			Region(regionUsedForInit).
			Build()
		if err != nil {
			reporter.Errorf("Error creating aws client for stack validation: %v", err)
			os.Exit(1)
		}
		return awsClient
	}
	return client
}

// Validations will validate if CF stack/users exist
func CheckStackReadyForCreateCluster(reporter *rprtr.Object, logger *logrus.Logger) {
	client := GetAWSClientForUserRegion(reporter, logger)
	reporter.Debugf("Validating cloudformation stack exists")
	stackExist, _, err := client.CheckStackReadyOrNotExisting(OsdCcsAdminStackName)
	if !stackExist || err != nil {
		reporter.Errorf("Cloudformation stack does not exist. Run `rosa init` first")
		os.Exit(1)
	}
	reporter.Debugf("cloudformation stack is valid!")
}
