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
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	semver "github.com/hashicorp/go-version"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/assets"
	"github.com/openshift/rosa/pkg/aws/tags"
)

var DefaultPrefix = "ManagedOpenShift"
var DefaultPolicyVersion = "4.9"

type Operator struct {
	Name                string
	Namespace           string
	ServiceAccountNames []string
}

var CredentialRequests map[string]Operator = map[string]Operator{
	"machine_api_aws_cloud_credentials": {
		Name:      "aws-cloud-credentials",
		Namespace: "openshift-machine-api",
		ServiceAccountNames: []string{
			"machine-api-controllers",
		},
	},
	"cloud_credential_operator_cloud_credential_operator_iam_ro_creds": {
		Name:      "cloud-credential-operator-iam-ro-creds",
		Namespace: "openshift-cloud-credential-operator",
		ServiceAccountNames: []string{
			"cloud-credential-operator",
		},
	},
	"image_registry_installer_cloud_credentials": {
		Name:      "installer-cloud-credentials",
		Namespace: "openshift-image-registry",
		ServiceAccountNames: []string{
			"cluster-image-registry-operator",
			"registry",
		},
	},
	"ingress_operator_cloud_credentials": {
		Name:      "cloud-credentials",
		Namespace: "openshift-ingress-operator",
		ServiceAccountNames: []string{
			"ingress-operator",
		},
	},
	"cluster_csi_drivers_ebs_cloud_credentials": {
		Name:      "ebs-cloud-credentials",
		Namespace: "openshift-cluster-csi-drivers",
		ServiceAccountNames: []string{
			"aws-ebs-csi-driver-operator",
			"aws-ebs-csi-driver-controller-sa",
		},
	},
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
	Policy     []Policy `json:"Policy,omitempty"`
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
)

var AccountRoles map[string]AccountRole = map[string]AccountRole{
	InstallerAccountRole:    {Name: "Installer", Flag: "role-arn"},
	ControlPlaneAccountRole: {Name: "ControlPlane", Flag: "controlplane-iam-role"},
	WorkerAccountRole:       {Name: "Worker", Flag: "worker-iam-role"},
	SupportAccountRole:      {Name: "Support", Flag: "support-role-arn"},
}

var roleTypeMap = map[string]string{
	"installer":             "Installer",
	"support":               "Support",
	"instance_controlplane": "Control plane",
	"instance_worker":       "Worker",
}

// PolicyDocument models an AWS IAM policy document
type PolicyDocument struct {
	ID string `json:"Id,omitempty"`
	// Specify the version of the policy language that you want to use.
	// As a best practice, use the latest 2012-10-17 version.
	Version string `json:"Version,omitempty"`
	// Use this main policy element as a container for the following elements.
	// You can include more than one statement in a policy.
	Statement []PolicyStatement `json:"Statement"`
}

// PolicyStatement models an AWS policy statement entry.
type PolicyStatement struct {
	// Include an optional statement ID to differentiate between your statements.
	Sid string `json:"Sid,omitempty"`
	// Use `Allow` or `Deny` to indicate whether the policy allows or denies access.
	Effect string `json:"Effect"`
	// If you create a resource-based policy, you must indicate the account, user, role, or
	// federated user to which you would like to allow or deny access. If you are creating an
	// IAM permissions policy to attach to a user or role, you cannot include this element.
	// The principal is implied as that user or role.
	Principal PolicyStatementPrincipal `json:"Principal,omitempty"`
	// Include a list of actions that the policy allows or denies.
	// (i.e. ec2:StartInstances, iam:ChangePassword)
	Action interface{} `json:"Action,omitempty"`
	// If you create an IAM permissions policy, you must specify a list of resources to which
	// the actions apply. If you create a resource-based policy, this element is optional. If
	// you do not include this element, then the resource to which the action applies is the
	// resource to which the policy is attached.
	Resource interface{} `json:"Resource,omitempty"`
}

type PolicyStatementPrincipal struct {
	// A service principal is an identifier that is used to grant permissions to a service.
	// The identifier for a service principal includes the service name, and is usually in the
	// following format: service-name.amazonaws.com
	Service []string `json:"Service,omitempty"`
	// You can specify an individual IAM role ARN (or array of role ARNs) as the principal.
	// In IAM roles, the Principal element in the role's trust policy specifies who can assume the role.
	// When you specify more than one principal in the element, you grant permissions to each principal.
	AWS interface{} `json:"AWS,omitempty"`
	// A federated principal uses a web identity token or SAML federation
	Federated string `json:"Federated,omitempty"`
}

