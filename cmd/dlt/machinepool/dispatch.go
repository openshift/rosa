package machinepool

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hyperfleetEnabled = hyperfleet.Enabled
	exitWithError     = func() { os.Exit(1) }
)

func dispatch(options *DeleteMachinepoolUserOptions) func(*cobra.Command, []string) {
	if hyperfleetEnabled() {
		r := reporter.CreateReporter()
		r.Errorf("This command is not yet supported with the Platform API")
		exitWithError()
		return nil
	}
	return rosa.DefaultRunner(rosa.RuntimeWithOCM(), DeleteMachinePoolRunner(options))
}
