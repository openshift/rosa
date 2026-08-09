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

// GcpFirewallRuleKind is the name of the type used to represent objects
// of type 'gcp_firewall_rule'.
const GcpFirewallRuleKind = "GcpFirewallRule"

// GcpFirewallRuleLinkKind is the name of the type used to represent links
// to objects of type 'gcp_firewall_rule'.
const GcpFirewallRuleLinkKind = "GcpFirewallRuleLink"

// GcpFirewallRuleNilKind is the name of the type used to nil references
// to objects of type 'gcp_firewall_rule'.
const GcpFirewallRuleNilKind = "GcpFirewallRuleNil"

// GcpFirewallRule represents the values of the 'gcp_firewall_rule' type.
//
// GCP firewall rule configuration for BYO (Bring Your Own) firewall workflow.
// These are organization-level resources that can be created before cluster
// provisioning and reused after cluster deletion.
type GcpFirewallRule struct {
	fieldSet_    []bool
	id           string
	href         string
	gcpNetwork   *GcpFirewallRuleGcpNetwork
	name         string
	organization *OrganizationLink
	profile      string
	status       *GcpFirewallRuleStatus
	template     *GcpFirewallRuleTemplateRef
	wifConfig    *WifConfig
}

// Kind returns the name of the type of the object.
func (o *GcpFirewallRule) Kind() string {
	if o == nil {
		return GcpFirewallRuleNilKind
	}
	if len(o.fieldSet_) > 0 && o.fieldSet_[0] {
		return GcpFirewallRuleLinkKind
	}
	return GcpFirewallRuleKind
}

// Link returns true if this is a link.
func (o *GcpFirewallRule) Link() bool {
	return o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0]
}

// ID returns the identifier of the object.
func (o *GcpFirewallRule) ID() string {
	if o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1] {
		return o.id
	}
	return ""
}

// GetID returns the identifier of the object and a flag indicating if the
// identifier has a value.
func (o *GcpFirewallRule) GetID() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1]
	if ok {
		value = o.id
	}
	return
}

// HREF returns the link to the object.
func (o *GcpFirewallRule) HREF() string {
	if o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2] {
		return o.href
	}
	return ""
}

// GetHREF returns the link of the object and a flag indicating if the
// link has a value.
func (o *GcpFirewallRule) GetHREF() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2]
	if ok {
		value = o.href
	}
	return
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *GcpFirewallRule) Empty() bool {
	if o == nil || len(o.fieldSet_) == 0 {
		return true
	}

	// Check all fields except the link flag (index 0)
	for i := 1; i < len(o.fieldSet_); i++ {
		if o.fieldSet_[i] {
			return false
		}
	}
	return true
}

// GcpNetwork returns the value of the 'gcp_network' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// GCP network configuration.
func (o *GcpFirewallRule) GcpNetwork() *GcpFirewallRuleGcpNetwork {
	if o != nil && len(o.fieldSet_) > 3 && o.fieldSet_[3] {
		return o.gcpNetwork
	}
	return nil
}

// GetGcpNetwork returns the value of the 'gcp_network' attribute and
// a flag indicating if the attribute has a value.
//
// GCP network configuration.
func (o *GcpFirewallRule) GetGcpNetwork() (value *GcpFirewallRuleGcpNetwork, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 3 && o.fieldSet_[3]
	if ok {
		value = o.gcpNetwork
	}
	return
}

// Name returns the value of the 'name' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Firewall rule name (unique per organization).
func (o *GcpFirewallRule) Name() string {
	if o != nil && len(o.fieldSet_) > 4 && o.fieldSet_[4] {
		return o.name
	}
	return ""
}

// GetName returns the value of the 'name' attribute and
// a flag indicating if the attribute has a value.
//
// Firewall rule name (unique per organization).
func (o *GcpFirewallRule) GetName() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 4 && o.fieldSet_[4]
	if ok {
		value = o.name
	}
	return
}

// Organization returns the value of the 'organization' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Link to the organization that owns this firewall rule.
func (o *GcpFirewallRule) Organization() *OrganizationLink {
	if o != nil && len(o.fieldSet_) > 5 && o.fieldSet_[5] {
		return o.organization
	}
	return nil
}

// GetOrganization returns the value of the 'organization' attribute and
// a flag indicating if the attribute has a value.
//
// Link to the organization that owns this firewall rule.
func (o *GcpFirewallRule) GetOrganization() (value *OrganizationLink, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 5 && o.fieldSet_[5]
	if ok {
		value = o.organization
	}
	return
}

// Profile returns the value of the 'profile' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Profile type (e.g. "public", "private").
func (o *GcpFirewallRule) Profile() string {
	if o != nil && len(o.fieldSet_) > 6 && o.fieldSet_[6] {
		return o.profile
	}
	return ""
}

