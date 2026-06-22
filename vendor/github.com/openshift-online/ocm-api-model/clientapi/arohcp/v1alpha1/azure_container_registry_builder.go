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

package v1alpha1 // github.com/openshift-online/ocm-api-model/clientapi/arohcp/v1alpha1

// Azure Container Registry configuration for an ARO HCP Cluster.
// Configures how Data Plane worker nodes authenticate container image
// pulls from Azure Container Registry (ACR).
type AzureContainerRegistryBuilder struct {
	fieldSet_   []bool
	credentials *AzureContainerRegistryCredentialsBuilder
}

// NewAzureContainerRegistry creates a new builder of 'azure_container_registry' objects.
func NewAzureContainerRegistry() *AzureContainerRegistryBuilder {
	return &AzureContainerRegistryBuilder{
		fieldSet_: make([]bool, 1),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *AzureContainerRegistryBuilder) Empty() bool {
	if b == nil || len(b.fieldSet_) == 0 {
		return true
	}
	for _, set := range b.fieldSet_ {
		if set {
			return false
		}
	}
	return true
}

// Credentials sets the value of the 'credentials' attribute to the given value.
//
// Azure Container Registry credentials configuration for an ARO HCP Cluster.
// Configures authentication for container image pulls from Azure Container
// Registry (ACR) on Data Plane worker nodes.
func (b *AzureContainerRegistryBuilder) Credentials(value *AzureContainerRegistryCredentialsBuilder) *AzureContainerRegistryBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 1)
	}
	b.credentials = value
	if value != nil {
		b.fieldSet_[0] = true
	} else {
		b.fieldSet_[0] = false
	}
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *AzureContainerRegistryBuilder) Copy(object *AzureContainerRegistry) *AzureContainerRegistryBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	if object.credentials != nil {
		b.credentials = NewAzureContainerRegistryCredentials().Copy(object.credentials)
	} else {
		b.credentials = nil
	}
	return b
}

// Build creates a 'azure_container_registry' object using the configuration stored in the builder.
func (b *AzureContainerRegistryBuilder) Build() (object *AzureContainerRegistry, err error) {
	object = new(AzureContainerRegistry)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	if b.credentials != nil {
		object.credentials, err = b.credentials.Build()
		if err != nil {
			return
		}
	}
	return
}
