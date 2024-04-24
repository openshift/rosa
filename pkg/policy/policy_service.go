/*
Copyright (c) 2024 Red Hat, Inc.

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

package policy

import (
	"fmt"

	"slices"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/ocm"
)

const (
	// quota code for managed policies per role
	QuotaCode = "L-0DA4ABF3"
)

type PolicyService interface {
	ValidateAttachOptions(roleName string, policyArns []string) error
	AutoAttachArbitraryPolicy(roleName string, policyArns []string, accountID, orgID string) (string, error)
	ManualAttachArbitraryPolicy(roleName string, policyArns []string, accountID, orgID string) string
}

type policyService struct {
	OCMClient *ocm.Client
	AWSClient aws.Client
}

func NewPolicyService(OCMClient *ocm.Client, AWSClient aws.Client) PolicyService {
	return &policyService{
		OCMClient: OCMClient,
		AWSClient: AWSClient,
	}
}

// ValidateAttachOptions validate the options:
// verify rolename and each policy arn are valid
// verify role is RedHat managed
// verify role has quota to attach the policies
func (p *policyService) ValidateAttachOptions(roleName string, policyArns []string) error {
	role, err := p.AWSClient.GetRoleByName(roleName)
	if err != nil {
		return fmt.Errorf(
			"Failed to find the role %s: %s",
			roleName, err,
		)
	}
	isRedHatManaged := false
	for _, tag := range role.Tags {
		if awsutil.ToString(tag.Key) == tags.RedHatManaged &&
			awsutil.ToString(tag.Value) == "true" {
			isRedHatManaged = true
			break
		}
	}
	if !isRedHatManaged {
		return fmt.Errorf("Cannot attach policies to non-ROSA roles")
	}

	err = validatePolicyQuota(p.AWSClient, roleName, policyArns)
	if err != nil {
		return err
	}
	for _, policyArn := range policyArns {
		_, err = p.AWSClient.IsPolicyExists(policyArn)
		if err != nil {
			return fmt.Errorf(
				"Failed to find the policy %s: %s",
				policyArn, err,
			)
		}
	}
	return nil
}

func (p *policyService) AutoAttachArbitraryPolicy(roleName string, policyArns []string,
	accountID, orgID string) (string, error) {
	output := ""
	for _, policyArn := range policyArns {
		err := p.AWSClient.AttachRolePolicy(roleName, policyArn)
		if err != nil {
			return output, fmt.Errorf("Failed to attach policy %s to role %s: %s",
				policyArn, roleName, err)
		}
		output = output + fmt.Sprintf("Attached policy '%s' to role '%s'\n", policyArn, roleName)
		p.OCMClient.LogEvent("ROSAAttachPolicyAuto", map[string]string{
			ocm.Account:      accountID,
			ocm.Organization: orgID,
			ocm.RoleName:     roleName,
			ocm.PolicyArn:    policyArn,
		})
	}
	return output, nil
}

func (p *policyService) ManualAttachArbitraryPolicy(roleName string, policyArns []string,
	accountID, orgID string) string {
	cmd := ""
	for _, policyArn := range policyArns {
		cmd = cmd + fmt.Sprintf("aws iam attach-role-policy --role-name %s --policy-arn %s\n",
			roleName, policyArn)
		p.OCMClient.LogEvent("ROSAAttachPolicyManual", map[string]string{
			ocm.Account:      accountID,
			ocm.Organization: orgID,
			ocm.RoleName:     roleName,
			ocm.PolicyArn:    policyArn,
		})
	}
	return cmd
}

func validatePolicyQuota(c aws.Client, roleName string, policyArns []string) error {
	quota, err := c.GetIAMServiceQuota(QuotaCode)
	if err != nil {
		return fmt.Errorf("Failed to get quota of policies per role: %s", err)
	}
	attachedPolicies, err := c.GetAttachedPolicy(&roleName)
	if err != nil {
		return fmt.Errorf("Failed to get attached policies of role %s: %s", roleName, err)
	}
	if len(policyArns)+len(attachedPolicies) > int(*quota.Quota.Value) {
		policySkipped := 0
		for _, attachedPolicy := range attachedPolicies {
			if slices.Contains(policyArns, attachedPolicy.PolicyArn) {
				policySkipped++
			}
		}
		if len(policyArns)+len(attachedPolicies)-policySkipped > int(*quota.Quota.Value) {
			return fmt.Errorf("Failed to attach policies due to quota limitations"+
				" (total limit: %d, expected: %d)",
				int(*quota.Quota.Value), len(policyArns)+len(attachedPolicies)-policySkipped)
		}
	}
	return nil
}
