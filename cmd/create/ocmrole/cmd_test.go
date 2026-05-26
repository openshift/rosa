/*
Copyright (c) 2024 Red Hat, Inc.

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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
)

func TestCreateOCMRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa create ocm-role")
}

func buildTestPolicies() map[string]*cmv1.AWSSTSPolicy {
	policies := map[string]*cmv1.AWSSTSPolicy{}

	trustPolicy, _ := cmv1.NewAWSSTSPolicy().
		ID("sts_ocm_trust_policy").
		Details(`{"Version":"2012-10-17","Statement":[]}`).
		Build()
	policies["sts_ocm_trust_policy"] = trustPolicy

	permPolicy, _ := cmv1.NewAWSSTSPolicy().
		ID("sts_ocm_permission_policy").
		Details(`{"Version":"2012-10-17","Statement":[]}`).
		ARN("arn:aws:iam::aws:policy/ROSAOCMPolicy").
		Build()
	policies["sts_ocm_permission_policy"] = permPolicy

	adminPolicy, _ := cmv1.NewAWSSTSPolicy().
		ID("sts_ocm_admin_permission_policy").
		Details(`{"Version":"2012-10-17","Statement":[]}`).
		ARN("arn:aws:iam::aws:policy/ROSAOCMAdminPolicy").
		Build()
	policies["sts_ocm_admin_permission_policy"] = adminPolicy

	noConsolePolicy, _ := cmv1.NewAWSSTSPolicy().
		ID("sts_ocm_no_console_permission_policy").
		Details(`{"Version":"2012-10-17","Statement":[]}`).
		ARN("arn:aws:iam::aws:policy/ROSAOCMNoConsolePolicy").
		Build()
	policies["sts_ocm_no_console_permission_policy"] = noConsolePolicy

	return policies
}

var _ = Describe("rosa create ocm-role", func() {
	Context("validateNoConsoleAdminExclusivity", func() {
		It("Returns no error when both are false", func() {
			err := validateNoConsoleAdminExclusivity(false, false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Returns no error when only admin is true", func() {
			err := validateNoConsoleAdminExclusivity(true, false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Returns no error when only noConsole is true", func() {
			err := validateNoConsoleAdminExclusivity(false, true)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Returns error when both are true", func() {
			err := validateNoConsoleAdminExclusivity(true, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("--no-console and --admin are mutually exclusive"))
		})
	})

	Context("buildCommands", func() {
		var (
			creator  *aws.Creator
			policies map[string]*cmv1.AWSSTSPolicy
		)

		BeforeEach(func() {
			creator = &aws.Creator{
				ARN:       "arn:aws:iam::111111111111:user/test",
				AccountID: "111111111111",
				Partition: "aws",
			}
			policies = buildTestPolicies()
		})

		It("Generates standard commands with no admin or no-console", func() {
			commands, err := buildCommands(
				"ManagedOpenShift", "ManagedOpenShift-OCM-Role-12345", "", "",
				creator, "production", false, false, false, false, policies,
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(commands).To(ContainSubstring("create-role"))
			Expect(commands).To(ContainSubstring("ManagedOpenShift-OCM-Role-12345"))
			Expect(commands).To(ContainSubstring("sts_ocm_permission_policy"))
			Expect(commands).ToNot(ContainSubstring(tags.AdminRole))
			Expect(commands).ToNot(ContainSubstring(tags.NoConsoleRole))
			Expect(commands).ToNot(ContainSubstring("ocm_no_console"))
			Expect(commands).To(ContainSubstring("rosa link ocm-role"))
		})

		It("Generates admin commands with admin tag and admin policy", func() {
			commands, err := buildCommands(
				"ManagedOpenShift", "ManagedOpenShift-OCM-Role-12345", "", "",
				creator, "production", true, false, false, false, policies,
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(commands).To(ContainSubstring(tags.AdminRole))
			Expect(commands).To(ContainSubstring("ocm_admin"))
			Expect(commands).ToNot(ContainSubstring(tags.NoConsoleRole))
		})

		It("Generates no-console commands with no-console tag and no-console policy", func() {
			commands, err := buildCommands(
				"ManagedOpenShift", "ManagedOpenShift-OCM-Role-12345", "", "",
				creator, "production", false, true, false, false, policies,
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(commands).To(ContainSubstring(tags.NoConsoleRole))
			Expect(commands).To(ContainSubstring("ocm_no_console"))
			Expect(commands).To(ContainSubstring("NoConsole-Policy"))
			Expect(commands).ToNot(ContainSubstring(tags.AdminRole))
			// Should NOT contain standard policy reference
			Expect(commands).ToNot(ContainSubstring("ManagedOpenShift-OCM-Role-12345-Policy"))
		})

		It("Uses managed policy ARN for no-console when managed policies enabled", func() {
			commands, err := buildCommands(
				"ManagedOpenShift", "ManagedOpenShift-OCM-Role-12345", "", "",
				creator, "production", false, true, true, false, policies,
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(commands).To(ContainSubstring("ROSAOCMNoConsolePolicy"))
			Expect(commands).To(ContainSubstring(tags.NoConsoleRole))
		})

		It("Includes link command with -y when autoConfirmLink is true", func() {
			commands, err := buildCommands(
				"ManagedOpenShift", "ManagedOpenShift-OCM-Role-12345", "", "",
				creator, "production", false, false, false, true, policies,
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(commands).To(ContainSubstring("rosa link ocm-role"))
			Expect(commands).To(ContainSubstring("-y"))
		})
	})
})
