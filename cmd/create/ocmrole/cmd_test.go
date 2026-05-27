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
	"os"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

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
		It("should generate no-console tag commands when profile is no-console", func() {
			commands, err := buildCommands(
				"test",
				"test-OCM-Role",
				"",
				"",
				creator,
				"production",
				ProfileNoConsole,
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
				ProfileNoConsole,
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
				ProfileAdmin,
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
				ProfileAdmin,
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
				ProfileStandard,
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
				ProfileNoConsole,
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
		err := generateOcmRolePolicyFiles(r, env, orgID, ProfileNoConsole, policies)
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
		err := generateOcmRolePolicyFiles(r, env, orgID, ProfileStandard, policies)
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
		err := generateOcmRolePolicyFiles(r, env, orgID, ProfileAdmin, policies)
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
		err := generateOcmRolePolicyFiles(r, env, orgID, ProfileNoConsole, policies)

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

			arn, exists, err := checkRoleExists(r, roleName, ProfileStandard, "auto")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error if existing role is admin", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(true, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)

			_, exists, err := checkRoleExists(r, roleName, ProfileStandard, "auto")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is an admin role"))
		})

		It("should error if existing role is no-console", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(true, nil)

			_, exists, err := checkRoleExists(r, roleName, ProfileStandard, "auto")

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

			arn, exists, err := checkRoleExists(r, roleName, ProfileAdmin, "auto")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error if existing role is no-console", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(true, nil)

			_, exists, err := checkRoleExists(r, roleName, ProfileAdmin, "auto")

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

			arn, exists, err := checkRoleExists(r, roleName, ProfileNoConsole, "auto")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal("arn:aws:iam::123456789012:role/test-role"))
		})

		It("should error if existing role is admin", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(true, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)

			_, exists, err := checkRoleExists(r, roleName, ProfileNoConsole, "auto")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is an admin role"))
		})

		It("should error if existing role is standard", func() {
			mockClient.EXPECT().CheckRoleExists(roleName).Return(true, "arn:aws:iam::123456789012:role/test-role", nil)
			mockClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			mockClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)

			_, exists, err := checkRoleExists(r, roleName, ProfileNoConsole, "auto")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the existing role is a standard role"))
		})
	})
})