func (c *awsClient) EnsureRole(name string, policy string, permissionsBoundary string,
	version string, tagList map[string]string) (string, error) {
	output, err := c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return c.createRole(name, policy, permissionsBoundary, tagList)
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

	policy, needsUpdate, err := c.updateAssumeRolePolicyPrincipals(policy, role)
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

func (c *awsClient) updateAssumeRolePolicyPrincipals(policy string, role *iam.Role) (string, bool, error) {
	oldPolicy, err := url.QueryUnescape(aws.StringValue(role.AssumeRolePolicyDocument))
	if err != nil {
		return policy, false, err
	}

	newPolicyDoc := PolicyDocument{}
	err = json.Unmarshal([]byte(policy), &newPolicyDoc)
	if err != nil {
		return policy, false, err
	}

	// Determine if role already contains trusted principal
	principals := []string{}
	hasMultiplePrincipals := false
	for _, statement := range newPolicyDoc.Statement {
		awsPrincipals := getAWSPrincipals(statement.Principal.AWS)
		// There is no AWS principal to add, nothing to do here
		if len(awsPrincipals) == 0 {
			return policy, false, nil
		}
		for _, trust := range awsPrincipals {
			// Trusted principal already exists, nothing to do here
			if strings.Contains(oldPolicy, trust) {
				return policy, false, nil
			}
			if strings.Contains(oldPolicy, `"AWS":[`) {
				hasMultiplePrincipals = true
			}
			principals = append(principals, trust)
		}
	}
	oldPrincipals := strings.Join(principals, `","`)

	// Extract existing trusted principals from existing role trust policy.
	// The AWS API is ambiguous faced with 1 vs many entries, so we cannot
	// unmarshal and have to resort to string matching...
	startSearch := `"AWS":"`
	endSearch := `"`
	if hasMultiplePrincipals {
		startSearch = `"AWS":["`
		endSearch = `"]`
	}
	start := strings.Index(oldPolicy, startSearch)
	if start >= 0 {
		start += len(startSearch)
		end := start + strings.Index(oldPolicy[start:], endSearch)
		if end >= start {
			principals = append(principals, strings.Split(oldPolicy[start:end], `","`)...)
		}
	}

	// Update assume role policy document to contain all trusted principals
	policy = strings.Replace(policy, oldPrincipals, strings.Join(principals, `","`), 1)

	return policy, true, nil
}

func (c *awsClient) createRole(name string, policy string, permissionsBoundary string,
	tagList map[string]string) (string, error) {
	if !RoleNameRE.MatchString(name) {
		return "", fmt.Errorf("Role name is invalid")
	}
	createRoleInput := &iam.CreateRoleInput{
		RoleName:                 aws.String(name),
		AssumeRolePolicyDocument: aws.String(policy),
		Tags:                     getTags(tagList),
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
	output, err := c.iamClient.ListRoleTags(&iam.ListRoleTagsInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		return false, err
	}

	return c.HasCompatibleVersionTags(output.Tags, version)
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
	version string, tagList map[string]string) (string, error) {
	output, err := c.iamClient.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return c.createPolicy(policyArn, document, tagList)
			default:
				return "", err
			}
		}
	}

	policyArn = aws.StringValue(output.Policy.Arn)

	isCompatible, err := c.isPolicyCompatible(policyArn, version)
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

func (c *awsClient) createPolicy(policyArn string, document string, tagList map[string]string) (string, error) {
	parsedArn, err := arn.Parse(policyArn)
	if err != nil {
		return "", err
	}
	policyName := strings.Split(parsedArn.Resource, "/")[1]

	output, err := c.iamClient.CreatePolicy(&iam.CreatePolicyInput{
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(document),
		Tags:           getTags(tagList),
	})
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

func (c *awsClient) isPolicyCompatible(policyArn string, version string) (bool, error) {
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

func ReadPolicyDocument(path string, args ...map[string]string) ([]byte, error) {
	bytes, err := assets.Asset(path)
	if err != nil {
		return nil, fmt.Errorf("Unable to load file %s: %s", path, err)
	}
	file := string(bytes)
	if len(args) > 0 {
		for key, val := range args[0] {
			file = strings.Replace(file, fmt.Sprintf("%%{%s}", key), val, -1)
		}
	}
	return []byte(file), nil
}

func parsePolicyDocument(path string) (PolicyDocument, error) {
	doc := PolicyDocument{}

	file, err := ReadPolicyDocument(path)
	if err != nil {
		return doc, err
	}

	err = json.Unmarshal(file, &doc)
	if err != nil {
		return doc, fmt.Errorf("Error unmarshalling statement: %s", err)
	}

	return doc, nil
}

func (c *awsClient) ListAccountRoles(version string) ([]Role, error) {
	accountRoles := []Role{}
	roles, err := c.ListRoles()
	if err != nil {
		return accountRoles, err
	}
	for _, role := range roles {
		if !checkIfROSARole(role.RoleName) {
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
			policiesOutput, err := c.iamClient.ListRolePolicies(&iam.ListRolePoliciesInput{
				RoleName: role.RoleName,
			})
			if err != nil {
				return nil, err
			}
			policies := []Policy{}
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
					PolicyDocument: policyDoc,
				}
				policies = append(policies, policy)
			}
			accountRole.Policy = policies
			accountRoles = append(accountRoles, accountRole)
		}
	}
	return accountRoles, nil
}

