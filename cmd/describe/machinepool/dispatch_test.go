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
	var originalDescribe func(*DescribeMachinepoolUserOptions, []string)

	BeforeEach(func() {
		originalEnabled = hyperfleetEnabled
		originalDescribe = hfDescribeMachinePool
	})

	AfterEach(func() {
		hyperfleetEnabled = originalEnabled
		hfDescribeMachinePool = originalDescribe
	})

	Context("when hyperfleet is disabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return false }
		})

		It("should return a runner function", func() {
			opts := &DescribeMachinepoolUserOptions{}
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
			hfDescribeMachinePool = func(*DescribeMachinepoolUserOptions, []string) { called = true }
			opts := &DescribeMachinepoolUserOptions{}
			dispatch(opts)(&cobra.Command{}, []string{"worker"})
			Expect(called).To(BeTrue())
		})
	})
})
