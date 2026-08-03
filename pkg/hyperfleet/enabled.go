// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package hyperfleet

// Enabled returns true when the CLI should dispatch to the Platform API (v2) path.
func Enabled() bool {
	return hyperfleetURL != ""
}
