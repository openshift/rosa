package addon

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/openshift/rosa/pkg/output"
	"github.com/spf13/cobra"
)

var _ = Describe("describe addon", func() {
	Context("describe addon command", func() {
		It("returns command", func() {
			cmd := NewDescribeAddonCommand()
			Expect(cmd).NotTo(BeNil())
		})
	})

	Context("execute describe addon", func() {
		// Full diff for long string to help debugging
		format.TruncatedDiff = false

		var cmd *cobra.Command

		BeforeEach(func() {
			output.SetOutput("")
			cmd = NewDescribeAddonCommand()
		})

		AfterEach(func() {
			output.SetOutput("")
		})

		It("Fails with no addon arg", func() {
			err := cmd.Args(cmd, []string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Expected exactly " +
				"one command line argument containing the identifier of the add-on"))
		})

		It("Succeeds with addon arg", func() {
			err := cmd.Args(cmd, []string{"name"})
			Expect(err).To(BeNil())
		})
	})
})
