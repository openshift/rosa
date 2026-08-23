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

// Status of an individual GCP firewall rule.
type GcpFirewallRulesStatusEntryBuilder struct {
	fieldSet_           []bool
	description         string
	name                string
	configuredCorrectly bool
	exists              bool
}

// NewGcpFirewallRulesStatusEntry creates a new builder of 'gcp_firewall_rules_status_entry' objects.
func NewGcpFirewallRulesStatusEntry() *GcpFirewallRulesStatusEntryBuilder {
	return &GcpFirewallRulesStatusEntryBuilder{
		fieldSet_: make([]bool, 4),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *GcpFirewallRulesStatusEntryBuilder) Empty() bool {
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

// ConfiguredCorrectly sets the value of the 'configured_correctly' attribute to the given value.
func (b *GcpFirewallRulesStatusEntryBuilder) ConfiguredCorrectly(value bool) *GcpFirewallRulesStatusEntryBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.configuredCorrectly = value
	b.fieldSet_[0] = true
	return b
}

// Description sets the value of the 'description' attribute to the given value.
func (b *GcpFirewallRulesStatusEntryBuilder) Description(value string) *GcpFirewallRulesStatusEntryBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.description = value
	b.fieldSet_[1] = true
	return b
}

// Exists sets the value of the 'exists' attribute to the given value.
func (b *GcpFirewallRulesStatusEntryBuilder) Exists(value bool) *GcpFirewallRulesStatusEntryBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.exists = value
	b.fieldSet_[2] = true
	return b
}

// Name sets the value of the 'name' attribute to the given value.
func (b *GcpFirewallRulesStatusEntryBuilder) Name(value string) *GcpFirewallRulesStatusEntryBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.name = value
	b.fieldSet_[3] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *GcpFirewallRulesStatusEntryBuilder) Copy(object *GcpFirewallRulesStatusEntry) *GcpFirewallRulesStatusEntryBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.configuredCorrectly = object.configuredCorrectly
	b.description = object.description
	b.exists = object.exists
	b.name = object.name
	return b
}

// Build creates a 'gcp_firewall_rules_status_entry' object using the configuration stored in the builder.
func (b *GcpFirewallRulesStatusEntryBuilder) Build() (object *GcpFirewallRulesStatusEntry, err error) {
	object = new(GcpFirewallRulesStatusEntry)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	object.configuredCorrectly = b.configuredCorrectly
	object.description = b.description
	object.exists = b.exists
	object.name = b.name
	return
}
