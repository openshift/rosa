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

// AzureContainerRegistryCredentials represents the values of the 'azure_container_registry_credentials' type.
//
// Azure Container Registry credentials configuration for an ARO HCP Cluster.
// Configures authentication for container image pulls from Azure Container
// Registry (ACR) on Data Plane worker nodes.
type AzureContainerRegistryCredentials struct {
	fieldSet_       []bool
	managedIdentity *AzureUserAssignedManagedIdentity
	type_           AzureContainerRegistryCredentialType
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *AzureContainerRegistryCredentials) Empty() bool {
	if o == nil || len(o.fieldSet_) == 0 {
		return true
	}
	for _, set := range o.fieldSet_ {
		if set {
			return false
		}
	}
	return true
}

// ManagedIdentity returns the value of the 'managed_identity' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// The user-assigned managed identity used for ACR image pulls
// on Data Plane worker nodes.
// The managed identity must be in the same Microsoft Entra tenant
// as the cluster's Azure Subscription.
// The managed identity can be in any Azure Subscription or location
// within the tenant.
// The Azure Resource Group Name specified as part of the Resource ID
// must be a different Resource Group Name than the one specified in
// `.azure.managed_resource_group_name`.
// Required when type is ManagedIdentity.
func (o *AzureContainerRegistryCredentials) ManagedIdentity() *AzureUserAssignedManagedIdentity {
	if o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0] {
		return o.managedIdentity
	}
	return nil
}

// GetManagedIdentity returns the value of the 'managed_identity' attribute and
// a flag indicating if the attribute has a value.
//
// The user-assigned managed identity used for ACR image pulls
// on Data Plane worker nodes.
// The managed identity must be in the same Microsoft Entra tenant
// as the cluster's Azure Subscription.
// The managed identity can be in any Azure Subscription or location
// within the tenant.
// The Azure Resource Group Name specified as part of the Resource ID
// must be a different Resource Group Name than the one specified in
// `.azure.managed_resource_group_name`.
// Required when type is ManagedIdentity.
func (o *AzureContainerRegistryCredentials) GetManagedIdentity() (value *AzureUserAssignedManagedIdentity, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0]
	if ok {
		value = o.managedIdentity
	}
	return
}

// Type returns the value of the 'type' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// The credential type used for ACR image pulls on Data Plane worker nodes.
// Required.
func (o *AzureContainerRegistryCredentials) Type() AzureContainerRegistryCredentialType {
	if o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1] {
		return o.type_
	}
	return AzureContainerRegistryCredentialType("")
}

// GetType returns the value of the 'type' attribute and
// a flag indicating if the attribute has a value.
//
// The credential type used for ACR image pulls on Data Plane worker nodes.
// Required.
func (o *AzureContainerRegistryCredentials) GetType() (value AzureContainerRegistryCredentialType, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1]
	if ok {
		value = o.type_
	}
	return
}

// AzureContainerRegistryCredentialsListKind is the name of the type used to represent list of objects of
// type 'azure_container_registry_credentials'.
const AzureContainerRegistryCredentialsListKind = "AzureContainerRegistryCredentialsList"

// AzureContainerRegistryCredentialsListLinkKind is the name of the type used to represent links to list
// of objects of type 'azure_container_registry_credentials'.
const AzureContainerRegistryCredentialsListLinkKind = "AzureContainerRegistryCredentialsListLink"

// AzureContainerRegistryCredentialsNilKind is the name of the type used to nil lists of objects of
// type 'azure_container_registry_credentials'.
const AzureContainerRegistryCredentialsListNilKind = "AzureContainerRegistryCredentialsListNil"

// AzureContainerRegistryCredentialsList is a list of values of the 'azure_container_registry_credentials' type.
type AzureContainerRegistryCredentialsList struct {
	href  string
	link  bool
	items []*AzureContainerRegistryCredentials
}

// Len returns the length of the list.
func (l *AzureContainerRegistryCredentialsList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Items sets the items of the list.
func (l *AzureContainerRegistryCredentialsList) SetLink(link bool) {
	l.link = link
}

// Items sets the items of the list.
func (l *AzureContainerRegistryCredentialsList) SetHREF(href string) {
	l.href = href
}

// Items sets the items of the list.
func (l *AzureContainerRegistryCredentialsList) SetItems(items []*AzureContainerRegistryCredentials) {
	l.items = items
}

// Items returns the items of the list.
func (l *AzureContainerRegistryCredentialsList) Items() []*AzureContainerRegistryCredentials {
	if l == nil {
		return nil
	}
	return l.items
}

// Empty returns true if the list is empty.
func (l *AzureContainerRegistryCredentialsList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *AzureContainerRegistryCredentialsList) Get(i int) *AzureContainerRegistryCredentials {
	if l == nil || i < 0 || i >= len(l.items) {
		return nil
	}
	return l.items[i]
}

// Slice returns an slice containing the items of the list. The returned slice is a
// copy of the one used internally, so it can be modified without affecting the
// internal representation.
//
// If you don't need to modify the returned slice consider using the Each or Range
// functions, as they don't need to allocate a new slice.
func (l *AzureContainerRegistryCredentialsList) Slice() []*AzureContainerRegistryCredentials {
	var slice []*AzureContainerRegistryCredentials
	if l == nil {
		slice = make([]*AzureContainerRegistryCredentials, 0)
	} else {
		slice = make([]*AzureContainerRegistryCredentials, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *AzureContainerRegistryCredentialsList) Each(f func(item *AzureContainerRegistryCredentials) bool) {
	if l == nil {
		return
	}
	for _, item := range l.items {
		if !f(item) {
			break
		}
	}
}

// Range runs the given function for each index and item of the list, in order. If
// the function returns false the iteration stops, otherwise it continues till all
// the elements of the list have been processed.
func (l *AzureContainerRegistryCredentialsList) Range(f func(index int, item *AzureContainerRegistryCredentials) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
