// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
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
			opts := &EditMachinepoolUserOptions{}
			runner := dispatch(opts)
			Expect(runner).NotTo(BeNil())
		})
	})

	Context("when hyperfleet is enabled", func() {
		var origEdit func(*EditMachinepoolUserOptions, *cobra.Command, []string)

		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return true }
			origEdit = hfEditMachinePool
		})

		AfterEach(func() {
			hfEditMachinePool = origEdit
		})

		It("should return a runner that calls the v2 handler", func() {
			called := false
			hfEditMachinePool = func(*EditMachinepoolUserOptions, *cobra.Command, []string) {
				called = true
			}
			opts := &EditMachinepoolUserOptions{}
			runner := dispatch(opts)
			Expect(runner).NotTo(BeNil())
			runner(&cobra.Command{}, []string{})
			Expect(called).To(BeTrue())
		})
	})
})
