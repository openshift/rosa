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
	awserr "github.com/openshift-online/ocm-common/pkg/aws/errors"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
)

const (
	// quota code for managed policies per role
	QuotaCode = "L-0DA4ABF3"
)

type PolicyService interface {
	ValidateAttachOptions(roleName string, policyArns []string) error
	AutoAttachArbitraryPolicy(reporter reporter.Logger, roleName string,
		policyArns []string, accountID, orgID string) error
	ManualAttachArbitraryPolicy(roleName string, policyArns []string, accountID, orgID string) string
	ValidateDetachOptions(roleName string, policyArns []string) error
	AutoDetachArbitraryPolicy(roleName string, policyArns []string, accountID, orgID string) (string, error)
	ManualDetachArbitraryPolicy(roleName string, policyArns []string, accountID, orgID string) (string, string, error)
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
	err := validateRoleAndPolicies(p.AWSClient, roleName, policyArns)
	if err != nil {
		return err
	}
	err = validatePolicyQuota(p.AWSClient, roleName, policyArns)
	if err != nil {
		return err
	}
	return nil
}

func (p *policyService) ValidateDetachOptions(roleName string, policyArns []string) error {
	return validateRoleAndPolicies(p.AWSClient, roleName, policyArns)
}

func (p *policyService) AutoAttachArbitraryPolicy(reporter reporter.Logger, roleName string, policyArns []string,
	accountID, orgID string) error {
	for _, policyArn := range policyArns {
		err := p.AWSClient.AttachRolePolicy(reporter, roleName, policyArn)
		if err != nil {
			return fmt.Errorf("Failed to attach policy %s to role %s: %s",
				policyArn, roleName, err)
		}
		p.OCMClient.LogEvent("ROSAAttachPolicyAuto", map[string]string{
			ocm.Account:      accountID,
			ocm.Organization: orgID,
			ocm.RoleName:     roleName,
			ocm.PolicyArn:    policyArn,
		})
	}
	return nil
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

func (p *policyService) AutoDetachArbitraryPolicy(roleName string, policyArns []string,
	accountID, orgID string) (string, error) {
	output := ""
	for _, policyArn := range policyArns {
		err := p.AWSClient.DetachRolePolicy(policyArn, roleName)
		if err != nil {
			if awserr.IsNoSuchEntityException(err) {
				output = output + fmt.Sprintf("The policy '%s' is currently not attached to role '%s'\n",
					policyArn, roleName)
			} else {
				return output, fmt.Errorf("Failed to detach policy '%s' from role '%s': %s",
					policyArn, roleName, err)
			}
		} else {
			output = output + fmt.Sprintf("Detached policy '%s' from role '%s'\n", policyArn, roleName)
			p.OCMClient.LogEvent("ROSADetachPolicyAuto", map[string]string{
				ocm.Account:      accountID,
				ocm.Organization: orgID,
				ocm.RoleName:     roleName,
				ocm.PolicyArn:    policyArn,
			})
		}
	}
	return output[:len(output)-1], nil
}

func (p *policyService) ManualDetachArbitraryPolicy(roleName string, policyArns []string,
	accountID, orgID string) (string, string, error) {
	cmd := ""
	warn := ""
	policies, err := p.AWSClient.ListAttachedRolePolicies(roleName)
	if err != nil {
		return cmd, warn, err
	}
	for _, policyArn := range policyArns {
		if slices.Contains(policies, policyArn) {
			cmd = cmd + fmt.Sprintf("aws iam detach-role-policy --role-name %s --policy-arn %s\n",
				roleName, policyArn)
			p.OCMClient.LogEvent("ROSADetachPolicyManual", map[string]string{
				ocm.Account:      accountID,
				ocm.Organization: orgID,
				ocm.RoleName:     roleName,
				ocm.PolicyArn:    policyArn,
			})
		} else {
			warn = warn + fmt.Sprintf("The policy '%s' is currently not attached to role '%s'\n",
				policyArn, roleName)
		}
	}
	return cmd, warn, nil
}

func validatePolicyQuota(c aws.Client, roleName string, policyArns []string) error {
	quota, err := c.GetIAMServiceQuota(QuotaCode)
	if err != nil {
		return fmt.Errorf("Failed to get quota of policies per role: %s", err)
	}
	attachedPolicies, err := c.GetAttachedPolicy(&roleName)
	if err != nil {
		return fmt.Errorf("Failed to get attached policies of role '%s': %s", roleName, err)
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

func validateRoleAndPolicies(c aws.Client, roleName string, policyArns []string) error {
	if !aws.RoleNameRE.MatchString(roleName) {
		return fmt.Errorf("Invalid role name '%s', expected a valid role name matching %s",
			roleName, aws.RoleNameRE.String())
	}
	role, err := c.GetRoleByName(roleName)
	if err != nil {
		return fmt.Errorf(
			"Failed to find the role '%s': %s",
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
		return fmt.Errorf("Cannot attach/detach policies to non-ROSA roles")
	}

	for _, policyArn := range policyArns {
		if !aws.PolicyArnRE.MatchString(policyArn) {
			return fmt.Errorf("Invalid policy arn '%s', expected a valid policy arn matching %s",
				policyArn, aws.PolicyArnRE.String())
		}
		_, err = c.IsPolicyExists(policyArn)
		if err != nil {
			return fmt.Errorf(
				"Failed to find the policy '%s': %s",
				policyArn, err,
			)
		}
	}
	return nil
}
