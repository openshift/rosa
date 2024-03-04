package rosa

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

// CommandRun defines a function that should be returned to implement the actual action of the command
type CommandRun func(ctx context.Context, r *Runtime, command *cobra.Command, args []string) error

// RuntimeVisitor defines a function that can visit the Runtime and configure it as required. Default implementation
// of this would exist to facilitate the most common scenarios e.g with OCM only
type RuntimeVisitor func(ctx context.Context, r *Runtime, command *cobra.Command, args []string) error

// This is the default runnable function for all commands. It takes care of instantiating the runtime and context
// and then invokes the runtime visitor and finally, the commandrun.
func DefaultRosaCommandRun(visitor RuntimeVisitor, commandRun CommandRun) func(command *cobra.Command, args []string) {
	return func(command *cobra.Command, args []string) {
		ctx := context.Background()
		runtime := NewRuntime()
		defer runtime.Cleanup()

		err := visitor(ctx, runtime, command, args)
		if err != nil {
			runtime.Reporter.Errorf(err.Error())
			os.Exit(1)
		}

		err = commandRun(ctx, runtime, command, args)
		if err != nil {
			runtime.Reporter.Errorf(err.Error())
			os.Exit(1)
		}
	}
}
