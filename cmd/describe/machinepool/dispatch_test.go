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
			opts := &DescribeMachinepoolUserOptions{}
			runner := dispatch(opts)
			Expect(runner).NotTo(BeNil())
		})
	})

	Context("when hyperfleet is enabled", func() {
		var origDescribe func(*DescribeMachinepoolUserOptions, []string)

		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return true }
			origDescribe = hfDescribeMachinePool
		})

		AfterEach(func() {
			hfDescribeMachinePool = origDescribe
		})

		It("should return a runner that calls the v2 handler", func() {
			called := false
			hfDescribeMachinePool = func(*DescribeMachinepoolUserOptions, []string) {
				called = true
			}
			opts := &DescribeMachinepoolUserOptions{}
			runner := dispatch(opts)
			Expect(runner).NotTo(BeNil())
			runner(&cobra.Command{}, []string{})
			Expect(called).To(BeTrue())
		})
	})
})
