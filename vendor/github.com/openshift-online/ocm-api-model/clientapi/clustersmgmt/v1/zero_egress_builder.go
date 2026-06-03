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

// Zero egress configuration.
type ZeroEgressBuilder struct {
	fieldSet_             []bool
	noProxyDefaultDomains []string
	enabled               bool
}

// NewZeroEgress creates a new builder of 'zero_egress' objects.
func NewZeroEgress() *ZeroEgressBuilder {
	return &ZeroEgressBuilder{
		fieldSet_: make([]bool, 2),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *ZeroEgressBuilder) Empty() bool {
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

// Enabled sets the value of the 'enabled' attribute to the given value.
func (b *ZeroEgressBuilder) Enabled(value bool) *ZeroEgressBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 2)
	}
	b.enabled = value
	b.fieldSet_[0] = true
	return b
}

// NoProxyDefaultDomains sets the value of the 'no_proxy_default_domains' attribute to the given values.
func (b *ZeroEgressBuilder) NoProxyDefaultDomains(values ...string) *ZeroEgressBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 2)
	}
	b.noProxyDefaultDomains = make([]string, len(values))
	copy(b.noProxyDefaultDomains, values)
	b.fieldSet_[1] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *ZeroEgressBuilder) Copy(object *ZeroEgress) *ZeroEgressBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.enabled = object.enabled
	if object.noProxyDefaultDomains != nil {
		b.noProxyDefaultDomains = make([]string, len(object.noProxyDefaultDomains))
		copy(b.noProxyDefaultDomains, object.noProxyDefaultDomains)
	} else {
		b.noProxyDefaultDomains = nil
	}
	return b
}

// Build creates a 'zero_egress' object using the configuration stored in the builder.
func (b *ZeroEgressBuilder) Build() (object *ZeroEgress, err error) {
	object = new(ZeroEgress)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	object.enabled = b.enabled
	if b.noProxyDefaultDomains != nil {
		object.noProxyDefaultDomains = make([]string, len(b.noProxyDefaultDomains))
		copy(object.noProxyDefaultDomains, b.noProxyDefaultDomains)
	}
	return
}