//Check if it is one of the ROSA account roles
func checkIfROSARole(roleName *string) bool {
	for _, prefix := range AccountRoles {
		if strings.Contains(aws.StringValue(roleName), prefix.Name) {
			return true
		}
	}
	return false
}

//Check if it is one of the ROSA account roles
func checkIfROSAOperatorRole(roleName *string) bool {
	for _, operatorRole := range CredentialRequests {
		if strings.Contains(aws.StringValue(roleName), operatorRole.Namespace) {
			return true
		}
	}
	return false
}

func getPolicyDocument(policyDocument *string) (PolicyDocument, error) {
	data := PolicyDocument{}
	if policyDocument != nil {
		val, err := url.QueryUnescape(aws.StringValue(policyDocument))
		if err != nil {
			return data, err
		}
		err = json.Unmarshal([]byte(val), &data)
		if err != nil {
			return data, err
		}
	}
	return data, nil
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

func (c *awsClient) deleteAccountRolePolicies(role *string) error {
	policies, err := c.getAccountRolePolicy(role)
	if err != nil {
		return err
	}
	for _, policyName := range policies {
		if policyName != "" {
			_, err = c.iamClient.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
				PolicyName: aws.String(policyName),
				RoleName:   role,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/**
get both role and inline policies
*/
func (c *awsClient) getAccountRolePolicy(role *string) ([]string, error) {
	policies := []string{}
	rolePolicyOutput, err := c.iamClient.GetRolePolicy(&iam.GetRolePolicyInput{RoleName: role,
		PolicyName: aws.String(fmt.Sprintf("%s-Policy", aws.StringValue(role)))})
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
	policies = append(policies, aws.StringValue(rolePolicyOutput.PolicyName))

	attachedPoliciesOutput, err := c.iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{RoleName: role})
	if err != nil {
		return nil, err
	}
	for _, policy := range attachedPoliciesOutput.AttachedPolicies {
		policies = append(policies, aws.StringValue(policy.PolicyName))
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

func (c *awsClient) GetOperatorRolesFromAccount(clusterID string) ([]string, error) {
	roleList := []string{}
	roles, err := c.ListRoles()
	if err != nil {
		return roleList, err
	}
	for _, role := range roles {
		if !checkIfROSAOperatorRole(role.RoleName) {
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
			awsPrincipal := getAWSPrincipals(statement.Principal.AWS)
			if len(awsPrincipal) > 1 {
				break
			}
			for _, a := range awsPrincipal {
				str := strings.Split(a, ":")
				if len(str) > 4 {
					if str[4] == JumpAccounts[env] {
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
		awsPrincipal := getAWSPrincipals(statement.Principal.AWS)
		for _, a := range awsPrincipal {
			str := strings.Split(a, ":")
			if len(str) > 4 {
				if str[4] == JumpAccounts[env] {
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
				return roleList, err
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
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-%s-Role", accountID, rolePrefix, prefix.Name)
		roleName := fmt.Sprintf("%s-%s-Role", rolePrefix, prefix.Name)

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
			RoleARN:  roleArn,
			RoleName: roleName,
			RoleType: prefix.Name,
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func getAWSPrincipals(awsPrincipal interface{}) []string {
	var awsArr []string
	if awsPrincipal == nil {
		return awsArr
	}
	switch reflect.TypeOf(awsPrincipal).Kind() {
	case reflect.Slice:
		value := reflect.ValueOf(awsPrincipal)
		awsArr = make([]string, value.Len())
		for i := 0; i < value.Len(); i++ {
			awsArr[i] = value.Index(i).Interface().(string)
		}
	case reflect.String:
		awsArr = make([]string, 1)
		awsArr[0] = awsPrincipal.(string)
	}
	return awsArr
}

func (c *awsClient) GetAccountRolePolicies(roles []string) (map[string]string, error) {
	roleMap := make(map[string]string)
	for _, role := range roles {
		policies, err := c.getAccountRolePolicy(aws.String(role))
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					continue
				}
			}
			return roleMap, err
		}
		policyStr := ""
		for _, policyName := range policies {
			if policyStr == "" {
				policyStr = policyName
			} else {
				policyStr = policyStr + "," + policyName
			}
		}
		roleMap[role] = policyStr
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
