// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("seaplane dispatch", func() {
	var origEnabled func() bool
	var origDelete func(*cobra.Command, *DeleteMachinepoolUserOptions, []string)

	BeforeEach(func() {
		origEnabled = seaplaneEnabled
		origDelete = seaplaneDeleteMachinePool
	})

	AfterEach(func() {
		seaplaneEnabled = origEnabled
		seaplaneDeleteMachinePool = origDelete
	})

	It("routes to seaplaneDeleteMachinePool when seaplane is enabled", func() {
		called := false
		seaplaneEnabled = func() bool { return true }
		seaplaneDeleteMachinePool = func(_ *cobra.Command, _ *DeleteMachinepoolUserOptions, _ []string) { called = true }

		cmd := NewDeleteMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})
