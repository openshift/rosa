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

package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	semver "github.com/hashicorp/go-version"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws/tags"
)

var DefaultPrefix = "ManagedOpenShift"

type Operator struct {
	Name                string
	Namespace           string
	RoleARN             string
	ServiceAccountNames []string
	MinVersion          string
}

type AccountRole struct {
	Name string
	Flag string
}

type Role struct {
	RoleType   string   `json:"RoleType,omitempty"`
	Version    string   `json:"Version,omitempty"`
	RolePrefix string   `json:"RolePrefix,omitempty"`
	RoleName   string   `json:"RoleName,omitempty"`
	RoleARN    string   `json:"RoleARN,omitempty"`
	Linked     string   `json:"Linked,omitempty"`
	Admin      string   `json:"Admin,omitempty"`
	Policy     []Policy `json:"Policy,omitempty"`
}

type PolicyDetail struct {
	PolicyName string
	PolicyArn  string
	PolicType  string
}

type Policy struct {
	PolicyName     string         `json:"PolicyName,omitempty"`
	PolicyDocument PolicyDocument `json:"PolicyDocument,omitempty"`
}

const (
	InstallerAccountRole    = "installer"
	ControlPlaneAccountRole = "instance_controlplane"
	WorkerAccountRole       = "instance_worker"
	SupportAccountRole      = "support"
	OCMRole                 = "OCM"
	OCMUserRole             = "User"
)

var AccountRoles map[string]AccountRole = map[string]AccountRole{
	InstallerAccountRole:    {Name: "Installer", Flag: "role-arn"},
	ControlPlaneAccountRole: {Name: "ControlPlane", Flag: "controlplane-iam-role"},
	WorkerAccountRole:       {Name: "Worker", Flag: "worker-iam-role"},
	SupportAccountRole:      {Name: "Support", Flag: "support-role-arn"},
}

var OCMUserRolePolicyFile = "ocm_user"
var OCMRolePolicyFile = "ocm"
var OCMAdminRolePolicyFile = "ocm_admin"

var roleTypeMap = map[string]string{
	"installer":             "Installer",
	"support":               "Support",
	"instance_controlplane": "Control plane",
	"instance_worker":       "Worker",
}

func (c *awsClient) EnsureRole(name string, policy string, permissionsBoundary string,
	version string, tagList map[string]string, path string) (string, error) {
	output, err := c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return c.createRole(name, policy, permissionsBoundary, tagList, path)
			default:
				return "", err
			}
		}
	}

	if permissionsBoundary != "" {
		_, err = c.iamClient.PutRolePermissionsBoundary(&iam.PutRolePermissionsBoundaryInput{
			RoleName:            aws.String(name),
			PermissionsBoundary: aws.String(permissionsBoundary),
		})
	} else if output.Role.PermissionsBoundary != nil {
		_, err = c.iamClient.DeleteRolePermissionsBoundary(&iam.DeleteRolePermissionsBoundaryInput{
			RoleName: aws.String(name),
		})
	}
	if err != nil {
		return "", err
	}

	role := output.Role
	roleArn := aws.StringValue(role.Arn)

	isCompatible, err := c.isRoleCompatible(name, version)
	if err != nil {
		return roleArn, err
	}

	policy, needsUpdate, err := updateAssumeRolePolicyPrincipals(policy, role)
	if err != nil {
		return roleArn, err
	}

	if needsUpdate || !isCompatible {
		_, err = c.iamClient.UpdateAssumeRolePolicy(&iam.UpdateAssumeRolePolicyInput{
			RoleName:       aws.String(name),
			PolicyDocument: aws.String(policy),
		})
		if err != nil {
			return roleArn, err
		}

		_, err = c.iamClient.TagRole(&iam.TagRoleInput{
			RoleName: aws.String(name),
			Tags:     getTags(tagList),
		})
		if err != nil {
			return roleArn, err
		}
	}

	return roleArn, nil
}

func (c *awsClient) ValidateRoleNameAvailable(name string) (err error) {
	_, err = c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(name),
	})
	if err == nil {
		// If we found an existing role with this name we want to error
		return fmt.Errorf("A role named '%s' already exists. "+
			"Please delete the existing role, or provide a different prefix", name)
	}

	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case iam.ErrCodeNoSuchEntityException:
			// This is what we want
			return nil
		}
	}
	return fmt.Errorf("Error validating role name '%s': %v", name, err)
}

