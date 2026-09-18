// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package create

import "testing"

func TestCreateIncludesSpotTerminationQueueCommand(t *testing.T) {
	t.Helper()

	for _, command := range Cmd.Commands() {
		if command.Name() == "spot-termination-queue" {
			return
		}
	}

	t.Fatalf("expected create root command to include spot-termination-queue subcommand")
}
