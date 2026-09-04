package machinepool

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

func dispatch(options *mpOpts.CreateMachinepoolUserOptions) func(*cobra.Command, []string) {
	if hyperfleet.Enabled() {
		r := reporter.CreateReporter()
		r.Errorf("This command is not yet supported with the Platform API")
		os.Exit(1)
	}
	return rosa.DefaultRunner(rosa.RuntimeWithOCM(), CreateMachinepoolRunner(options))
}
