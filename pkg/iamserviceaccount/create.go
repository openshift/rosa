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
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"

	rosaerrors "github.com/openshift/rosa/pkg/errors"
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

// createValidators enforces each domain invariant for
// CreateIAMServiceAccountRequest independently. Keeping each check as a
// small, separately testable function makes it easy to see which invariants
// exist and to add or remove one without touching the others. Each
// validator returns its own violations directly (nil if none), rather than
// pre-joining them, so Validate() can build one flat list and call
// errors.Join exactly once instead of nesting a join inside a join.
var createValidators = []func(*CreateIAMServiceAccountRequest) []error{
	validateClusterName,
	validateOIDCProviderARN,
	validateServiceAccountsPresent,
	validateServiceAccountIdentifiers,
	validatePolicies,
	validateRoleName,
	validateGovcloud,
}

// Validate checks that the request contains all required fields and that
// service account names and namespaces are syntactically valid. It runs
// every validator in createValidators and joins their errors, so a caller
// sees every violation at once instead of fixing them one at a time.
func (r *CreateIAMServiceAccountRequest) Validate() error {
	var errs []error
	for _, validate := range createValidators {
		errs = append(errs, validate(r)...)
	}
	return errors.Join(errs...)
}

func validateClusterName(r *CreateIAMServiceAccountRequest) []error {
	if r.ClusterName == "" {
		return []error{&rosaerrors.ValidationError{Field: "ClusterName", Message: "cluster name is required"}}
	}
	return nil
}

func validateOIDCProviderARN(r *CreateIAMServiceAccountRequest) []error {
	if r.OIDCProviderARN == "" {
		return []error{&rosaerrors.ValidationError{Field: "OIDCProviderARN", Message: "OIDC provider ARN is required"}}
	}
	return nil
}

func validateServiceAccountsPresent(r *CreateIAMServiceAccountRequest) []error {
	if len(r.ServiceAccounts) == 0 {
		return []error{
			&rosaerrors.ValidationError{Field: "ServiceAccounts", Message: "at least one service account is required"},
		}
	}
	return nil
}

func validateServiceAccountIdentifiers(r *CreateIAMServiceAccountRequest) []error {
	var errs []error
	for _, sa := range r.ServiceAccounts {
		if err := ValidateServiceAccountName(sa.Name); err != nil {
			errs = append(errs, &rosaerrors.ValidationError{
				Field:   "ServiceAccounts",
				Message: fmt.Sprintf("invalid service account name %q", sa.Name),
				Err:     err,
			})
		}
		if err := ValidateNamespaceName(sa.Namespace); err != nil {
			errs = append(errs, &rosaerrors.ValidationError{
				Field:   "ServiceAccounts",
				Message: fmt.Sprintf("invalid namespace %q for service account %q", sa.Namespace, sa.Name),
				Err:     err,
			})
		}
	}
	return errs
}

func validatePolicies(r *CreateIAMServiceAccountRequest) []error {
	var errs []error
	if len(r.PolicyARNs) == 0 && r.InlinePolicy == nil {
		errs = append(errs, &rosaerrors.ValidationError{
			Message: "at least one policy ARN or inline policy is required",
		})
	}
	if r.InlinePolicy != nil {
		if *r.InlinePolicy == "" {
			errs = append(errs, &rosaerrors.ValidationError{
				Field: "InlinePolicy", Message: "inline policy must not be empty when provided",
			})
		} else if !json.Valid([]byte(*r.InlinePolicy)) {
			errs = append(errs, &rosaerrors.ValidationError{
				Field: "InlinePolicy", Message: "inline policy must be valid JSON",
			})
		}
	}
	for i, policyARN := range r.PolicyARNs {
		if policyARN == "" {
			errs = append(errs, &rosaerrors.ValidationError{
				Field:   "PolicyARNs",
				Message: fmt.Sprintf("policy ARN at index %d is empty", i),
			})
			continue
		}
		if _, err := arn.Parse(policyARN); err != nil {
			errs = append(errs, &rosaerrors.ValidationError{
				Field:   "PolicyARNs",
				Message: fmt.Sprintf("policy ARN at index %d is invalid", i),
				Err:     err,
			})
		}
	}
	return errs
}

func validateRoleName(r *CreateIAMServiceAccountRequest) []error {
	var errs []error
	if r.RoleName != nil && strings.TrimSpace(*r.RoleName) == "" {
		errs = append(errs, &rosaerrors.ValidationError{
			Field: "RoleName", Message: "role name must not be blank when provided",
		})
	}
	if r.RoleName == nil && len(r.ServiceAccounts) > 1 {
		errs = append(errs, &rosaerrors.ValidationError{
			Field: "RoleName", Message: "role name is required when specifying multiple service accounts",
		})
	}
	return errs
}

func validateGovcloud(r *CreateIAMServiceAccountRequest) []error {
	if !r.IsGovcloud {
		return nil
	}
	var errs []error
	if r.AccountID == "" {
		errs = append(errs, &rosaerrors.ValidationError{
			Field: "AccountID", Message: "account ID is required for GovCloud environments",
		})
	}
	if r.Partition == "" {
		errs = append(errs, &rosaerrors.ValidationError{
			Field: "Partition", Message: "partition is required for GovCloud environments",
		})
	}
	return errs
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
		return nil, &rosaerrors.ValidationError{Err: fmt.Errorf("invalid request: %w", err)}
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
