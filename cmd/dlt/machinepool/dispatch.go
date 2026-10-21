// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

var (
	seaplaneEnabled = seaplane.Enabled
)

func dispatch(options *DeleteMachinepoolUserOptions) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, argv []string) {
		if seaplaneEnabled() {
			seaplaneDeleteMachinePool(cmd, options, argv)
			return
		}
		rosa.DefaultRunner(rosa.RuntimeWithOCM(), DeleteMachinePoolRunner(options))(cmd, argv)
	}
}
