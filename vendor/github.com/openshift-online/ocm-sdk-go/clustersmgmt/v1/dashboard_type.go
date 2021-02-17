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

// DashboardKind is the name of the type used to represent objects
// of type 'dashboard'.
const DashboardKind = "Dashboard"

// DashboardLinkKind is the name of the type used to represent links
// to objects of type 'dashboard'.
const DashboardLinkKind = "DashboardLink"

// DashboardNilKind is the name of the type used to nil references
// to objects of type 'dashboard'.
const DashboardNilKind = "DashboardNil"

// Dashboard represents the values of the 'dashboard' type.
//
// Collection of metrics intended to render a graphical dashboard.
type Dashboard struct {
	bitmap_ uint32
	id      string
	href    string
	metrics []*Metric
	name    string
}

// Kind returns the name of the type of the object.
func (o *Dashboard) Kind() string {
	if o == nil {
		return DashboardNilKind
	}
	if o.bitmap_&1 != 0 {
		return DashboardLinkKind
	}
	return DashboardKind
}

// Link returns true iif this is a link.
func (o *Dashboard) Link() bool {
	return o != nil && o.bitmap_&1 != 0
}

// ID returns the identifier of the object.
func (o *Dashboard) ID() string {
	if o != nil && o.bitmap_&2 != 0 {
		return o.id
	}
	return ""
}

// GetID returns the identifier of the object and a flag indicating if the
// identifier has a value.
func (o *Dashboard) GetID() (value string, ok bool) {
	ok = o != nil && o.bitmap_&2 != 0
	if ok {
		value = o.id
	}
	return
}

// HREF returns the link to the object.
func (o *Dashboard) HREF() string {
	if o != nil && o.bitmap_&4 != 0 {
		return o.href
	}
	return ""
}

// GetHREF returns the link of the object and a flag indicating if the
// link has a value.
func (o *Dashboard) GetHREF() (value string, ok bool) {
	ok = o != nil && o.bitmap_&4 != 0
	if ok {
		value = o.href
	}
	return
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *Dashboard) Empty() bool {
	return o == nil || o.bitmap_&^1 == 0
}

// Metrics returns the value of the 'metrics' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Metrics included in the dashboard.
func (o *Dashboard) Metrics() []*Metric {
	if o != nil && o.bitmap_&8 != 0 {
		return o.metrics
	}
	return nil
}

// GetMetrics returns the value of the 'metrics' attribute and
// a flag indicating if the attribute has a value.
//
// Metrics included in the dashboard.
func (o *Dashboard) GetMetrics() (value []*Metric, ok bool) {
	ok = o != nil && o.bitmap_&8 != 0
	if ok {
		value = o.metrics
	}
	return
}

// Name returns the value of the 'name' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Name of the dashboard.
func (o *Dashboard) Name() string {
	if o != nil && o.bitmap_&16 != 0 {
		return o.name
	}
	return ""
}

// GetName returns the value of the 'name' attribute and
// a flag indicating if the attribute has a value.
//
// Name of the dashboard.
func (o *Dashboard) GetName() (value string, ok bool) {
	ok = o != nil && o.bitmap_&16 != 0
	if ok {
		value = o.name
	}
	return
}

// DashboardListKind is the name of the type used to represent list of objects of
// type 'dashboard'.
const DashboardListKind = "DashboardList"

// DashboardListLinkKind is the name of the type used to represent links to list
// of objects of type 'dashboard'.
const DashboardListLinkKind = "DashboardListLink"

// DashboardNilKind is the name of the type used to nil lists of objects of
// type 'dashboard'.
const DashboardListNilKind = "DashboardListNil"

// DashboardList is a list of values of the 'dashboard' type.
type DashboardList struct {
	href  string
	link  bool
	items []*Dashboard
}

// Kind returns the name of the type of the object.
func (l *DashboardList) Kind() string {
	if l == nil {
		return DashboardListNilKind
	}
	if l.link {
		return DashboardListLinkKind
	}
	return DashboardListKind
}

// Link returns true iif this is a link.
func (l *DashboardList) Link() bool {
	return l != nil && l.link
}

// HREF returns the link to the list.
func (l *DashboardList) HREF() string {
	if l != nil {
		return l.href
	}
	return ""
}

// GetHREF returns the link of the list and a flag indicating if the
// link has a value.
func (l *DashboardList) GetHREF() (value string, ok bool) {
	ok = l != nil && l.href != ""
	if ok {
		value = l.href
	}
	return
}

// Len returns the length of the list.
func (l *DashboardList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Empty returns true if the list is empty.
func (l *DashboardList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *DashboardList) Get(i int) *Dashboard {
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
func (l *DashboardList) Slice() []*Dashboard {
	var slice []*Dashboard
	if l == nil {
		slice = make([]*Dashboard, 0)
	} else {
		slice = make([]*Dashboard, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *DashboardList) Each(f func(item *Dashboard) bool) {
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
func (l *DashboardList) Range(f func(index int, item *Dashboard) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
