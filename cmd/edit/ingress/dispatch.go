// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package ingress

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

func dispatch(cmd *cobra.Command, argv []string) {
	if seaplane.Enabled() {
		rosa.DefaultRunner(rosa.RuntimeWithSeaplane(), editSeaplaneComponentRoutesRunner())(cmd, argv)
		return
	}
	run(cmd, argv)
}