func (c *awsClient) createRole(name string, policy string, permissionsBoundary string,
	tagList map[string]string, path string) (string, error) {
	if !RoleNameRE.MatchString(name) {
		return "", fmt.Errorf("Role name is invalid")
	}
	createRoleInput := &iam.CreateRoleInput{
		RoleName:                 aws.String(name),
		AssumeRolePolicyDocument: aws.String(policy),
		Tags:                     getTags(tagList),
	}
	if path != "" {
		createRoleInput.Path = aws.String(path)
	}
	if permissionsBoundary != "" {
		createRoleInput.PermissionsBoundary = aws.String(permissionsBoundary)
	}
	output, err := c.iamClient.CreateRole(createRoleInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeEntityAlreadyExistsException:
				return "", nil
			}
		}
		return "", err
	}
	return aws.StringValue(output.Role.Arn), nil
}

func (c *awsClient) isRoleCompatible(name string, version string) (bool, error) {
	// Ignore if there is no version
	if version == "" {
		return true, nil
	}
	output, err := c.iamClient.ListRoleTags(&iam.ListRoleTagsInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		return false, err
	}

	return c.hasCompatibleMajorMinorVersionTags(output.Tags, version)
}

func (c *awsClient) PutRolePolicy(roleName string, policyName string, policy string) error {
	_, err := c.iamClient.PutRolePolicy(&iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(policy),
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *awsClient) EnsurePolicy(policyArn string, document string,
	version string, tagList map[string]string, path string) (string, error) {
	output, err := c.IsPolicyExists(policyArn)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return c.createPolicy(policyArn, document, tagList, path)
			default:
				return "", err
			}
		}
	}

	policyArn = aws.StringValue(output.Policy.Arn)

	isCompatible, err := c.IsPolicyCompatible(policyArn, version)
	if err != nil {
		return policyArn, err
	}

	if !isCompatible {
		_, err = c.iamClient.CreatePolicyVersion(&iam.CreatePolicyVersionInput{
			PolicyArn:      aws.String(policyArn),
			PolicyDocument: aws.String(document),
			SetAsDefault:   aws.Bool(true),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeLimitExceededException:
					return "", fmt.Errorf("Managed policy limit exceeded. Please delete the old ones "+
						"from your aws account for policy '%s' and try again. %v", policyArn, aerr.Message())
				default:
					return "", err
				}
			}
			return policyArn, err
		}

		_, err = c.iamClient.TagPolicy(&iam.TagPolicyInput{
			PolicyArn: aws.String(policyArn),
			Tags:      getTags(tagList),
		})
		if err != nil {
			return policyArn, err
		}
	}

	return policyArn, nil
}

func (c *awsClient) IsPolicyExists(policyArn string) (*iam.GetPolicyOutput, error) {
	output, err := c.iamClient.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: aws.String(policyArn),
	})
	return output, err
}

func (c *awsClient) IsRolePolicyExists(roleName string, policyName string) (*iam.GetRolePolicyOutput, error) {
	output, err := c.iamClient.GetRolePolicy(&iam.GetRolePolicyInput{
		PolicyName: aws.String(policyName),
		RoleName:   aws.String(roleName),
	})
	return output, err
}

func (c *awsClient) createPolicy(policyArn string, document string, tagList map[string]string,
	path string) (string, error) {
	policyName, err := GetResourceIdFromARN(policyArn)
	if err != nil {
		return "", err
	}
	createPolicyInput := &iam.CreatePolicyInput{
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(document),
		Tags:           getTags(tagList),
	}
	if path != "" {
		createPolicyInput.Path = aws.String(path)
	}

	output, err := c.iamClient.CreatePolicy(createPolicyInput)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeEntityAlreadyExistsException:
				return policyArn, nil
			}
		}
		return "", err
	}
	return aws.StringValue(output.Policy.Arn), nil
}

