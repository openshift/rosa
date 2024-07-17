package aws_client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openshift-online/ocm-common/pkg/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

var (
	DEFAULT_RETRIES        = 3
	DEFAULT_RETRY_DURATION = 60 * time.Second
)

func (client *AWSClient) CreateRole(roleName string,
	assumeRolePolicyDocument string,
	permissionBoundry string,
	tags map[string]string,
	path string,
) (role types.Role, err error) {
	var roleTags []types.Tag
	for tagKey, tagValue := range tags {
		roleTags = append(roleTags, types.Tag{
			Key:   &tagKey,
			Value: &tagValue,
		})
	}
	description := "This is created role for ocm-qe automation testing"
	input := &iam.CreateRoleInput{
		RoleName:                 &roleName,
		AssumeRolePolicyDocument: &assumeRolePolicyDocument,
		Description:              &description,
	}
	if path != "" {
		input.Path = &path
	}
	if permissionBoundry != "" {
		input.PermissionsBoundary = &permissionBoundry
	}
	if len(tags) != 0 {
		input.Tags = roleTags
	}
	var resp *iam.CreateRoleOutput
	resp, err = client.IamClient.CreateRole(context.TODO(), input)
	if err == nil && resp != nil {
		role = *resp.Role
		err = client.WaitForResourceExisting("role-"+*resp.Role.RoleName, 10) // add a prefix to meet the resourceExisting split rule

	}
	return
}

func (client *AWSClient) CreateRoleAndAttachPolicy(roleName string,
	assumeRolePolicyDocument string,
	permissionBoundry string,
	tags map[string]string,
	path string,
	policyArn string) (types.Role, error) {
	role, err := client.CreateRole(roleName, assumeRolePolicyDocument, permissionBoundry, tags, path)
	if err == nil {
		err = client.AttachPolicy(*role.RoleName, policyArn, DEFAULT_RETRIES, DEFAULT_RETRY_DURATION)
	}
	return role, err
}

func (client *AWSClient) GetRole(roleName string) (*types.Role, error) {
	input := &iam.GetRoleInput{
		RoleName: &roleName,
	}
	out, err := client.IamClient.GetRole(context.TODO(), input)
	return out.Role, err
}
func (client *AWSClient) DeleteRole(roleName string) error {

	input := &iam.DeleteRoleInput{
		RoleName: &roleName,
	}
	_, err := client.IamClient.DeleteRole(context.TODO(), input)
	return err
}

func (client *AWSClient) DeleteRoleAndPolicy(roleName string, managedPolicy bool) error {
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	}
	output, err := client.IamClient.ListAttachedRolePolicies(client.ClientContext, input)
	if err != nil {
		return err
	}

	fmt.Println(output.AttachedPolicies)
	for _, policy := range output.AttachedPolicies {
		err = client.DetachIAMPolicy(roleName, *policy.PolicyArn)
		if err != nil {
			return err
		}
		if !managedPolicy {
			err = client.DeletePolicy(*policy.PolicyArn)
			if err != nil {
				return err
			}
		}

	}
	err = client.DeleteRole(roleName)
	return err
}

func (client *AWSClient) ListRoles() ([]types.Role, error) {
	input := &iam.ListRolesInput{}
	out, err := client.IamClient.ListRoles(context.TODO(), input)
	return out.Roles, err
}

func (client *AWSClient) IsPolicyAttachedToRole(roleName string, policyArn string) (bool, error) {
	policies, err := client.ListAttachedRolePolicies(roleName)
	if err != nil {
		return false, err
	}
	for _, policy := range policies {
		if aws.ToString(policy.PolicyArn) == policyArn {
			return true, nil
		}
	}
	return false, nil
}

func (client *AWSClient) ListAttachedRolePolicies(roleName string) ([]types.AttachedPolicy, error) {
	policies := []types.AttachedPolicy{}
	policyLister := iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	}
	policyOut, err := client.IamClient.ListAttachedRolePolicies(context.TODO(), &policyLister)
	if err != nil {
		return policies, err
	}
	return policyOut.AttachedPolicies, nil
}

func (client *AWSClient) DetachRolePolicies(roleName string) error {
	policies, err := client.ListAttachedRolePolicies(roleName)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		policyDetacher := iam.DetachRolePolicyInput{
			PolicyArn: policy.PolicyArn,
			RoleName:  &roleName,
		}
		_, err := client.IamClient.DetachRolePolicy(context.TODO(), &policyDetacher)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *AWSClient) DeleteRoleInstanceProfiles(roleName string) error {
	inProfileLister := iam.ListInstanceProfilesForRoleInput{
		RoleName: &roleName,
	}
	out, err := client.IamClient.ListInstanceProfilesForRole(context.TODO(), &inProfileLister)
	if err != nil {
		return err
	}
	for _, inProfile := range out.InstanceProfiles {
		profileDeleter := iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: inProfile.InstanceProfileName,
			RoleName:            &roleName,
		}
		_, err = client.IamClient.RemoveRoleFromInstanceProfile(context.TODO(), &profileDeleter)
		if err != nil {
			return err
		}
	}

	return nil
}

