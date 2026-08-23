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

// GcpFirewallRulesStatusEntry represents the values of the 'gcp_firewall_rules_status_entry' type.
//
// Status of an individual GCP firewall rule.
type GcpFirewallRulesStatusEntry struct {
	fieldSet_           []bool
	description         string
	name                string
	configuredCorrectly bool
	exists              bool
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *GcpFirewallRulesStatusEntry) Empty() bool {
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

// ConfiguredCorrectly returns the value of the 'configured_correctly' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Whether the firewall rule is configured correctly.
func (o *GcpFirewallRulesStatusEntry) ConfiguredCorrectly() bool {
	if o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0] {
		return o.configuredCorrectly
	}
	return false
}

// GetConfiguredCorrectly returns the value of the 'configured_correctly' attribute and
// a flag indicating if the attribute has a value.
//
// Whether the firewall rule is configured correctly.
func (o *GcpFirewallRulesStatusEntry) GetConfiguredCorrectly() (value bool, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0]
	if ok {
		value = o.configuredCorrectly
	}
	return
}

// Description returns the value of the 'description' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Description of each firewall rule
func (o *GcpFirewallRulesStatusEntry) Description() string {
	if o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1] {
		return o.description
	}
	return ""
}

// GetDescription returns the value of the 'description' attribute and
// a flag indicating if the attribute has a value.
//
// Description of each firewall rule
func (o *GcpFirewallRulesStatusEntry) GetDescription() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1]
	if ok {
		value = o.description
	}
	return
}

// Exists returns the value of the 'exists' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Whether the firewall rule exists in GCP.
func (o *GcpFirewallRulesStatusEntry) Exists() bool {
	if o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2] {
		return o.exists
	}
	return false
}

// GetExists returns the value of the 'exists' attribute and
// a flag indicating if the attribute has a value.
//
// Whether the firewall rule exists in GCP.
func (o *GcpFirewallRulesStatusEntry) GetExists() (value bool, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2]
	if ok {
		value = o.exists
	}
	return
}

// Name returns the value of the 'name' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Name of the firewall rule in GCP.
func (o *GcpFirewallRulesStatusEntry) Name() string {
	if o != nil && len(o.fieldSet_) > 3 && o.fieldSet_[3] {
		return o.name
	}
	return ""
}

// GetName returns the value of the 'name' attribute and
// a flag indicating if the attribute has a value.
//
// Name of the firewall rule in GCP.
func (o *GcpFirewallRulesStatusEntry) GetName() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 3 && o.fieldSet_[3]
	if ok {
		value = o.name
	}
	return
}

// GcpFirewallRulesStatusEntryListKind is the name of the type used to represent list of objects of
// type 'gcp_firewall_rules_status_entry'.
const GcpFirewallRulesStatusEntryListKind = "GcpFirewallRulesStatusEntryList"

// GcpFirewallRulesStatusEntryListLinkKind is the name of the type used to represent links to list
// of objects of type 'gcp_firewall_rules_status_entry'.
const GcpFirewallRulesStatusEntryListLinkKind = "GcpFirewallRulesStatusEntryListLink"

// GcpFirewallRulesStatusEntryNilKind is the name of the type used to nil lists of objects of
// type 'gcp_firewall_rules_status_entry'.
const GcpFirewallRulesStatusEntryListNilKind = "GcpFirewallRulesStatusEntryListNil"

// GcpFirewallRulesStatusEntryList is a list of values of the 'gcp_firewall_rules_status_entry' type.
type GcpFirewallRulesStatusEntryList struct {
	href  string
	link  bool
	items []*GcpFirewallRulesStatusEntry
}

// Len returns the length of the list.
func (l *GcpFirewallRulesStatusEntryList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Items sets the items of the list.
func (l *GcpFirewallRulesStatusEntryList) SetLink(link bool) {
	l.link = link
}

// Items sets the items of the list.
func (l *GcpFirewallRulesStatusEntryList) SetHREF(href string) {
	l.href = href
}

// Items sets the items of the list.
func (l *GcpFirewallRulesStatusEntryList) SetItems(items []*GcpFirewallRulesStatusEntry) {
	l.items = items
}

// Items returns the items of the list.
func (l *GcpFirewallRulesStatusEntryList) Items() []*GcpFirewallRulesStatusEntry {
	if l == nil {
		return nil
	}
	return l.items
}

// Empty returns true if the list is empty.
func (l *GcpFirewallRulesStatusEntryList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *GcpFirewallRulesStatusEntryList) Get(i int) *GcpFirewallRulesStatusEntry {
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
func (l *GcpFirewallRulesStatusEntryList) Slice() []*GcpFirewallRulesStatusEntry {
	var slice []*GcpFirewallRulesStatusEntry
	if l == nil {
		slice = make([]*GcpFirewallRulesStatusEntry, 0)
	} else {
		slice = make([]*GcpFirewallRulesStatusEntry, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *GcpFirewallRulesStatusEntryList) Each(f func(item *GcpFirewallRulesStatusEntry) bool) {
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
func (l *GcpFirewallRulesStatusEntryList) Range(f func(index int, item *GcpFirewallRulesStatusEntry) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
