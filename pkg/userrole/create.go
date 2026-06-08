/*
Copyright (c) 2021 Red Hat, Inc.

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

package userrole

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/reporter"
)

// Input holds the values collected for user role creation flows.
type Input struct {
	Prefix              string
	PermissionsBoundary string
	Path                string
	Mode                string
}

// Validate checks user role input collected from flags or interactive prompts.
func Validate(input Input) error {
	if len(input.Prefix) > 32 {
		return fmt.Errorf("expected a prefix with no more than 32 characters")
	}
	if !aws.RoleNameRE.MatchString(input.Prefix) {
		return fmt.Errorf("expected a valid role prefix matching %s", aws.RoleNameRE.String())
	}
	if input.PermissionsBoundary != "" {
		if err := aws.ARNValidator(input.PermissionsBoundary); err != nil {
			return fmt.Errorf("expected a valid policy ARN for permissions boundary: %s", err)
		}
	}
	if input.Path != "" && !aws.ARNPath.MatchString(input.Path) {
		return fmt.Errorf("the specified value for path is invalid. " +
			"It must begin and end with '/' and contain only alphanumeric characters and/or '/' characters")
	}
	if input.Mode != "" && input.Mode != interactive.ModeAuto && input.Mode != interactive.ModeManual {
		return fmt.Errorf("invalid mode. Allowed values are %s", interactive.Modes)
	}
	return nil
}

// RoleARN returns the ARN that would be created for the given input.
func RoleARN(prefix, path, userName string, creator *aws.Creator) string {
	roleName := aws.GetUserRoleName(prefix, aws.OCMUserRole, userName)
	return aws.GetRoleARN(creator.AccountID, roleName, path, creator.Partition)
}

// RoleName returns the IAM role name that would be created for the given input.
func RoleName(prefix, userName string) string {
	return aws.GetUserRoleName(prefix, aws.OCMUserRole, userName)
}

// BuildCommands returns the manual-mode AWS CLI and link commands for the user role.
func BuildCommands(prefix, path, userName string, creator *aws.Creator, env, permissionsBoundary string) string {
	commands := []string{}
	roleName := aws.GetUserRoleName(prefix, aws.OCMUserRole, userName)
	roleARN := aws.GetRoleARN(creator.AccountID, roleName, path, creator.Partition)
	iamTags := map[string]string{
		tags.RolePrefix:    prefix,
		tags.RoleType:      aws.OCMUserRole,
		tags.Environment:   env,
		tags.RedHatManaged: "true",
	}
	createRole := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreateRole).
		AddParam(awscb.RoleName, roleName).
		AddParam(awscb.AssumeRolePolicyDocument,
			fmt.Sprintf("file://sts_%s_trust_policy.json", aws.OCMUserRolePolicyFile)).
		AddParam(awscb.PermissionsBoundary, permissionsBoundary).
		AddTags(iamTags).
		AddParam(awscb.Path, path).
		Build()
	linkRole := fmt.Sprintf("rosa link user-role --role-arn %s", roleARN)
	commands = append(commands, createRole, linkRole)
	return awscb.JoinCommands(commands)
}

// GeneratePolicyFiles writes the trust policy document required for manual mode.
func GeneratePolicyFiles(reporter reporter.Logger, env, partition, accountID string,
	policies map[string]*cmv1.AWSSTSPolicy) error {
	filename := fmt.Sprintf("sts_%s_trust_policy", aws.OCMUserRolePolicyFile)
	policyDetail := aws.GetPolicyDetails(policies, filename)
	policy := aws.InterpolatePolicyDocument(partition, policyDetail, map[string]string{
		"partition":      partition,
		"aws_account_id": aws.GetJumpAccount(env),
		"ocm_account_id": accountID,
	})

	filename = aws.GetFormattedFileName(filename)
	reporter.Debugf("Saving '%s' to the current directory", filename)
	return helper.SaveDocument(policy, filename)
}
