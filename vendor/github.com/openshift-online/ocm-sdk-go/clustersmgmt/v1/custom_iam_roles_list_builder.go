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

package v1 // github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1

// CustomIAMRolesListBuilder contains the data and logic needed to build
// 'custom_IAM_roles' objects.
type CustomIAMRolesListBuilder struct {
	items []*CustomIAMRolesBuilder
}

// NewCustomIAMRolesList creates a new builder of 'custom_IAM_roles' objects.
func NewCustomIAMRolesList() *CustomIAMRolesListBuilder {
	return new(CustomIAMRolesListBuilder)
}

// Items sets the items of the list.
func (b *CustomIAMRolesListBuilder) Items(values ...*CustomIAMRolesBuilder) *CustomIAMRolesListBuilder {
	b.items = make([]*CustomIAMRolesBuilder, len(values))
	copy(b.items, values)
	return b
}

// Copy copies the items of the given list into this builder, discarding any previous items.
func (b *CustomIAMRolesListBuilder) Copy(list *CustomIAMRolesList) *CustomIAMRolesListBuilder {
	if list == nil || list.items == nil {
		b.items = nil
	} else {
		b.items = make([]*CustomIAMRolesBuilder, len(list.items))
		for i, v := range list.items {
			b.items[i] = NewCustomIAMRoles().Copy(v)
		}
	}
	return b
}

// Build creates a list of 'custom_IAM_roles' objects using the
// configuration stored in the builder.
func (b *CustomIAMRolesListBuilder) Build() (list *CustomIAMRolesList, err error) {
	items := make([]*CustomIAMRoles, len(b.items))
	for i, item := range b.items {
		items[i], err = item.Build()
		if err != nil {
			return
		}
	}
	list = new(CustomIAMRolesList)
	list.items = items
	return
}
