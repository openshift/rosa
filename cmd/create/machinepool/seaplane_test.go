// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
)

var _ = Describe("seaplane dispatch", func() {
	var origEnabled func() bool
	var origCreate func(*mpOpts.CreateMachinepoolUserOptions, []string, *cobra.Command)

	BeforeEach(func() {
		origEnabled = seaplaneEnabled
		origCreate = seaplaneCreateMachinePool
	})

	AfterEach(func() {
		seaplaneEnabled = origEnabled
		seaplaneCreateMachinePool = origCreate
	})

	It("routes to seaplaneCreateMachinePool when seaplane is enabled", func() {
		called := false
		seaplaneEnabled = func() bool { return true }
		seaplaneCreateMachinePool = func(_ *mpOpts.CreateMachinepoolUserOptions, _ []string, _ *cobra.Command) { called = true }

		cmd := NewCreateMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})
