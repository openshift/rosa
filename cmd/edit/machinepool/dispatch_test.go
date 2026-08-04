// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Dispatch", func() {
	var originalEnabled func() bool
	var originalEdit func(*EditMachinepoolUserOptions, *cobra.Command, []string)

	BeforeEach(func() {
		originalEnabled = hyperfleetEnabled
		originalEdit = hfEditMachinePool
	})

	AfterEach(func() {
		hyperfleetEnabled = originalEnabled
		hfEditMachinePool = originalEdit
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
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return true }
		})

		It("routes to the Platform API handler", func() {
			called := false
			hfEditMachinePool = func(*EditMachinepoolUserOptions, *cobra.Command, []string) { called = true }
			opts := &EditMachinepoolUserOptions{}
			cmd := &cobra.Command{}
			dispatch(opts)(cmd, []string{"worker"})
			Expect(called).To(BeTrue())
		})
	})
})