func (c *awsClient) IsPolicyCompatible(policyArn string, version string) (bool, error) {
	output, err := c.iamClient.ListPolicyTags(&iam.ListPolicyTagsInput{
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		return false, err
	}

	return c.HasCompatibleVersionTags(output.Tags, version)
}

func (c *awsClient) HasCompatibleVersionTags(iamTags []*iam.Tag, version string) (bool, error) {
	if len(iamTags) == 0 {
		return false, nil
	}
	for _, tag := range iamTags {
		if aws.StringValue(tag.Key) == tags.OpenShiftVersion {
			if version == aws.StringValue(tag.Value) {
				return true, nil
			}
			wantedVersion, err := semver.NewVersion(version)
			if err != nil {
				return false, err
			}
			currentVersion, err := semver.NewVersion(aws.StringValue(tag.Value))
			if err != nil {
				return false, err
			}
			return currentVersion.GreaterThanOrEqual(wantedVersion), nil
		}
	}
	return false, nil
}

func (c *awsClient) hasCompatibleMajorMinorVersionTags(iamTags []*iam.Tag, version string) (bool, error) {
	if len(iamTags) == 0 {
		return false, nil
	}
	for _, tag := range iamTags {
		if aws.StringValue(tag.Key) == tags.OpenShiftVersion {
			if version == aws.StringValue(tag.Value) {
				return true, nil
			}

			upgradeVersion, err := semver.NewVersion(version)
			if err != nil {
				return false, err
			}

			currentVersion, err := semver.NewVersion(aws.StringValue(tag.Value))
			if err != nil {
				return false, err
			}

			upgradeVersionSegments := upgradeVersion.Segments64()
			c, err := semver.NewConstraint(fmt.Sprintf(">= %d.%d",
				upgradeVersionSegments[0], upgradeVersionSegments[1]))
			if err != nil {
				return false, err
			}
			return c.Check(currentVersion), nil
		}
	}
	return false, nil
}

func (c *awsClient) AttachRolePolicy(roleName string, policyARN string) error {
	_, err := c.iamClient.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyARN),
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *awsClient) FindRoleARNs(roleType string, version string) ([]string, error) {
	roleARNs := []string{}
	roles, err := c.ListRoles()
	if err != nil {
		return roleARNs, err
	}
	for _, role := range roles {
		if !strings.Contains(aws.StringValue(role.RoleName), AccountRoles[roleType].Name) {
			continue
		}
		listRoleTagsOutput, err := c.iamClient.ListRoleTags(&iam.ListRoleTagsInput{
			RoleName: role.RoleName,
		})
		if err != nil {
			return roleARNs, err
		}
		skip := false
		isTagged := false
		for _, tag := range listRoleTagsOutput.Tags {
			tagValue := aws.StringValue(tag.Value)
			switch aws.StringValue(tag.Key) {
			case tags.RoleType:
				isTagged = true
				if tagValue != roleType {
					skip = true
					break
				}
			case tags.OpenShiftVersion:
				isTagged = true
				clusterVersion, err := semver.NewVersion(version)
				if err != nil {
					skip = true
					break
				}
				policyVersion, err := semver.NewVersion(tagValue)
				if err != nil {
					skip = true
					break
				}
				if policyVersion.LessThan(clusterVersion) {
					skip = true
					break
				}
			}
		}
		if isTagged && !skip {
			roleARNs = append(roleARNs, aws.StringValue(role.Arn))
		}
	}
	return roleARNs, nil
}

func (c *awsClient) ListRoles() ([]*iam.Role, error) {
	roles := []*iam.Role{}
	err := c.iamClient.ListRolesPages(&iam.ListRolesInput{}, func(page *iam.ListRolesOutput, lastPage bool) bool {
		roles = append(roles, page.Roles...)
		return aws.BoolValue(page.IsTruncated)
	})
	return roles, err
}

func (c *awsClient) FindPolicyARN(operator Operator, version string) (string, error) {
	policies := []*iam.Policy{}
	err := c.iamClient.ListPoliciesPages(&iam.ListPoliciesInput{
		Scope: aws.String(iam.PolicyScopeTypeLocal),
	}, func(page *iam.ListPoliciesOutput, lastPage bool) bool {
		policies = append(policies, page.Policies...)
		return aws.BoolValue(page.IsTruncated)
	})
	if err != nil {
		return "", err
	}
	for _, policy := range policies {
		listPolicyTagsOutput, err := c.iamClient.ListPolicyTags(&iam.ListPolicyTagsInput{
			PolicyArn: policy.Arn,
		})
		if err != nil {
			return "", err
		}
		skip := false
		isTagged := false
		for _, tag := range listPolicyTagsOutput.Tags {
			tagValue := aws.StringValue(tag.Value)
			switch aws.StringValue(tag.Key) {
			case "operator_namespace":
				isTagged = true
				if tagValue != operator.Namespace {
					skip = true
					break
				}
			case "operator_name":
				isTagged = true
				if tagValue != operator.Name {
					skip = true
					break
				}
			case tags.OpenShiftVersion:
				isTagged = true
				if tagValue != version {
					skip = true
					break
				}
			}
		}
		if isTagged && !skip {
			return aws.StringValue(policy.Arn), nil
		}
	}
	return "", nil
}

