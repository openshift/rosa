package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("hyperfleet dispatch", func() {
	var (
		origEnabled    func() bool
		origListClusters func()
	)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origListClusters = hfListClusters
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfListClusters = origListClusters
	})

	It("routes to hfListClusters when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfListClusters = func() { called = true }

		run(nil, nil)

		Expect(called).To(BeTrue())
	})
})
