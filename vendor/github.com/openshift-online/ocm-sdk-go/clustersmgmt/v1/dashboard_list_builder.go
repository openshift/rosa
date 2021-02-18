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

// DashboardListBuilder contains the data and logic needed to build
// 'dashboard' objects.
type DashboardListBuilder struct {
	items []*DashboardBuilder
}

// NewDashboardList creates a new builder of 'dashboard' objects.
func NewDashboardList() *DashboardListBuilder {
	return new(DashboardListBuilder)
}

// Items sets the items of the list.
func (b *DashboardListBuilder) Items(values ...*DashboardBuilder) *DashboardListBuilder {
	b.items = make([]*DashboardBuilder, len(values))
	copy(b.items, values)
	return b
}

// Copy copies the items of the given list into this builder, discarding any previous items.
func (b *DashboardListBuilder) Copy(list *DashboardList) *DashboardListBuilder {
	if list == nil || list.items == nil {
		b.items = nil
	} else {
		b.items = make([]*DashboardBuilder, len(list.items))
		for i, v := range list.items {
			b.items[i] = NewDashboard().Copy(v)
		}
	}
	return b
}

// Build creates a list of 'dashboard' objects using the
// configuration stored in the builder.
func (b *DashboardListBuilder) Build() (list *DashboardList, err error) {
	items := make([]*Dashboard, len(b.items))
	for i, item := range b.items {
		items[i], err = item.Build()
		if err != nil {
			return
		}
	}
	list = new(DashboardList)
	list.items = items
	return
}
