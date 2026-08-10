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

// GCP network configuration for a firewall rule.
type GcpFirewallRuleGcpNetworkBuilder struct {
	fieldSet_   []bool
	machineCidr string
	projectId   string
	vpcName     string
}

// NewGcpFirewallRuleGcpNetwork creates a new builder of 'gcp_firewall_rule_gcp_network' objects.
func NewGcpFirewallRuleGcpNetwork() *GcpFirewallRuleGcpNetworkBuilder {
	return &GcpFirewallRuleGcpNetworkBuilder{
		fieldSet_: make([]bool, 3),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *GcpFirewallRuleGcpNetworkBuilder) Empty() bool {
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

// MachineCidr sets the value of the 'machine_cidr' attribute to the given value.
func (b *GcpFirewallRuleGcpNetworkBuilder) MachineCidr(value string) *GcpFirewallRuleGcpNetworkBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 3)
	}
	b.machineCidr = value
	b.fieldSet_[0] = true
	return b
}

// ProjectId sets the value of the 'project_id' attribute to the given value.
func (b *GcpFirewallRuleGcpNetworkBuilder) ProjectId(value string) *GcpFirewallRuleGcpNetworkBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 3)
	}
	b.projectId = value
	b.fieldSet_[1] = true
	return b
}

// VpcName sets the value of the 'vpc_name' attribute to the given value.
func (b *GcpFirewallRuleGcpNetworkBuilder) VpcName(value string) *GcpFirewallRuleGcpNetworkBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 3)
	}
	b.vpcName = value
	b.fieldSet_[2] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *GcpFirewallRuleGcpNetworkBuilder) Copy(object *GcpFirewallRuleGcpNetwork) *GcpFirewallRuleGcpNetworkBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.machineCidr = object.machineCidr
	b.projectId = object.projectId
	b.vpcName = object.vpcName
	return b
}

// Build creates a 'gcp_firewall_rule_gcp_network' object using the configuration stored in the builder.
func (b *GcpFirewallRuleGcpNetworkBuilder) Build() (object *GcpFirewallRuleGcpNetwork, err error) {
	object = new(GcpFirewallRuleGcpNetwork)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	object.machineCidr = b.machineCidr
	object.projectId = b.projectId
	object.vpcName = b.vpcName
	return
}
