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
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	rosaerrors "github.com/openshift/rosa/pkg/errors"
)

// mockCreateIAMServiceAccountClient is a test double for CreateIAMServiceAccountClient
// that records calls and returns configurable responses.
type mockCreateIAMServiceAccountClient struct {
	ensureRoleFn       func(name, policy, permBoundary, version string, tags map[string]string, path string, managed bool) (string, error)
	attachRolePolicyFn func(roleName, policyARN string) error
	putRolePolicyFn    func(roleName, policyName, policy string) error

	// recorded calls for verification
	ensureRoleCalls       []ensureRoleCall
	attachRolePolicyCalls []attachRolePolicyCall
	putRolePolicyCalls    []putRolePolicyCall
}

type ensureRoleCall struct {
	Context             context.Context
	Name                string
	Policy              string
	PermissionsBoundary string
	Version             string
	Tags                map[string]string
	Path                string
	ManagedPolicies     bool
}

type attachRolePolicyCall struct {
	Context   context.Context
	RoleName  string
	PolicyARN string
}

type putRolePolicyCall struct {
	Context    context.Context
	RoleName   string
	PolicyName string
	Policy     string
}

func (f *mockCreateIAMServiceAccountClient) EnsureRole(ctx context.Context, name, policy, permBoundary, version string,
	tags map[string]string, path string, managed bool) (string, error) {
	f.ensureRoleCalls = append(f.ensureRoleCalls, ensureRoleCall{
		Context: ctx, Name: name, Policy: policy, PermissionsBoundary: permBoundary,
		Version: version, Tags: tags, Path: path, ManagedPolicies: managed,
	})
	if f.ensureRoleFn != nil {
		return f.ensureRoleFn(name, policy, permBoundary, version, tags, path, managed)
	}
	return fmt.Sprintf("arn:aws:iam::123456789012:role/%s", name), nil
}

func (f *mockCreateIAMServiceAccountClient) AttachRolePolicy(ctx context.Context, roleName, policyARN string) error {
	f.attachRolePolicyCalls = append(f.attachRolePolicyCalls, attachRolePolicyCall{
		Context: ctx, RoleName: roleName, PolicyARN: policyARN,
	})
	if f.attachRolePolicyFn != nil {
		return f.attachRolePolicyFn(roleName, policyARN)
	}
	return nil
}

func (f *mockCreateIAMServiceAccountClient) PutRolePolicy(ctx context.Context, roleName, policyName, policy string) error {
	f.putRolePolicyCalls = append(f.putRolePolicyCalls, putRolePolicyCall{
		Context: ctx, RoleName: roleName, PolicyName: policyName, Policy: policy,
	})
	if f.putRolePolicyFn != nil {
		return f.putRolePolicyFn(roleName, policyName, policy)
	}
	return nil
}

func expectNoClientCalls(client *mockCreateIAMServiceAccountClient) {
	Expect(client.ensureRoleCalls).To(BeEmpty())
	Expect(client.attachRolePolicyCalls).To(BeEmpty())
	Expect(client.putRolePolicyCalls).To(BeEmpty())
}

// flattenLeafErrors returns the individual violations inside err, peeling
// through the single-cause wrapping layers a workflow function adds around
// Validate()'s result (the aggregate *rosaerrors.ValidationError, then its
// fmt.Errorf "invalid request: %w" wrap) to reach the errors.Join underneath.
// Validate() builds that join from one flat list (see createValidators in
// create.go), so there is exactly one Unwrap() []error to expand -- never a
// join nested inside another join -- and no recursion is needed.
func flattenLeafErrors(err error) []error {
	for err != nil {
		if joined, ok := err.(interface{ Unwrap() []error }); ok {
			return joined.Unwrap()
		}
		wrapped, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		// A type can implement Unwrap() error yet have nothing to unwrap
		// (e.g. a *rosaerrors.ValidationError leaf with no cause set) --
		// that err is itself the leaf, not an empty branch.
		inner := wrapped.Unwrap()
		if inner == nil {
			break
		}
		err = inner
	}
	if err == nil {
		return nil
	}
	return []error{err}
}

