package aws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/openshift/moactl/assets"
)

// PolicyStatement models an AWS policy statement entry.
type PolicyStatement struct {
	Sid string `json:"sid,omitempty"`
	// Effect indicates if this policy statement is to Allow or Deny.
	Effect string `json:"effect"`
	// Action describes the particular AWS service actions that should be allowed or denied.
	// (i.e. ec2:StartInstances, iam:ChangePassword)
	Action []string `json:"action"`
	// Resource specifies the object(s) this statement should apply to. (or "*" for all)
	Resource interface{} `json:"resource"`
}

// PolicyDocument models an AWS IAM policy document
type PolicyDocument struct {
	Version   string            `json:"version,omitempty"`
	ID        string            `json:"id,omitempty"`
	Statement []PolicyStatement `json:"statement"`
}

// SimulateParams captures any additional details that should be used
// when simulating permissions.
type SimulateParams struct {
	Region string
}

// checkPermissionsUsingQueryClient will use queryClient to query whether the credentials in targetClient can perform
// the actions listed in the statementEntries. queryClient will need iam:GetUser and iam:SimulatePrincipalPolicy
func checkPermissionsUsingQueryClient(queryClient, targetClient *awsClient, policyDocument PolicyDocument,
	params *SimulateParams) (bool, error) {
	// Ignoring isRoot here since we only warn the user that its not best practice to use it.
	// TODO: Add a check for isRoot in the initializer
	targetUser, _, err := getClientDetails(targetClient)
	if err != nil {
		return false, fmt.Errorf("Error gathering AWS credentials details: %v", err)
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

	err = queryClient.iamClient.SimulatePrincipalPolicyPages(input,
		func(response *iam.SimulatePolicyResponse, lastPage bool) bool {
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
		return false, fmt.Errorf("Error simulating policy: %v", err)
	}

	if !allClear {
		return false, fmt.Errorf("Actions not allowed with tested credentials: %v", failedActions)
	}

	return true, nil
}

func validatePolicyDocuments(queryClient, targetClient *awsClient, policyDocuments []PolicyDocument,
	sParams *SimulateParams) (bool, error) {
	for _, policyDocument := range policyDocuments {
		permissionsOk, err := checkPermissionsUsingQueryClient(queryClient, targetClient, policyDocument, sParams)
		if err != nil {
			return false, err
		}
		if !permissionsOk {
			return false, fmt.Errorf("Unable to validate permissions in %s", policyDocument.ID)
		}
	}

	return true, nil
}

// readSCPPolicy reads a SCP policy from file into a Policy Document
// SCP policies are structured the same as a IAM Policy Document the contain
// IAM Policy Statements
func readSCPPolicy(policyDocumentPath string) PolicyDocument {
	policyDocumentFile, err := assets.Asset(policyDocumentPath)
	if err != nil {
		fmt.Println(fmt.Errorf("Unable to load file: %s", policyDocumentPath))
	}

	policyDocument := PolicyDocument{}

	err = json.Unmarshal(policyDocumentFile, &policyDocument)
	if err != nil {
		fmt.Println(fmt.Errorf("Error unmarshalling statement: %v", err))
	}

	return policyDocument
}
