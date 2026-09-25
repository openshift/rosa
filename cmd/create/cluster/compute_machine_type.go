/*
Copyright (c) 2026 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"github.com/openshift/rosa/pkg/helper"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
)

// resolveComputeMachineTypeDefault picks the compute machine type to send on create
// when the user did not pass --compute-machine-type.
//
// Hosted control plane (HCP): leave empty so Clusters Service applies rosaHcpDefaults
// (m7i.xlarge, including regional overrides). Sending the classic osd-4 flavour default
// (m5.xlarge) would override that server-side default.
//
// Classic: keep using the flavour-provided default.
func resolveComputeMachineTypeDefault(isHostedCP bool, flavourDefault string) string {
	if isHostedCP {
		return ""
	}
	return flavourDefault
}

// interactiveComputeMachineTypeDefault picks a prompt default that must appear in
// availableIDs when possible. Prefers m7i.xlarge for HCP to match CS / machine pool defaults.
func interactiveComputeMachineTypeDefault(isHostedCP bool, flavourDefault string, availableIDs []string) string {
	if !isHostedCP {
		return flavourDefault
	}
	if helper.Contains(availableIDs, mpOpts.DefaultInstanceType) {
		return mpOpts.DefaultInstanceType
	}
	if flavourDefault != "" && helper.Contains(availableIDs, flavourDefault) {
		return flavourDefault
	}
	if len(availableIDs) > 0 {
		return availableIDs[0]
	}
	return mpOpts.DefaultInstanceType
}
