// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Dispatch", func() {
	var cmd *cobra.Command
	var runCalled bool
	var v2Called bool
	var originalEnabled func() bool
	var originalRun func(*cobra.Command, []string)
	var originalList func()

	BeforeEach(func() {
		cmd = &cobra.Command{}
		runCalled = false
		v2Called = false
		originalEnabled = hyperfleetEnabled
		originalRun = runV1
		originalList = hfListClusters

		runV1 = func(*cobra.Command, []string) {
			runCalled = true
		}
		hfListClusters = func() { v2Called = true }
	})

	AfterEach(func() {
		// Reset to production defaults
		runV1 = originalRun
		hfListClusters = originalList
		hyperfleetEnabled = originalEnabled
	})

	Context("when hyperfleet is disabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return false }
		})

		It("should call the v1 run function", func() {
			dispatch(cmd, []string{})
			Expect(runCalled).To(BeTrue())
		})

	})

	Context("when hyperfleet is enabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return true }
		})

		It("should not call the v1 run function", func() {
			dispatch(cmd, []string{})
			Expect(runCalled).To(BeFalse())
		})

		It("should call the v2 list handler", func() {
			dispatch(cmd, []string{})
			Expect(v2Called).To(BeTrue())
			Expect(runCalled).To(BeFalse())
		})
	})
})