func (client *AWSClient) CreateIAMRole(roleName string, ProdENVTrustedRole string, StageENVTrustedRole string, StageIssuerTrustedRole string, policyArn string,
	externalID ...string) (types.Role, error) {
	statement := map[string]interface{}{
		"Effect": "Allow",
		"Principal": map[string]interface{}{
			"Service": "ec2.amazonaws.com",
			"AWS": []string{
				ProdENVTrustedRole,
				StageENVTrustedRole,
				StageIssuerTrustedRole,
			},
		},
		"Action": "sts:AssumeRole",
	}

	if len(externalID) == 1 {
		statement["Condition"] = map[string]map[string]string{
			"StringEquals": {
				"sts:ExternalId": "aaaa",
			},
		}
	}

	assumeRolePolicyDocument, err := completeRolePolicyDocument(statement)
	if err != nil {
		fmt.Println("Failed to convert Role Policy Document into JSON: ", err)
		return types.Role{}, err
	}

	return client.CreateRoleAndAttachPolicy(roleName, string(assumeRolePolicyDocument), "", make(map[string]string), "/", policyArn)
}

func (client *AWSClient) CreateRegularRole(roleName string, policyArn string) (types.Role, error) {

	statement := map[string]interface{}{
		"Effect": "Allow",
		"Principal": map[string]interface{}{
			"Service": "ec2.amazonaws.com",
		},
		"Action": "sts:AssumeRole",
	}

	assumeRolePolicyDocument, err := completeRolePolicyDocument(statement)
	if err != nil {
		fmt.Println("Failed to convert Role Policy Document into JSON: ", err)
		return types.Role{}, err
	}
	return client.CreateRoleAndAttachPolicy(roleName, assumeRolePolicyDocument, "", make(map[string]string), "/", policyArn)
}

func (client *AWSClient) CreateRoleForAuditLogForward(roleName, awsAccountID string, oidcEndpointURL string, policyArn string) (types.Role, error) {
	statement := map[string]interface{}{
		"Effect": "Allow",
		"Principal": map[string]interface{}{
			"Federated": fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s", awsAccountID, oidcEndpointURL),
		},
		"Action": "sts:AssumeRoleWithWebIdentity",
		"Condition": map[string]interface{}{
			"StringEquals": map[string]interface{}{
				fmt.Sprintf("%s:sub", oidcEndpointURL): "system:serviceaccount:openshift-config-managed:cloudwatch-audit-exporter",
			},
		},
	}

	assumeRolePolicyDocument, err := completeRolePolicyDocument(statement)
	if err != nil {
		fmt.Println("Failed to convert Role Policy Document into JSON: ", err)
		return types.Role{}, err
	}

	return client.CreateRoleAndAttachPolicy(roleName, string(assumeRolePolicyDocument), "", make(map[string]string), "/", policyArn)
}

func (client *AWSClient) CreatePolicy(policyName string, statements ...map[string]interface{}) (string, error) {
	timeCreation := time.Now().Local().String()
	description := fmt.Sprintf("Created by OCM QE at %s", timeCreation)
	document := map[string]interface{}{
		"Version":   "2012-10-17",
		"Statement": []map[string]interface{}{},
	}
	if len(statements) != 0 {
		for _, statement := range statements {
			document["Statement"] = append(document["Statement"].([]map[string]interface{}), statement)
		}
	}
	documentBytes, err := json.Marshal(document)
	if err != nil {
		err = fmt.Errorf("error to unmarshal the statement to string: %v", err)
		return "", err
	}
	documentStr := string(documentBytes)
	policyCreator := iam.CreatePolicyInput{
		PolicyDocument: &documentStr,
		PolicyName:     &policyName,
		Description:    &description,
	}
	outRes, err := client.IamClient.CreatePolicy(context.TODO(), &policyCreator)
	if err != nil {
		return "", err
	}
	policyArn := *outRes.Policy.Arn
	return policyArn, err
}

func (client *AWSClient) CreatePolicyForAuditLogForward(policyName string) (string, error) {

	statement := map[string]interface{}{
		"Effect":   "Allow",
		"Resource": "arn:aws:logs:*:*:*",
		"Action": []string{
			"logs:PutLogEvents",
			"logs:CreateLogGroup",
			"logs:PutRetentionPolicy",
			"logs:CreateLogStream",
			"logs:DescribeLogGroups",
			"logs:DescribeLogStreams",
		},
	}
	return client.CreatePolicy(policyName, statement)
}

