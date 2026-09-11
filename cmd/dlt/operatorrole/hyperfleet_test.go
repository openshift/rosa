package operatorrole

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("hyperfleet dispatch", func() {
	var origEnabled func() bool
	var origDelete func()

	BeforeEach(func() {
		origEnabled = hfEnabled
		origDelete = hfDeleteOperatorRoles
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfDeleteOperatorRoles = origDelete
	})

	It("routes to hfDeleteOperatorRoles when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfDeleteOperatorRoles = func() { called = true }

		run(nil, nil)

		Expect(called).To(BeTrue())
	})
})