// GetProfile returns the value of the 'profile' attribute and
// a flag indicating if the attribute has a value.
//
// Profile type (e.g. "public", "private").
func (o *GcpFirewallRule) GetProfile() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 6 && o.fieldSet_[6]
	if ok {
		value = o.profile
	}
	return
}

// Status returns the value of the 'status' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Status containing the rendered firewall rules.
func (o *GcpFirewallRule) Status() *GcpFirewallRuleStatus {
	if o != nil && len(o.fieldSet_) > 7 && o.fieldSet_[7] {
		return o.status
	}
	return nil
}

// GetStatus returns the value of the 'status' attribute and
// a flag indicating if the attribute has a value.
//
// Status containing the rendered firewall rules.
func (o *GcpFirewallRule) GetStatus() (value *GcpFirewallRuleStatus, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 7 && o.fieldSet_[7]
	if ok {
		value = o.status
	}
	return
}

// Template returns the value of the 'template' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Template version and profile reference.
func (o *GcpFirewallRule) Template() *GcpFirewallRuleTemplateRef {
	if o != nil && len(o.fieldSet_) > 8 && o.fieldSet_[8] {
		return o.template
	}
	return nil
}

// GetTemplate returns the value of the 'template' attribute and
// a flag indicating if the attribute has a value.
//
// Template version and profile reference.
func (o *GcpFirewallRule) GetTemplate() (value *GcpFirewallRuleTemplateRef, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 8 && o.fieldSet_[8]
	if ok {
		value = o.template
	}
	return
}

// WifConfig returns the value of the 'wif_config' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Link to the WIF config used by this firewall rule.
func (o *GcpFirewallRule) WifConfig() *WifConfig {
	if o != nil && len(o.fieldSet_) > 9 && o.fieldSet_[9] {
		return o.wifConfig
	}
	return nil
}

// GetWifConfig returns the value of the 'wif_config' attribute and
// a flag indicating if the attribute has a value.
//
// Link to the WIF config used by this firewall rule.
func (o *GcpFirewallRule) GetWifConfig() (value *WifConfig, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 9 && o.fieldSet_[9]
	if ok {
		value = o.wifConfig
	}
	return
}

// GcpFirewallRuleListKind is the name of the type used to represent list of objects of
// type 'gcp_firewall_rule'.
const GcpFirewallRuleListKind = "GcpFirewallRuleList"

// GcpFirewallRuleListLinkKind is the name of the type used to represent links to list
// of objects of type 'gcp_firewall_rule'.
const GcpFirewallRuleListLinkKind = "GcpFirewallRuleListLink"

// GcpFirewallRuleNilKind is the name of the type used to nil lists of objects of
// type 'gcp_firewall_rule'.
const GcpFirewallRuleListNilKind = "GcpFirewallRuleListNil"

// GcpFirewallRuleList is a list of values of the 'gcp_firewall_rule' type.
type GcpFirewallRuleList struct {
	href  string
	link  bool
	items []*GcpFirewallRule
}

// Kind returns the name of the type of the object.
func (l *GcpFirewallRuleList) Kind() string {
	if l == nil {
		return GcpFirewallRuleListNilKind
	}
	if l.link {
		return GcpFirewallRuleListLinkKind
	}
	return GcpFirewallRuleListKind
}

// Link returns true iif this is a link.
func (l *GcpFirewallRuleList) Link() bool {
	return l != nil && l.link
}

// HREF returns the link to the list.
func (l *GcpFirewallRuleList) HREF() string {
	if l != nil {
		return l.href
	}
	return ""
}

// GetHREF returns the link of the list and a flag indicating if the
// link has a value.
func (l *GcpFirewallRuleList) GetHREF() (value string, ok bool) {
	ok = l != nil && l.href != ""
	if ok {
		value = l.href
	}
	return
}

// Len returns the length of the list.
func (l *GcpFirewallRuleList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Items sets the items of the list.
func (l *GcpFirewallRuleList) SetLink(link bool) {
	l.link = link
}

// Items sets the items of the list.
func (l *GcpFirewallRuleList) SetHREF(href string) {
	l.href = href
}

// Items sets the items of the list.
func (l *GcpFirewallRuleList) SetItems(items []*GcpFirewallRule) {
	l.items = items
}

// Items returns the items of the list.
func (l *GcpFirewallRuleList) Items() []*GcpFirewallRule {
	if l == nil {
		return nil
	}
	return l.items
}

// Empty returns true if the list is empty.
func (l *GcpFirewallRuleList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *GcpFirewallRuleList) Get(i int) *GcpFirewallRule {
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
func (l *GcpFirewallRuleList) Slice() []*GcpFirewallRule {
	var slice []*GcpFirewallRule
	if l == nil {
		slice = make([]*GcpFirewallRule, 0)
	} else {
		slice = make([]*GcpFirewallRule, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *GcpFirewallRuleList) Each(f func(item *GcpFirewallRule) bool) {
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
func (l *GcpFirewallRuleList) Range(f func(index int, item *GcpFirewallRule) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
