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
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	semver "github.com/hashicorp/go-version"

	"github.com/openshift/rosa/assets"
	"github.com/openshift/rosa/pkg/aws/tags"
)

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

var AccountRoles map[string]AccountRole = map[string]AccountRole{
	"installer":             {Name: "Installer", Flag: "role-arn"},
	"instance_controlplane": {Name: "ControlPlane", Flag: "master-iam-role"},
	"instance_worker":       {Name: "Worker", Flag: "worker-iam-role"},
	"support":               {Name: "Support", Flag: "support-role-arn"},
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
	Principal PolicyStatementPrincipal `json:"Principal"`
	// Include a list of actions that the policy allows or denies.
	// (i.e. ec2:StartInstances, iam:ChangePassword)
	Action []string `json:"Action"`
	// If you create an IAM permissions policy, you must specify a list of resources to which
	// the actions apply. If you create a resource-based policy, this element is optional. If
	// you do not include this element, then the resource to which the action applies is the
	// resource to which the policy is attached.
	Resource []string `json:"Resource"`
}

type PolicyStatementPrincipal struct {
	// A service principal is an identifier that is used to grant permissions to a service.
	// The identifier for a service principal includes the service name, and is usually in the
	// following format: service-name.amazonaws.com
	Service []string `json:"Service"`
	// You can specify an individual IAM role ARN (or array of role ARNs) as the principal.
	// In IAM roles, the Principal element in the role's trust policy specifies who can assume the role.
	// When you specify more than one principal in the element, you grant permissions to each principal.
	AWS []string `json:"AWS"`
	// A federated principal uses a web identity token or SAML federation
	Federated string `json:"Federated"`
}

func (c *awsClient) EnsureRole(name string, policy string, version string, tagList map[string]string) (string, error) {
	output, err := c.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return c.createRole(name, policy, tagList)
			default:
				return "", err
			}
		}
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
		// There is no AWS principal to add, nothing to do here
		if len(statement.Principal.AWS) == 0 {
			return policy, false, nil
		}
		for _, trust := range statement.Principal.AWS {
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

func (c *awsClient) createRole(name string, policy string, tagList map[string]string) (string, error) {
	if !RoleNameRE.MatchString(name) {
		return "", fmt.Errorf("Role name is invalid")
	}
	output, err := c.iamClient.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(name),
		AssumeRolePolicyDocument: aws.String(policy),
		Tags:                     getTags(tagList),
	})
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
