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
	"os"
	"testing"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	internalocmrole "github.com/openshift/rosa/internal/ocmrole"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

func TestOCMRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCM Role suite")
}

var _ = Describe("internalocmrole.RoleProfile constants", func() {
	It("Should have correct profile values", func() {
		Expect(internalocmrole.ProfileStandard).To(Equal(internalocmrole.RoleProfile("standard")))
		Expect(internalocmrole.ProfileAdmin).To(Equal(internalocmrole.RoleProfile("admin")))
		Expect(internalocmrole.ProfileNoConsole).To(Equal(internalocmrole.RoleProfile("no-console")))
	})
})

var _ = Describe("internalocmrole.DetermineProfile", func() {
	It("should return internalocmrole.ProfileAdmin when isAdmin is true", func() {
		profile := internalocmrole.DetermineProfile(true, false)
		Expect(profile).To(Equal(internalocmrole.ProfileAdmin))
	})

	It("should return internalocmrole.ProfileAdmin when both isAdmin and isNoConsole are true", func() {
		// Admin takes precedence
		profile := internalocmrole.DetermineProfile(true, true)
		Expect(profile).To(Equal(internalocmrole.ProfileAdmin))
	})

	It("should return internalocmrole.ProfileNoConsole when isNoConsole is true and isAdmin is false", func() {
		profile := internalocmrole.DetermineProfile(false, true)
		Expect(profile).To(Equal(internalocmrole.ProfileNoConsole))
	})

	It("should return internalocmrole.ProfileStandard when both are false", func() {
		profile := internalocmrole.DetermineProfile(false, false)
		Expect(profile).To(Equal(internalocmrole.ProfileStandard))
	})
})

