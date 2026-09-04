package cluster

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/reporter"
)

func dispatch(cmd *cobra.Command, args []string) {
	if hyperfleet.Enabled() {
		r := reporter.CreateReporter()
		r.Errorf("This command is not yet supported with the Platform API")
		os.Exit(1)
	}
	run(cmd, args)
}
