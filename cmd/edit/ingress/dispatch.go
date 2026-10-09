// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package ingress

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

func dispatch(cmd *cobra.Command, argv []string) {
	if hyperfleet.Enabled() {
		rosa.DefaultRunner(rosa.RuntimeWithHyperFleet(), editHyperfleetComponentRoutesRunner())(cmd, argv)
		return
	}
	run(cmd, argv)
}
