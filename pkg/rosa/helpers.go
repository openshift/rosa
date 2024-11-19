package rosa

import (
	"os"

	"github.com/spf13/cobra"
)

func HostedClusterOnlyFlag(r *Runtime, cmd *cobra.Command, flagName string) {
	isFlagSet := cmd.Flags().Changed(flagName)
	if isFlagSet {
		r.Reporter.Errorf("Setting the `%s` flag is only supported for hosted clusters", flagName)
		os.Exit(1)
	}
}
