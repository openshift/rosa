package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("seaplane dispatch", func() {
	var origEnabled func() bool
	var origEdit func(*EditMachinepoolUserOptions, *cobra.Command, []string)

	BeforeEach(func() {
		origEnabled = seaplaneEnabled
		origEdit = seaplaneEditMachinePool
	})

	AfterEach(func() {
		seaplaneEnabled = origEnabled
		seaplaneEditMachinePool = origEdit
	})

	It("routes to seaplaneEditMachinePool when seaplane is enabled", func() {
		called := false
		seaplaneEnabled = func() bool { return true }
		seaplaneEditMachinePool = func(_ *EditMachinepoolUserOptions, _ *cobra.Command, _ []string) { called = true }

		cmd := NewEditMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})
