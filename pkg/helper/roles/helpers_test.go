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

package roles

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRolesHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Roles Helper Suite")
}

var _ = Describe("Roles Helper", func() {
	Context("Validate Additional Allowed Principals", func() {
		It("should pass when valid ARNs", func() {
			aapARNs := []string{
				"arn:aws:iam::123456789012:role/role1",
				"arn:aws:iam::123456789012:role/role2",
			}
			err := ValidateAdditionalAllowedPrincipals(aapARNs)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("should error when containing duplicate ARNs", func() {
			aapARNs := []string{
				"arn:aws:iam::123456789012:role/role1",
				"arn:aws:iam::123456789012:role/role2",
				"arn:aws:iam::123456789012:role/role1",
			}
			err := ValidateAdditionalAllowedPrincipals(aapARNs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"Invalid additional allowed principals list, " +
					"duplicate key 'arn:aws:iam::123456789012:role/role1' found"))
		})

		It("should error when contain invalid ARN", func() {
			aapARNs := []string{
				"arn:aws:iam::123456789012:role/role1",
				"foobar",
			}
			err := ValidateAdditionalAllowedPrincipals(aapARNs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Expected valid ARNs for additional allowed principals list"))
		})

	})
})
