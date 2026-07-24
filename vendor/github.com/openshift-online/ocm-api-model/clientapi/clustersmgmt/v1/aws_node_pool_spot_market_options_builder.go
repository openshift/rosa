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

package v1 // github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1

// Spot market options for AWS node pool.
type AwsNodePoolSpotMarketOptionsBuilder struct {
	fieldSet_ []bool
	id        string
	href      string
	maxPrice  string
}

// NewAwsNodePoolSpotMarketOptions creates a new builder of 'aws_node_pool_spot_market_options' objects.
func NewAwsNodePoolSpotMarketOptions() *AwsNodePoolSpotMarketOptionsBuilder {
	return &AwsNodePoolSpotMarketOptionsBuilder{
		fieldSet_: make([]bool, 4),
	}
}

// Link sets the flag that indicates if this is a link.
func (b *AwsNodePoolSpotMarketOptionsBuilder) Link(value bool) *AwsNodePoolSpotMarketOptionsBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.fieldSet_[0] = true
	return b
}

// ID sets the identifier of the object.
func (b *AwsNodePoolSpotMarketOptionsBuilder) ID(value string) *AwsNodePoolSpotMarketOptionsBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.id = value
	b.fieldSet_[1] = true
	return b
}

// HREF sets the link to the object.
func (b *AwsNodePoolSpotMarketOptionsBuilder) HREF(value string) *AwsNodePoolSpotMarketOptionsBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.href = value
	b.fieldSet_[2] = true
	return b
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *AwsNodePoolSpotMarketOptionsBuilder) Empty() bool {
	if b == nil || len(b.fieldSet_) == 0 {
		return true
	}
	// Check all fields except the link flag (index 0)
	for i := 1; i < len(b.fieldSet_); i++ {
		if b.fieldSet_[i] {
			return false
		}
	}
	return true
}

// MaxPrice sets the value of the 'max_price' attribute to the given value.
func (b *AwsNodePoolSpotMarketOptionsBuilder) MaxPrice(value string) *AwsNodePoolSpotMarketOptionsBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.maxPrice = value
	b.fieldSet_[3] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *AwsNodePoolSpotMarketOptionsBuilder) Copy(object *AwsNodePoolSpotMarketOptions) *AwsNodePoolSpotMarketOptionsBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.id = object.id
	b.href = object.href
	b.maxPrice = object.maxPrice
	return b
}

// Build creates a 'aws_node_pool_spot_market_options' object using the configuration stored in the builder.
func (b *AwsNodePoolSpotMarketOptionsBuilder) Build() (object *AwsNodePoolSpotMarketOptions, err error) {
	object = new(AwsNodePoolSpotMarketOptions)
	object.id = b.id
	object.href = b.href
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	object.maxPrice = b.maxPrice
	return
}
