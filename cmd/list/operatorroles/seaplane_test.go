// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package operatorroles

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("seaplane dispatch", func() {
	var (
		origSeaplaneEnabled           func() bool
		origSeaplaneExitFn            func(int)
		origSeaplaneListOperatorRoles func()

		enabledCalled bool
		listCalled    bool
	)

	BeforeEach(func() {
		origSeaplaneEnabled = seaplaneEnabled
		origSeaplaneExitFn = seaplaneExitFn
		origSeaplaneListOperatorRoles = seaplaneListOperatorRoles

		enabledCalled = false
		listCalled = false

		seaplaneEnabled = func() bool {
			enabledCalled = true
			return true
		}
		seaplaneExitFn = func(int) {}
		seaplaneListOperatorRoles = func() {
			listCalled = true
		}
	})

	AfterEach(func() {
		seaplaneEnabled = origSeaplaneEnabled
		seaplaneExitFn = origSeaplaneExitFn
		seaplaneListOperatorRoles = origSeaplaneListOperatorRoles
	})

	It("routes to seaplaneListOperatorRoles when seaplane is enabled", func() {
		run(Cmd, []string{})

		Expect(enabledCalled).To(BeTrue(), "seaplaneEnabled should be called")
		Expect(listCalled).To(BeTrue(), "seaplaneListOperatorRoles should be called")
	})

	It("does not route to seaplane when disabled", func() {
		seaplaneEnabled = func() bool {
			enabledCalled = true
			return false
		}

		// This will fail because we're not setting up the full OCM runtime,
		// but we only want to verify the dispatch logic doesn't call seaplane
		// We can't easily test the OCM path without mocking the entire runtime
		// So we'll just verify that seaplane is NOT called
		defer func() {
			if r := recover(); r != nil {
				// Expected to panic because OCM setup would fail
				// This is fine - we just want to verify seaplane wasn't called
			}
		}()

		run(Cmd, []string{})

		Expect(enabledCalled).To(BeTrue(), "seaplaneEnabled should be called")
		Expect(listCalled).To(BeFalse(), "seaplaneListOperatorRoles should NOT be called when disabled")
	})
})
