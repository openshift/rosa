package version

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	verify "github.com/openshift/rosa/cmd/verify/rosa"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

func TestVersionCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa version command")
}

var _ = Describe("Run Command", func() {
	var cmd *cobra.Command
	var r *rosa.Runtime

	var delegateInvokeCount int

	BeforeEach(func() {
		delegateInvokeCount = 0
		delegateCommand = func(cmd *cobra.Command, args []string) {
			delegateInvokeCount++
		}

		cmd = makeCmd()
		initFlags(cmd)

		r = rosa.NewRuntime()
		DeferCleanup(r.Cleanup)
	})

	It("It only prints version and build information", func() {
		args.clientOnly = true

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)

		Expect(err).ToNot(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(ContainSubstring("%s (Build: %s)\n", info.Version, info.Build))
		Expect(delegateInvokeCount).To(Equal(0))
	})

	It("Prints version information and invokes delegate command", func() {
		args.clientOnly = false

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)

		Expect(err).ToNot(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(ContainSubstring("%s (Build: %s)\n", info.Version, info.Build))
		Expect(delegateInvokeCount).To(Equal(1))
	})

	It("Prints verbose information and invokes delegate command", func() {
		args.clientOnly = false
		args.verbose = true

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)

		Expect(err).ToNot(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(ContainSubstring("%s (Build: %s)\n"+
			"Information and download locations:\n\t%s\n\t%s\n",
			info.Version, info.Build,
			verify.ConsoleLatestFolder,
			verify.DownloadLatestMirrorFolder,
		))
		Expect(delegateInvokeCount).To(Equal(1))
	})
})
