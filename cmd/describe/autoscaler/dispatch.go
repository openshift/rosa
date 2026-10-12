// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package autoscaler

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

func dispatch() func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, argv []string) {
		if hyperfleet.Enabled() {
			rosa.DefaultRunner(rosa.RuntimeWithHyperFleet(), describeHyperfleetAutoscalerRunner())(cmd, argv)
			return
		}
		rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeAutoscalerRunner())(cmd, argv)
	}
}
