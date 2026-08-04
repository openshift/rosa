package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("hyperfleet dispatch", func() {
	var origEnabled func() bool
	var origDelete func(*DeleteMachinepoolUserOptions, []string)

	BeforeEach(func() {
		origEnabled = hyperfleetEnabled
		origDelete = hfDeleteMachinePool
	})

	AfterEach(func() {
		hyperfleetEnabled = origEnabled
		hfDeleteMachinePool = origDelete
	})

	It("routes to hfDeleteMachinePool when hyperfleet is enabled", func() {
		called := false
		hyperfleetEnabled = func() bool { return true }
		hfDeleteMachinePool = func(_ *DeleteMachinepoolUserOptions, _ []string) { called = true }

		cmd := NewDeleteMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})
