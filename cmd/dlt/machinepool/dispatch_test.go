package machinepool

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/hyperfleet"
)

var _ = Describe("Dispatch", func() {
	var exitCalled bool
	var exitCode int

	BeforeEach(func() {
		exitCalled = false
		exitCode = 0

		// Override exit function
		exitWithError = func() {
			exitCalled = true
			exitCode = 1
		}
	})

	AfterEach(func() {
		// Reset to production defaults
		exitWithError = func() { os.Exit(1) }
		hyperfleetEnabled = hyperfleet.Enabled
	})

	Context("when hyperfleet is disabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return false }
		})

		It("should return a runner function", func() {
			opts := &DeleteMachinepoolUserOptions{}
			runner := dispatch(opts)
			Expect(runner).NotTo(BeNil())
		})

		It("should not call exit", func() {
			opts := &DeleteMachinepoolUserOptions{}
			dispatch(opts)
			Expect(exitCalled).To(BeFalse())
		})
	})

	Context("when hyperfleet is enabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return true }
		})

		It("should call exit with status 1", func() {
			opts := &DeleteMachinepoolUserOptions{}
			dispatch(opts)
			Expect(exitCalled).To(BeTrue())
			Expect(exitCode).To(Equal(1))
		})

		It("should return nil instead of a runner", func() {
			opts := &DeleteMachinepoolUserOptions{}
			runner := dispatch(opts)
			Expect(exitCalled).To(BeTrue())
			Expect(runner).To(BeNil())
		})
	})
})
