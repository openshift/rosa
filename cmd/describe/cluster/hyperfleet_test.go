package cluster

import (
	"github.com/spf13/cobra"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("hyperfleet dispatch", func() {
	var (
		origEnabled         func() bool
		origDescribeCluster func(*cobra.Command, []string)
	)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origDescribeCluster = hfDescribeCluster
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfDescribeCluster = origDescribeCluster
	})

	It("routes to hfDescribeCluster when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfDescribeCluster = func(_ *cobra.Command, _ []string) { called = true }

		run(Cmd, nil)

		Expect(called).To(BeTrue())
	})
})
