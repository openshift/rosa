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

// AzureContainerRegistryBuilder contains the data and logic needed to build 'azure_container_registry' objects.
//
// Azure Container Registry configuration for an ARO HCP Cluster.
// Configures how Data Plane worker nodes authenticate container image
// pulls from Azure Container Registry (ACR).
type AzureContainerRegistryBuilder = api_v1alpha1.AzureContainerRegistryBuilder

// NewAzureContainerRegistry creates a new builder of 'azure_container_registry' objects.
var NewAzureContainerRegistry = api_v1alpha1.NewAzureContainerRegistry
