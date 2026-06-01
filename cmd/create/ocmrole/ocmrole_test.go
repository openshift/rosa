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

func TestOCMRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCM Role suite")
}

var _ = Describe("RoleProfile constants", func() {
	It("Should have correct profile values", func() {
		Expect(ProfileStandard).To(Equal(RoleProfile("standard")))
		Expect(ProfileAdmin).To(Equal(RoleProfile("admin")))
		Expect(ProfileNoConsole).To(Equal(RoleProfile("no-console")))
	})
})

var _ = Describe("determineProfile", func() {
	It("should return ProfileAdmin when isAdmin is true", func() {
		profile := determineProfile(true, false)
		Expect(profile).To(Equal(ProfileAdmin))
	})

	It("should return ProfileAdmin when both isAdmin and isNoConsole are true", func() {
		// Admin takes precedence
		profile := determineProfile(true, true)
		Expect(profile).To(Equal(ProfileAdmin))
	})

	It("should return ProfileNoConsole when isNoConsole is true and isAdmin is false", func() {
		profile := determineProfile(false, true)
		Expect(profile).To(Equal(ProfileNoConsole))
	})

	It("should return ProfileStandard when both are false", func() {
		profile := determineProfile(false, false)
		Expect(profile).To(Equal(ProfileStandard))
	})
})
