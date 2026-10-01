package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("hyperfleet dispatch", func() {
	var origEnabled func() bool
	var origList func(*cobra.Command, []string)

	BeforeEach(func() {
		origEnabled = hyperfleetEnabled
		origList = hfListMachinePools
	})

	AfterEach(func() {
		hyperfleetEnabled = origEnabled
		hfListMachinePools = origList
	})

	It("routes to hfListMachinePools when hyperfleet is enabled", func() {
		called := false
		hyperfleetEnabled = func() bool { return true }
		hfListMachinePools = func(_ *cobra.Command, _ []string) { called = true }

		cmd := NewListMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})
