/*
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

package iamserviceaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

// CreateIAMServiceAccountRequest contains the resolved inputs for creating
// an IAM role bound to one or more Kubernetes service accounts via OIDC.
//
// The CLI layer constructs this after resolving flags, interactive prompts,
// and AWS lookups (OIDC provider ARN, creator partition).
type CreateIAMServiceAccountRequest struct {
	ClusterName     string
	OIDCProviderARN string
	ServiceAccounts []ServiceAccountIdentifier

	// RoleName is optional. When nil and a single service account is
	// provided, the workflow generates a name from the cluster, namespace,
	// and service account. When multiple service accounts are provided,
	// RoleName is required.
	RoleName *string

	PolicyARNs          []string
	InlinePolicy        *string // resolved policy document, not a file:// reference
	PermissionsBoundary *string
	Path                *string

	// AWS account context, resolved by the CLI layer via GetCreator.
	AccountID  string
	Partition  string
	IsGovcloud bool
}

// Validate checks that the request contains all required fields and that
// service account names and namespaces are syntactically valid.
func (r *CreateIAMServiceAccountRequest) Validate() error {
	if r.ClusterName == "" {
		return fmt.Errorf("cluster name is required")
	}
	if r.OIDCProviderARN == "" {
		return fmt.Errorf("OIDC provider ARN is required")
	}
	if len(r.ServiceAccounts) == 0 {
		return fmt.Errorf("at least one service account is required")
	}
	for _, sa := range r.ServiceAccounts {
		if err := ValidateServiceAccountName(sa.Name); err != nil {
			return fmt.Errorf("invalid service account name %q: %w", sa.Name, err)
		}
		if err := ValidateNamespaceName(sa.Namespace); err != nil {
			return fmt.Errorf("invalid namespace %q for service account %q: %w", sa.Namespace, sa.Name, err)
		}
	}
	if len(r.PolicyARNs) == 0 && r.InlinePolicy == nil {
		return fmt.Errorf("at least one policy ARN or inline policy is required")
	}
	if r.InlinePolicy != nil && *r.InlinePolicy == "" {
		return fmt.Errorf("inline policy must not be empty when provided")
	}
	if r.InlinePolicy != nil && !json.Valid([]byte(*r.InlinePolicy)) {
		return fmt.Errorf("inline policy must be valid JSON")
	}
	for i, policyARN := range r.PolicyARNs {
		if policyARN == "" {
			return fmt.Errorf("policy ARN at index %d is empty", i)
		}
		if _, err := arn.Parse(policyARN); err != nil {
			return fmt.Errorf("policy ARN at index %d is invalid: %w", i, err)
		}
	}
	if r.RoleName != nil && strings.TrimSpace(*r.RoleName) == "" {
		return fmt.Errorf("role name must not be blank when provided")
	}
	if r.RoleName == nil && len(r.ServiceAccounts) > 1 {
		return fmt.Errorf("role name is required when specifying multiple service accounts")
	}
	if r.IsGovcloud {
		if r.AccountID == "" {
			return fmt.Errorf("account ID is required for GovCloud environments")
		}
		if r.Partition == "" {
			return fmt.Errorf("partition is required for GovCloud environments")
		}
	}
	return nil
}

// CreateIAMServiceAccountResult contains the structured outcome of creating
// an IAM role for Kubernetes service account OIDC federation.
//
// The CLI layer uses this to format output for the user. The result carries
// domain values only, not preformatted console messages.
type CreateIAMServiceAccountResult struct {
	RoleName string
	RoleARN  string

	// AttachedPolicyARNs lists the managed policies that were attached to the role.
	AttachedPolicyARNs []string

	// InlinePolicyName is the name of the inline policy that was attached to
	// the role. Empty when no inline policy was requested.
	InlinePolicyName string
}

// Service is the default implementation of IAMServiceAccountService.
type Service struct{}

// CreateIAMServiceAccount validates the request, creates an IAM role bound
// to one or more Kubernetes service accounts via OIDC federation, attaches
// the requested policies, and returns a structured result.
func (s *Service) CreateIAMServiceAccount(
	ctx context.Context,
	client CreateIAMServiceAccountClient,
	req CreateIAMServiceAccountRequest,
) (*CreateIAMServiceAccountResult, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Generate role name if not provided
	var roleName string
	if req.RoleName != nil {
		roleName = *req.RoleName
	} else {
		roleName = GenerateRoleName(
			req.ClusterName,
			req.ServiceAccounts[0].Namespace,
			req.ServiceAccounts[0].Name,
		)
	}

	// Generate the OIDC trust policy for the service accounts
	trustPolicy := GenerateTrustPolicyMultiple(req.OIDCProviderARN, req.ServiceAccounts)

	// Generate default tags for the role
	tags := GenerateDefaultTags(
		req.ClusterName,
		req.ServiceAccounts[0].Namespace,
		req.ServiceAccounts[0].Name,
	)

	// Resolve optional string fields for the AWS call
	permissionsBoundary := ""
	if req.PermissionsBoundary != nil {
		permissionsBoundary = *req.PermissionsBoundary
	}
	path := ""
	if req.Path != nil {
		path = *req.Path
	}

	// Create or update the IAM role
	managedPolicies := false
	roleARN, err := client.EnsureRole(
		ctx,
		roleName,
		trustPolicy,
		permissionsBoundary,
		"", // version is unused for service account roles
		tags,
		path,
		managedPolicies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	// For GovCloud/FedRAMP environments, the ARN returned by EnsureRole may
	// use the wrong partition. Reconstruct it using the caller's partition.
	if req.IsGovcloud {
		roleARN = GetRoleARN(req.AccountID, roleName, path, req.Partition)
	}

	// Attach managed policies
	for _, policyARN := range req.PolicyARNs {
		if err := client.AttachRolePolicy(ctx, roleName, policyARN); err != nil {
			return nil, fmt.Errorf("failed to attach policy '%s' to role '%s': %w", policyARN, roleName, err)
		}
	}

	// Attach inline policy if provided
	var inlinePolicyName string
	if req.InlinePolicy != nil {
		inlinePolicyName = fmt.Sprintf("%s-inline-policy", roleName)
		if err := client.PutRolePolicy(ctx, roleName, inlinePolicyName, *req.InlinePolicy); err != nil {
			return nil, fmt.Errorf("failed to attach inline policy to role '%s': %w", roleName, err)
		}
	}

	return &CreateIAMServiceAccountResult{
		RoleName:           roleName,
		RoleARN:            roleARN,
		AttachedPolicyARNs: req.PolicyARNs,
		InlinePolicyName:   inlinePolicyName,
	}, nil
}
