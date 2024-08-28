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

// NodePoolStateBuilder contains the data and logic needed to build 'node_pool_state' objects.
//
// Representation of the status of a node pool.
type NodePoolStateBuilder struct {
	bitmap_ uint32
	id      string
	href    string
	details string
	value   NodePoolStateValues
}

// NewNodePoolState creates a new builder of 'node_pool_state' objects.
func NewNodePoolState() *NodePoolStateBuilder {
	return &NodePoolStateBuilder{}
}

// Link sets the flag that indicates if this is a link.
func (b *NodePoolStateBuilder) Link(value bool) *NodePoolStateBuilder {
	b.bitmap_ |= 1
	return b
}

// ID sets the identifier of the object.
func (b *NodePoolStateBuilder) ID(value string) *NodePoolStateBuilder {
	b.id = value
	b.bitmap_ |= 2
	return b
}

// HREF sets the link to the object.
func (b *NodePoolStateBuilder) HREF(value string) *NodePoolStateBuilder {
	b.href = value
	b.bitmap_ |= 4
	return b
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *NodePoolStateBuilder) Empty() bool {
	return b == nil || b.bitmap_&^1 == 0
}

// Details sets the value of the 'details' attribute to the given value.
func (b *NodePoolStateBuilder) Details(value string) *NodePoolStateBuilder {
	b.details = value
	b.bitmap_ |= 8
	return b
}

// Value sets the value of the 'value' attribute to the given value.
//
// Overall state of a node pool.
func (b *NodePoolStateBuilder) Value(value NodePoolStateValues) *NodePoolStateBuilder {
	b.value = value
	b.bitmap_ |= 16
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *NodePoolStateBuilder) Copy(object *NodePoolState) *NodePoolStateBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.id = object.id
	b.href = object.href
	b.details = object.details
	b.value = object.value
	return b
}

// Build creates a 'node_pool_state' object using the configuration stored in the builder.
func (b *NodePoolStateBuilder) Build() (object *NodePoolState, err error) {
	object = new(NodePoolState)
	object.id = b.id
	object.href = b.href
	object.bitmap_ = b.bitmap_
	object.details = b.details
	object.value = b.value
	return
}
