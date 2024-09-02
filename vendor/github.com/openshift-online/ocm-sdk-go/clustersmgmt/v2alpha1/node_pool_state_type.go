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

// NodePoolStateKind is the name of the type used to represent objects
// of type 'node_pool_state'.
const NodePoolStateKind = "NodePoolState"

// NodePoolStateLinkKind is the name of the type used to represent links
// to objects of type 'node_pool_state'.
const NodePoolStateLinkKind = "NodePoolStateLink"

// NodePoolStateNilKind is the name of the type used to nil references
// to objects of type 'node_pool_state'.
const NodePoolStateNilKind = "NodePoolStateNil"

// NodePoolState represents the values of the 'node_pool_state' type.
//
// Representation of the status of a node pool.
type NodePoolState struct {
	bitmap_ uint32
	id      string
	href    string
	details string
	value   NodePoolStateValues
}

// Kind returns the name of the type of the object.
func (o *NodePoolState) Kind() string {
	if o == nil {
		return NodePoolStateNilKind
	}
	if o.bitmap_&1 != 0 {
		return NodePoolStateLinkKind
	}
	return NodePoolStateKind
}

// Link returns true iif this is a link.
func (o *NodePoolState) Link() bool {
	return o != nil && o.bitmap_&1 != 0
}

// ID returns the identifier of the object.
func (o *NodePoolState) ID() string {
	if o != nil && o.bitmap_&2 != 0 {
		return o.id
	}
	return ""
}

// GetID returns the identifier of the object and a flag indicating if the
// identifier has a value.
func (o *NodePoolState) GetID() (value string, ok bool) {
	ok = o != nil && o.bitmap_&2 != 0
	if ok {
		value = o.id
	}
	return
}

// HREF returns the link to the object.
func (o *NodePoolState) HREF() string {
	if o != nil && o.bitmap_&4 != 0 {
		return o.href
	}
	return ""
}

// GetHREF returns the link of the object and a flag indicating if the
// link has a value.
func (o *NodePoolState) GetHREF() (value string, ok bool) {
	ok = o != nil && o.bitmap_&4 != 0
	if ok {
		value = o.href
	}
	return
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *NodePoolState) Empty() bool {
	return o == nil || o.bitmap_&^1 == 0
}

// Details returns the value of the 'details' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Detailed user friendly status for node pool state.
func (o *NodePoolState) Details() string {
	if o != nil && o.bitmap_&8 != 0 {
		return o.details
	}
	return ""
}

// GetDetails returns the value of the 'details' attribute and
// a flag indicating if the attribute has a value.
//
// Detailed user friendly status for node pool state.
func (o *NodePoolState) GetDetails() (value string, ok bool) {
	ok = o != nil && o.bitmap_&8 != 0
	if ok {
		value = o.details
	}
	return
}

// Value returns the value of the 'value' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// The current state of the node pool.
func (o *NodePoolState) Value() NodePoolStateValues {
	if o != nil && o.bitmap_&16 != 0 {
		return o.value
	}
	return NodePoolStateValues("")
}

// GetValue returns the value of the 'value' attribute and
// a flag indicating if the attribute has a value.
//
// The current state of the node pool.
func (o *NodePoolState) GetValue() (value NodePoolStateValues, ok bool) {
	ok = o != nil && o.bitmap_&16 != 0
	if ok {
		value = o.value
	}
	return
}

// NodePoolStateListKind is the name of the type used to represent list of objects of
// type 'node_pool_state'.
const NodePoolStateListKind = "NodePoolStateList"

// NodePoolStateListLinkKind is the name of the type used to represent links to list
// of objects of type 'node_pool_state'.
const NodePoolStateListLinkKind = "NodePoolStateListLink"

// NodePoolStateNilKind is the name of the type used to nil lists of objects of
// type 'node_pool_state'.
const NodePoolStateListNilKind = "NodePoolStateListNil"

// NodePoolStateList is a list of values of the 'node_pool_state' type.
type NodePoolStateList struct {
	href  string
	link  bool
	items []*NodePoolState
}

// Kind returns the name of the type of the object.
func (l *NodePoolStateList) Kind() string {
	if l == nil {
		return NodePoolStateListNilKind
	}
	if l.link {
		return NodePoolStateListLinkKind
	}
	return NodePoolStateListKind
}

// Link returns true iif this is a link.
func (l *NodePoolStateList) Link() bool {
	return l != nil && l.link
}

// HREF returns the link to the list.
func (l *NodePoolStateList) HREF() string {
	if l != nil {
		return l.href
	}
	return ""
}

// GetHREF returns the link of the list and a flag indicating if the
// link has a value.
func (l *NodePoolStateList) GetHREF() (value string, ok bool) {
	ok = l != nil && l.href != ""
	if ok {
		value = l.href
	}
	return
}

// Len returns the length of the list.
func (l *NodePoolStateList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Empty returns true if the list is empty.
func (l *NodePoolStateList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *NodePoolStateList) Get(i int) *NodePoolState {
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
func (l *NodePoolStateList) Slice() []*NodePoolState {
	var slice []*NodePoolState
	if l == nil {
		slice = make([]*NodePoolState, 0)
	} else {
		slice = make([]*NodePoolState, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *NodePoolStateList) Each(f func(item *NodePoolState) bool) {
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
func (l *NodePoolStateList) Range(f func(index int, item *NodePoolState) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