func getTags(tagList map[string]string) []*iam.Tag {
	iamTags := []*iam.Tag{}
	for k, v := range tagList {
		iamTags = append(iamTags, &iam.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return iamTags
}

func roleHasTag(roleTags []*iam.Tag, tagKey string, tagValue string) bool {
	for _, tag := range roleTags {
		if aws.StringValue(tag.Key) == tagKey && aws.StringValue(tag.Value) == tagValue {
			return true
		}
	}

	return false
}

func IsOCMRole(roleName *string) bool {
	return strings.Contains(aws.StringValue(roleName), fmt.Sprintf("%s-Role", OCMRole))
}

// IsUserRole checks the role tags in addition to the role name, because the word 'user' is common
func (c *awsClient) IsUserRole(roleName *string) (bool, error) {
	if strings.Contains(aws.StringValue(roleName), OCMUserRole) {
		roleTags, err := c.iamClient.ListRoleTags(&iam.ListRoleTagsInput{
			RoleName: roleName,
		})
		if err != nil {
			return false, err
		}

		return roleHasTag(roleTags.Tags, tags.RoleType, OCMUserRole), nil
	}

	return false, nil
}

func (c *awsClient) ListUserRoles() ([]Role, error) {
	var userRoles []Role
	roles, err := c.ListRoles()
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		isUserRole, err := c.IsUserRole(role.RoleName)
		if err != nil {
			return nil, err
		}

		if isUserRole {
			var userRole Role
			userRole.RoleName = aws.StringValue(role.RoleName)
			userRole.RoleARN = aws.StringValue(role.Arn)

			userRoles = append(userRoles, userRole)
		}
	}

	return userRoles, nil
}

func (c *awsClient) ListOCMRoles() ([]Role, error) {
	var ocmRoles []Role
	roles, err := c.ListRoles()
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if IsOCMRole(role.RoleName) {
			var ocmRole Role
			ocmRole.RoleName = aws.StringValue(role.RoleName)
			ocmRole.RoleARN = aws.StringValue(role.Arn)

			roleTags, err := c.iamClient.ListRoleTags(&iam.ListRoleTagsInput{
				RoleName: role.RoleName,
			})
			if err != nil {
				return nil, err
			}
			if roleHasTag(roleTags.Tags, tags.AdminRole, "true") {
				ocmRole.Admin = "Yes"
			} else {
				ocmRole.Admin = "No"
			}

			ocmRoles = append(ocmRoles, ocmRole)
		}
	}

	return ocmRoles, nil
}

func (c *awsClient) listPolicies(role *iam.Role) ([]Policy, error) {
	policiesOutput, err := c.iamClient.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName: role.RoleName,
	})
	if err != nil {
		return nil, err
	}

	var policies []Policy
	for _, policyName := range policiesOutput.PolicyNames {
		policyOutput, err := c.iamClient.GetRolePolicy(&iam.GetRolePolicyInput{
			PolicyName: policyName,
			RoleName:   role.RoleName,
		})
		if err != nil {
			return nil, err
		}
		policyDoc, err := getPolicyDocument(policyOutput.PolicyDocument)
		if err != nil {
			return nil, err
		}
		policy := Policy{
			PolicyName:     aws.StringValue(policyOutput.PolicyName),
			PolicyDocument: *policyDoc,
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

func (c *awsClient) ListAccountRoles(version string) ([]Role, error) {
	accountRoles := []Role{}
	roles, err := c.ListRoles()
	if err != nil {
		return accountRoles, err
	}
	for _, role := range roles {
		if !checkIfAccountRole(role.RoleName) {
			continue
		}
		accountRole := Role{}
		listRoleTagsOutput, err := c.iamClient.ListRoleTags(&iam.ListRoleTagsInput{
			RoleName: role.RoleName,
		})
		if err != nil {
			return accountRoles, err
		}

		isTagged := false
		skip := false
		for _, tag := range listRoleTagsOutput.Tags {
			switch aws.StringValue(tag.Key) {
			case tags.RoleType:
				isTagged = true
				accountRole.RoleType = roleTypeMap[aws.StringValue(tag.Value)]
			case tags.OpenShiftVersion:
				tagValue := aws.StringValue(tag.Value)
				if version != "" && tagValue != version {
					skip = true
					break
				}
				isTagged = true
				accountRole.Version = tagValue
			}
		}
		if isTagged && !skip {
			accountRole.RoleName = aws.StringValue(role.RoleName)
			accountRole.RoleARN = aws.StringValue(role.Arn)
			policies, err := c.listPolicies(role)
			if err != nil {
				return nil, err
			}
			accountRole.Policy = policies

			accountRoles = append(accountRoles, accountRole)
		}
	}
	return accountRoles, nil
}

//Check if it is one of the ROSA account roles
func checkIfAccountRole(roleName *string) bool {
	for _, prefix := range AccountRoles {
		if strings.Contains(aws.StringValue(roleName), prefix.Name) {
			return true
		}
	}
	return false
}

//Check if it is one of the ROSA account roles
func checkIfROSAOperatorRole(roleName *string, credRequest map[string]*cmv1.STSOperator) bool {
	for _, operatorRole := range credRequest {
		if strings.Contains(aws.StringValue(roleName), operatorRole.Namespace()) {
			return true
		}
	}
	return false
}

func (c *awsClient) DeleteOperatorRole(roleName string) error {
	role := aws.String(roleName)
	err := c.detachOperatorRolePolicies(role)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Printf("Policies does not exists for role '%s'",
					roleName)
			}
		}
		return err
	}
	return c.DeleteRole(roleName, role)
}

