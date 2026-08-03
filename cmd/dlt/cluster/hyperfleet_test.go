package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("hyperfleet dispatch", func() {
	var (
		origEnabled    func() bool
		origRunCluster func()
	)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origRunCluster = hfDeleteCluster
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfDeleteCluster = origRunCluster
	})

	It("routes to hfDeleteCluster when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfDeleteCluster = func() { called = true }

		run(nil, nil)

		Expect(called).To(BeTrue())
	})
})
