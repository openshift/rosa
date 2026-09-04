package cluster

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
)

var _ = Describe("Dispatch", func() {
	var cmd *cobra.Command
	var runCalled bool
	var exitCalled bool
	var exitCode int

	BeforeEach(func() {
		cmd = &cobra.Command{}
		runCalled = false
		exitCalled = false
		exitCode = 0

		// Override v1 runner to avoid actual execution
		runV1 = func(*cobra.Command, []string) {
			runCalled = true
		}

		// Override exit function
		exitWithError = func() {
			exitCalled = true
			exitCode = 1
		}
	})

	AfterEach(func() {
		// Reset to production defaults
		runV1 = run
		exitWithError = func() { os.Exit(1) }
		hyperfleetEnabled = hyperfleet.Enabled
	})

	Context("when hyperfleet is disabled", func() {
		BeforeEach(func() {
			hyperfleetEnabled = func() bool { return false }
		})

		It("should call the v1 run function", func() {
			dispatch(cmd, []string{})
			Expect(runCalled).To(BeTrue())
		})

		It("should not call exit", func() {
			dispatch(cmd, []string{})
			Expect(exitCalled).To(BeFalse())
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

		It("should call exit with status 1", func() {
			dispatch(cmd, []string{})
			Expect(exitCalled).To(BeTrue())
			Expect(exitCode).To(Equal(1))
		})
	})
})
