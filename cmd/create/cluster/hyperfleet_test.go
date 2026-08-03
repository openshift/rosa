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
		origRunCluster = hfCreateCluster
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfCreateCluster = origRunCluster
	})

	It("routes to hfCreateCluster when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfCreateCluster = func() { called = true }

		run(nil, nil)

		Expect(called).To(BeTrue())
	})
})
