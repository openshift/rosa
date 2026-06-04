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
	"testing"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	r          *rosa.Runtime
	ctrl       *gomock.Controller
	mockClient *aws.MockClient
)

func TestInternalOCMRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal OCM Role Suite")
}

var _ = Describe("CheckRoleExistsInternal", func() {
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = aws.NewMockClient(ctrl)
		r = &rosa.Runtime{
			AWSClient: mockClient,
			Reporter:  reporter.CreateReporter(),
			Creator: &aws.Creator{
				ARN:       "arn:aws:iam::123456789012:user/test",
				AccountID: "123456789012",
				Partition: "aws",
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("when role does not exist", func() {
		It("should return exists=false for any profile", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(false, "", nil)

			roleARN, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileStandard, "auto", "/")

			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(roleARN).To(BeEmpty())
		})
	})

	Context("when requesting standard profile", func() {
		It("should succeed if existing role is standard", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(false, nil)

			roleARN, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileStandard, "auto", "/")

			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error if existing role is admin", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(true, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(false, nil)

			roleARN, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileStandard, "auto", "/")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/test-role"))
			Expect(err.Error()).To(ContainSubstring("the existing role is an admin role"))
		})

		It("should error if existing role is no-console", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(true, nil)

			_, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileStandard, "auto", "/")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is a no-console role"))
		})
	})

	Context("when requesting admin profile", func() {
		It("should succeed if existing role is admin (idempotent)", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(true, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(false, nil)

			roleARN, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileAdmin, "auto", "/")

			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error if existing role is no-console", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(true, nil)

			_, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileAdmin, "auto", "/")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is a no-console role"))
		})

		It("should succeed with self-healing if existing role has admin policy but missing tag", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies("test-role").Return([]string{
				"arn:aws:iam::123456789012:policy/test-role-Policy",
				"arn:aws:iam::123456789012:policy/test-role-Admin-Policy",
			}, nil)
			mockClient.EXPECT().AddRoleTag("test-role", "rosa_admin_role", "true").Return(nil)

			roleARN, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileAdmin, "auto", "/")

			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})
	})

	Context("when requesting no-console profile", func() {
		It("should succeed if existing role is no-console", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(true, nil)
			mockClient.EXPECT().ListAttachedRolePolicies("test-role").Return([]string{
				"arn:aws:iam::123456789012:policy/test-role-NoConsole-Policy",
			}, nil)

			roleARN, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileNoConsole, "auto", "/")

			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error if existing role is admin", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(true, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(false, nil)

			_, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileNoConsole, "auto", "/")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is an admin role"))
		})

		It("should error if existing role is standard", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies("test-role").Return([]string{
				"arn:aws:iam::123456789012:policy/ManagedOpenShift-OCM-Role-Policy",
			}, nil)

			_, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileNoConsole, "auto", "/")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is a standard role"))
		})

		It("should self-heal when no-console policy exists but tag is missing", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies("test-role").Return([]string{
				"arn:aws:iam::123456789012:policy/test-role-NoConsole-Policy",
			}, nil)
			mockClient.EXPECT().AddRoleTag("test-role", "rosa_no_console_role", "true").Return(nil)

			roleARN, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileNoConsole, "auto", "/")

			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error when self-healing tag addition fails", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies("test-role").Return([]string{
				"arn:aws:iam::123456789012:policy/test-role-NoConsole-Policy",
			}, nil)
			mockClient.EXPECT().AddRoleTag("test-role", "rosa_no_console_role", "true").Return(
				fmt.Errorf("tag operation failed"))

			_, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileNoConsole, "auto", "/")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("failed to add no-console role tag"))
		})

		It("should error when no-console tag exists but policy is not attached", func() {
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(true, nil)
			mockClient.EXPECT().ListAttachedRolePolicies("test-role").Return([]string{
				"arn:aws:iam::123456789012:policy/test-role-Policy",
			}, nil)

			_, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileNoConsole, "auto", "/")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the role has the no-console tag but the no-console policy is not attached"))
		})
	})

	Context("when dealing with custom role paths", func() {
		It("should self-heal no-console role with custom path", func() {
			customPath := "/custom/path/"
			mockClient.EXPECT().CheckRoleExists("test-role").Return(true, "arn:aws:iam::123456789012:role/custom/path/test-role", nil)
			mockClient.EXPECT().IsAdminRole("test-role").Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole("test-role").Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies("test-role").Return([]string{
				"arn:aws:iam::123456789012:policy/custom/path/test-role-NoConsole-Policy",
			}, nil)
			mockClient.EXPECT().AddRoleTag("test-role", "rosa_no_console_role", "true").Return(nil)

			roleARN, exists, err := CheckRoleExistsInternal(r, "test-role", ProfileNoConsole, "auto", customPath)

			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/custom/path/test-role"))
		})
	})
})

