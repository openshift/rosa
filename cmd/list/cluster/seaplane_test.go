package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("seaplane dispatch", func() {
	var (
		origEnabled      func() bool
		origListClusters func()
	)

	BeforeEach(func() {
		origEnabled = seaplaneEnabled
		origListClusters = seaplaneListClusters
	})

	AfterEach(func() {
		seaplaneEnabled = origEnabled
		seaplaneListClusters = origListClusters
	})

	It("routes to seaplaneListClusters when seaplane is enabled", func() {
		called := false
		seaplaneEnabled = func() bool { return true }
		seaplaneListClusters = func() { called = true }

		dispatch(nil, nil)

		Expect(called).To(BeTrue())
	})
})
