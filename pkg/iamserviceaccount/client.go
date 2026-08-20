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
)

// CreateIAMServiceAccountClient defines the IAM operations needed by the
// CreateIAMServiceAccount workflow.
//
// This is a narrow interface over the subset of AWS IAM operations that
// workflow uses. It is not shared with any other workflow; a future list,
// describe, or delete workflow gets its own client interface scoped to what
// it calls, not a method added here. See "Client Interfaces Are Scoped Per
// Workflow, Not Per Package" in guidelines/workflow-conventions.md.
//
// TODO: It is not yet implemented or wired up anywhere; the CLI command in
// cmd/create/iamserviceaccount still calls aws.Client directly. When that
// command is migrated to call IAMServiceAccountService, the CLI layer will
// provide an adapter around aws.Client (which passes reporter.Logger
// through to EnsureRole and AttachRolePolicy) so that the core workflows
// stay free of reporter and CLI dependencies.
type CreateIAMServiceAccountClient interface {
	// EnsureRole creates or updates an IAM role with the given trust policy and tags.
	EnsureRole(ctx context.Context, name string, policy string, permissionsBoundary string,
		version string, tagList map[string]string, path string, managedPolicies bool) (string, error)

	// AttachRolePolicy attaches a managed policy to a role.
	AttachRolePolicy(ctx context.Context, roleName string, policyARN string) error

	// PutRolePolicy creates or updates an inline policy on a role.
	PutRolePolicy(ctx context.Context, roleName string, policyName string, policy string) error
}

// IAMServiceAccountService defines the workflow operations for managing
// IAM roles bound to Kubernetes service accounts. The CLI layer calls
// these methods after constructing a resolved Request.
type IAMServiceAccountService interface {
	CreateIAMServiceAccount(ctx context.Context, client CreateIAMServiceAccountClient,
		req CreateIAMServiceAccountRequest) (*CreateIAMServiceAccountResult, error)
}
