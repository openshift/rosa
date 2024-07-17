package aws_client

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) CreateIAMPolicy(policyName string, policyDocument string, tags map[string]string) (*types.Policy, error) {
	var policyTags []types.Tag
	for tagKey, tagValue := range tags {
		policyTags = append(policyTags, types.Tag{
			Key:   &tagKey,
			Value: &tagValue,
		})
	}
	description := "Policy for ocm-qe testing"
	input := &iam.CreatePolicyInput{
		PolicyName:     &policyName,
		PolicyDocument: &policyDocument,
		Tags:           policyTags,
		Description:    &description,
	}
	output, err := client.IamClient.CreatePolicy(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	err = client.WaitForResourceExisting("policy-"+*output.Policy.Arn, 10) // add a prefix to meet the resourceExisting split rule
	return output.Policy, err
}

func (client *AWSClient) GetIAMPolicy(policyArn string) (*types.Policy, error) {
	input := &iam.GetPolicyInput{
		PolicyArn: &policyArn,
	}
	out, err := client.IamClient.GetPolicy(context.TODO(), input)
	return out.Policy, err
}

func (client *AWSClient) DeleteIAMPolicy(arn string) error {
	input := &iam.DeletePolicyInput{
		PolicyArn: &arn,
	}
	err := client.DeletePolicyVersions(arn)
	if err != nil {
		return err
	}
	_, err = client.IamClient.DeletePolicy(context.TODO(), input)
	return err
}

func (client *AWSClient) AttachIAMPolicy(roleName string, policyArn string) error {
	input := &iam.AttachRolePolicyInput{
		PolicyArn: &policyArn,
		RoleName:  &roleName,
	}
	_, err := client.IamClient.AttachRolePolicy(context.TODO(), input)
	return err

}
func (client *AWSClient) DetachIAMPolicy(roleAName string, policyArn string) error {
	input := &iam.DetachRolePolicyInput{
		RoleName:  &roleAName,
		PolicyArn: &policyArn,
	}
	_, err := client.IamClient.DetachRolePolicy(context.TODO(), input)
	return err
}
func (client *AWSClient) GetCustomerIAMPolicies() ([]types.Policy, error) {

	maxItem := int32(1000)
	input := &iam.ListPoliciesInput{
		Scope:    "Local",
		MaxItems: &maxItem,
	}
	out, err := client.IamClient.ListPolicies(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	return out.Policies, err

}
func CleanByOutDate(policy types.Policy) bool {
	now := time.Now().UTC()
	return policy.CreateDate.Add(7 * time.Hour * 24).Before(now)
}

func CleanByName(policy types.Policy) bool {
	return strings.Contains(*policy.PolicyName, "sdq-ci-")
}

func (client *AWSClient) FilterNeedCleanPolicies(cleanRule func(types.Policy) bool) ([]types.Policy, error) {
	needClean := []types.Policy{}

	policies, err := client.GetCustomerIAMPolicies()
	if err != nil {
		return needClean, err
	}
	for _, policy := range policies {
		if cleanRule(policy) {

			needClean = append(needClean, policy)
		}
	}
	return needClean, nil
}

func (client *AWSClient) DeletePolicy(arn string) error {
	input := &iam.DeletePolicyInput{
		PolicyArn: &arn,
	}
	err := client.DeletePolicyVersions(arn)
	if err != nil {
		return err
	}
	_, err = client.IamClient.DeletePolicy(context.TODO(), input)
	return err
}

func (client *AWSClient) DeletePolicyVersions(policyArn string) error {
	input := &iam.ListPolicyVersionsInput{
		PolicyArn: &policyArn,
	}
	out, err := client.IamClient.ListPolicyVersions(context.TODO(), input)
	if err != nil {
		return err
	}
	for _, version := range out.Versions {
		if version.IsDefaultVersion {
			continue
		}
		input := &iam.DeletePolicyVersionInput{
			PolicyArn: &policyArn,
			VersionId: version.VersionId,
		}
		_, err = client.IamClient.DeletePolicyVersion(context.TODO(), input)
		if err != nil {
			return err
		}
	}
	return nil
}
func (client *AWSClient) CleanPolicies(cleanRule func(types.Policy) bool) error {
	policies, err := client.FilterNeedCleanPolicies(cleanRule)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		if *policy.AttachmentCount == 0 {
			log.LogInfo("Can be deleted: %s", *policy.Arn)
			err = client.DeletePolicy(*policy.Arn)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (client *AWSClient) TagPolicy(policyArn string, tags map[string]string) error {
	var policyTags []types.Tag
	for tagKey, tagValue := range tags {
		policyTags = append(policyTags, types.Tag{
			Key:   &tagKey,
			Value: &tagValue,
		})
	}
	input := &iam.TagPolicyInput{
		PolicyArn: &policyArn,
		Tags:      policyTags,
	}
	_, err := client.IamClient.TagPolicy(context.TODO(), input)
	return err
}

func (client *AWSClient) UntagPolicy(policyArn string, tagKeys []string) error {
	input := &iam.UntagPolicyInput{
		PolicyArn: &policyArn,
		TagKeys:   tagKeys,
	}
	_, err := client.IamClient.UntagPolicy(context.TODO(), input)
	return err
}