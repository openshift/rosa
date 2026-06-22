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

// Azure Container Registry credentials configuration for an ARO HCP Cluster.
// Configures authentication for container image pulls from Azure Container
// Registry (ACR) on Data Plane worker nodes.
type AzureContainerRegistryCredentialsBuilder struct {
	fieldSet_       []bool
	managedIdentity *AzureUserAssignedManagedIdentityBuilder
	type_           AzureContainerRegistryCredentialType
}

// NewAzureContainerRegistryCredentials creates a new builder of 'azure_container_registry_credentials' objects.
func NewAzureContainerRegistryCredentials() *AzureContainerRegistryCredentialsBuilder {
	return &AzureContainerRegistryCredentialsBuilder{
		fieldSet_: make([]bool, 2),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *AzureContainerRegistryCredentialsBuilder) Empty() bool {
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

// ManagedIdentity sets the value of the 'managed_identity' attribute to the given value.
//
// Identifies a user-assigned managed identity by its ARM resource ID.
func (b *AzureContainerRegistryCredentialsBuilder) ManagedIdentity(value *AzureUserAssignedManagedIdentityBuilder) *AzureContainerRegistryCredentialsBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 2)
	}
	b.managedIdentity = value
	if value != nil {
		b.fieldSet_[0] = true
	} else {
		b.fieldSet_[0] = false
	}
	return b
}

// Type sets the value of the 'type' attribute to the given value.
//
// The type of credential used for Azure Container Registry image pulls.
func (b *AzureContainerRegistryCredentialsBuilder) Type(value AzureContainerRegistryCredentialType) *AzureContainerRegistryCredentialsBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 2)
	}
	b.type_ = value
	b.fieldSet_[1] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *AzureContainerRegistryCredentialsBuilder) Copy(object *AzureContainerRegistryCredentials) *AzureContainerRegistryCredentialsBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	if object.managedIdentity != nil {
		b.managedIdentity = NewAzureUserAssignedManagedIdentity().Copy(object.managedIdentity)
	} else {
		b.managedIdentity = nil
	}
	b.type_ = object.type_
	return b
}

// Build creates a 'azure_container_registry_credentials' object using the configuration stored in the builder.
func (b *AzureContainerRegistryCredentialsBuilder) Build() (object *AzureContainerRegistryCredentials, err error) {
	object = new(AzureContainerRegistryCredentials)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	if b.managedIdentity != nil {
		object.managedIdentity, err = b.managedIdentity.Build()
		if err != nil {
			return
		}
	}
	object.type_ = b.type_
	return
}
