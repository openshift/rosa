package version

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/info"
)

func TestVersionCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa version command")
}

var _ = Describe("Run Command", func() {

	var delegateInvokeCount int
	var buf *bytes.Buffer

	BeforeEach(func() {
		buf = new(bytes.Buffer)
		writer = buf
		delegateInvokeCount = 0
		delegateCommand = func(cmd *cobra.Command, args []string) {
			delegateInvokeCount++
		}
	})

	When("Run in client-only mode", func() {
		It("It only prints version and build information", func() {
			args.clientOnly = true

			Cmd.Execute()

			Expect(buf.String()).To(Equal(fmt.Sprintf("%s (Build: %s)\n", info.Version, info.Build)))
			Expect(delegateInvokeCount).To(Equal(0))
		})
	})

	When("Run without client-only", func() {

		It("Prints version information and invokes delegate command", func() {
			args.clientOnly = false
			Cmd.Execute()

			Expect(buf.String()).To(Equal(fmt.Sprintf("%s (Build: %s)\n", info.Version, info.Build)))
			Expect(delegateInvokeCount).To(Equal(1))
		})
	})
})
