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

// Allowed protocol and ports for a GCP firewall rule.
type GcpFirewallRuleAllowedBuilder struct {
	fieldSet_  []bool
	ipProtocol string
	ports      []string
}

// NewGcpFirewallRuleAllowed creates a new builder of 'gcp_firewall_rule_allowed' objects.
func NewGcpFirewallRuleAllowed() *GcpFirewallRuleAllowedBuilder {
	return &GcpFirewallRuleAllowedBuilder{
		fieldSet_: make([]bool, 2),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *GcpFirewallRuleAllowedBuilder) Empty() bool {
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

// IPProtocol sets the value of the 'IP_protocol' attribute to the given value.
func (b *GcpFirewallRuleAllowedBuilder) IPProtocol(value string) *GcpFirewallRuleAllowedBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 2)
	}
	b.ipProtocol = value
	b.fieldSet_[0] = true
	return b
}

// Ports sets the value of the 'ports' attribute to the given values.
func (b *GcpFirewallRuleAllowedBuilder) Ports(values ...string) *GcpFirewallRuleAllowedBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 2)
	}
	b.ports = make([]string, len(values))
	copy(b.ports, values)
	b.fieldSet_[1] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *GcpFirewallRuleAllowedBuilder) Copy(object *GcpFirewallRuleAllowed) *GcpFirewallRuleAllowedBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.ipProtocol = object.ipProtocol
	if object.ports != nil {
		b.ports = make([]string, len(object.ports))
		copy(b.ports, object.ports)
	} else {
		b.ports = nil
	}
	return b
}

// Build creates a 'gcp_firewall_rule_allowed' object using the configuration stored in the builder.
func (b *GcpFirewallRuleAllowedBuilder) Build() (object *GcpFirewallRuleAllowed, err error) {
	object = new(GcpFirewallRuleAllowed)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	object.ipProtocol = b.ipProtocol
	if b.ports != nil {
		object.ports = make([]string, len(b.ports))
		copy(object.ports, b.ports)
	}
	return
}
