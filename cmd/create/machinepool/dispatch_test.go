// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
)

var _ = Describe("Dispatch", func() {
	AfterEach(func() {
		hyperfleetEnabled = hyperfleet.Enabled
	})

	Context("when hyperfleet is disabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return false }
		})

		It("should return a runner function", func() {
			opts := &mpOpts.CreateMachinepoolUserOptions{}
			runner := dispatch(opts)
			Expect(runner).NotTo(BeNil())
		})
	})

	Context("when hyperfleet is enabled", func() {
		var origCreate func(*mpOpts.CreateMachinepoolUserOptions, []string, *cobra.Command)

		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return true }
			origCreate = hfCreateMachinePool
		})

		AfterEach(func() {
			hfCreateMachinePool = origCreate
		})

		It("should return a runner that calls the v2 handler", func() {
			called := false
			hfCreateMachinePool = func(*mpOpts.CreateMachinepoolUserOptions, []string, *cobra.Command) {
				called = true
			}
			opts := &mpOpts.CreateMachinepoolUserOptions{}
			runner := dispatch(opts)
			Expect(runner).NotTo(BeNil())
			runner(&cobra.Command{}, []string{})
			Expect(called).To(BeTrue())
		})
	})
})
