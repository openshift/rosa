// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package autoscaler

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

func dispatch(args *clusterautoscaler.AutoscalerArgs) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, argv []string) {
		if seaplane.Enabled() {
			rosa.DefaultRunner(rosa.RuntimeWithSeaplane(), editSeaplaneAutoscalerRunner(args))(cmd, argv)
			return
		}
		rosa.DefaultRunner(rosa.RuntimeWithOCM(), EditAutoscalerRunner(args))(cmd, argv)
	}
}
