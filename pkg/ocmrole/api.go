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
	"fmt"

	internalocmrole "github.com/openshift/rosa/internal/ocmrole"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

type RoleProfile = internalocmrole.RoleProfile

const (
	ProfileStandard  = internalocmrole.ProfileStandard
	ProfileAdmin     = internalocmrole.ProfileAdmin
	ProfileNoConsole = internalocmrole.ProfileNoConsole
)

// GetOrCreateOCMRole gets an existing OCM role or creates it if it doesn't exist (idempotent operation).
//
// Behavior:
//   - If the role exists with the correct profile: returns it immediately (created=false)
//   - If the role exists with wrong profile: returns an error
//   - If the role doesn't exist: creates it with the specified configuration (created=true)
//
// When checking existing roles, this function performs self-healing for policy/tag mismatches
// (e.g., admin policy attached but tag missing).
//
// Profile Mismatch Handling:
// If a role exists but with an incompatible profile (e.g., role is admin but standard was requested),
// this function returns an error describing the mismatch. Upgrading or downgrading role profiles is
// the caller's responsibility.
//
// This function does NOT link the role to OCM organization (caller should use OCMClient.LinkOrgToRole after).
//
// Returns:
//   - roleARN: ARN of the role (whether it existed or was created)
//   - created: true if the role was created by this call, false if it already existed
//   - error: nil on success, or an error describing the issue
func GetOrCreateOCMRole(
	r *rosa.Runtime,
	prefix string,
	profile RoleProfile,
	permissionsBoundary string,
	path string,
	managedPolicies bool,
) (string, bool, error) {
	// Validate runtime
	if r == nil {
		return "", false, fmt.Errorf("runtime cannot be nil")
	}
	if r.AWSClient == nil {
		return "", false, fmt.Errorf("AWS client not initialized in runtime")
	}
	if r.Creator == nil {
		return "", false, fmt.Errorf("creator not initialized in runtime")
	}
	if r.Reporter == nil {
		return "", false, fmt.Errorf("reporter not initialized in runtime")
	}
	if r.OCMClient == nil {
		return "", false, fmt.Errorf("OCM client not initialized in runtime")
	}
	if prefix == "" {
		return "", false, fmt.Errorf("prefix cannot be empty")
	}
	if profile != ProfileStandard && profile != ProfileAdmin && profile != ProfileNoConsole {
		return "", false, fmt.Errorf("profile must be one of: %s, %s, %s", ProfileStandard, ProfileAdmin, ProfileNoConsole)
	}

	if path == "" {
		path = "/"
	}

	// Get current OCM organization
	orgID, externalID, err := r.OCMClient.GetCurrentOrganization()
	if err != nil {
		return "", false, fmt.Errorf("failed to get organization account: %w", err)
	}

	roleName := aws.GetOCMRoleName(prefix, aws.OCMRole, externalID)

	// Check if role already exists
	roleARN, exists, err := internalocmrole.CheckRoleExistsInternal(r, roleName, profile, interactive.ModeAuto, path)
	if err != nil {
		return "", false, err
	}
	if exists {
		return roleARN, false, nil
	}

	// Role doesn't exist - need to create it
	policies, err := r.OCMClient.GetPolicies("OCMRole")
	if err != nil {
		return "", false, fmt.Errorf("failed to get OCM policies: %w", err)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		return "", false, fmt.Errorf("failed to determine OCM environment: %w", err)
	}

	// Validate no-console policy exists before creating role
	// This prevents orphaned IAM roles when the policy is missing or malformed
	if profile == ProfileNoConsole {
		filename := fmt.Sprintf("sts_%s_permission_policy", aws.OCMNoConsoleRolePolicyFile)
		policy, ok := policies[filename]
		// For managed policies, validate ARN exists
		// For customer-managed policies, validate Details exists (ARN is constructed later)
		if !ok || (managedPolicies && policy.ARN() == "") || (!managedPolicies && policy.Details() == "") {
			return "", false, fmt.Errorf("the no-console OCM role profile is not yet enabled for your Organization")
		}
	}

	roleARN, err = internalocmrole.CreateRolesInternal(r, prefix, roleName, path, permissionsBoundary,
		orgID, env, profile, policies, managedPolicies)
	if err != nil {
		return "", false, err
	}

	return roleARN, true, nil
}
