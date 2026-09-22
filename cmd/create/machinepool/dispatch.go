// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hyperfleetEnabled = hyperfleet.Enabled
)

func dispatch(options *mpOpts.CreateMachinepoolUserOptions) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, argv []string) {
		if hyperfleetEnabled() {
			hfCreateMachinePool(options, argv, cmd)
			return
		}
		rosa.DefaultRunner(rosa.RuntimeWithOCM(), CreateMachinepoolRunner(options))(cmd, argv)
	}
}