func (c *awsClient) DeleteRole(role string, r *string) error {
	_, err := c.iamClient.DeleteRole(&iam.DeleteRoleInput{RoleName: r})
	if err != nil {
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					return fmt.Errorf("operator role '%s' does not exists.skipping",
						role)
				}
			}
			return err
		}
	}
	return nil
}

func (c *awsClient) GetInstanceProfilesForRole(r string) ([]string, error) {
	instanceProfiles := []string{}
	profiles, err := c.iamClient.ListInstanceProfilesForRole(&iam.ListInstanceProfilesForRoleInput{
		RoleName: aws.String(r),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return instanceProfiles, nil
			}
		}
		return nil, err
	}
	for _, profile := range profiles.InstanceProfiles {
		instanceProfiles = append(instanceProfiles, aws.StringValue(profile.InstanceProfileName))
	}
	return instanceProfiles, nil
}

func (c *awsClient) DeleteAccountRole(roleName string) error {
	role := aws.String(roleName)
	err := c.deleteAccountRolePolicies(role)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				//do nothing
			default:
				return err
			}
		} else {
			return err
		}
	}
	return c.DeleteRole(roleName, role)
}

func (c *awsClient) detachAttachedRolePolicies(role *string) error {
	attachedPoliciesOutput, err := c.iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: role,
	})
	if err != nil {
		return err
	}
	for _, policy := range attachedPoliciesOutput.AttachedPolicies {
		_, err = c.iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
			PolicyArn: policy.PolicyArn,
			RoleName:  role,
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					continue
				}
			}
			return err
		}
	}

	return nil
}

func (c *awsClient) DeleteInlineRolePolicies(role string) error {
	listRolePolicyOutput, err := c.iamClient.ListRolePolicies(&iam.ListRolePoliciesInput{RoleName: aws.String(role)})
	if err != nil {
		return err
	}
	for _, policyName := range listRolePolicyOutput.PolicyNames {
		_, err = c.iamClient.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
			PolicyName: policyName,
			RoleName:   aws.String(role),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					continue
				}
			}
			return err
		}
	}

	return nil
}

func (c *awsClient) deleteAccountRolePolicies(role *string) error {
	err := c.detachAttachedRolePolicies(role)
	if err != nil {
		return err
	}
	err = c.DeleteInlineRolePolicies(aws.StringValue(role))
	if err != nil {
		return err
	}

	return nil
}
func (c *awsClient) GetAttachedPolicy(role *string) ([]PolicyDetail, error) {
	policies := []PolicyDetail{}
	attachedPoliciesOutput, err := c.iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{RoleName: role})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				break
			default:
				return policies, err
			}
		} else {
			return policies, err
		}
	}

	for _, policy := range attachedPoliciesOutput.AttachedPolicies {
		policyDetail := PolicyDetail{
			PolicyName: aws.StringValue(policy.PolicyName),
			PolicyArn:  aws.StringValue(policy.PolicyArn),
			PolicType:  Attached,
		}
		policies = append(policies, policyDetail)
	}

	rolePolicyOutput, err := c.iamClient.ListRolePolicies(&iam.ListRolePoliciesInput{RoleName: role})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				break
			default:
				return policies, err
			}
		} else {
			return policies, err
		}
	}
	for _, policy := range rolePolicyOutput.PolicyNames {
		policyDetail := PolicyDetail{
			PolicyName: aws.StringValue(policy),
			PolicType:  Inline,
		}
		policies = append(policies, policyDetail)
	}

	return policies, nil
}

