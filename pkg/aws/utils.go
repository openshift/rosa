package aws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
	"gitlab.cee.redhat.com/service/moactl/assets"
)

// SimulateParams captures any additional details that should be used
// when simulating permissions.
type SimulateParams struct {
	Region string
}

// CheckPermissionsUsingQueryClient will use queryClient to query whether the credentials in targetClient can perform the actions
// listed in the statementEntries. queryClient will need iam:GetUser and iam:SimulatePrincipalPolicy
func CheckPermissionsUsingQueryClient(queryClient, targetClient *awsClient, policyDocument PolicyDocument,
	params *SimulateParams) (bool, error) {

	// Ignoring isRoot here since we only warn the user that its not best pratice to use it.
	// TODO: Add a check for isRoot in the initalizer
	targetUser, _, err := getClientDetails(targetClient)
	if err != nil {
		return false, fmt.Errorf("error gathering AWS credentials details: %v", err)
	}

	allowList := []*string{}
	for _, statement := range policyDocument.Statement {
		for _, action := range statement.Action {
			allowList = append(allowList, aws.String(action))
		}
	}

	input := &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: targetUser.Arn,
		ActionNames:     allowList,
		ContextEntries:  []*iam.ContextEntry{},
	}

	if params != nil {
		if params.Region != "" {
			input.ContextEntries = append(input.ContextEntries, &iam.ContextEntry{
				ContextKeyName:   aws.String("aws:RequestedRegion"),
				ContextKeyType:   aws.String("stringList"),
				ContextKeyValues: []*string{aws.String(params.Region)},
			})
		}
	}

	// Either all actions are allowed and we'll return 'true', or it's a failure
	allClear := true
	// Collect all failed actions
	var failedActions []string

	err = queryClient.iamClient.SimulatePrincipalPolicyPages(input, func(response *iam.SimulatePolicyResponse, lastPage bool) bool {

		for _, result := range response.EvaluationResults {
			if *result.EvalDecision != "allowed" {
				// Don't bail out after the first failure, so we can log the full list
				// of failed/denied actions
				failedActions = append(failedActions, *result.EvalActionName)
				allClear = false
			}
		}
		return !lastPage
	})
	if err != nil {
		return false, fmt.Errorf("error simulating policy: %v", err)
	}

	if !allClear {
		return false, fmt.Errorf("actions not allowed with tested credentials: %v", failedActions)
	}

	return true, nil

}

// getClientDetails will return the *iam.User associated with the provided client's credentials,
// a boolean indicating whether the user is the 'root' account, and any error encountered
// while trying to gather the info.
func getClientDetails(awsClient *awsClient) (*iam.User, bool, error) {
	rootUser := false

	user, err := awsClient.iamClient.GetUser(nil)
	if err != nil {
		return nil, rootUser, fmt.Errorf("error querying username: %v", err)
	}

	// Detect whether the AWS account's root user is being used
	parsed, err := arn.Parse(*user.User.Arn)
	if err != nil {
		return nil, rootUser, fmt.Errorf("error parsing user's ARN: %v", err)
	}
	if parsed.AccountID == *user.User.UserId {
		rootUser = true
	}

	return user.User, rootUser, nil
}

// generatePolicyDocument generates an IAM policy Document from a list of actions
func generatePolicyDocument(actions []string, id *string) PolicyDocument {
	policyDocument := PolicyDocument{}

	if id != nil {
		policyDocument.ID = aws.StringValue(id)
	}

	for _, action := range actions {
		policyStatement := PolicyStatement{
			Effect:   "Allow",
			Action:   []string{action},
			Resource: []string{"*"},
		}
		policyDocument.Statement = append(policyDocument.Statement, policyStatement)
	}

	return policyDocument
}

// ReadSCPPolicy reads a SCP policy from file into a Policy Document
// SCP policies are structured the same as a IAM Policy Document the contain
// IAM Policy Statements
func readSCPPolicy(policyDocumentPath string) PolicyDocument {

	policyDocumentFile, err := assets.Asset(policyDocumentPath)
	if err != nil {
		fmt.Println(fmt.Errorf("Unable to load file: %s", policyDocumentPath))
	}

	policyDocument := PolicyDocument{}

	err = json.Unmarshal([]byte(policyDocumentFile), &policyDocument)
	if err != nil {
		fmt.Println(fmt.Errorf("Error unmarshalling statement: %v", err))
	}

	return policyDocument
}

// Build cloudformation stack input
func buildStackInput(cfTemplateBody, stackName string) *cloudformation.CreateStackInput {
	// Special cloudformation capabilities are required to craete IAM resources in AWS
	cfCapabilityIAM := "CAPABILITY_IAM"
	cfCapabilityNamedIAM := "CAPABILITY_NAMED_IAM"
	cfTemplateCapabilities := []*string{&cfCapabilityIAM, &cfCapabilityNamedIAM}

	return &cloudformation.CreateStackInput{
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
		return "", fmt.Errorf("unable to read cloudformation template: %s", err)
	}

	return string(cfTemplate), nil
}

func validatePolicyDocuments(queryClient, targetClient *awsClient, policyDocuments []PolicyDocument, sParams *SimulateParams) (bool, error) {
	permissionsOk := true

	for _, policyDocument := range policyDocuments {
		permissionsOk, err := CheckPermissionsUsingQueryClient(queryClient, targetClient, policyDocument, sParams)
		if err != nil {
			return false, err
		}
		if !permissionsOk {
			return false, fmt.Errorf("Unable to validate permissions in %s", policyDocument.ID)
		}

	}

	return permissionsOk, nil
}
