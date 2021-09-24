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

	"github.com/openshift/rosa/assets"
	"github.com/openshift/rosa/pkg/aws/tags"
)

var DefaultPrefix = "ManagedOpenShift"

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

var AccountRoles map[string]AccountRole = map[string]AccountRole{
	"installer":             {Name: "Installer", Flag: "role-arn"},
	"instance_controlplane": {Name: "ControlPlane", Flag: "master-iam-role"},
	"instance_worker":       {Name: "Worker", Flag: "worker-iam-role"},
	"support":               {Name: "Support", Flag: "support-role-arn"},
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
	Federated interface{} `json:"Federated,omitempty"`
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
		awsPrinciples := getAWSPrinciple(statement.Principal.AWS)
		// There is no AWS principal to add, nothing to do here
		if len(awsPrinciples) == 0 {
			return policy, false, nil
		}
		for _, trust := range awsPrinciples {
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

	return hasCompatibleTags(output.Tags, version)
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

	return hasCompatibleTags(output.Tags, version)
}

func hasCompatibleTags(iamTags []*iam.Tag, version string) (bool, error) {
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
				if tagValue != version {
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


/**
1) Get the trust
2) Check with the account
3) Yes -->
*/

func (c *awsClient) GetAccountRolesForCurrentEnv(env string) ([]string, error) {
	roleList := []string{}
	roles, err := c.ListRoles()
	if err != nil {
		return roleList, err
	}
	for _, role := range roles {
		if role.RoleName == nil{
			continue
		}
		//reporter.Infof("%v",aws.StringValue(role.Arn))
		if !strings.Contains(aws.StringValue(role.RoleName), ("Installer-Role")) {
			continue
		}
		policyDoc, err := getPolicyDocument(role.AssumeRolePolicyDocument)
		if err != nil {
			return roleList, err
		}
		statements := policyDoc.Statement
		for _, statement := range statements {
			awsArr := statement.Principal.AWS
			awsPriciple := getAWSPrinciple(awsArr)
			for _, a := range awsPriciple {
				str := strings.Split(a, "::")
				if len(str) > 1 {
					c := strings.Split(str[1], ":")
					if c[0] == JumpAccounts[env] {
						roles:=buildRoles(aws.StringValue(role.RoleName))
						roleList = append(roleList, roles...)
						break
					}
				}
			}
		}
	}

	return roleList, nil
}


func buildRoles(roleName string) []string{
	roles := []string{}
	role := strings.Split(roleName,"-Installer-Role")[0]
	for _, prefix := range AccountRoles {
		roles = append(roles, fmt.Sprintf("%s-%s",role,prefix.Name))
	}
	return roles
}


func getAWSPrinciple(awsPrinciple interface{}) []string {
	var awsArr []string
	switch reflect.TypeOf(awsPrinciple).Kind() {
	case reflect.Slice:
		value := reflect.ValueOf(awsPrinciple)
		awsArr = make([]string, value.Len())
		for i := 0; i < value.Len(); i++ {
			awsArr[i] = value.Index(i).Interface().(string)
		}
	case reflect.String:
		awsArr = make([]string, 1)
		awsArr[0] = awsPrinciple.(string)
	}
	return awsArr

}