func (c *awsClient) detachOperatorRolePolicies(role *string) error {
	// get attached role policies as operator roles have managed policies
	policiesOutput, err := c.iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: role,
	})
	if err != nil {
		return err
	}
	for _, policy := range policiesOutput.AttachedPolicies {
		_, err := c.iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{PolicyArn: policy.PolicyArn, RoleName: role})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *awsClient) GetOperatorRolesFromAccount(clusterID string,
	credRequest map[string]*cmv1.STSOperator) ([]string, error) {
	roleList := []string{}
	roles, err := c.ListRoles()
	if err != nil {
		return roleList, err
	}
	for _, role := range roles {
		if !checkIfROSAOperatorRole(role.RoleName, credRequest) {
			continue
		}
		listRoleTagsOutput, err := c.iamClient.ListRoleTags(&iam.ListRoleTagsInput{
			RoleName: role.RoleName,
		})
		if err != nil {
			return roleList, err
		}
		isTagged := false
		for _, tag := range listRoleTagsOutput.Tags {
			switch aws.StringValue(tag.Key) {
			case tags.ClusterID:
				if aws.StringValue(tag.Value) == clusterID {
					isTagged = true
					break
				}
			}
		}
		if isTagged {
			roleList = append(roleList, aws.StringValue(role.RoleName))
		}
	}
	return roleList, nil
}

func (c *awsClient) GetPolicies(roles []string) (map[string][]string, error) {
	roleMap := make(map[string][]string)
	for _, role := range roles {
		policyArr := []string{}
		policiesOutput, err := c.iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(role),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					continue
				}
			}
			return roleMap, err
		}
		for _, policy := range policiesOutput.AttachedPolicies {
			policyArr = append(policyArr, aws.StringValue(policy.PolicyArn))
		}
		roleMap[role] = policyArr
	}
	return roleMap, nil
}

func (c *awsClient) GetAccountRolesForCurrentEnv(env string, accountID string) ([]Role, error) {
	roleList := []Role{}
	roles, err := c.ListRoles()
	if err != nil {
		return roleList, err
	}
	for _, role := range roles {
		if role.RoleName == nil {
			continue
		}
		if !strings.Contains(aws.StringValue(role.RoleName), ("Installer-Role")) {
			continue
		}
		policyDoc, err := getPolicyDocument(role.AssumeRolePolicyDocument)
		if err != nil {
			return roleList, err
		}
		statements := policyDoc.Statement
		for _, statement := range statements {
			awsPrincipal := statement.GetAWSPrincipals()
			if len(awsPrincipal) > 1 {
				break
			}
			for _, a := range awsPrincipal {
				str := strings.Split(a, ":")
				if len(str) > 4 {
					if str[4] == GetJumpAccount(env) {
						roles, err := c.buildRoles(aws.StringValue(role.RoleName), accountID)
						if err != nil {
							return roleList, err
						}
						roleList = append(roleList, roles...)
						break
					}
				}
			}
		}
	}
	return roleList, nil
}

func (c *awsClient) GetAccountRoleForCurrentEnv(env string, roleName string) (Role, error) {
	role := Role{}
	// This is done to ensure user did not provide invalid role before we check for installer role
	accountRoleResponse, err := c.iamClient.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					return role, errors.NotFound.Errorf("Role '%s' not found", roleName)
				}
			}
		}
		return role, err
	}

	assumePolicyDoc := accountRoleResponse.Role.AssumeRolePolicyDocument
	if !strings.Contains(roleName, ("Installer-Role")) {
		installerRoleResponse, err := c.checkInstallerRoleExists(roleName)
		if err != nil {
			return role, err
		}
		if installerRoleResponse == nil {
			return Role{
				RoleARN:  aws.StringValue(accountRoleResponse.Role.Arn),
				RoleName: roleName,
			}, nil
		}
		assumePolicyDoc = installerRoleResponse.AssumeRolePolicyDocument
	}
	policyDoc, err := getPolicyDocument(assumePolicyDoc)
	if err != nil {
		return role, err
	}
	statements := policyDoc.Statement
	for _, statement := range statements {
		awsPrincipal := statement.GetAWSPrincipals()
		for _, a := range awsPrincipal {
			str := strings.Split(a, ":")
			if len(str) > 4 {
				if str[4] == GetJumpAccount(env) {
					r := Role{
						RoleARN:  aws.StringValue(accountRoleResponse.Role.Arn),
						RoleName: roleName,
					}
					return r, nil
				}
			}
		}
	}
	return role, nil
}

