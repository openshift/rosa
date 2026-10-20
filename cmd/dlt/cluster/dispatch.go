// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/seaplane"
)

var (
	seaplaneEnabled = seaplane.Enabled
	runV1           = run
)

func dispatch(cmd *cobra.Command, args []string) {
	if seaplaneEnabled() {
		seaplaneDeleteCluster(cmd)
		return
	}
	runV1(cmd, args)
}
