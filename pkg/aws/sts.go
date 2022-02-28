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
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/pkg/errors"
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

	return "", errors.Errorf("Couldn't extract the role name from role ARN")
}

func (c *awsClient) ValidateRoleARNAccountIDMatchCallerAccountID(roleARN string) error {
	creator, err := c.GetCreator()
	if err != nil {
		return errors.Errorf("Failed to get AWS creator: %v", err)
	}

	parsedARN, err := arn.Parse(roleARN)
	if err != nil {
		return errors.Errorf("%s", err)
	}

	if creator.AccountID != parsedARN.AccountID {
		return errors.Errorf("Role ARN '%s' doesn't match the user's account ID '%s'", roleARN, creator.AccountID)
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
