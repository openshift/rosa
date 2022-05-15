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

// STSOperatorBuilder contains the data and logic needed to build 'STS_operator' objects.
//
// Representation of an sts operator
type STSOperatorBuilder struct {
	bitmap_             uint32
	maxVersion          string
	minVersion          string
	name                string
	namespace           string
	serviceAccountNames []string
}

// NewSTSOperator creates a new builder of 'STS_operator' objects.
func NewSTSOperator() *STSOperatorBuilder {
	return &STSOperatorBuilder{}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *STSOperatorBuilder) Empty() bool {
	return b == nil || b.bitmap_ == 0
}

// MaxVersion sets the value of the 'max_version' attribute to the given value.
//
//
func (b *STSOperatorBuilder) MaxVersion(value string) *STSOperatorBuilder {
	b.maxVersion = value
	b.bitmap_ |= 1
	return b
}

// MinVersion sets the value of the 'min_version' attribute to the given value.
//
//
func (b *STSOperatorBuilder) MinVersion(value string) *STSOperatorBuilder {
	b.minVersion = value
	b.bitmap_ |= 2
	return b
}

// Name sets the value of the 'name' attribute to the given value.
//
//
func (b *STSOperatorBuilder) Name(value string) *STSOperatorBuilder {
	b.name = value
	b.bitmap_ |= 4
	return b
}

// Namespace sets the value of the 'namespace' attribute to the given value.
//
//
func (b *STSOperatorBuilder) Namespace(value string) *STSOperatorBuilder {
	b.namespace = value
	b.bitmap_ |= 8
	return b
}

// ServiceAccountNames sets the value of the 'service_account_names' attribute to the given values.
//
//
func (b *STSOperatorBuilder) ServiceAccountNames(values ...string) *STSOperatorBuilder {
	b.serviceAccountNames = make([]string, len(values))
	copy(b.serviceAccountNames, values)
	b.bitmap_ |= 16
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *STSOperatorBuilder) Copy(object *STSOperator) *STSOperatorBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.maxVersion = object.maxVersion
	b.minVersion = object.minVersion
	b.name = object.name
	b.namespace = object.namespace
	if object.serviceAccountNames != nil {
		b.serviceAccountNames = make([]string, len(object.serviceAccountNames))
		copy(b.serviceAccountNames, object.serviceAccountNames)
	} else {
		b.serviceAccountNames = nil
	}
	return b
}

// Build creates a 'STS_operator' object using the configuration stored in the builder.
func (b *STSOperatorBuilder) Build() (object *STSOperator, err error) {
	object = new(STSOperator)
	object.bitmap_ = b.bitmap_
	object.maxVersion = b.maxVersion
	object.minVersion = b.minVersion
	object.name = b.name
	object.namespace = b.namespace
	if b.serviceAccountNames != nil {
		object.serviceAccountNames = make([]string, len(b.serviceAccountNames))
		copy(object.serviceAccountNames, b.serviceAccountNames)
	}
	return
}
