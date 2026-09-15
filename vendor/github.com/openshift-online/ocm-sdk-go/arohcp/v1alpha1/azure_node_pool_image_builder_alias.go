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

package v1alpha1 // github.com/openshift-online/ocm-sdk-go/arohcp/v1alpha1

import (
	api_v1alpha1 "github.com/openshift-online/ocm-api-model/clientapi/arohcp/v1alpha1"
)

// AzureNodePoolImageBuilder contains the data and logic needed to build 'azure_node_pool_image' objects.
//
// Specifies the Azure Marketplace image to use for the Nodes of the Node Pool.
// When specified, the provided image is used instead of the default RHCOS image.
// All four fields must be provided together.
// Optional during creation. Immutable.
type AzureNodePoolImageBuilder = api_v1alpha1.AzureNodePoolImageBuilder

// NewAzureNodePoolImage creates a new builder of 'azure_node_pool_image' objects.
var NewAzureNodePoolImage = api_v1alpha1.NewAzureNodePoolImage
