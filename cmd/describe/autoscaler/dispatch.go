// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package autoscaler

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

func dispatch() func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, argv []string) {
		if seaplane.Enabled() {
			rosa.DefaultRunner(rosa.RuntimeWithSeaplane(), describeSeaplaneAutoscalerRunner())(cmd, argv)
			return
		}
		rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeAutoscalerRunner())(cmd, argv)
	}
}