var _ = Describe("CreateRolesInternal", func() {
	var (
		policies map[string]*cmv1.AWSSTSPolicy
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = aws.NewMockClient(ctrl)
		r = &rosa.Runtime{
			AWSClient: mockClient,
			Reporter:  reporter.CreateReporter(),
			Creator: &aws.Creator{
				ARN:       "arn:aws:iam::123456789012:user/test",
				AccountID: "123456789012",
				Partition: "aws",
			},
		}

		// Create test policies
		trustPolicy, _ := cmv1.NewAWSSTSPolicy().
			ID("sts_ocm_trust_policy").
			Details(`{"Version":"2012-10-17","Statement":[]}`).
			Build()
		standardPolicy, _ := cmv1.NewAWSSTSPolicy().
			ID("sts_ocm_permission_policy").
			Details(`{"Version":"2012-10-17","Statement":[]}`).
			Build()
		adminPolicy, _ := cmv1.NewAWSSTSPolicy().
			ID("sts_ocm_admin_permission_policy").
			Details(`{"Version":"2012-10-17","Statement":[]}`).
			Build()
		noConsolePolicy, _ := cmv1.NewAWSSTSPolicy().
			ID("sts_ocm_no_console_permission_policy").
			Details(`{"Version":"2012-10-17","Statement":[]}`).
			Build()

		policies = map[string]*cmv1.AWSSTSPolicy{
			"sts_ocm_trust_policy":                 trustPolicy,
			"sts_ocm_permission_policy":            standardPolicy,
			"sts_ocm_admin_permission_policy":      adminPolicy,
			"sts_ocm_no_console_permission_policy": noConsolePolicy,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should create standard OCM role successfully", func() {
		mockClient.EXPECT().EnsureRole(
			gomock.Any(), // reporter
			gomock.Eq("test-role"),
			gomock.Any(), // trust policy
			gomock.Eq(""),
			gomock.Eq(""),
			gomock.Any(), // tags
			gomock.Eq("/"),
			gomock.Eq(false),
		).Return("arn:aws:iam::123456789012:role/test-role", nil)

		mockClient.EXPECT().EnsurePolicy(
			gomock.Any(), // policy ARN
			gomock.Any(), // policy document
			gomock.Eq(""),
			gomock.Any(), // tags
			gomock.Eq("/"),
		).Return("arn:aws:iam::123456789012:policy/test-role-Policy", nil)

		mockClient.EXPECT().AttachRolePolicy(
			gomock.Any(),
			gomock.Eq("test-role"),
			gomock.Any(),
		).Return(nil)

		roleARN, err := CreateRolesInternal(r, "test-prefix", "test-role", "/", "",
			"org-123", "production", ProfileStandard, policies, false)

		Expect(err).ToNot(HaveOccurred())
		Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/test-role"))
	})

	It("should create admin OCM role with two policies", func() {
		mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("arn:aws:iam::123456789012:role/test-role", nil)

		// Standard policy
		mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("arn:aws:iam::123456789012:policy/test-role-Policy", nil)
		mockClient.EXPECT().AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		// Admin policy
		mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("arn:aws:iam::123456789012:policy/test-role-Admin-Policy", nil)
		mockClient.EXPECT().AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		// Admin tag
		mockClient.EXPECT().AddRoleTag("test-role", "rosa_admin_role", "true").Return(nil)

		roleARN, err := CreateRolesInternal(r, "test-prefix", "test-role", "/", "",
			"org-123", "production", ProfileAdmin, policies, false)

		Expect(err).ToNot(HaveOccurred())
		Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/test-role"))
	})

	It("should create no-console OCM role with policy and tag", func() {
		mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("arn:aws:iam::123456789012:role/test-role", nil)

		// NoConsole policy
		mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("arn:aws:iam::123456789012:policy/test-role-NoConsole-Policy", nil)
		mockClient.EXPECT().AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		// NoConsole tag
		mockClient.EXPECT().AddRoleTag("test-role", "rosa_no_console_role", "true").Return(nil)

		roleARN, err := CreateRolesInternal(r, "test-prefix", "test-role", "/", "",
			"org-123", "production", ProfileNoConsole, policies, false)

		Expect(err).ToNot(HaveOccurred())
		Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/test-role"))
	})

	It("should fail when role creation fails", func() {
		mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", fmt.Errorf("role creation failed"))

		_, err := CreateRolesInternal(r, "test-prefix", "test-role", "/", "",
			"org-123", "production", ProfileStandard, policies, false)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("role creation failed"))
	})

	It("should fail when policy creation fails", func() {
		mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("arn:aws:iam::123456789012:role/test-role", nil)

		mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", fmt.Errorf("policy creation failed"))

		_, err := CreateRolesInternal(r, "test-prefix", "test-role", "/", "",
			"org-123", "production", ProfileStandard, policies, false)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("policy creation failed"))
	})

	It("should fail when policy attachment fails", func() {
		mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("arn:aws:iam::123456789012:role/test-role", nil)

		mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("arn:aws:iam::123456789012:policy/test-role-Policy", nil)

		mockClient.EXPECT().AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(fmt.Errorf("attachment failed"))

		_, err := CreateRolesInternal(r, "test-prefix", "test-role", "/", "",
			"org-123", "production", ProfileStandard, policies, false)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("attachment failed"))
	})
})
