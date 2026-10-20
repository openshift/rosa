package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("seaplane dispatch", func() {
	var origEnabled func() bool
	var origList func(*cobra.Command, []string)

	BeforeEach(func() {
		origEnabled = seaplaneEnabled
		origList = seaplaneListMachinePools
	})

	AfterEach(func() {
		seaplaneEnabled = origEnabled
		seaplaneListMachinePools = origList
	})

	It("routes to seaplaneListMachinePools when seaplane is enabled", func() {
		called := false
		seaplaneEnabled = func() bool { return true }
		seaplaneListMachinePools = func(_ *cobra.Command, _ []string) { called = true }

		cmd := NewListMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})
