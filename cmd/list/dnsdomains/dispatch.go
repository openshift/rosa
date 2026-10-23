// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package dnsdomains

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/seaplane"
)

var (
	seaplaneEnabled = seaplane.Enabled
	runPlatformAPI  = runV2
	runOCM          = run
)

func dispatch(cmd *cobra.Command, argv []string) {
	if seaplaneEnabled() {
		runPlatformAPI(cmd, argv)
		return
	}
	runOCM(cmd, argv)
}
