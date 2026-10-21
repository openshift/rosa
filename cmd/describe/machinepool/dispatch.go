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

func dispatch(options *DescribeMachinepoolUserOptions) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, argv []string) {
		if seaplaneEnabled() {
			seaplaneDescribeMachinePool(options, argv)
			return
		}
		rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeMachinePoolRunner(options))(cmd, argv)
	}
}
