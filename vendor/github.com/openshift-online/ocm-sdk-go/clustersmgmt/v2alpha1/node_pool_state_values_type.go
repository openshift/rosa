/*
Copyright (c) 2020 Red Hat, Inc.

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

// IMPORTANT: This file has been generated automatically, refrain from modifying it manually as all
// your changes will be lost when the file is generated again.

package v2alpha1 // github.com/openshift-online/ocm-sdk-go/clustersmgmt/v2alpha1

// NodePoolStateValues represents the values of the 'node_pool_state_values' enumerated type.
type NodePoolStateValues string

const (
	// The node pool is still being created.
	NodePoolStateValuesCreating NodePoolStateValues = "creating"
	// The node pool is being uninstalled.
	NodePoolStateValuesDeleting NodePoolStateValues = "deleting"
	// Error during installation or change.
	NodePoolStateValuesError NodePoolStateValues = "error"
	// The node pool is pending resources before being provisioned.
	NodePoolStateValuesPending NodePoolStateValues = "pending"
	// The node pool is ready to use.
	NodePoolStateValuesReady NodePoolStateValues = "ready"
	// The state of the node pool is unknown.
	NodePoolStateValuesUnknown NodePoolStateValues = "unknown"
	// The state of the node pool is unknown.
	NodePoolStateValuesUpdating NodePoolStateValues = "updating"
	// The node pool is validating user input.
	NodePoolStateValuesValidating NodePoolStateValues = "validating"
)
