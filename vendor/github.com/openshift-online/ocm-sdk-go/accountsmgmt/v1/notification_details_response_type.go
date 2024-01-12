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

package v1 // github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1

// NotificationDetailsResponse represents the values of the 'notification_details_response' type.
//
// This struct is a request to get a templated email to a user related to this.
// subscription/cluster.
type NotificationDetailsResponse struct {
	bitmap_       uint32
	associates    []string
	externalOrgID string
	recipients    []string
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *NotificationDetailsResponse) Empty() bool {
	return o == nil || o.bitmap_ == 0
}

// Associates returns the value of the 'associates' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Indicates a list of associates email address.
func (o *NotificationDetailsResponse) Associates() []string {
	if o != nil && o.bitmap_&1 != 0 {
		return o.associates
	}
	return nil
}

// GetAssociates returns the value of the 'associates' attribute and
// a flag indicating if the attribute has a value.
//
// Indicates a list of associates email address.
func (o *NotificationDetailsResponse) GetAssociates() (value []string, ok bool) {
	ok = o != nil && o.bitmap_&1 != 0
	if ok {
		value = o.associates
	}
	return
}

// ExternalOrgID returns the value of the 'external_org_ID' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Indicates the external organization id of the subscription.
func (o *NotificationDetailsResponse) ExternalOrgID() string {
	if o != nil && o.bitmap_&2 != 0 {
		return o.externalOrgID
	}
	return ""
}

// GetExternalOrgID returns the value of the 'external_org_ID' attribute and
// a flag indicating if the attribute has a value.
//
// Indicates the external organization id of the subscription.
func (o *NotificationDetailsResponse) GetExternalOrgID() (value string, ok bool) {
	ok = o != nil && o.bitmap_&2 != 0
	if ok {
		value = o.externalOrgID
	}
	return
}

// Recipients returns the value of the 'recipients' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Indicates a list of recipients username.
func (o *NotificationDetailsResponse) Recipients() []string {
	if o != nil && o.bitmap_&4 != 0 {
		return o.recipients
	}
	return nil
}

// GetRecipients returns the value of the 'recipients' attribute and
// a flag indicating if the attribute has a value.
//
// Indicates a list of recipients username.
func (o *NotificationDetailsResponse) GetRecipients() (value []string, ok bool) {
	ok = o != nil && o.bitmap_&4 != 0
	if ok {
		value = o.recipients
	}
	return
}

// NotificationDetailsResponseListKind is the name of the type used to represent list of objects of
// type 'notification_details_response'.
const NotificationDetailsResponseListKind = "NotificationDetailsResponseList"

// NotificationDetailsResponseListLinkKind is the name of the type used to represent links to list
// of objects of type 'notification_details_response'.
const NotificationDetailsResponseListLinkKind = "NotificationDetailsResponseListLink"

// NotificationDetailsResponseNilKind is the name of the type used to nil lists of objects of
// type 'notification_details_response'.
const NotificationDetailsResponseListNilKind = "NotificationDetailsResponseListNil"

// NotificationDetailsResponseList is a list of values of the 'notification_details_response' type.
type NotificationDetailsResponseList struct {
	href  string
	link  bool
	items []*NotificationDetailsResponse
}

// Len returns the length of the list.
func (l *NotificationDetailsResponseList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Empty returns true if the list is empty.
func (l *NotificationDetailsResponseList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *NotificationDetailsResponseList) Get(i int) *NotificationDetailsResponse {
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
func (l *NotificationDetailsResponseList) Slice() []*NotificationDetailsResponse {
	var slice []*NotificationDetailsResponse
	if l == nil {
		slice = make([]*NotificationDetailsResponse, 0)
	} else {
		slice = make([]*NotificationDetailsResponse, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *NotificationDetailsResponseList) Each(f func(item *NotificationDetailsResponse) bool) {
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
func (l *NotificationDetailsResponseList) Range(f func(index int, item *NotificationDetailsResponse) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
