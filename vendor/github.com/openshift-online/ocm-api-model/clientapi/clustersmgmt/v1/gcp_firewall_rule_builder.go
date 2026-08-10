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

// GCP firewall rule configuration for BYO (Bring Your Own) firewall workflow.
// These are organization-level resources that can be created before cluster
// provisioning and reused after cluster deletion.
type GcpFirewallRuleBuilder struct {
	fieldSet_    []bool
	id           string
	href         string
	gcpNetwork   *GcpFirewallRuleGcpNetworkBuilder
	name         string
	organization *OrganizationLinkBuilder
	profile      string
	status       *GcpFirewallRuleStatusBuilder
	template     *GcpFirewallRuleTemplateRefBuilder
	wifConfig    *WifConfigBuilder
}

// NewGcpFirewallRule creates a new builder of 'gcp_firewall_rule' objects.
func NewGcpFirewallRule() *GcpFirewallRuleBuilder {
	return &GcpFirewallRuleBuilder{
		fieldSet_: make([]bool, 10),
	}
}

// Link sets the flag that indicates if this is a link.
func (b *GcpFirewallRuleBuilder) Link(value bool) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.fieldSet_[0] = true
	return b
}

// ID sets the identifier of the object.
func (b *GcpFirewallRuleBuilder) ID(value string) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.id = value
	b.fieldSet_[1] = true
	return b
}

// HREF sets the link to the object.
func (b *GcpFirewallRuleBuilder) HREF(value string) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.href = value
	b.fieldSet_[2] = true
	return b
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *GcpFirewallRuleBuilder) Empty() bool {
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

// GcpNetwork sets the value of the 'gcp_network' attribute to the given value.
//
// GCP network configuration for a firewall rule.
func (b *GcpFirewallRuleBuilder) GcpNetwork(value *GcpFirewallRuleGcpNetworkBuilder) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.gcpNetwork = value
	if value != nil {
		b.fieldSet_[3] = true
	} else {
		b.fieldSet_[3] = false
	}
	return b
}

// Name sets the value of the 'name' attribute to the given value.
func (b *GcpFirewallRuleBuilder) Name(value string) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.name = value
	b.fieldSet_[4] = true
	return b
}

// Organization sets the value of the 'organization' attribute to the given value.
//
// Definition of an organization link.
func (b *GcpFirewallRuleBuilder) Organization(value *OrganizationLinkBuilder) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.organization = value
	if value != nil {
		b.fieldSet_[5] = true
	} else {
		b.fieldSet_[5] = false
	}
	return b
}

// Profile sets the value of the 'profile' attribute to the given value.
func (b *GcpFirewallRuleBuilder) Profile(value string) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.profile = value
	b.fieldSet_[6] = true
	return b
}

// Status sets the value of the 'status' attribute to the given value.
//
// Status of a GCP firewall rule set, containing the rendered rules.
func (b *GcpFirewallRuleBuilder) Status(value *GcpFirewallRuleStatusBuilder) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.status = value
	if value != nil {
		b.fieldSet_[7] = true
	} else {
		b.fieldSet_[7] = false
	}
	return b
}

// Template sets the value of the 'template' attribute to the given value.
//
// Template version and profile reference for a firewall rule.
func (b *GcpFirewallRuleBuilder) Template(value *GcpFirewallRuleTemplateRefBuilder) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.template = value
	if value != nil {
		b.fieldSet_[8] = true
	} else {
		b.fieldSet_[8] = false
	}
	return b
}

// WifConfig sets the value of the 'wif_config' attribute to the given value.
//
// Definition of an wif_config resource.
func (b *GcpFirewallRuleBuilder) WifConfig(value *WifConfigBuilder) *GcpFirewallRuleBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 10)
	}
	b.wifConfig = value
	if value != nil {
		b.fieldSet_[9] = true
	} else {
		b.fieldSet_[9] = false
	}
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *GcpFirewallRuleBuilder) Copy(object *GcpFirewallRule) *GcpFirewallRuleBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.id = object.id
	b.href = object.href
	if object.gcpNetwork != nil {
		b.gcpNetwork = NewGcpFirewallRuleGcpNetwork().Copy(object.gcpNetwork)
	} else {
		b.gcpNetwork = nil
	}
	b.name = object.name
	if object.organization != nil {
		b.organization = NewOrganizationLink().Copy(object.organization)
	} else {
		b.organization = nil
	}
	b.profile = object.profile
	if object.status != nil {
		b.status = NewGcpFirewallRuleStatus().Copy(object.status)
	} else {
		b.status = nil
	}
	if object.template != nil {
		b.template = NewGcpFirewallRuleTemplateRef().Copy(object.template)
	} else {
		b.template = nil
	}
	if object.wifConfig != nil {
		b.wifConfig = NewWifConfig().Copy(object.wifConfig)
	} else {
		b.wifConfig = nil
	}
	return b
}

// Build creates a 'gcp_firewall_rule' object using the configuration stored in the builder.
func (b *GcpFirewallRuleBuilder) Build() (object *GcpFirewallRule, err error) {
	object = new(GcpFirewallRule)
	object.id = b.id
	object.href = b.href
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	if b.gcpNetwork != nil {
		object.gcpNetwork, err = b.gcpNetwork.Build()
		if err != nil {
			return
		}
	}
	object.name = b.name
	if b.organization != nil {
		object.organization, err = b.organization.Build()
		if err != nil {
			return
		}
	}
	object.profile = b.profile
	if b.status != nil {
		object.status, err = b.status.Build()
		if err != nil {
			return
		}
	}
	if b.template != nil {
		object.template, err = b.template.Build()
		if err != nil {
			return
		}
	}
	if b.wifConfig != nil {
		object.wifConfig, err = b.wifConfig.Build()
		if err != nil {
			return
		}
	}
	return
}
