// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"github.com/spf13/cobra"

	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

var (
	seaplaneEnabled = seaplane.Enabled
)

func dispatch(options *mpOpts.CreateMachinepoolUserOptions) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, argv []string) {
		if seaplaneEnabled() {
			seaplaneCreateMachinePool(options, argv, cmd)
			return
		}
		rosa.DefaultRunner(rosa.RuntimeWithOCM(), CreateMachinepoolRunner(options))(cmd, argv)
	}
}
