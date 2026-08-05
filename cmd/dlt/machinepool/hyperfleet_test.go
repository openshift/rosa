package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("hyperfleet dispatch", func() {
	var origEnabled func() bool
	var origDelete func(*cobra.Command, *DeleteMachinepoolUserOptions, []string)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origDelete = hfDeleteMachinePool
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfDeleteMachinePool = origDelete
	})

	It("routes to hfDeleteMachinePool when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfDeleteMachinePool = func(_ *cobra.Command, _ *DeleteMachinepoolUserOptions, _ []string) { called = true }

		cmd := NewDeleteMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})
