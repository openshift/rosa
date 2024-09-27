package Network

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("BuildMachinePoolCreateCommandWithOptions", func() {
	var (
		cmd *cobra.Command
	)

	BeforeEach(func() {
		cmd, _ = BuildNetworkCommandWithOptions()
	})

	It("should create a command with the expected use, short, long, and example descriptions", func() {
		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Example).To(Equal(example))
	})
})
