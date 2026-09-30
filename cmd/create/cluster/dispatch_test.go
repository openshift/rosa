// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
)

var _ = Describe("Dispatch", func() {
	var cmd *cobra.Command
	var runCalled bool
	var v2Called bool
	var origV2 func(*cobra.Command)

	BeforeEach(func() {
		cmd = &cobra.Command{}
		runCalled = false
		v2Called = false
		origV2 = hfCreateCluster

		runV1 = func(*cobra.Command, []string) {
			runCalled = true
		}
		hfCreateCluster = func(*cobra.Command) {
			v2Called = true
		}
	})

	AfterEach(func() {
		runV1 = run
		hfCreateCluster = origV2
		hyperfleetEnabled = hyperfleet.Enabled
	})

	Context("when hyperfleet is disabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return false }
		})

		It("should call the v1 run function", func() {
			dispatch(cmd, []string{})
			Expect(runCalled).To(BeTrue())
			Expect(v2Called).To(BeFalse())
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

		It("should call the v2 runner", func() {
			dispatch(cmd, []string{})
			Expect(v2Called).To(BeTrue())
		})
	})
})
