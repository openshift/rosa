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

// BillingModelItem represents the values of the 'billing_model_item' type.
//
// BillingModelItem that represents a billing model (defined in pkg/api/billing_types.go). Using BillingModelItem to keep backwards compatibility as we already have a BillingModel enum defined.
type BillingModelItem struct {
	bitmap_     uint32
	href        string
	description string
	displayName string
	id          string
	marketplace string
	model       string
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *BillingModelItem) Empty() bool {
	return o == nil || o.bitmap_ == 0
}

// HREF returns the value of the 'HREF' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// HREF for the billing model
func (o *BillingModelItem) HREF() string {
	if o != nil && o.bitmap_&1 != 0 {
		return o.href
	}
	return ""
}

// GetHREF returns the value of the 'HREF' attribute and
// a flag indicating if the attribute has a value.
//
// HREF for the billing model
func (o *BillingModelItem) GetHREF() (value string, ok bool) {
	ok = o != nil && o.bitmap_&1 != 0
	if ok {
		value = o.href
	}
	return
}

// Description returns the value of the 'description' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Single line description of the billing model for better understanding.
func (o *BillingModelItem) Description() string {
	if o != nil && o.bitmap_&2 != 0 {
		return o.description
	}
	return ""
}

// GetDescription returns the value of the 'description' attribute and
// a flag indicating if the attribute has a value.
//
// Single line description of the billing model for better understanding.
func (o *BillingModelItem) GetDescription() (value string, ok bool) {
	ok = o != nil && o.bitmap_&2 != 0
	if ok {
		value = o.description
	}
	return
}

// DisplayName returns the value of the 'display_name' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// User friendly display name of the billing model.
func (o *BillingModelItem) DisplayName() string {
	if o != nil && o.bitmap_&4 != 0 {
		return o.displayName
	}
	return ""
}

// GetDisplayName returns the value of the 'display_name' attribute and
// a flag indicating if the attribute has a value.
//
// User friendly display name of the billing model.
func (o *BillingModelItem) GetDisplayName() (value string, ok bool) {
	ok = o != nil && o.bitmap_&4 != 0
	if ok {
		value = o.displayName
	}
	return
}

// Id returns the value of the 'id' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Id for the BillingModel.
func (o *BillingModelItem) Id() string {
	if o != nil && o.bitmap_&8 != 0 {
		return o.id
	}
	return ""
}

// GetId returns the value of the 'id' attribute and
// a flag indicating if the attribute has a value.
//
// Id for the BillingModel.
func (o *BillingModelItem) GetId() (value string, ok bool) {
	ok = o != nil && o.bitmap_&8 != 0
	if ok {
		value = o.id
	}
	return
}

// Marketplace returns the value of the 'marketplace' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Indicates the marketplace of the billing model. e.g. gcp, aws, etc.
func (o *BillingModelItem) Marketplace() string {
	if o != nil && o.bitmap_&16 != 0 {
		return o.marketplace
	}
	return ""
}

// GetMarketplace returns the value of the 'marketplace' attribute and
// a flag indicating if the attribute has a value.
//
// Indicates the marketplace of the billing model. e.g. gcp, aws, etc.
func (o *BillingModelItem) GetMarketplace() (value string, ok bool) {
	ok = o != nil && o.bitmap_&16 != 0
	if ok {
		value = o.marketplace
	}
	return
}

// Model returns the value of the 'model' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Model type of the BillingModel. e.g. standard, marketplace, marketplace-aws, etc.
func (o *BillingModelItem) Model() string {
	if o != nil && o.bitmap_&32 != 0 {
		return o.model
	}
	return ""
}

// GetModel returns the value of the 'model' attribute and
// a flag indicating if the attribute has a value.
//
// Model type of the BillingModel. e.g. standard, marketplace, marketplace-aws, etc.
func (o *BillingModelItem) GetModel() (value string, ok bool) {
	ok = o != nil && o.bitmap_&32 != 0
	if ok {
		value = o.model
	}
	return
}

// BillingModelItemListKind is the name of the type used to represent list of objects of
// type 'billing_model_item'.
const BillingModelItemListKind = "BillingModelItemList"

// BillingModelItemListLinkKind is the name of the type used to represent links to list
// of objects of type 'billing_model_item'.
const BillingModelItemListLinkKind = "BillingModelItemListLink"

// BillingModelItemNilKind is the name of the type used to nil lists of objects of
// type 'billing_model_item'.
const BillingModelItemListNilKind = "BillingModelItemListNil"

// BillingModelItemList is a list of values of the 'billing_model_item' type.
type BillingModelItemList struct {
	href  string
	link  bool
	items []*BillingModelItem
}

// Len returns the length of the list.
func (l *BillingModelItemList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Empty returns true if the list is empty.
func (l *BillingModelItemList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *BillingModelItemList) Get(i int) *BillingModelItem {
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
func (l *BillingModelItemList) Slice() []*BillingModelItem {
	var slice []*BillingModelItem
	if l == nil {
		slice = make([]*BillingModelItem, 0)
	} else {
		slice = make([]*BillingModelItem, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *BillingModelItemList) Each(f func(item *BillingModelItem) bool) {
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
func (l *BillingModelItemList) Range(f func(index int, item *BillingModelItem) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
