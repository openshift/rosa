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
			runner := dispatch()
			Expect(runner).NotTo(BeNil())
		})
	})

	Context("when hyperfleet is enabled", func() {
		var origList func(*cobra.Command, []string)

		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return true }
			origList = hfListMachinePools
		})

		AfterEach(func() {
			hfListMachinePools = origList
		})

		It("should return a runner that calls the v2 handler", func() {
			called := false
			hfListMachinePools = func(*cobra.Command, []string) {
				called = true
			}
			runner := dispatch()
			Expect(runner).NotTo(BeNil())
			runner(&cobra.Command{}, []string{})
			Expect(called).To(BeTrue())
		})
	})
})
