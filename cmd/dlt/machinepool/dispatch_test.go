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
	var originalDelete func(*cobra.Command, *DeleteMachinepoolUserOptions, []string)

	BeforeEach(func() {
		originalEnabled = seaplaneEnabled
		originalDelete = seaplaneDeleteMachinePool
	})

	AfterEach(func() {
		seaplaneEnabled = originalEnabled
		seaplaneDeleteMachinePool = originalDelete
	})

	Context("when seaplane is disabled", func() {
		BeforeEach(func() {
			seaplaneEnabled = func() bool { return false }
		})

		It("should return a runner function", func() {
			opts := &DeleteMachinepoolUserOptions{}
			runner := dispatch(opts)
			Expect(runner).NotTo(BeNil())
		})

	})

	Context("when seaplane is enabled", func() {
		BeforeEach(func() {
			seaplaneEnabled = func() bool { return true }
		})

		It("routes to the Platform API handler", func() {
			called := false
			seaplaneDeleteMachinePool = func(*cobra.Command, *DeleteMachinepoolUserOptions, []string) {
				called = true
			}
			opts := &DeleteMachinepoolUserOptions{}
			dispatch(opts)(&cobra.Command{}, []string{"worker"})
			Expect(called).To(BeTrue())
		})
	})
})
