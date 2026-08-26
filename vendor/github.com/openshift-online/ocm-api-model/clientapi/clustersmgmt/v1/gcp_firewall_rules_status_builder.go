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

// Status of GCP firewall rules verification.
type GcpFirewallRulesStatusBuilder struct {
	fieldSet_   []bool
	description string
	kind        string
	rules       []*GcpFirewallRulesStatusEntryBuilder
	state       string
}

// NewGcpFirewallRulesStatus creates a new builder of 'gcp_firewall_rules_status' objects.
func NewGcpFirewallRulesStatus() *GcpFirewallRulesStatusBuilder {
	return &GcpFirewallRulesStatusBuilder{
		fieldSet_: make([]bool, 4),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *GcpFirewallRulesStatusBuilder) Empty() bool {
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

// Description sets the value of the 'description' attribute to the given value.
func (b *GcpFirewallRulesStatusBuilder) Description(value string) *GcpFirewallRulesStatusBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.description = value
	b.fieldSet_[0] = true
	return b
}

// Kind sets the value of the 'kind' attribute to the given value.
func (b *GcpFirewallRulesStatusBuilder) Kind(value string) *GcpFirewallRulesStatusBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.kind = value
	b.fieldSet_[1] = true
	return b
}

// Rules sets the value of the 'rules' attribute to the given values.
func (b *GcpFirewallRulesStatusBuilder) Rules(values ...*GcpFirewallRulesStatusEntryBuilder) *GcpFirewallRulesStatusBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.rules = make([]*GcpFirewallRulesStatusEntryBuilder, len(values))
	copy(b.rules, values)
	b.fieldSet_[2] = true
	return b
}

// State sets the value of the 'state' attribute to the given value.
func (b *GcpFirewallRulesStatusBuilder) State(value string) *GcpFirewallRulesStatusBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.state = value
	b.fieldSet_[3] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *GcpFirewallRulesStatusBuilder) Copy(object *GcpFirewallRulesStatus) *GcpFirewallRulesStatusBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.description = object.description
	b.kind = object.kind
	if object.rules != nil {
		b.rules = make([]*GcpFirewallRulesStatusEntryBuilder, len(object.rules))
		for i, v := range object.rules {
			b.rules[i] = NewGcpFirewallRulesStatusEntry().Copy(v)
		}
	} else {
		b.rules = nil
	}
	b.state = object.state
	return b
}

// Build creates a 'gcp_firewall_rules_status' object using the configuration stored in the builder.
func (b *GcpFirewallRulesStatusBuilder) Build() (object *GcpFirewallRulesStatus, err error) {
	object = new(GcpFirewallRulesStatus)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	object.description = b.description
	object.kind = b.kind
	if b.rules != nil {
		object.rules = make([]*GcpFirewallRulesStatusEntry, len(b.rules))
		for i, v := range b.rules {
			object.rules[i], err = v.Build()
			if err != nil {
				return
			}
		}
	}
	object.state = b.state
	return
}