func (c *awsClient) checkInstallerRoleExists(roleName string) (*iam.Role, error) {
	rolePrefix := ""
	for _, prefix := range AccountRoles {
		p := fmt.Sprintf("%s-Role", prefix.Name)
		if strings.Contains(roleName, p) {
			rolePrefix = strings.Split(roleName, p)[0]
		}
	}
	installerRole := fmt.Sprintf("%s%s-Role", rolePrefix, "Installer")
	installerRoleResponse, err := c.iamClient.GetRole(&iam.GetRoleInput{RoleName: aws.String(installerRole)})
	//We try our best to determine the environment based on the trust policy in the installer
	//If the installer role is deleted we can assume that there is no cluster using the role
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return nil, nil
			default:
				return nil, err
			}
		}
		return nil, err
	}

	return installerRoleResponse.Role, nil
}

func (c *awsClient) GetAccountRoleForCurrentEnvWithPrefix(env string, rolePrefix string) ([]Role, error) {
	roleList := []Role{}
	for _, prefix := range AccountRoles {
		role, err := c.GetAccountRoleForCurrentEnv(env, fmt.Sprintf("%s-%s-Role", rolePrefix, prefix.Name))
		if err != nil {
			if errors.GetType(err) != errors.NotFound {
				return nil, err
			}
		}
		roleList = append(roleList, role)
	}
	return roleList, nil
}

func (c *awsClient) buildRoles(roleName string, accountID string) ([]Role, error) {
	roles := []Role{}
	rolePrefix := strings.Split(roleName, "-Installer-Role")[0]
	for _, prefix := range AccountRoles {
		roleName := fmt.Sprintf("%s-%s-Role", rolePrefix, prefix.Name)
		roleARN := GetRoleARN(accountID, roleName, "")

		if prefix.Name != "Installer" {
			_, err := c.iamClient.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case iam.ErrCodeNoSuchEntityException:
						continue
					}
				}
				return roles, err
			}
		}
		role := Role{
			RoleARN:  roleARN,
			RoleName: roleName,
			RoleType: prefix.Name,
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (c *awsClient) GetAccountRolePolicies(roles []string) (map[string][]PolicyDetail, error) {
	roleMap := make(map[string][]PolicyDetail)
	for _, role := range roles {
		policies, err := c.GetAttachedPolicy(aws.String(role))
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					continue
				}
			}
			return roleMap, err
		}
		roleMap[role] = policies
	}
	return roleMap, nil
}

func (c *awsClient) GetOpenIDConnectProvider(clusterID string) (string, error) {
	providers, err := c.iamClient.ListOpenIDConnectProviders(&iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return "", err
	}
	for _, provider := range providers.OpenIDConnectProviderList {
		providerValue := aws.StringValue(provider.Arn)
		connectProvider, err := c.iamClient.GetOpenIDConnectProvider(&iam.GetOpenIDConnectProviderInput{
			OpenIDConnectProviderArn: provider.Arn,
		})
		if err != nil {
			return "", err
		}
		isTagged := false
		for _, providerTag := range connectProvider.Tags {
			switch aws.StringValue(providerTag.Key) {
			case tags.ClusterID:
				if aws.StringValue(providerTag.Value) == clusterID {
					isTagged = true
					break
				}
			}
		}
		if isTagged {
			return providerValue, nil
		}
		if strings.Contains(providerValue, clusterID) {
			return providerValue, nil
		}
	}
	return "", nil
}

func (c *awsClient) GetRoleARNPath(prefix string) (string, error) {
	for _, accountRole := range AccountRoles {
		roleName := fmt.Sprintf("%s-%s-Role", prefix, accountRole.Name)
		role, err := c.iamClient.GetRole(&iam.GetRoleInput{
			RoleName: aws.String(roleName),
		})
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return "", errors.NotFound.Errorf("Roles with the prefix'%s' not found", prefix)
			}
		}
		return GetRolePath(aws.StringValue(role.Role.Arn))
	}
	return "", nil
}

func (c *awsClient) IsUpgradedNeededForAccountRolePolicies(prefix string, version string) (bool, error) {
	for _, accountRole := range AccountRoles {
		roleName := fmt.Sprintf("%s-%s-Role", prefix, accountRole.Name)
		role, err := c.iamClient.GetRole(&iam.GetRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					return false, errors.NotFound.Errorf("Roles with the prefix '%s' not found", prefix)
				}
			}
			return false, err
		}
		isCompatible, err := c.validateRoleUpgradeVersionCompatibility(aws.StringValue(role.Role.RoleName),
			version)

		if err != nil {
			return false, err
		}
		if !isCompatible {
			return true, nil
		}
	}
	return false, nil
}

