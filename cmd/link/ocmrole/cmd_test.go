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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLinkOCMRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Link OCM Role suite")
}

var _ = Describe("nonCompliantRoleNameWarning", func() {
	It("names the expected naming convention without echoing caller-supplied values", func() {
		message := nonCompliantRoleNameWarning()

		Expect(message).To(
			ContainSubstring("<prefix>-OCM-Role-<ocm-organization-external-id>"),
			"expected the warning to describe the naming convention generically")
		Expect(message).To(
			ContainSubstring("organization-not-authorized"),
			"expected the warning to explain the downstream OCM failure")
	})
})

var _ = Describe("unknownExternalIDWarning", func() {
	It("explains that the naming convention could not be checked, without asserting the role name is wrong", func() {
		message := unknownExternalIDWarning()

		Expect(message).To(
			ContainSubstring("could not be determined"),
			"expected the warning to explain that the organization's external ID is unknown")
		Expect(message).To(
			ContainSubstring("<prefix>-OCM-Role-<ocm-organization-external-id>"),
			"expected the warning to describe the naming convention generically")
		Expect(message).NotTo(
			ContainSubstring("does not match"),
			"expected the warning to avoid asserting the role name itself is wrong")
	})
})
