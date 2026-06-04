/*
Copyright (c) 2026 Red Hat, Inc.

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

package ocmrole

import (
	"errors"
	"fmt"
	"slices"

	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
)

// ErrRoleExistsWrongProfile is returned when a role exists but with a different profile than requested
var ErrRoleExistsWrongProfile = errors.New("role exists with different profile")

// RoleProfile defines the type of OCM role to create
type RoleProfile string

const (
	ProfileStandard  RoleProfile = "standard"
	ProfileAdmin     RoleProfile = "admin"
	ProfileNoConsole RoleProfile = "no-console"
)

func DetermineProfile(isAdmin, isNoConsole bool) RoleProfile {
	switch {
	case isAdmin:
		return ProfileAdmin
	case isNoConsole:
		return ProfileNoConsole
	default:
		return ProfileStandard
	}
}

// CheckRoleExistsInternal is a prompt-free version of checkRoleExists that returns typed errors
// instead of prompting the user. This function is used by both the CLI (via checkRoleExists wrapper)
// and by CreateOCMRole for programmatic use.
//
// Returns:
//   - roleARN: The ARN of the existing role (if exists=true)
//   - exists: Whether the role exists
//   - error: Any error encountered, including ErrRoleExistsWrongProfile when role exists but profile doesn't match
func CheckRoleExistsInternal(r *rosa.Runtime, roleName string, profile RoleProfile,
	mode string, rolePath string,
) (string, bool, error) {
	exists, roleARN, err := r.AWSClient.CheckRoleExists(roleName)
	if err != nil {
		return "", false, err
	}
	if exists {
		isExistingRoleAdmin, err := r.AWSClient.IsAdminRole(roleName)
		if err != nil {
			return "", true, err
		}
		isExistingRoleNoConsole, err := r.AWSClient.IsNoConsoleRole(roleName)
		if err != nil {
			return "", true, err
		}

		r.Reporter.Warnf("Role '%s' already exists", roleName)

		switch profile {
		case ProfileStandard:
			if isExistingRoleAdmin {
				return roleARN, true, fmt.Errorf("the existing role is an admin role."+
					" To remove admin capabilities please delete the admin policy and the '%s' tag",
					tags.AdminRole)
			}
			if isExistingRoleNoConsole {
				return roleARN, true, fmt.Errorf("the existing role is a no-console role." +
					" To use standard permissions please delete the role and recreate it")
			}
			return roleARN, true, nil

		case ProfileAdmin:
			if isExistingRoleNoConsole {
				return roleARN, true, fmt.Errorf("the existing role is a no-console role." +
					" To use admin permissions please delete the role and recreate it")
			}

			if isExistingRoleAdmin {
				return roleARN, true, nil
			}

			// Role appears to be standard - check if admin policy is actually attached (self-healing)
			attachedPolicies, err := r.AWSClient.ListAttachedRolePolicies(roleName)
			if err != nil {
				return "", true, err
			}

			// Check if admin policy is attached (exact ARN match)
			expectedAdminPolicyARN := aws.GetAdminPolicyARN(r.Creator.Partition, r.Creator.AccountID, roleName, rolePath)
			if slices.Contains(attachedPolicies, expectedAdminPolicyARN) {
				// Self-heal: admin policy exists but tag is missing
				r.Reporter.Debugf("Admin policy found but tag missing - adding tag")
				err = r.AWSClient.AddRoleTag(roleName, tags.AdminRole, "true")
				if err != nil {
					return "", true, fmt.Errorf("failed to add admin role tag: %w", err)
				}
				return roleARN, true, nil
			}

			if mode == interactive.ModeAuto {
				return roleARN, true, fmt.Errorf("%w: role is standard, requested admin", ErrRoleExistsWrongProfile)
			}

		case ProfileNoConsole:
			if isExistingRoleAdmin {
				return roleARN, true, fmt.Errorf("the existing role is an admin role." +
					" To use no-console permissions please delete the role and recreate it")
			}

			if isExistingRoleNoConsole {
				// Verify no-console policy is actually attached
				attachedPolicies, err := r.AWSClient.ListAttachedRolePolicies(roleName)
				if err != nil {
					return "", true, err
				}

				expectedNoConsolePolicyARN := aws.GetNoConsolePolicyARN(
					r.Creator.Partition, r.Creator.AccountID, roleName, rolePath)
				if !slices.Contains(attachedPolicies, expectedNoConsolePolicyARN) {
					// Tag exists but policy is missing - incomplete manual run
					return "", true, fmt.Errorf("the role has the no-console tag but the no-console policy is not attached." +
						" Please attach the policy or remove the tag and recreate the role")
				}

				return roleARN, true, nil
			}

			// Role appears to be standard - check if no-console policy is actually attached (self-healing)
			attachedPolicies, err := r.AWSClient.ListAttachedRolePolicies(roleName)
			if err != nil {
				return "", true, err
			}

			// Check if no-console policy is attached (exact ARN match)
			expectedNoConsolePolicyARN := aws.GetNoConsolePolicyARN(r.Creator.Partition, r.Creator.AccountID, roleName, rolePath)
			if slices.Contains(attachedPolicies, expectedNoConsolePolicyARN) {
				// Self-heal: no-console policy exists but tag is missing
				r.Reporter.Debugf("No-console policy found but tag missing - adding tag")
				err = r.AWSClient.AddRoleTag(roleName, tags.NoConsoleRole, "true")
				if err != nil {
					return "", true, fmt.Errorf("failed to add no-console role tag: %w", err)
				}
				return roleARN, true, nil
			}

			// Existing is standard, cannot convert
			return roleARN, true, fmt.Errorf("the existing role is a standard role." +
				" To use no-console permissions please delete the role and recreate it")

		default:
			// Should never reach here if validation is done at boundaries
			return "", false, fmt.Errorf("invalid profile: %s (must be one of: %s, %s, %s)",
				profile, ProfileStandard, ProfileAdmin, ProfileNoConsole)
		}
	}

	return "", false, nil
}

// CreateRolesInternal is a prompt-free version of createRoles that performs the actual IAM role creation.
// This function assumes the existence check has already been done by the caller and the role doesn't exist.
// It creates the role, attaches policies, and adds appropriate tags based on the profile.
//
// This function is used by both the CLI (via createRoles wrapper) and by CreateOCMRole for programmatic use.
func CreateRolesInternal(r *rosa.Runtime, prefix string, roleName string, rolePath string,
	permissionsBoundary string, orgID string, env string, profile RoleProfile,
	policies map[string]*cmv1.AWSSTSPolicy, managedPolicies bool,
) (string, error) {
	var policyARN string
	var err error

	if profile != ProfileNoConsole {
		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies, fmt.Sprintf("sts_%s_permission_policy", aws.OCMRolePolicyFile))
			if err != nil {
				return "", err
			}
		} else {
			policyARN = aws.GetPolicyArnWithSuffix(r.Creator.Partition, r.Creator.AccountID, roleName, rolePath)
		}
	}

	// Build trust policy
	filename := fmt.Sprintf("sts_%s_trust_policy", aws.OCMRolePolicyFile)
	policyDetail := aws.GetPolicyDetails(policies, filename)
	policy := aws.InterpolatePolicyDocument(r.Creator.Partition, policyDetail, map[string]string{
		"partition":           r.Creator.Partition,
		"aws_account_id":      aws.GetJumpAccount(env),
		"ocm_organization_id": orgID,
	})

	// Build IAM tags
	iamTags := map[string]string{
		tags.RolePrefix:    prefix,
		tags.RoleType:      aws.OCMRole,
		tags.Environment:   env,
		tags.RedHatManaged: tags.True,
	}
	if managedPolicies {
		iamTags[common.ManagedPolicies] = tags.True
	}

	// Verify profile is valid before creating any AWS resources
	if profile != ProfileStandard && profile != ProfileAdmin && profile != ProfileNoConsole {
		return "", fmt.Errorf("invalid profile: %s (must be one of: %s, %s, %s)",
			profile, ProfileStandard, ProfileAdmin, ProfileNoConsole)
	}

	r.Reporter.Debugf("Creating role '%s'", roleName)

	roleARN, err := r.AWSClient.EnsureRole(r.Reporter, roleName, policy, permissionsBoundary,
		"", iamTags, rolePath, false)
	if err != nil {
		return "", err
	}
	r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

	switch profile {
	case ProfileStandard:
		filename = fmt.Sprintf("sts_%s_permission_policy", aws.OCMRolePolicyFile)
		policyDetail = aws.GetPolicyDetails(policies, filename)
		err = CreatePermissionPolicy(r, policyARN, iamTags, roleName, rolePath, policyDetail, managedPolicies)
		if err != nil {
			return "", err
		}

	case ProfileAdmin:
		// standard policy first
		filename = fmt.Sprintf("sts_%s_permission_policy", aws.OCMRolePolicyFile)
		policyDetail = aws.GetPolicyDetails(policies, filename)
		err = CreatePermissionPolicy(r, policyARN, iamTags, roleName, rolePath, policyDetail, managedPolicies)
		if err != nil {
			return "", err
		}

		// create and attach the admin policy to the role
		filename = fmt.Sprintf("sts_%s_permission_policy", aws.OCMAdminRolePolicyFile)
		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies, filename)
			if err != nil {
				return "", err
			}
		} else {
			policyARN = aws.GetAdminPolicyARN(r.Creator.Partition, r.Creator.AccountID, roleName, rolePath)
		}
		iamTags[tags.AdminRole] = tags.True
		policyDetail = aws.GetPolicyDetails(policies, filename)
		err = CreatePermissionPolicy(r, policyARN, iamTags, roleName, rolePath, policyDetail, managedPolicies)
		if err != nil {
			return "", err
		}

		// tag role with admin tag
		err = r.AWSClient.AddRoleTag(roleName, tags.AdminRole, tags.True)
		if err != nil {
			return "", err
		}

	case ProfileNoConsole:
		filename = fmt.Sprintf("sts_%s_permission_policy", aws.OCMNoConsoleRolePolicyFile)

		// create and attach the no-console policy to the role
		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies, filename)
			if err != nil {
				return "", err
			}
		} else {
			policyARN = aws.GetNoConsolePolicyARN(r.Creator.Partition, r.Creator.AccountID, roleName, rolePath)
		}
		iamTags[tags.NoConsoleRole] = tags.True
		policyDetail = aws.GetPolicyDetails(policies, filename)
		err = CreatePermissionPolicy(r, policyARN, iamTags, roleName, rolePath, policyDetail, managedPolicies)
		if err != nil {
			return "", err
		}

		// tag role with no-console tag
		err = r.AWSClient.AddRoleTag(roleName, tags.NoConsoleRole, tags.True)
		if err != nil {
			return "", err
		}

	default:
		// Should never reach here if validation is done at boundaries
		return "", fmt.Errorf("invalid profile: %s (must be one of: %s, %s, %s)",
			profile, ProfileStandard, ProfileAdmin, ProfileNoConsole)
	}

	return roleARN, nil
}

// CreatePermissionPolicy creates and attaches a permission policy to an IAM role
func CreatePermissionPolicy(r *rosa.Runtime, policyARN string,
	iamTags map[string]string, roleName string, rolePath string, policyDetail string, managedPolicies bool,
) error {
	r.Reporter.Debugf("Creating permission policy '%s'", policyARN)
	if !managedPolicies {
		var err error
		policyARN, err = r.AWSClient.EnsurePolicy(policyARN, policyDetail, "", iamTags, rolePath)
		if err != nil {
			return err
		}
	}

	r.Reporter.Debugf("Attaching permission policy to role '%s'", roleName)
	err := r.AWSClient.AttachRolePolicy(r.Reporter, roleName, policyARN)
	if err != nil {
		return err
	}

	return nil
}
