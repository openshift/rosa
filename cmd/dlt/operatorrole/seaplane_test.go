package operatorrole

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("seaplane dispatch", func() {
	var origEnabled func() bool
	var origDelete func()

	BeforeEach(func() {
		origEnabled = seaplaneEnabled
		origDelete = seaplaneDeleteOperatorRoles
	})

	AfterEach(func() {
		seaplaneEnabled = origEnabled
		seaplaneDeleteOperatorRoles = origDelete
	})

	It("routes to seaplaneDeleteOperatorRoles when seaplane is enabled", func() {
		called := false
		seaplaneEnabled = func() bool { return true }
		seaplaneDeleteOperatorRoles = func() { called = true }

		run(nil, nil)

		Expect(called).To(BeTrue())
	})
})
