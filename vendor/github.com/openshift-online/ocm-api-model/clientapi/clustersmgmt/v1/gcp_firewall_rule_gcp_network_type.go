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

// GcpFirewallRuleGcpNetwork represents the values of the 'gcp_firewall_rule_gcp_network' type.
//
// GCP network configuration for a firewall rule.
type GcpFirewallRuleGcpNetwork struct {
	fieldSet_   []bool
	machineCidr string
	projectId   string
	vpcName     string
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *GcpFirewallRuleGcpNetwork) Empty() bool {
	if o == nil || len(o.fieldSet_) == 0 {
		return true
	}
	for _, set := range o.fieldSet_ {
		if set {
			return false
		}
	}
	return true
}

// MachineCidr returns the value of the 'machine_cidr' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Machine CIDR block.
func (o *GcpFirewallRuleGcpNetwork) MachineCidr() string {
	if o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0] {
		return o.machineCidr
	}
	return ""
}

// GetMachineCidr returns the value of the 'machine_cidr' attribute and
// a flag indicating if the attribute has a value.
//
// Machine CIDR block.
func (o *GcpFirewallRuleGcpNetwork) GetMachineCidr() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0]
	if ok {
		value = o.machineCidr
	}
	return
}

// ProjectId returns the value of the 'project_id' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// GCP project ID.
func (o *GcpFirewallRuleGcpNetwork) ProjectId() string {
	if o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1] {
		return o.projectId
	}
	return ""
}

// GetProjectId returns the value of the 'project_id' attribute and
// a flag indicating if the attribute has a value.
//
// GCP project ID.
func (o *GcpFirewallRuleGcpNetwork) GetProjectId() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1]
	if ok {
		value = o.projectId
	}
	return
}

// VpcName returns the value of the 'vpc_name' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// VPC network name.
func (o *GcpFirewallRuleGcpNetwork) VpcName() string {
	if o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2] {
		return o.vpcName
	}
	return ""
}

// GetVpcName returns the value of the 'vpc_name' attribute and
// a flag indicating if the attribute has a value.
//
// VPC network name.
func (o *GcpFirewallRuleGcpNetwork) GetVpcName() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2]
	if ok {
		value = o.vpcName
	}
	return
}

// GcpFirewallRuleGcpNetworkListKind is the name of the type used to represent list of objects of
// type 'gcp_firewall_rule_gcp_network'.
const GcpFirewallRuleGcpNetworkListKind = "GcpFirewallRuleGcpNetworkList"

// GcpFirewallRuleGcpNetworkListLinkKind is the name of the type used to represent links to list
// of objects of type 'gcp_firewall_rule_gcp_network'.
const GcpFirewallRuleGcpNetworkListLinkKind = "GcpFirewallRuleGcpNetworkListLink"

// GcpFirewallRuleGcpNetworkNilKind is the name of the type used to nil lists of objects of
// type 'gcp_firewall_rule_gcp_network'.
const GcpFirewallRuleGcpNetworkListNilKind = "GcpFirewallRuleGcpNetworkListNil"

// GcpFirewallRuleGcpNetworkList is a list of values of the 'gcp_firewall_rule_gcp_network' type.
type GcpFirewallRuleGcpNetworkList struct {
	href  string
	link  bool
	items []*GcpFirewallRuleGcpNetwork
}

// Len returns the length of the list.
func (l *GcpFirewallRuleGcpNetworkList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Items sets the items of the list.
func (l *GcpFirewallRuleGcpNetworkList) SetLink(link bool) {
	l.link = link
}

// Items sets the items of the list.
func (l *GcpFirewallRuleGcpNetworkList) SetHREF(href string) {
	l.href = href
}

// Items sets the items of the list.
func (l *GcpFirewallRuleGcpNetworkList) SetItems(items []*GcpFirewallRuleGcpNetwork) {
	l.items = items
}

// Items returns the items of the list.
func (l *GcpFirewallRuleGcpNetworkList) Items() []*GcpFirewallRuleGcpNetwork {
	if l == nil {
		return nil
	}
	return l.items
}

// Empty returns true if the list is empty.
func (l *GcpFirewallRuleGcpNetworkList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *GcpFirewallRuleGcpNetworkList) Get(i int) *GcpFirewallRuleGcpNetwork {
	if l == nil || i < 0 || i >= len(l.items) {
		return nil
	}
	return l.items[i]
}

// Slice returns an slice containing the items of the list. The returned slice is a
// copy of the one used internally, so it can be modified without affecting the
// internal representation.
//
// If you don't need to modify the returned slice consider using the Each or Range
// functions, as they don't need to allocate a new slice.
func (l *GcpFirewallRuleGcpNetworkList) Slice() []*GcpFirewallRuleGcpNetwork {
	var slice []*GcpFirewallRuleGcpNetwork
	if l == nil {
		slice = make([]*GcpFirewallRuleGcpNetwork, 0)
	} else {
		slice = make([]*GcpFirewallRuleGcpNetwork, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *GcpFirewallRuleGcpNetworkList) Each(f func(item *GcpFirewallRuleGcpNetwork) bool) {
	if l == nil {
		return
	}
	for _, item := range l.items {
		if !f(item) {
			break
		}
	}
}

// Range runs the given function for each index and item of the list, in order. If
// the function returns false the iteration stops, otherwise it continues till all
// the elements of the list have been processed.
func (l *GcpFirewallRuleGcpNetworkList) Range(f func(index int, item *GcpFirewallRuleGcpNetwork) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
