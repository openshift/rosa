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

// A single rendered GCP firewall rule specification.
type GcpFirewallRuleSpecBuilder struct {
	fieldSet_             []bool
	allowed               []*GcpFirewallRuleAllowedBuilder
	direction             string
	name                  string
	network               string
	priority              int
	sourceRanges          []string
	sourceServiceAccounts []string
	targetServiceAccounts []string
}

// NewGcpFirewallRuleSpec creates a new builder of 'gcp_firewall_rule_spec' objects.
func NewGcpFirewallRuleSpec() *GcpFirewallRuleSpecBuilder {
	return &GcpFirewallRuleSpecBuilder{
		fieldSet_: make([]bool, 8),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *GcpFirewallRuleSpecBuilder) Empty() bool {
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

// Allowed sets the value of the 'allowed' attribute to the given values.
func (b *GcpFirewallRuleSpecBuilder) Allowed(values ...*GcpFirewallRuleAllowedBuilder) *GcpFirewallRuleSpecBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 8)
	}
	b.allowed = make([]*GcpFirewallRuleAllowedBuilder, len(values))
	copy(b.allowed, values)
	b.fieldSet_[0] = true
	return b
}

// Direction sets the value of the 'direction' attribute to the given value.
func (b *GcpFirewallRuleSpecBuilder) Direction(value string) *GcpFirewallRuleSpecBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 8)
	}
	b.direction = value
	b.fieldSet_[1] = true
	return b
}

// Name sets the value of the 'name' attribute to the given value.
func (b *GcpFirewallRuleSpecBuilder) Name(value string) *GcpFirewallRuleSpecBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 8)
	}
	b.name = value
	b.fieldSet_[2] = true
	return b
}

// Network sets the value of the 'network' attribute to the given value.
func (b *GcpFirewallRuleSpecBuilder) Network(value string) *GcpFirewallRuleSpecBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 8)
	}
	b.network = value
	b.fieldSet_[3] = true
	return b
}

// Priority sets the value of the 'priority' attribute to the given value.
func (b *GcpFirewallRuleSpecBuilder) Priority(value int) *GcpFirewallRuleSpecBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 8)
	}
	b.priority = value
	b.fieldSet_[4] = true
	return b
}

// SourceRanges sets the value of the 'source_ranges' attribute to the given values.
func (b *GcpFirewallRuleSpecBuilder) SourceRanges(values ...string) *GcpFirewallRuleSpecBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 8)
	}
	b.sourceRanges = make([]string, len(values))
	copy(b.sourceRanges, values)
	b.fieldSet_[5] = true
	return b
}

// SourceServiceAccounts sets the value of the 'source_service_accounts' attribute to the given values.
func (b *GcpFirewallRuleSpecBuilder) SourceServiceAccounts(values ...string) *GcpFirewallRuleSpecBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 8)
	}
	b.sourceServiceAccounts = make([]string, len(values))
	copy(b.sourceServiceAccounts, values)
	b.fieldSet_[6] = true
	return b
}

// TargetServiceAccounts sets the value of the 'target_service_accounts' attribute to the given values.
func (b *GcpFirewallRuleSpecBuilder) TargetServiceAccounts(values ...string) *GcpFirewallRuleSpecBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 8)
	}
	b.targetServiceAccounts = make([]string, len(values))
	copy(b.targetServiceAccounts, values)
	b.fieldSet_[7] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *GcpFirewallRuleSpecBuilder) Copy(object *GcpFirewallRuleSpec) *GcpFirewallRuleSpecBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	if object.allowed != nil {
		b.allowed = make([]*GcpFirewallRuleAllowedBuilder, len(object.allowed))
		for i, v := range object.allowed {
			b.allowed[i] = NewGcpFirewallRuleAllowed().Copy(v)
		}
	} else {
		b.allowed = nil
	}
	b.direction = object.direction
	b.name = object.name
	b.network = object.network
	b.priority = object.priority
	if object.sourceRanges != nil {
		b.sourceRanges = make([]string, len(object.sourceRanges))
		copy(b.sourceRanges, object.sourceRanges)
	} else {
		b.sourceRanges = nil
	}
	if object.sourceServiceAccounts != nil {
		b.sourceServiceAccounts = make([]string, len(object.sourceServiceAccounts))
		copy(b.sourceServiceAccounts, object.sourceServiceAccounts)
	} else {
		b.sourceServiceAccounts = nil
	}
	if object.targetServiceAccounts != nil {
		b.targetServiceAccounts = make([]string, len(object.targetServiceAccounts))
		copy(b.targetServiceAccounts, object.targetServiceAccounts)
	} else {
		b.targetServiceAccounts = nil
	}
	return b
}

// Build creates a 'gcp_firewall_rule_spec' object using the configuration stored in the builder.
func (b *GcpFirewallRuleSpecBuilder) Build() (object *GcpFirewallRuleSpec, err error) {
	object = new(GcpFirewallRuleSpec)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	if b.allowed != nil {
		object.allowed = make([]*GcpFirewallRuleAllowed, len(b.allowed))
		for i, v := range b.allowed {
			object.allowed[i], err = v.Build()
			if err != nil {
				return
			}
		}
	}
	object.direction = b.direction
	object.name = b.name
	object.network = b.network
	object.priority = b.priority
	if b.sourceRanges != nil {
		object.sourceRanges = make([]string, len(b.sourceRanges))
		copy(object.sourceRanges, b.sourceRanges)
	}
	if b.sourceServiceAccounts != nil {
		object.sourceServiceAccounts = make([]string, len(b.sourceServiceAccounts))
		copy(object.sourceServiceAccounts, b.sourceServiceAccounts)
	}
	if b.targetServiceAccounts != nil {
		object.targetServiceAccounts = make([]string, len(b.targetServiceAccounts))
		copy(object.targetServiceAccounts, b.targetServiceAccounts)
	}
	return
}
