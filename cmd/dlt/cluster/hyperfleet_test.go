package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("hyperfleet dispatch", func() {
	var (
		origEnabled    func() bool
		origRunCluster func(*cobra.Command)
	)

	BeforeEach(func() {
		origEnabled = hyperfleetEnabled
		origRunCluster = hfDeleteCluster
	})

	AfterEach(func() {
		hyperfleetEnabled = origEnabled
		hfDeleteCluster = origRunCluster
	})

	It("routes to hfDeleteCluster when hyperfleet is enabled", func() {
		called := false
		hyperfleetEnabled = func() bool { return true }
		hfDeleteCluster = func(*cobra.Command) { called = true }

		dispatch(nil, nil)

		Expect(called).To(BeTrue())
	})
})