func completeRolePolicyDocument(statement map[string]interface{}) (string, error) {
	rolePolicyDocument := map[string]interface{}{
		"Version":   "2012-10-17",
		"Statement": statement,
	}

	assumeRolePolicyDocument, err := json.Marshal(rolePolicyDocument)
	return string(assumeRolePolicyDocument), err
}

func (client *AWSClient) AttachPolicy(roleName string, policyArn string, retries int, retryIntervalInSeconds time.Duration) error {
	policyAttach := iam.AttachRolePolicyInput{
		PolicyArn: &policyArn,
		RoleName:  &roleName,
	}
	_, err := client.IamClient.AttachRolePolicy(context.TODO(), &policyAttach)
	if err != nil {
		return err
	}

	attached, err := client.PolicyAttachedToRole(roleName, policyArn)

	if err != nil {
		return err
	}
	if attached {
		return nil
	}

	start := 0

	for start < retries {
		attached, err := client.PolicyAttachedToRole(roleName, policyArn)
		if err != nil && start == retries {
			return err
		}
		if attached {
			return nil
		}
		time.Sleep(retryIntervalInSeconds)
		start++
	}
	return fmt.Errorf("failed to attach policy to role but no errors were thrown, please investigate")
}

func (client *AWSClient) PolicyAttachedToRole(roleName string, policyArn string) (bool, error) {
	policies, err := client.ListRoleAttachedPolicies(roleName)
	if err != nil {
		return false, err
	}
	for _, policy := range policies {
		if *policy.PolicyArn == policyArn {
			return true, nil
		}
	}
	return false, nil
}

func (client *AWSClient) ListRoleAttachedPolicies(roleName string) ([]types.AttachedPolicy, error) {
	policies := []types.AttachedPolicy{}
	policyLister := iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	}
	policyOut, err := client.IamClient.ListAttachedRolePolicies(context.TODO(), &policyLister)
	if err != nil {
		return policies, err
	}
	policies = policyOut.AttachedPolicies
	return policies, nil
}
func (client *AWSClient) TagRole(roleName string, tags map[string]string) error {
	var roleTags []types.Tag
	for tagKey, tagValue := range tags {
		roleTags = append(roleTags, types.Tag{
			Key:   &tagKey,
			Value: &tagValue,
		})
	}
	input := &iam.TagRoleInput{
		RoleName: &roleName,
		Tags:     roleTags,
	}
	_, err := client.IamClient.TagRole(context.TODO(), input)
	return err
}

func (client *AWSClient) UntagRole(roleName string, tagKeys []string) error {
	input := &iam.UntagRoleInput{
		RoleName: &roleName,
		TagKeys:  tagKeys,
	}
	_, err := client.IamClient.UntagRole(context.TODO(), input)
	return err
}

func (client *AWSClient) CreateRoleForSharedVPC(roleName, installerRoleArn string, ingressOperatorRoleArn string) (types.Role, error) {
	statement := map[string]interface{}{
		"Sid":    "Statement1",
		"Effect": "Allow",
		"Principal": map[string]interface{}{
			"AWS": []string{installerRoleArn, ingressOperatorRoleArn},
		},
		"Action": "sts:AssumeRole",
	}

	assumeRolePolicyDocument, err := completeRolePolicyDocument(statement)
	if err != nil {
		log.LogError("Failed to convert Role Policy Document into JSON: %s", err.Error())
		return types.Role{}, err
	}

	return client.CreateRole(roleName, string(assumeRolePolicyDocument), "", make(map[string]string), "/")
}

func (client *AWSClient) CreatePolicyForSharedVPC(policyName string) (string, error) {
	statement := map[string]interface{}{
		"Sid":    "Statement1",
		"Effect": "Allow",
		"Action": []string{
			"route53:GetChange",
			"route53:GetHostedZone",
			"route53:ChangeResourceRecordSets",
			"route53:ListHostedZones",
			"route53:ListHostedZonesByName",
			"route53:ListResourceRecordSets",
			"route53:ChangeTagsForResource",
			"route53:GetAccountLimit",
			"route53:ListTagsForResource",
			"route53:UpdateHostedZoneComment",
			"tag:GetResources",
			"tag:UntagResources",
		},
		"Resource": "*",
	}
	return client.CreatePolicy(policyName, statement)
}

func (client *AWSClient) CreateRoleForAdditionalPrincipals(roleName string, installerRoleArn string) (types.Role, error) {
	statement := map[string]interface{}{
		"Sid":    "Statement1",
		"Effect": "Allow",
		"Principal": map[string]interface{}{
			"AWS": []string{installerRoleArn},
		},
		"Action": "sts:AssumeRole",
	}

	assumeRolePolicyDocument, err := completeRolePolicyDocument(statement)
	if err != nil {
		log.LogError("Failed to convert Role Policy Document into JSON: %s", err.Error())
		return types.Role{}, err
	}

	return client.CreateRole(roleName, string(assumeRolePolicyDocument), "", make(map[string]string), "/")
}
