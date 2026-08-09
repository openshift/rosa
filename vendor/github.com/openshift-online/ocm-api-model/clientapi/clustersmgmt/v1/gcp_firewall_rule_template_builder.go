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

// GCP firewall rule template definition.
// Templates define the set of firewall rules that will be created for a cluster.
// Each template has a version and one or more profiles (e.g. "public", "private").
type GcpFirewallRuleTemplateBuilder struct {
	fieldSet_   []bool
	description string
	href        string
	profile     string
	version     string
}

// NewGcpFirewallRuleTemplate creates a new builder of 'gcp_firewall_rule_template' objects.
func NewGcpFirewallRuleTemplate() *GcpFirewallRuleTemplateBuilder {
	return &GcpFirewallRuleTemplateBuilder{
		fieldSet_: make([]bool, 4),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *GcpFirewallRuleTemplateBuilder) Empty() bool {
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
func (b *GcpFirewallRuleTemplateBuilder) Description(value string) *GcpFirewallRuleTemplateBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.description = value
	b.fieldSet_[0] = true
	return b
}

// Href sets the value of the 'href' attribute to the given value.
func (b *GcpFirewallRuleTemplateBuilder) Href(value string) *GcpFirewallRuleTemplateBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.href = value
	b.fieldSet_[1] = true
	return b
}

// Profile sets the value of the 'profile' attribute to the given value.
func (b *GcpFirewallRuleTemplateBuilder) Profile(value string) *GcpFirewallRuleTemplateBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.profile = value
	b.fieldSet_[2] = true
	return b
}

// Version sets the value of the 'version' attribute to the given value.
func (b *GcpFirewallRuleTemplateBuilder) Version(value string) *GcpFirewallRuleTemplateBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 4)
	}
	b.version = value
	b.fieldSet_[3] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *GcpFirewallRuleTemplateBuilder) Copy(object *GcpFirewallRuleTemplate) *GcpFirewallRuleTemplateBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.description = object.description
	b.href = object.href
	b.profile = object.profile
	b.version = object.version
	return b
}

// Build creates a 'gcp_firewall_rule_template' object using the configuration stored in the builder.
func (b *GcpFirewallRuleTemplateBuilder) Build() (object *GcpFirewallRuleTemplate, err error) {
	object = new(GcpFirewallRuleTemplate)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	object.description = b.description
	object.href = b.href
	object.profile = b.profile
	object.version = b.version
	return
}
