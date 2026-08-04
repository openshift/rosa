package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
)

var _ = Describe("hyperfleet dispatch", func() {
	var origEnabled func() bool
	var origCreate func(*mpOpts.CreateMachinepoolUserOptions, []string)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origCreate = hfCreateMachinePool
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfCreateMachinePool = origCreate
	})

	It("routes to hfCreateMachinePool when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfCreateMachinePool = func(_ *mpOpts.CreateMachinepoolUserOptions, _ []string) { called = true }

		cmd := NewCreateMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})
