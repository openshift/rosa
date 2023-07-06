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

// ClusterStsSupportRoleListBuilder contains the data and logic needed to build
// 'cluster_sts_support_role' objects.
type ClusterStsSupportRoleListBuilder struct {
	items []*ClusterStsSupportRoleBuilder
}

// NewClusterStsSupportRoleList creates a new builder of 'cluster_sts_support_role' objects.
func NewClusterStsSupportRoleList() *ClusterStsSupportRoleListBuilder {
	return new(ClusterStsSupportRoleListBuilder)
}

// Items sets the items of the list.
func (b *ClusterStsSupportRoleListBuilder) Items(values ...*ClusterStsSupportRoleBuilder) *ClusterStsSupportRoleListBuilder {
	b.items = make([]*ClusterStsSupportRoleBuilder, len(values))
	copy(b.items, values)
	return b
}

// Empty returns true if the list is empty.
func (b *ClusterStsSupportRoleListBuilder) Empty() bool {
	return b == nil || len(b.items) == 0
}

// Copy copies the items of the given list into this builder, discarding any previous items.
func (b *ClusterStsSupportRoleListBuilder) Copy(list *ClusterStsSupportRoleList) *ClusterStsSupportRoleListBuilder {
	if list == nil || list.items == nil {
		b.items = nil
	} else {
		b.items = make([]*ClusterStsSupportRoleBuilder, len(list.items))
		for i, v := range list.items {
			b.items[i] = NewClusterStsSupportRole().Copy(v)
		}
	}
	return b
}

// Build creates a list of 'cluster_sts_support_role' objects using the
// configuration stored in the builder.
func (b *ClusterStsSupportRoleListBuilder) Build() (list *ClusterStsSupportRoleList, err error) {
	items := make([]*ClusterStsSupportRole, len(b.items))
	for i, item := range b.items {
		items[i], err = item.Build()
		if err != nil {
			return
		}
	}
	list = new(ClusterStsSupportRoleList)
	list.items = items
	return
}
