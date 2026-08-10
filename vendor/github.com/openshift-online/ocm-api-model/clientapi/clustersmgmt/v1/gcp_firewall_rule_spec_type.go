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

// GcpFirewallRuleSpec represents the values of the 'gcp_firewall_rule_spec' type.
//
// A single rendered GCP firewall rule specification.
type GcpFirewallRuleSpec struct {
	fieldSet_             []bool
	allowed               []*GcpFirewallRuleAllowed
	direction             string
	name                  string
	network               string
	priority              int
	sourceRanges          []string
	sourceServiceAccounts []string
	targetServiceAccounts []string
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *GcpFirewallRuleSpec) Empty() bool {
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

// Allowed returns the value of the 'allowed' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Allowed protocols and ports.
func (o *GcpFirewallRuleSpec) Allowed() []*GcpFirewallRuleAllowed {
	if o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0] {
		return o.allowed
	}
	return nil
}

// GetAllowed returns the value of the 'allowed' attribute and
// a flag indicating if the attribute has a value.
//
// Allowed protocols and ports.
func (o *GcpFirewallRuleSpec) GetAllowed() (value []*GcpFirewallRuleAllowed, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0]
	if ok {
		value = o.allowed
	}
	return
}

// Direction returns the value of the 'direction' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Direction of traffic (INGRESS or EGRESS).
func (o *GcpFirewallRuleSpec) Direction() string {
	if o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1] {
		return o.direction
	}
	return ""
}

// GetDirection returns the value of the 'direction' attribute and
// a flag indicating if the attribute has a value.
//
// Direction of traffic (INGRESS or EGRESS).
func (o *GcpFirewallRuleSpec) GetDirection() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1]
	if ok {
		value = o.direction
	}
	return
}

// Name returns the value of the 'name' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Name of the firewall rule.
func (o *GcpFirewallRuleSpec) Name() string {
	if o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2] {
		return o.name
	}
	return ""
}

// GetName returns the value of the 'name' attribute and
// a flag indicating if the attribute has a value.
//
// Name of the firewall rule.
func (o *GcpFirewallRuleSpec) GetName() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2]
	if ok {
		value = o.name
	}
	return
}

// Network returns the value of the 'network' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// GCP network URL.
func (o *GcpFirewallRuleSpec) Network() string {
	if o != nil && len(o.fieldSet_) > 3 && o.fieldSet_[3] {
		return o.network
	}
	return ""
}

// GetNetwork returns the value of the 'network' attribute and
// a flag indicating if the attribute has a value.
//
// GCP network URL.
func (o *GcpFirewallRuleSpec) GetNetwork() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 3 && o.fieldSet_[3]
	if ok {
		value = o.network
	}
	return
}

// Priority returns the value of the 'priority' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Priority of the rule (0-65535).
func (o *GcpFirewallRuleSpec) Priority() int {
	if o != nil && len(o.fieldSet_) > 4 && o.fieldSet_[4] {
		return o.priority
	}
	return 0
}

// GetPriority returns the value of the 'priority' attribute and
// a flag indicating if the attribute has a value.
//
// Priority of the rule (0-65535).
func (o *GcpFirewallRuleSpec) GetPriority() (value int, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 4 && o.fieldSet_[4]
	if ok {
		value = o.priority
	}
	return
}

// SourceRanges returns the value of the 'source_ranges' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Source IP CIDR ranges.
func (o *GcpFirewallRuleSpec) SourceRanges() []string {
	if o != nil && len(o.fieldSet_) > 5 && o.fieldSet_[5] {
		return o.sourceRanges
	}
	return nil
}

// GetSourceRanges returns the value of the 'source_ranges' attribute and
// a flag indicating if the attribute has a value.
//
// Source IP CIDR ranges.
func (o *GcpFirewallRuleSpec) GetSourceRanges() (value []string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 5 && o.fieldSet_[5]
	if ok {
		value = o.sourceRanges
	}
	return
}

// SourceServiceAccounts returns the value of the 'source_service_accounts' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Source service accounts.
func (o *GcpFirewallRuleSpec) SourceServiceAccounts() []string {
	if o != nil && len(o.fieldSet_) > 6 && o.fieldSet_[6] {
		return o.sourceServiceAccounts
	}
	return nil
}

// GetSourceServiceAccounts returns the value of the 'source_service_accounts' attribute and
// a flag indicating if the attribute has a value.
//
// Source service accounts.
func (o *GcpFirewallRuleSpec) GetSourceServiceAccounts() (value []string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 6 && o.fieldSet_[6]
	if ok {
		value = o.sourceServiceAccounts
	}
	return
}

// TargetServiceAccounts returns the value of the 'target_service_accounts' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Target service accounts.
func (o *GcpFirewallRuleSpec) TargetServiceAccounts() []string {
	if o != nil && len(o.fieldSet_) > 7 && o.fieldSet_[7] {
		return o.targetServiceAccounts
	}
	return nil
}

// GetTargetServiceAccounts returns the value of the 'target_service_accounts' attribute and
// a flag indicating if the attribute has a value.
//
// Target service accounts.
func (o *GcpFirewallRuleSpec) GetTargetServiceAccounts() (value []string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 7 && o.fieldSet_[7]
	if ok {
		value = o.targetServiceAccounts
	}
	return
}

// GcpFirewallRuleSpecListKind is the name of the type used to represent list of objects of
// type 'gcp_firewall_rule_spec'.
const GcpFirewallRuleSpecListKind = "GcpFirewallRuleSpecList"

// GcpFirewallRuleSpecListLinkKind is the name of the type used to represent links to list
// of objects of type 'gcp_firewall_rule_spec'.
const GcpFirewallRuleSpecListLinkKind = "GcpFirewallRuleSpecListLink"

// GcpFirewallRuleSpecNilKind is the name of the type used to nil lists of objects of
// type 'gcp_firewall_rule_spec'.
const GcpFirewallRuleSpecListNilKind = "GcpFirewallRuleSpecListNil"

// GcpFirewallRuleSpecList is a list of values of the 'gcp_firewall_rule_spec' type.
type GcpFirewallRuleSpecList struct {
	href  string
	link  bool
	items []*GcpFirewallRuleSpec
}

// Len returns the length of the list.
func (l *GcpFirewallRuleSpecList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Items sets the items of the list.
func (l *GcpFirewallRuleSpecList) SetLink(link bool) {
	l.link = link
}

// Items sets the items of the list.
func (l *GcpFirewallRuleSpecList) SetHREF(href string) {
	l.href = href
}

// Items sets the items of the list.
func (l *GcpFirewallRuleSpecList) SetItems(items []*GcpFirewallRuleSpec) {
	l.items = items
}

// Items returns the items of the list.
func (l *GcpFirewallRuleSpecList) Items() []*GcpFirewallRuleSpec {
	if l == nil {
		return nil
	}
	return l.items
}

// Empty returns true if the list is empty.
func (l *GcpFirewallRuleSpecList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *GcpFirewallRuleSpecList) Get(i int) *GcpFirewallRuleSpec {
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
func (l *GcpFirewallRuleSpecList) Slice() []*GcpFirewallRuleSpec {
	var slice []*GcpFirewallRuleSpec
	if l == nil {
		slice = make([]*GcpFirewallRuleSpec, 0)
	} else {
		slice = make([]*GcpFirewallRuleSpec, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *GcpFirewallRuleSpecList) Each(f func(item *GcpFirewallRuleSpec) bool) {
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
func (l *GcpFirewallRuleSpecList) Range(f func(index int, item *GcpFirewallRuleSpec) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
