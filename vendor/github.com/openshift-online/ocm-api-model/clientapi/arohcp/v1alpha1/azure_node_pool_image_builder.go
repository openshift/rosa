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

// Specifies the Azure Marketplace image to use for the Nodes of the Node Pool.
// When specified, the provided image is used instead of the default RHCOS image.
// All four fields must be provided together.
// Optional during creation. Immutable.
type AzureNodePoolImageBuilder struct {
	fieldSet_ []bool
	sku       string
	offer     string
	publisher string
	version   string
}

// NewAzureNodePoolImage creates a new builder of 'azure_node_pool_image' objects.
func NewAzureNodePoolImage() *AzureNodePoolImageBuilder {
	return &AzureNodePoolImageBuilder{
		fieldSet_: make([]bool, 4),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *AzureNodePoolImageBuilder) Empty() bool {
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

// SKU sets the value of the 'SKU' attribute to the given value.
func (b *AzureNodePoolImageBuilder) SKU(value string) *AzureNodePoolImageBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.sku = value
	b.fieldSet_[0] = true
	return b
}

// Offer sets the value of the 'offer' attribute to the given value.
func (b *AzureNodePoolImageBuilder) Offer(value string) *AzureNodePoolImageBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.offer = value
	b.fieldSet_[1] = true
	return b
}

// Publisher sets the value of the 'publisher' attribute to the given value.
func (b *AzureNodePoolImageBuilder) Publisher(value string) *AzureNodePoolImageBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.publisher = value
	b.fieldSet_[2] = true
	return b
}

// Version sets the value of the 'version' attribute to the given value.
func (b *AzureNodePoolImageBuilder) Version(value string) *AzureNodePoolImageBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.version = value
	b.fieldSet_[3] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *AzureNodePoolImageBuilder) Copy(object *AzureNodePoolImage) *AzureNodePoolImageBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.sku = object.sku
	b.offer = object.offer
	b.publisher = object.publisher
	b.version = object.version
	return b
}

// Build creates a 'azure_node_pool_image' object using the configuration stored in the builder.
func (b *AzureNodePoolImageBuilder) Build() (object *AzureNodePoolImage, err error) {
	object = new(AzureNodePoolImage)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	object.sku = b.sku
	object.offer = b.offer
	object.publisher = b.publisher
	object.version = b.version
	return
}
