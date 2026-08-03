package cluster

import (
	"github.com/spf13/cobra"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("hyperfleet dispatch", func() {
	var (
		origEnabled    func() bool
		origEditCluster func(*cobra.Command)
	)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origEditCluster = hfEditCluster
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfEditCluster = origEditCluster
	})

	It("routes to hfEditCluster when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfEditCluster = func(_ *cobra.Command) { called = true }

		run(makeCmd(), nil)

		Expect(called).To(BeTrue())
	})
})