func (c *awsClient) IsUpgradedNeededForAccountRolePoliciesForCluster(cluster *cmv1.Cluster,
	version string) (bool, error) {

	roles := []string{}
	roles = append(roles, cluster.AWS().STS().RoleARN(), cluster.AWS().STS().SupportRoleARN(),
		cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN(), cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN())

	for _, accountRoleARN := range roles {
		role, err := c.GetRoleByARN(accountRoleARN)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					return false, errors.NotFound.Errorf("'%s' role not found", accountRoleARN)
				}
			}
			return false, err
		}
		isCompatible, err := c.validateRoleUpgradeVersionCompatibility(aws.StringValue(role.RoleName),
			version)

		if err != nil {
			return false, err
		}
		if !isCompatible {
			return true, nil
		}
	}
	return false, nil
}

func (c *awsClient) UpdateTag(roleName string, defaultPolicyVersion string) error {
	return c.AddRoleTag(roleName, tags.OpenShiftVersion, defaultPolicyVersion)
}

func (c *awsClient) AddRoleTag(roleName string, key string, value string) error {
	role, err := c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return err
	}
	_, err = c.iamClient.TagRole(&iam.TagRoleInput{
		RoleName: role.Role.RoleName,
		Tags: []*iam.Tag{
			{
				Key:   aws.String(key),
				Value: aws.String(value),
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *awsClient) IsUpgradedNeededForOperatorRolePolicies(cluster *cmv1.Cluster, accountID string, version string) (
	bool, error) {
	for _, operator := range cluster.AWS().STS().OperatorIAMRoles() {
		roleName, err := GetResourceIdFromARN(operator.RoleARN())
		if err != nil {
			return false, err
		}
		_, err = c.iamClient.GetRole(&iam.GetRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					return false, errors.NotFound.Errorf("Operator Role '%s' does not exists for the "+
						"cluster '%s'", roleName, cluster.ID())
				}
			}
			return false, err
		}
		isCompatible, err := c.validateRoleUpgradeVersionCompatibility(roleName, version)
		if err != nil {
			return false, err
		}
		if !isCompatible {
			return true, nil
		}
	}
	return false, nil
}

func (c *awsClient) IsUpgradedNeededForOperatorRolePoliciesUsingPrefix(prefix string, accountID string,
	version string, credRequests map[string]*cmv1.STSOperator, path string) (bool, error) {
	for _, operator := range credRequests {
		policyARN := GetOperatorPolicyARN(accountID, prefix, operator.Namespace(), operator.Name(), path)
		_, err := c.IsPolicyExists(policyARN)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					return true, nil
				default:
					return false, err
				}
			}
		}
		isCompatible, err := c.isRolePoliciesCompatibleForUpgrade(policyARN, version)
		if err != nil {
			return false, err
		}
		if !isCompatible {
			return true, nil
		}
	}
	return false, nil
}

func (c *awsClient) validateRoleUpgradeVersionCompatibility(roleName string,
	version string) (bool, error) {
	attachedPolicies, err := c.GetAttachedPolicy(aws.String(roleName))
	if err != nil {
		return false, err
	}
	isAttachedPolicyExists := false
	for _, attachedPolicy := range attachedPolicies {
		if attachedPolicy.PolicType == Inline {
			continue
		}
		isAttachedPolicyExists = true
		isCompatible, err := c.isRolePoliciesCompatibleForUpgrade(attachedPolicy.PolicyArn, version)
		if err != nil {
			return false, errors.Errorf("Failed to validate role polices : %v", err)
		}
		if !isCompatible {
			return false, nil
		}
	}
	if !isAttachedPolicyExists {
		return false, nil
	}
	return true, nil
}

func (c *awsClient) isRolePoliciesCompatibleForUpgrade(policyARN string, version string) (bool, error) {
	policyTagOutput, err := c.iamClient.ListPolicyTags(&iam.ListPolicyTagsInput{
		PolicyArn: aws.String(policyARN),
	})
	if err != nil {
		return false, err
	}
	return c.hasCompatibleMajorMinorVersionTags(policyTagOutput.Tags, version)
}

func (c *awsClient) GetAccountRoleVersion(roleName string) (string, error) {
	role, err := c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return "", err
	}
	_, version := GetTagValues(role.Role.Tags)
	return version, nil
}

func (c *awsClient) IsAdminRole(roleName string) (bool, error) {
	role, err := c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return false, err
	}

	for _, tag := range role.Role.Tags {
		if aws.StringValue(tag.Key) == tags.AdminRole && aws.StringValue(tag.Value) == "true" {
			return true, nil
		}
	}

	return false, nil
}
