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

// IAMServiceAccountClient defines the IAM operations needed by service
// account workflows.
//
// This is a narrow interface over the subset of AWS IAM operations that
// the workflows use. The CLI layer satisfies it by adapting aws.Client
// (which passes reporter.Logger through to EnsureRole and AttachRolePolicy)
// so that the core workflows stay free of reporter and CLI dependencies.
type IAMServiceAccountClient interface {
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
	CreateIAMServiceAccount(ctx context.Context, client IAMServiceAccountClient,
		req *CreateIAMServiceAccountRequest) (*CreateIAMServiceAccountResult, error)
}
