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

// GcpFirewallRulesStatus represents the values of the 'gcp_firewall_rules_status' type.
//
// Status of GCP firewall rules verification.
type GcpFirewallRulesStatus struct {
	fieldSet_   []bool
	description string
	kind        string
	rules       []*GcpFirewallRulesStatusEntry
	state       string
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *GcpFirewallRulesStatus) Empty() bool {
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

// Description returns the value of the 'description' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Description of the current status.
func (o *GcpFirewallRulesStatus) Description() string {
	if o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0] {
		return o.description
	}
	return ""
}

// GetDescription returns the value of the 'description' attribute and
// a flag indicating if the attribute has a value.
//
// Description of the current status.
func (o *GcpFirewallRulesStatus) GetDescription() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0]
	if ok {
		value = o.description
	}
	return
}

// Kind returns the value of the 'kind' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Kind is the resource type identifier.
func (o *GcpFirewallRulesStatus) Kind() string {
	if o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1] {
		return o.kind
	}
	return ""
}

// GetKind returns the value of the 'kind' attribute and
// a flag indicating if the attribute has a value.
//
// Kind is the resource type identifier.
func (o *GcpFirewallRulesStatus) GetKind() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1]
	if ok {
		value = o.kind
	}
	return
}

// Rules returns the value of the 'rules' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Individual firewall rule status entries.
func (o *GcpFirewallRulesStatus) Rules() []*GcpFirewallRulesStatusEntry {
	if o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2] {
		return o.rules
	}
	return nil
}

// GetRules returns the value of the 'rules' attribute and
// a flag indicating if the attribute has a value.
//
// Individual firewall rule status entries.
func (o *GcpFirewallRulesStatus) GetRules() (value []*GcpFirewallRulesStatusEntry, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2]
	if ok {
		value = o.rules
	}
	return
}

// State returns the value of the 'state' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Overall state of the firewall rules (e.g. "ready", "incomplete", "not_found").
func (o *GcpFirewallRulesStatus) State() string {
	if o != nil && len(o.fieldSet_) > 3 && o.fieldSet_[3] {
		return o.state
	}
	return ""
}

// GetState returns the value of the 'state' attribute and
// a flag indicating if the attribute has a value.
//
// Overall state of the firewall rules (e.g. "ready", "incomplete", "not_found").
func (o *GcpFirewallRulesStatus) GetState() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 3 && o.fieldSet_[3]
	if ok {
		value = o.state
	}
	return
}

// GcpFirewallRulesStatusListKind is the name of the type used to represent list of objects of
// type 'gcp_firewall_rules_status'.
const GcpFirewallRulesStatusListKind = "GcpFirewallRulesStatusList"

// GcpFirewallRulesStatusListLinkKind is the name of the type used to represent links to list
// of objects of type 'gcp_firewall_rules_status'.
const GcpFirewallRulesStatusListLinkKind = "GcpFirewallRulesStatusListLink"

// GcpFirewallRulesStatusNilKind is the name of the type used to nil lists of objects of
// type 'gcp_firewall_rules_status'.
const GcpFirewallRulesStatusListNilKind = "GcpFirewallRulesStatusListNil"

// GcpFirewallRulesStatusList is a list of values of the 'gcp_firewall_rules_status' type.
type GcpFirewallRulesStatusList struct {
	href  string
	link  bool
	items []*GcpFirewallRulesStatus
}

// Len returns the length of the list.
func (l *GcpFirewallRulesStatusList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Items sets the items of the list.
func (l *GcpFirewallRulesStatusList) SetLink(link bool) {
	l.link = link
}

// Items sets the items of the list.
func (l *GcpFirewallRulesStatusList) SetHREF(href string) {
	l.href = href
}

// Items sets the items of the list.
func (l *GcpFirewallRulesStatusList) SetItems(items []*GcpFirewallRulesStatus) {
	l.items = items
}

// Items returns the items of the list.
func (l *GcpFirewallRulesStatusList) Items() []*GcpFirewallRulesStatus {
	if l == nil {
		return nil
	}
	return l.items
}

// Empty returns true if the list is empty.
func (l *GcpFirewallRulesStatusList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *GcpFirewallRulesStatusList) Get(i int) *GcpFirewallRulesStatus {
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
func (l *GcpFirewallRulesStatusList) Slice() []*GcpFirewallRulesStatus {
	var slice []*GcpFirewallRulesStatus
	if l == nil {
		slice = make([]*GcpFirewallRulesStatus, 0)
	} else {
		slice = make([]*GcpFirewallRulesStatus, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *GcpFirewallRulesStatusList) Each(f func(item *GcpFirewallRulesStatus) bool) {
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
func (l *GcpFirewallRulesStatusList) Range(f func(index int, item *GcpFirewallRulesStatus) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