// soleValidationError returns the single *rosaerrors.ValidationError leaf
// expected in err, for tests that assert on Field.
func soleValidationError(err error) *rosaerrors.ValidationError {
	leaves := flattenLeafErrors(err)
	ExpectWithOffset(1, leaves).To(HaveLen(1))
	validationErr, ok := leaves[0].(*rosaerrors.ValidationError)
	ExpectWithOffset(1, ok).To(BeTrue(), "expected a *rosaerrors.ValidationError leaf, got: %T", leaves[0])
	return validationErr
}

var _ = Describe("CreateIAMServiceAccount", func() {
	var (
		service *Service
		client  *mockCreateIAMServiceAccountClient
		req     CreateIAMServiceAccountRequest
	)

	BeforeEach(func() {
		service = &Service{}
		client = &mockCreateIAMServiceAccountClient{}
		req = CreateIAMServiceAccountRequest{
			ClusterName:     "my-cluster",
			OIDCProviderARN: "arn:aws:iam::123456789012:oidc-provider/rh-oidc.s3.us-east-1.amazonaws.com/abc123",
			ServiceAccounts: []ServiceAccountIdentifier{
				{Name: "my-app", Namespace: "default"},
			},
			PolicyARNs: []string{"arn:aws:iam::123456789012:policy/my-policy"},
			AccountID:  "123456789012",
			Partition:  "aws",
		}
	})

	Context("successful creation", func() {
		It("should create a role with generated name for a single service account", func() {
			result, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).ToNot(HaveOccurred())

			// Role name should be auto-generated
			Expect(result.RoleName).To(Equal("my-cluster-default-my-app-role"))
			Expect(result.RoleARN).To(ContainSubstring("my-cluster-default-my-app-role"))
			Expect(result.AttachedPolicyARNs).To(Equal(req.PolicyARNs))
			Expect(result.InlinePolicyName).To(BeEmpty())

			// Verify EnsureRole was called with correct trust policy and tags
			Expect(client.ensureRoleCalls).To(HaveLen(1))
			call := client.ensureRoleCalls[0]
			Expect(call.Name).To(Equal("my-cluster-default-my-app-role"))
			Expect(call.Policy).To(ContainSubstring("sts:AssumeRoleWithWebIdentity"))
			Expect(call.Policy).To(ContainSubstring("system:serviceaccount:default:my-app"))
			Expect(call.Tags).To(HaveKeyWithValue(ClusterTagKey, "my-cluster"))
			Expect(call.Tags).To(HaveKeyWithValue(NamespaceTagKey, "default"))
			Expect(call.Tags).To(HaveKeyWithValue(ServiceAccountTagKey, "my-app"))
			Expect(call.ManagedPolicies).To(BeFalse())

			// Verify managed policy was attached
			Expect(client.attachRolePolicyCalls).To(HaveLen(1))
			Expect(client.attachRolePolicyCalls[0].PolicyARN).To(Equal("arn:aws:iam::123456789012:policy/my-policy"))
		})

		It("should use explicit role name when provided", func() {
			roleName := "custom-role-name"
			req.RoleName = &roleName
			result, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).ToNot(HaveOccurred())

			Expect(result.RoleName).To(Equal("custom-role-name"))
			Expect(client.ensureRoleCalls[0].Name).To(Equal("custom-role-name"))
		})

		It("should attach multiple managed policies", func() {
			req.PolicyARNs = []string{
				"arn:aws:iam::123456789012:policy/policy-one",
				"arn:aws:iam::123456789012:policy/policy-two",
				"arn:aws:iam::123456789012:policy/policy-three",
			}

			result, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).ToNot(HaveOccurred())

			Expect(result.AttachedPolicyARNs).To(HaveLen(3))
			Expect(client.attachRolePolicyCalls).To(HaveLen(3))
			Expect(client.attachRolePolicyCalls[0].PolicyARN).To(Equal("arn:aws:iam::123456789012:policy/policy-one"))
			Expect(client.attachRolePolicyCalls[1].PolicyARN).To(Equal("arn:aws:iam::123456789012:policy/policy-two"))
			Expect(client.attachRolePolicyCalls[2].PolicyARN).To(Equal("arn:aws:iam::123456789012:policy/policy-three"))
		})

		It("should attach inline policy when provided", func() {
			inlinePolicy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}`
			req.InlinePolicy = &inlinePolicy

			result, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).ToNot(HaveOccurred())

			Expect(result.InlinePolicyName).To(Equal("my-cluster-default-my-app-role-inline-policy"))
			Expect(client.putRolePolicyCalls).To(HaveLen(1))
			Expect(client.putRolePolicyCalls[0].PolicyName).To(Equal("my-cluster-default-my-app-role-inline-policy"))
			Expect(client.putRolePolicyCalls[0].Policy).To(Equal(inlinePolicy))
		})

		It("should propagate the caller's context to every client call", func() {
			inlinePolicy := `{"Version":"2012-10-17","Statement":[]}`
			req.InlinePolicy = &inlinePolicy

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			_, err := service.CreateIAMServiceAccount(ctx, client, req)
			Expect(err).ToNot(HaveOccurred())

			Expect(client.ensureRoleCalls).To(HaveLen(1))
			Expect(client.ensureRoleCalls[0].Context).To(Equal(ctx))
			Expect(client.attachRolePolicyCalls).To(HaveLen(1))
			Expect(client.attachRolePolicyCalls[0].Context).To(Equal(ctx))
			Expect(client.putRolePolicyCalls).To(HaveLen(1))
			Expect(client.putRolePolicyCalls[0].Context).To(Equal(ctx))
		})

		It("should not call PutRolePolicy when no inline policy is provided", func() {
			result, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).ToNot(HaveOccurred())

			Expect(result.InlinePolicyName).To(BeEmpty())
			Expect(client.putRolePolicyCalls).To(BeEmpty())
		})

		It("should pass permissions boundary and path through to EnsureRole", func() {
			boundary := "arn:aws:iam::123456789012:policy/boundary"
			path := "/rosa/"
			req.PermissionsBoundary = &boundary
			req.Path = &path

			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).ToNot(HaveOccurred())

			call := client.ensureRoleCalls[0]
			Expect(call.PermissionsBoundary).To(Equal("arn:aws:iam::123456789012:policy/boundary"))
			Expect(call.Path).To(Equal("/rosa/"))
		})

		It("should override role ARN for GovCloud environments", func() {
			req.IsGovcloud = true
			req.AccountID = "111222333444"
			req.Partition = "aws-us-gov"

			result, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).ToNot(HaveOccurred())

			// The ARN should be reconstructed with the GovCloud partition
			Expect(result.RoleARN).To(Equal("arn:aws-us-gov:iam::111222333444:role/my-cluster-default-my-app-role"))
		})

		It("should handle multiple service accounts with explicit role name", func() {
			roleName := "shared-role"
			req.RoleName = &roleName
			req.ServiceAccounts = []ServiceAccountIdentifier{
				{Name: "controller-manager", Namespace: "my-operator"},
				{Name: "controller-cluster", Namespace: "my-operator"},
			}

			result, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).ToNot(HaveOccurred())

			Expect(result.RoleName).To(Equal("shared-role"))

			// Trust policy should contain both service accounts
			trustPolicy := client.ensureRoleCalls[0].Policy
			Expect(trustPolicy).To(ContainSubstring("system:serviceaccount:my-operator:controller-manager"))
			Expect(trustPolicy).To(ContainSubstring("system:serviceaccount:my-operator:controller-cluster"))
		})
	})

	Context("validation errors", func() {
		It("should reject request with no cluster name", func() {
			req.ClusterName = ""
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster name is required"))
			Expect(soleValidationError(err).Field).To(Equal("ClusterName"))
			expectNoClientCalls(client)
		})

		It("should reject request with no OIDC provider ARN", func() {
			req.OIDCProviderARN = ""
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("OIDC provider ARN is required"))
			Expect(soleValidationError(err).Field).To(Equal("OIDCProviderARN"))
			expectNoClientCalls(client)
		})

		It("should reject request with no service accounts", func() {
			req.ServiceAccounts = nil
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one service account is required"))
			Expect(soleValidationError(err).Field).To(Equal("ServiceAccounts"))
			expectNoClientCalls(client)
		})

		It("should reject request with no policies", func() {
			req.PolicyARNs = nil
			req.InlinePolicy = nil
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one policy ARN or inline policy is required"))
			// This check spans two fields (PolicyARNs, InlinePolicy), so it
			// intentionally carries no single Field.
			Expect(soleValidationError(err).Field).To(BeEmpty())
			expectNoClientCalls(client)
		})

		It("should report every violation when multiple invariants are broken", func() {
			req.ClusterName = ""
			req.OIDCProviderARN = ""
			req.PolicyARNs = nil
			req.InlinePolicy = nil
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster name is required"))
			Expect(err.Error()).To(ContainSubstring("OIDC provider ARN is required"))
			Expect(err.Error()).To(ContainSubstring("at least one policy ARN or inline policy is required"))

			leaves := flattenLeafErrors(err)
			Expect(leaves).To(HaveLen(3), "expected exactly the three broken invariants, got: %v", leaves)
			expectNoClientCalls(client)
		})

		It("should reject request with blank policy ARN", func() {
			req.PolicyARNs = []string{"arn:aws:iam::123456789012:policy/valid", ""}
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("policy ARN at index 1 is empty"))
			Expect(soleValidationError(err).Field).To(Equal("PolicyARNs"))
			expectNoClientCalls(client)
		})

		It("should reject request with empty inline policy string", func() {
			emptyPolicy := ""
			req.InlinePolicy = &emptyPolicy
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("inline policy must not be empty when provided"))
			Expect(soleValidationError(err).Field).To(Equal("InlinePolicy"))
			expectNoClientCalls(client)
		})

		It("should reject request with malformed policy ARN", func() {
			req.PolicyARNs = []string{"not-an-arn"}
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("policy ARN at index 0 is invalid"))
			validationErr := soleValidationError(err)
			Expect(validationErr.Field).To(Equal("PolicyARNs"))
			Expect(validationErr.Err).To(HaveOccurred(), "the arn.Parse cause must be preserved, not flattened into the message")
			expectNoClientCalls(client)
		})

		It("should reject request with invalid JSON inline policy", func() {
			invalidPolicy := `{"Version": "2012-10-17", "Statement": [`
			req.InlinePolicy = &invalidPolicy
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("inline policy must be valid JSON"))
			Expect(soleValidationError(err).Field).To(Equal("InlinePolicy"))
			expectNoClientCalls(client)
		})

		It("should reject request with invalid service account name", func() {
			req.ServiceAccounts = []ServiceAccountIdentifier{
				{Name: "INVALID-NAME", Namespace: "default"},
			}
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid service account name"))
			Expect(soleValidationError(err).Field).To(Equal("ServiceAccounts"))
			expectNoClientCalls(client)
		})

		It("should reject request with invalid namespace name", func() {
			req.ServiceAccounts = []ServiceAccountIdentifier{
				{Name: "my-app", Namespace: "INVALID-NS"},
			}
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid namespace"))
			Expect(soleValidationError(err).Field).To(Equal("ServiceAccounts"))
			expectNoClientCalls(client)
		})

		It("should reject GovCloud request with empty account ID", func() {
			req.IsGovcloud = true
			req.AccountID = ""
			req.Partition = "aws-us-gov"
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("account ID is required for GovCloud"))
			Expect(soleValidationError(err).Field).To(Equal("AccountID"))
			expectNoClientCalls(client)
		})

		It("should reject GovCloud request with empty partition", func() {
			req.IsGovcloud = true
			req.AccountID = "111222333444"
			req.Partition = ""
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("partition is required for GovCloud"))
			Expect(soleValidationError(err).Field).To(Equal("Partition"))
			expectNoClientCalls(client)
		})

		It("should report both GovCloud violations when account ID and partition are empty", func() {
			req.IsGovcloud = true
			req.AccountID = ""
			req.Partition = ""
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("account ID is required for GovCloud"))
			Expect(err.Error()).To(ContainSubstring("partition is required for GovCloud"))

			leaves := flattenLeafErrors(err)
			Expect(leaves).To(HaveLen(2), "expected both GovCloud violations, got: %v", leaves)
			expectNoClientCalls(client)
		})

		It("should reject multiple service accounts without explicit role name", func() {
			req.RoleName = nil
			req.ServiceAccounts = []ServiceAccountIdentifier{
				{Name: "app-one", Namespace: "default"},
				{Name: "app-two", Namespace: "default"},
			}
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("role name is required when specifying multiple service accounts"))
			Expect(soleValidationError(err).Field).To(Equal("RoleName"))
			expectNoClientCalls(client)
		})

		It("should reject blank explicit role name", func() {
			blankRoleName := "   "
			req.RoleName = &blankRoleName
			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("role name must not be blank when provided"))
			Expect(soleValidationError(err).Field).To(Equal("RoleName"))
			expectNoClientCalls(client)
		})

		It("should allow inline-only policy (no managed policy ARNs)", func() {
			req.PolicyARNs = nil
			inlinePolicy := `{"Version":"2012-10-17","Statement":[]}`
			req.InlinePolicy = &inlinePolicy

			result, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.InlinePolicyName).ToNot(BeEmpty())
			Expect(client.attachRolePolicyCalls).To(BeEmpty())
		})

		It("should classify a Validate() failure as a rosaerrors.ValidationError", func() {
			req.ClusterName = ""

			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())

			var validationErr *rosaerrors.ValidationError
			Expect(errors.As(err, &validationErr)).To(BeTrue())
			expectNoClientCalls(client)
		})
	})

	Context("AWS client errors", func() {
		It("should propagate EnsureRole errors", func() {
			errEnsureRole := errors.New("access denied")
			client.ensureRoleFn = func(string, string, string, string, map[string]string, string, bool) (string, error) {
				return "", errEnsureRole
			}

			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, errEnsureRole)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("failed to create role"))

			// An operational AWS failure must not be mistaken for an invalid
			// request: it happens only after Validate() already succeeded.
			var validationErr *rosaerrors.ValidationError
			Expect(errors.As(err, &validationErr)).To(BeFalse())

			// Should not attempt to attach policies after role creation fails
			Expect(client.attachRolePolicyCalls).To(BeEmpty())
		})

		It("should propagate AttachRolePolicy errors", func() {
			errAttachRolePolicy := errors.New("policy not found")
			client.attachRolePolicyFn = func(string, string) error {
				return errAttachRolePolicy
			}

			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, errAttachRolePolicy)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("failed to attach policy"))
		})

		It("should propagate PutRolePolicy errors", func() {
			inlinePolicy := `{"Version":"2012-10-17","Statement":[]}`
			req.InlinePolicy = &inlinePolicy
			errPutRolePolicy := errors.New("malformed policy document")
			client.putRolePolicyFn = func(string, string, string) error {
				return errPutRolePolicy
			}

			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, errPutRolePolicy)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("failed to attach inline policy"))
		})

		It("should stop attaching policies after first failure", func() {
			callCount := 0
			client.attachRolePolicyFn = func(string, string) error {
				callCount++
				if callCount == 2 {
					return fmt.Errorf("second policy failed")
				}
				return nil
			}
			req.PolicyARNs = []string{
				"arn:aws:iam::123456789012:policy/policy-one",
				"arn:aws:iam::123456789012:policy/policy-two",
				"arn:aws:iam::123456789012:policy/policy-three",
			}

			_, err := service.CreateIAMServiceAccount(context.Background(), client, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("second policy failed"))

			// Should have stopped after the second policy failed
			Expect(client.attachRolePolicyCalls).To(HaveLen(2))
		})
	})
})
