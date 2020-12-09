package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/openshift/rosa/assets"
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
