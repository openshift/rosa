// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
)

var _ = Describe("Dispatch", func() {
	var originalEnabled func() bool
	var originalCreate func(*mpOpts.CreateMachinepoolUserOptions, []string, *cobra.Command)

	BeforeEach(func() {
		originalEnabled = seaplaneEnabled
		originalCreate = seaplaneCreateMachinePool
	})

	AfterEach(func() {
		seaplaneEnabled = originalEnabled
		seaplaneCreateMachinePool = originalCreate
	})

	Context("when seaplane is disabled", func() {
		BeforeEach(func() {
			seaplaneEnabled = func() bool { return false }
		})

		It("should return a runner function", func() {
			opts := &mpOpts.CreateMachinepoolUserOptions{}
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
			seaplaneCreateMachinePool = func(*mpOpts.CreateMachinepoolUserOptions, []string, *cobra.Command) {
				called = true
			}
			opts := &mpOpts.CreateMachinepoolUserOptions{}
			dispatch(opts)(&cobra.Command{}, []string{"worker"})
			Expect(called).To(BeTrue())
		})
	})
})
