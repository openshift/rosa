package rosa

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

func TestDefaultRunner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "default command runner")
}

var _ = Describe("Runner Tests", func() {

	It("Invokes RuntimeVisitor and CommandRunner", func() {
		visited := false
		visitor := func(ctx context.Context, runtime *Runtime, command *cobra.Command, args []string) {
			visited = true
		}

		run := false
		runner := func(ctx context.Context, runtime *Runtime, command *cobra.Command, args []string) error {
			run = true
			return nil
		}

		DefaultRunner(visitor, runner)(nil, nil)

		Expect(visited).To(BeTrue())
		Expect(run).To(BeTrue())
	})

	It("Invokes Only CommandRunner if no RuntimeVisitor supplied", func() {
		run := false
		runner := func(ctx context.Context, runtime *Runtime, command *cobra.Command, args []string) error {
			run = true
			return nil
		}

		DefaultRunner(nil, runner)(nil, nil)

		Expect(run).To(BeTrue())
	})
})
