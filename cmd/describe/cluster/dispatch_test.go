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
	var originalDescribe func(*cobra.Command, []string)

	BeforeEach(func() {
		cmd = &cobra.Command{}
		runCalled = false
		v2Called = false
		originalEnabled = seaplaneEnabled
		originalRun = runV1
		originalDescribe = seaplaneDescribeCluster

		runV1 = func(*cobra.Command, []string) {
			runCalled = true
		}
		seaplaneDescribeCluster = func(*cobra.Command, []string) { v2Called = true }
	})

	AfterEach(func() {
		runV1 = originalRun
		seaplaneDescribeCluster = originalDescribe
		seaplaneEnabled = originalEnabled
	})

	Context("when seaplane is disabled", func() {
		BeforeEach(func() {
			seaplaneEnabled = func() bool { return false }
		})

		It("should call the v1 run function", func() {
			dispatch(cmd, []string{})
			Expect(runCalled).To(BeTrue())
			Expect(v2Called).To(BeFalse())
		})

	})

	Context("when seaplane is enabled", func() {
		BeforeEach(func() {
			seaplaneEnabled = func() bool { return true }
		})

		It("should not call the v1 run function", func() {
			dispatch(cmd, []string{})
			Expect(runCalled).To(BeFalse())
		})

		It("should call the v2 describe handler", func() {
			dispatch(cmd, []string{})
			Expect(v2Called).To(BeTrue())
			Expect(runCalled).To(BeFalse())
		})
	})
})
