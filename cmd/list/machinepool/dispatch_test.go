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
	var originalList func(*cobra.Command, []string)

	BeforeEach(func() {
		originalEnabled = hyperfleetEnabled
		originalList = hfListMachinePools
	})

	AfterEach(func() {
		hyperfleetEnabled = originalEnabled
		hfListMachinePools = originalList
	})

	Context("when hyperfleet is disabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return false }
		})

		It("should return a runner function", func() {
			runner := dispatch()
			Expect(runner).NotTo(BeNil())
		})

	})

	Context("when hyperfleet is enabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return true }
		})

		It("routes to the Platform API handler", func() {
			called := false
			hfListMachinePools = func(*cobra.Command, []string) { called = true }
			dispatch()(&cobra.Command{}, []string{"worker"})
			Expect(called).To(BeTrue())
		})
	})
})
