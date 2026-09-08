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

// Package rolebridge adapts pkg/aws.Client's IAM role and policy methods to
// the context.Context-based signatures that core workflow packages declare
// in their own <Verb><Resource>Client interfaces (see "Client Interfaces
// Are Scoped Per Workflow, Not Per Package" in
// guidelines/workflow-conventions.md).
//
// aws.Client's EnsureRole and AttachRolePolicy take a reporter.Logger
// instead of a context.Context, and PutRolePolicy takes neither, because
// aws.Client predates the Request/Result/Service workflow pattern. This
// package exists only to bridge that mismatch, in one place, for every
// workflow that needs these operations, instead of each workflow's CLI
// command hand-rolling its own copy of the same translation. Go's
// structural typing lets this single concrete type satisfy any number of
// distinct workflow client interfaces without them referencing each other.
//
// Delete this package once EnsureRole and AttachRolePolicy stop taking a
// reporter.Logger (see the "Needs split" entry for pkg/aws in
// guidelines/refactor/pkg-architecture.md) -- at that point workflows can
// depend on aws.Client directly.
//
// The context.Context each method accepts is currently discarded: none of
// the underlying aws.Client methods take one, so cancellation and deadlines
// set by a caller do not propagate to the AWS calls this package makes.
package rolebridge

import (
	"context"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/reporter" //nolint:depguard
)

// Client wraps an aws.Client and a reporter.Logger to present the IAM role
// and policy operations with context.Context-based signatures.
type Client struct {
	client   aws.Client
	reporter reporter.Logger
}

// New returns a Client that forwards to client, using reporter for the
// aws.Client methods that require one.
func New(client aws.Client, reporter reporter.Logger) *Client {
	return &Client{client: client, reporter: reporter}
}

// EnsureRole creates or updates an IAM role with the given trust policy and tags.
func (c *Client) EnsureRole(_ context.Context, name, policy, permissionsBoundary,
	version string, tagList map[string]string, path string, managedPolicies bool) (string, error) {
	return c.client.EnsureRole(c.reporter, name, policy, permissionsBoundary, version, tagList, path, managedPolicies)
}

// AttachRolePolicy attaches a managed policy to a role.
func (c *Client) AttachRolePolicy(_ context.Context, roleName, policyARN string) error {
	return c.client.AttachRolePolicy(c.reporter, roleName, policyARN)
}

// PutRolePolicy creates or updates an inline policy on a role.
func (c *Client) PutRolePolicy(_ context.Context, roleName, policyName, policy string) error {
	return c.client.PutRolePolicy(roleName, policyName, policy)
}
