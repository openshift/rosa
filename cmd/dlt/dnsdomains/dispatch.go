package dnsdomains

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
)

var (
	hyperfleetEnabled = hyperfleet.Enabled
	runPlatformAPI    = runV2
	runOCM            = run
)

func dispatch(cmd *cobra.Command, argv []string) {
	if hyperfleetEnabled() {
		runPlatformAPI(cmd, argv)
		return
	}
	runOCM(cmd, argv)
}
