/*
Copyright (c) 2022 Red Hat, Inc.

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
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws/tags"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

func (c *awsClient) DeleteUserRole(roleName string) error {
	err := c.detachAttachedRolePolicies(aws.String(roleName))
	if err != nil {
		return err
	}

	err = c.deletePermissionsBoundary(roleName)
	if err != nil {
		return err
	}

	return c.DeleteRole(roleName, aws.String(roleName))
}

func (c *awsClient) DeleteOCMRole(roleName string) error {
	err := c.deleteOCMRolePolicies(roleName)
	if err != nil {
		return err
	}

	err = c.deletePermissionsBoundary(roleName)
	if err != nil {
		return err
	}

	return c.DeleteRole(roleName, aws.String(roleName))
}

func RoleARNToRoleName(roleARN string) (string, error) {
	parsedARN, err := arn.Parse(roleARN)
	if err != nil {
		return "", err
	}

	// ARN = arn:aws:iam::123456789123:role/prefix-OCM-Role-12345678 --> ARN.Resource = role/prefix-OCM-Role-12541229
	ARNResourceSubStr := strings.SplitN(parsedARN.Resource, "/", 2)
	if len(ARNResourceSubStr) > 1 {
		return ARNResourceSubStr[1], nil
	}

	return "", fmt.Errorf("couldn't extract the role name from role ARN")
}

func (c *awsClient) ValidateRoleARNAccountIDMatchCallerAccountID(roleARN string) error {
	creator, err := c.GetCreator()
	if err != nil {
		return fmt.Errorf("failed to get AWS creator: %v", err)
	}

	parsedARN, err := arn.Parse(roleARN)
	if err != nil {
		return err
	}

	if creator.AccountID != parsedARN.AccountID {
		return fmt.Errorf("role ARN '%s' doesn't match the user's account ID '%s'", roleARN, creator.AccountID)
	}

	return nil
}

func (c *awsClient) HasPermissionsBoundary(roleName string) (bool, error) {
	output, err := c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return false, err
	}

	return output.Role.PermissionsBoundary != nil, nil
}

func (c *awsClient) deletePermissionsBoundary(roleName string) error {
	output, err := c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return err
	}

	if output.Role.PermissionsBoundary != nil {
		_, err := c.iamClient.DeleteRolePermissionsBoundary(&iam.DeleteRolePermissionsBoundaryInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *awsClient) deleteOCMRolePolicies(roleName string) error {
	policiesOutput, err := c.iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return err
	}

	for _, policy := range policiesOutput.AttachedPolicies {
		_, err := c.iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
			PolicyArn: policy.PolicyArn,
			RoleName:  aws.String(roleName),
		})
		if err != nil {
			return err
		}

		_, err = c.iamClient.DeletePolicy(&iam.DeletePolicyInput{PolicyArn: policy.PolicyArn})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == iam.ErrCodeDeleteConflictException { // policy is attached to another entity
					continue
				}
			}
			return err
		}
	}

	return nil
}

func SortRolesByLinkedRole(roles []Role) {
	sort.SliceStable(roles, func(i, j int) bool {
		return roles[i].Linked == "Yes" && roles[j].Linked == "No"
	})
}

func UpgradeOperatorPolicies(reporter *rprtr.Object, awsClient Client, accountID string,
	prefix string, policies map[string]string, defaultPolicyVersion string,
	credRequests map[string]*cmv1.STSOperator) error {
	for credrequest, operator := range credRequests {
		policyARN := GetOperatorPolicyARN(accountID, prefix, operator.Namespace(), operator.Name())
		filename := fmt.Sprintf("openshift_%s_policy", credrequest)
		policy := policies[filename]
		policyARN, err := awsClient.EnsurePolicy(policyARN, policy,
			defaultPolicyVersion, map[string]string{
				tags.OpenShiftVersion: defaultPolicyVersion,
				tags.RolePrefix:       prefix,
				"operator_namespace":  operator.Namespace(),
				"operator_name":       operator.Name(),
			})
		if err != nil {
			return err
		}
		reporter.Infof("Upgraded policy with ARN '%s' to version '%s'", policyARN, defaultPolicyVersion)
	}
	return nil
}

func BuildOperatorRoleCommands(prefix string, accountID string, awsClient Client,
	defaultPolicyVersion string, credRequests map[string]*cmv1.STSOperator) []string {
	commands := []string{}
	for credrequest, operator := range credRequests {
		policyARN := GetOperatorPolicyARN(accountID, prefix, operator.Namespace(), operator.Name())
		_, err := awsClient.IsPolicyExists(policyARN)
		if err != nil {
			name := GetPolicyName(prefix, operator.Namespace(), operator.Name())
			iamTags := fmt.Sprintf(
				"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
				tags.OpenShiftVersion, defaultPolicyVersion,
				tags.RolePrefix, prefix,
				"operator_namespace", operator.Namespace(),
				"operator_name", operator.Name(),
			)
			createPolicy := fmt.Sprintf("aws iam create-policy \\\n"+
				"\t--policy-name %s \\\n"+
				"\t--policy-document file://openshift_%s_policy.json \\\n"+
				"\t--tags %s",
				name, credrequest, iamTags)
			commands = append(commands, createPolicy)
		} else {
			policTags := fmt.Sprintf(
				"Key=%s,Value=%s",
				tags.OpenShiftVersion, defaultPolicyVersion,
			)
			createPolicy := fmt.Sprintf("aws iam create-policy-version \\\n"+
				"\t--policy-arn %s \\\n"+
				"\t--policy-document file://openshift_%s_policy.json \\\n"+
				"\t--set-as-default",
				policyARN, credrequest)
			tagPolicy := fmt.Sprintf("aws iam tag-policy \\\n"+
				"\t--tags %s \\\n"+
				"\t--policy-arn %s",
				policTags, policyARN)
			commands = append(commands, createPolicy, tagPolicy)
		}
	}
	return commands
}