var _ = Describe("buildCommands", func() {
	var (
		creator  *aws.Creator
		policies map[string]*cmv1.AWSSTSPolicy
	)

	BeforeEach(func() {
		creator = &aws.Creator{
			ARN:       "arn:aws:iam::123456789012:user/test",
			AccountID: "123456789012",
			Partition: "aws",
		}

		standardPolicy, err := (&cmv1.AWSSTSPolicyBuilder{}).
			ARN("arn:aws:iam::123456789012:policy/standard").
			Details(`{"Version":"2012-10-17","Statement":[]}`).
			Build()
		Expect(err).NotTo(HaveOccurred())
		adminPolicy, err := (&cmv1.AWSSTSPolicyBuilder{}).
			ARN("arn:aws:iam::123456789012:policy/admin").
			Details(`{"Version":"2012-10-17","Statement":[]}`).
			Build()
		Expect(err).NotTo(HaveOccurred())
		noConsolePolicy, err := (&cmv1.AWSSTSPolicyBuilder{}).
			ARN("arn:aws:iam::123456789012:policy/no-console").
			Details(`{"Version":"2012-10-17","Statement":[]}`).
			Build()
		Expect(err).NotTo(HaveOccurred())

		policies = map[string]*cmv1.AWSSTSPolicy{
			"sts_ocm_permission_policy":            standardPolicy,
			"sts_ocm_admin_permission_policy":      adminPolicy,
			"sts_ocm_no_console_permission_policy": noConsolePolicy,
		}
	})

	Context("Manual mode command generation", func() {
		It("should include no-console tag when profile is no-console", func() {
			commands, err := buildCommands(
				"test",
				"test-OCM-Role",
				"",
				"",
				creator,
				"production",
				internalocmrole.ProfileNoConsole,
				true, // managedPolicies
				false,
				policies,
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(commands).ToNot(BeEmpty())
			Expect(commands).To(ContainSubstring("Key=rosa_no_console_role,Value=true"))
			Expect(commands).To(ContainSubstring("test-OCM-Role"))
		})

		It("should not generate admin tag when profile is no-console", func() {
			commands, err := buildCommands(
				"test",
				"test-OCM-Role",
				"",
				"",
				creator,
				"production",
				internalocmrole.ProfileNoConsole,
				true, // managedPolicies
				false,
				policies,
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(commands).ToNot(BeEmpty())
			Expect(commands).ToNot(ContainSubstring("Key=rosa_admin_role,Value=true"))
		})

		It("should generate admin tag commands when profile is admin", func() {
			commands, err := buildCommands(
				"test",
				"test-OCM-Role",
				"",
				"",
				creator,
				"production",
				internalocmrole.ProfileAdmin,
				true, // managedPolicies
				false,
				policies,
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(commands).ToNot(BeEmpty())
			Expect(commands).To(ContainSubstring("Key=rosa_admin_role,Value=true"))
			Expect(commands).To(ContainSubstring("test-OCM-Role"))
		})

		It("should not generate no-console tag when profile is admin", func() {
			commands, err := buildCommands(
				"test",
				"test-OCM-Role",
				"",
				"",
				creator,
				"production",
				internalocmrole.ProfileAdmin,
				true, // managedPolicies
				false,
				policies,
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(commands).ToNot(BeEmpty())
			Expect(commands).ToNot(ContainSubstring("Key=rosa_no_console_role,Value=true"))
		})

		It("should not generate special tags for standard role", func() {
			commands, err := buildCommands(
				"test",
				"test-OCM-Role",
				"",
				"",
				creator,
				"production",
				internalocmrole.ProfileStandard,
				true, // managedPolicies
				false,
				policies,
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(commands).ToNot(BeEmpty())
			Expect(commands).ToNot(ContainSubstring("Key=rosa_admin_role,Value=true"))
			Expect(commands).ToNot(ContainSubstring("Key=rosa_no_console_role,Value=true"))
		})

		It("should generate attach-role-policy command for no-console role", func() {
			commands, err := buildCommands(
				"test",
				"test-OCM-Role",
				"",
				"",
				creator,
				"production",
				internalocmrole.ProfileNoConsole,
				true, // managedPolicies
				false,
				policies,
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(commands).ToNot(BeEmpty())
			Expect(commands).To(ContainSubstring("attach-role-policy"))
			Expect(commands).To(ContainSubstring("arn:aws:iam::123456789012:policy/no-console"))
		})
	})
})

var _ = Describe("generateOcmRolePolicyFiles", func() {
	var (
		r          *rosa.Runtime
		env        string
		orgID      string
		policies   map[string]*cmv1.AWSSTSPolicy
		tempDir    string
		originalWd string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "rosa-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Save current directory and change to temp
		originalWd, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		err = os.Chdir(tempDir)
		Expect(err).ToNot(HaveOccurred())

		env = "production"
		orgID = "test-org-123"

		trustPolicy, err := (&cmv1.AWSSTSPolicyBuilder{}).
			Details(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::710019948333:root"},"Action":"sts:AssumeRole"}]}`).
			Build()
		Expect(err).ToNot(HaveOccurred())
		standardPolicy, err := (&cmv1.AWSSTSPolicyBuilder{}).
			Details(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"sts:AssumeRole","Resource":"*"}]}`).
			Build()
		Expect(err).ToNot(HaveOccurred())
		adminPolicy, err := (&cmv1.AWSSTSPolicyBuilder{}).
			Details(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`).
			Build()
		Expect(err).ToNot(HaveOccurred())
		noConsolePolicy, err := (&cmv1.AWSSTSPolicyBuilder{}).
			Details(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"ec2:Describe*","Resource":"*"}]}`).
			Build()
		Expect(err).ToNot(HaveOccurred())

		policies = map[string]*cmv1.AWSSTSPolicy{
			"sts_ocm_trust_policy":                 trustPolicy,
			"sts_ocm_permission_policy":            standardPolicy,
			"sts_ocm_admin_permission_policy":      adminPolicy,
			"sts_ocm_no_console_permission_policy": noConsolePolicy,
		}

		r = rosa.NewRuntime()
		r.Creator = &aws.Creator{
			Partition: "aws",
			AccountID: "123456789012",
		}
	})

	AfterEach(func() {
		// Clean up temp directory
		err := os.Chdir(originalWd)
		Expect(err).ToNot(HaveOccurred())
		err = os.RemoveAll(tempDir)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should generate no-console permission policy file when profile is no-console", func() {
		err := generateOcmRolePolicyFiles(r, env, orgID, internalocmrole.ProfileNoConsole, policies)
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat("sts_ocm_no_console_permission_policy.json")
		Expect(err).ToNot(HaveOccurred(), "no-console policy file should exist")

		_, err = os.Stat("sts_ocm_permission_policy.json")
		Expect(os.IsNotExist(err)).To(BeTrue(), "standard policy file should not exist")

		_, err = os.Stat("sts_ocm_trust_policy.json")
		Expect(err).ToNot(HaveOccurred(), "trust policy file should exist")

		_, err = os.Stat("sts_ocm_admin_permission_policy.json")
		Expect(os.IsNotExist(err)).To(BeTrue(), "admin policy file should not exist")
	})

	It("should generate standard permission policy file when profile is standard", func() {
		err := generateOcmRolePolicyFiles(r, env, orgID, internalocmrole.ProfileStandard, policies)
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat("sts_ocm_permission_policy.json")
		Expect(err).ToNot(HaveOccurred(), "standard policy file should exist")

		_, err = os.Stat("sts_ocm_no_console_permission_policy.json")
		Expect(os.IsNotExist(err)).To(BeTrue(), "no-console policy file should not exist")

		_, err = os.Stat("sts_ocm_trust_policy.json")
		Expect(err).ToNot(HaveOccurred(), "trust policy file should exist")

		_, err = os.Stat("sts_ocm_admin_permission_policy.json")
		Expect(os.IsNotExist(err)).To(BeTrue(), "admin policy file should not exist")
	})

	It("should generate admin policy file when profile is admin", func() {
		err := generateOcmRolePolicyFiles(r, env, orgID, internalocmrole.ProfileAdmin, policies)
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat("sts_ocm_admin_permission_policy.json")
		Expect(err).ToNot(HaveOccurred(), "admin policy file should exist")

		_, err = os.Stat("sts_ocm_permission_policy.json")
		Expect(err).ToNot(HaveOccurred(), "standard policy file should exist")

		_, err = os.Stat("sts_ocm_trust_policy.json")
		Expect(err).ToNot(HaveOccurred(), "trust policy file should exist")

		_, err = os.Stat("sts_ocm_no_console_permission_policy.json")
		Expect(os.IsNotExist(err)).To(BeTrue(), "no-console policy file should not exist")
	})

	It("should generate no-console files successfully when policy is available", func() {
		err := generateOcmRolePolicyFiles(r, env, orgID, internalocmrole.ProfileNoConsole, policies)

		Expect(err).NotTo(HaveOccurred())
		// Verify no-console permission policy file was created
		fileContent, err := os.ReadFile("sts_ocm_no_console_permission_policy.json")
		Expect(err).NotTo(HaveOccurred())
		Expect(fileContent).NotTo(BeEmpty())
	})
})

var _ = Describe("checkRoleExists", func() {
	var (
		r          *rosa.Runtime
		roleName   string
		ctrl       *gomock.Controller
		mockClient *aws.MockClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = aws.NewMockClient(ctrl)
		r = rosa.NewRuntime()
		r.AWSClient = mockClient
		r.Reporter = reporter.CreateReporter()
		r.Creator = &aws.Creator{
			ARN:       "arn:aws:iam::123456789012:user/test",
			AccountID: "123456789012",
			Partition: "aws",
		}
		roleName = "test-role"
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("when requesting standard profile", func() {
		It("should succeed if existing role is standard", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)

			arn, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileStandard, "auto", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error if existing role is admin", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(true, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)

			_, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileStandard, "auto", "")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is an admin role"))
		})

		It("should error if existing role is no-console", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(true, nil)

			_, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileStandard, "auto", "")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is a no-console role"))
		})
	})

	Context("when requesting admin profile", func() {
		It("should succeed if existing role is admin (idempotent)", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(true, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)

			arn, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileAdmin, "auto", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error if existing role is no-console", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(true, nil)

			_, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileAdmin, "auto", "")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is a no-console role"))
		})

		// Note: Admin → Standard upgrade scenario with prompt is harder to test
		// because it requires mocking confirm.Prompt which exits on 'No'
	})

	Context("when requesting no-console profile", func() {
		It("should succeed if existing role is no-console", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(true, nil)
			mockClient.EXPECT().ListAttachedRolePolicies(roleName).Return([]string{
				"arn:aws:iam::123456789012:policy/test-role-NoConsole-Policy",
			}, nil)

			arn, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileNoConsole, "auto", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error if existing role is admin", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(true, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)

			_, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileNoConsole, "auto", "")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is an admin role"))
		})

		It("should error if existing role is standard", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies(roleName).Return([]string{
				"arn:aws:iam::123456789012:policy/ManagedOpenShift-OCM-Role-Policy",
			}, nil)

			_, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileNoConsole, "auto", "")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is a standard role"))
		})

		It("should self-heal when no-console policy exists but tag is missing", func() {
			noConsolePolicyARN := "arn:aws:iam::123456789012:policy/test-role-NoConsole-Policy"
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies(roleName).Return([]string{
				noConsolePolicyARN,
			}, nil)
			mockClient.EXPECT().AddRoleTag(roleName, "rosa_no_console_role", "true").Return(nil)

			arn, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileNoConsole, "auto", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error when self-healing tag addition fails", func() {
			noConsolePolicyARN := "arn:aws:iam::123456789012:policy/test-role-NoConsole-Policy"
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies(roleName).Return([]string{
				noConsolePolicyARN,
			}, nil)
			mockClient.EXPECT().AddRoleTag(roleName, "rosa_no_console_role", "true").Return(
				fmt.Errorf("tag operation failed"))

			_, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileNoConsole, "auto", "")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("failed to add no-console role tag"))
		})

		It("should error when no-console tag exists but policy is not attached", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(true, nil)
			mockClient.EXPECT().ListAttachedRolePolicies(roleName).Return([]string{
				"arn:aws:iam::123456789012:policy/test-role-Policy", // standard policy, not no-console
			}, nil)

			_, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileNoConsole, "auto", "")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the role has the no-console tag but the no-console policy is not attached"))
		})
	})

	Context("when requesting admin profile with self-healing", func() {
		It("should self-heal when admin policy exists but tag is missing", func() {
			adminPolicyARN := "arn:aws:iam::123456789012:policy/test-role-Admin-Policy"
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies(roleName).Return([]string{
				"arn:aws:iam::123456789012:policy/test-role-Policy",
				adminPolicyARN,
			}, nil)
			mockClient.EXPECT().AddRoleTag(roleName, "rosa_admin_role", "true").Return(nil)

			arn, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileAdmin, "auto", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error when self-healing tag addition fails for admin", func() {
			adminPolicyARN := "arn:aws:iam::123456789012:policy/test-role-Admin-Policy"
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies(roleName).Return([]string{
				"arn:aws:iam::123456789012:policy/test-role-Policy",
				adminPolicyARN,
			}, nil)
			mockClient.EXPECT().AddRoleTag(roleName, "rosa_admin_role", "true").Return(
				fmt.Errorf("tag operation failed"))

			_, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileAdmin, "auto", "")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("failed to add admin role tag"))
		})

		It("should self-heal no-console with custom rolePath", func() {
			customPath := "/custom/path/"
			noConsolePolicyARN := "arn:aws:iam::123456789012:policy/custom/path/test-role-NoConsole-Policy"
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/custom/path/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)
			mockClient.EXPECT().ListAttachedRolePolicies(roleName).Return([]string{
				noConsolePolicyARN,
			}, nil)
			mockClient.EXPECT().AddRoleTag(roleName, "rosa_no_console_role", "true").Return(nil)

			arn, exists, err := checkRoleExists(r, roleName, internalocmrole.ProfileNoConsole, "auto", customPath)

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:role/custom/path/test-role"))
		})
	})
})
