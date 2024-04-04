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

package v1 // github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1

// AwsOidcThumbprint represents the values of the 'aws_oidc_thumbprint' type.
//
// Thumbprint of the cluster's OpenID Connect identity provider
type AwsOidcThumbprint struct {
	bitmap_    uint32
	issuerUrl  string
	thumbprint string
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *AwsOidcThumbprint) Empty() bool {
	return o == nil || o.bitmap_ == 0
}

// IssuerUrl returns the value of the 'issuer_url' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Issuer URL
func (o *AwsOidcThumbprint) IssuerUrl() string {
	if o != nil && o.bitmap_&1 != 0 {
		return o.issuerUrl
	}
	return ""
}

// GetIssuerUrl returns the value of the 'issuer_url' attribute and
// a flag indicating if the attribute has a value.
//
// Issuer URL
func (o *AwsOidcThumbprint) GetIssuerUrl() (value string, ok bool) {
	ok = o != nil && o.bitmap_&1 != 0
	if ok {
		value = o.issuerUrl
	}
	return
}

// Thumbprint returns the value of the 'thumbprint' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// SHA1 of the certificate of the top intermediate CA in the certificate authority chain
func (o *AwsOidcThumbprint) Thumbprint() string {
	if o != nil && o.bitmap_&2 != 0 {
		return o.thumbprint
	}
	return ""
}

// GetThumbprint returns the value of the 'thumbprint' attribute and
// a flag indicating if the attribute has a value.
//
// SHA1 of the certificate of the top intermediate CA in the certificate authority chain
func (o *AwsOidcThumbprint) GetThumbprint() (value string, ok bool) {
	ok = o != nil && o.bitmap_&2 != 0
	if ok {
		value = o.thumbprint
	}
	return
}

// AwsOidcThumbprintListKind is the name of the type used to represent list of objects of
// type 'aws_oidc_thumbprint'.
const AwsOidcThumbprintListKind = "AwsOidcThumbprintList"

// AwsOidcThumbprintListLinkKind is the name of the type used to represent links to list
// of objects of type 'aws_oidc_thumbprint'.
const AwsOidcThumbprintListLinkKind = "AwsOidcThumbprintListLink"

// AwsOidcThumbprintNilKind is the name of the type used to nil lists of objects of
// type 'aws_oidc_thumbprint'.
const AwsOidcThumbprintListNilKind = "AwsOidcThumbprintListNil"

// AwsOidcThumbprintList is a list of values of the 'aws_oidc_thumbprint' type.
type AwsOidcThumbprintList struct {
	href  string
	link  bool
	items []*AwsOidcThumbprint
}

// Len returns the length of the list.
func (l *AwsOidcThumbprintList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Empty returns true if the list is empty.
func (l *AwsOidcThumbprintList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *AwsOidcThumbprintList) Get(i int) *AwsOidcThumbprint {
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
func (l *AwsOidcThumbprintList) Slice() []*AwsOidcThumbprint {
	var slice []*AwsOidcThumbprint
	if l == nil {
		slice = make([]*AwsOidcThumbprint, 0)
	} else {
		slice = make([]*AwsOidcThumbprint, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *AwsOidcThumbprintList) Each(f func(item *AwsOidcThumbprint) bool) {
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
func (l *AwsOidcThumbprintList) Range(f func(index int, item *AwsOidcThumbprint) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
