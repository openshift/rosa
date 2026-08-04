package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("hyperfleet dispatch", func() {
	var origEnabled func() bool
	var origEdit func(*EditMachinepoolUserOptions, *cobra.Command, []string)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origEdit = hfEditMachinePool
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfEditMachinePool = origEdit
	})

	It("routes to hfEditMachinePool when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfEditMachinePool = func(_ *EditMachinepoolUserOptions, _ *cobra.Command, _ []string) { called = true }

		cmd := NewEditMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})
