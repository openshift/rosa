package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("seaplane dispatch", func() {
	var (
		origEnabled     func() bool
		origEditCluster func(*cobra.Command)
	)

	BeforeEach(func() {
		origEnabled = seaplaneEnabled
		origEditCluster = seaplaneEditCluster
	})

	AfterEach(func() {
		seaplaneEnabled = origEnabled
		seaplaneEditCluster = origEditCluster
	})

	It("routes to seaplaneEditCluster when seaplane is enabled", func() {
		called := false
		seaplaneEnabled = func() bool { return true }
		seaplaneEditCluster = func(_ *cobra.Command) { called = true }

		dispatch(makeCmd(), nil)

		Expect(called).To(BeTrue())
	})
})
