package cluster

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/reporter"
)

var (
	hyperfleetEnabled = hyperfleet.Enabled
	runV1             = run
	exitWithError     = func() { os.Exit(1) }
)

func dispatch(cmd *cobra.Command, args []string) {
	if hyperfleetEnabled() {
		r := reporter.CreateReporter()
		r.Errorf("This command is not yet supported with the Platform API")
		exitWithError()
		return
	}
	runV1(cmd, args)
}
