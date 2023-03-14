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

import (
	time "time"
)

// HostedOidcConfig represents the values of the 'hosted_oidc_config' type.
//
// Contains the necessary attributes to support oidc configuration hosting under Red Hat.
type HostedOidcConfig struct {
	bitmap_                 uint32
	href                    string
	id                      string
	creationTimestamp       time.Time
	installerRoleArn        string
	oidcEndpointUrl         string
	oidcFolderName          string
	oidcPrivateKeySecretArn string
	organizationId          string
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *HostedOidcConfig) Empty() bool {
	return o == nil || o.bitmap_ == 0
}

// HREF returns the value of the 'HREF' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// HREF for the hosted oidc config, filled in response.
func (o *HostedOidcConfig) HREF() string {
	if o != nil && o.bitmap_&1 != 0 {
		return o.href
	}
	return ""
}

// GetHREF returns the value of the 'HREF' attribute and
// a flag indicating if the attribute has a value.
//
// HREF for the hosted oidc config, filled in response.
func (o *HostedOidcConfig) GetHREF() (value string, ok bool) {
	ok = o != nil && o.bitmap_&1 != 0
	if ok {
		value = o.href
	}
	return
}

// ID returns the value of the 'ID' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// ID for the hosted oidc config, filled in response.
func (o *HostedOidcConfig) ID() string {
	if o != nil && o.bitmap_&2 != 0 {
		return o.id
	}
	return ""
}

// GetID returns the value of the 'ID' attribute and
// a flag indicating if the attribute has a value.
//
// ID for the hosted oidc config, filled in response.
func (o *HostedOidcConfig) GetID() (value string, ok bool) {
	ok = o != nil && o.bitmap_&2 != 0
	if ok {
		value = o.id
	}
	return
}

// CreationTimestamp returns the value of the 'creation_timestamp' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Creation timestamp, filled in response.
func (o *HostedOidcConfig) CreationTimestamp() time.Time {
	if o != nil && o.bitmap_&4 != 0 {
		return o.creationTimestamp
	}
	return time.Time{}
}

// GetCreationTimestamp returns the value of the 'creation_timestamp' attribute and
// a flag indicating if the attribute has a value.
//
// Creation timestamp, filled in response.
func (o *HostedOidcConfig) GetCreationTimestamp() (value time.Time, ok bool) {
	ok = o != nil && o.bitmap_&4 != 0
	if ok {
		value = o.creationTimestamp
	}
	return
}

// InstallerRoleArn returns the value of the 'installer_role_arn' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// ARN of the AWS role to assume when installing the cluster, supplied in request.
func (o *HostedOidcConfig) InstallerRoleArn() string {
	if o != nil && o.bitmap_&8 != 0 {
		return o.installerRoleArn
	}
	return ""
}

// GetInstallerRoleArn returns the value of the 'installer_role_arn' attribute and
// a flag indicating if the attribute has a value.
//
// ARN of the AWS role to assume when installing the cluster, supplied in request.
func (o *HostedOidcConfig) GetInstallerRoleArn() (value string, ok bool) {
	ok = o != nil && o.bitmap_&8 != 0
	if ok {
		value = o.installerRoleArn
	}
	return
}

// OidcEndpointUrl returns the value of the 'oidc_endpoint_url' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Oidc endpoint URL, filled in response base on secret arn for the private key.
func (o *HostedOidcConfig) OidcEndpointUrl() string {
	if o != nil && o.bitmap_&16 != 0 {
		return o.oidcEndpointUrl
	}
	return ""
}

// GetOidcEndpointUrl returns the value of the 'oidc_endpoint_url' attribute and
// a flag indicating if the attribute has a value.
//
// Oidc endpoint URL, filled in response base on secret arn for the private key.
func (o *HostedOidcConfig) GetOidcEndpointUrl() (value string, ok bool) {
	ok = o != nil && o.bitmap_&16 != 0
	if ok {
		value = o.oidcEndpointUrl
	}
	return
}

// OidcFolderName returns the value of the 'oidc_folder_name' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Name for the oidc folder, filled in response based on secret arn private key.
func (o *HostedOidcConfig) OidcFolderName() string {
	if o != nil && o.bitmap_&32 != 0 {
		return o.oidcFolderName
	}
	return ""
}

// GetOidcFolderName returns the value of the 'oidc_folder_name' attribute and
// a flag indicating if the attribute has a value.
//
// Name for the oidc folder, filled in response based on secret arn private key.
func (o *HostedOidcConfig) GetOidcFolderName() (value string, ok bool) {
	ok = o != nil && o.bitmap_&32 != 0
	if ok {
		value = o.oidcFolderName
	}
	return
}

// OidcPrivateKeySecretArn returns the value of the 'oidc_private_key_secret_arn' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Secrets Manager ARN for the OIDC private key key, supplied in request.
func (o *HostedOidcConfig) OidcPrivateKeySecretArn() string {
	if o != nil && o.bitmap_&64 != 0 {
		return o.oidcPrivateKeySecretArn
	}
	return ""
}

// GetOidcPrivateKeySecretArn returns the value of the 'oidc_private_key_secret_arn' attribute and
// a flag indicating if the attribute has a value.
//
// Secrets Manager ARN for the OIDC private key key, supplied in request.
func (o *HostedOidcConfig) GetOidcPrivateKeySecretArn() (value string, ok bool) {
	ok = o != nil && o.bitmap_&64 != 0
	if ok {
		value = o.oidcPrivateKeySecretArn
	}
	return
}

// OrganizationId returns the value of the 'organization_id' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Organization ID, filled in response respecting token provided.
func (o *HostedOidcConfig) OrganizationId() string {
	if o != nil && o.bitmap_&128 != 0 {
		return o.organizationId
	}
	return ""
}

// GetOrganizationId returns the value of the 'organization_id' attribute and
// a flag indicating if the attribute has a value.
//
// Organization ID, filled in response respecting token provided.
func (o *HostedOidcConfig) GetOrganizationId() (value string, ok bool) {
	ok = o != nil && o.bitmap_&128 != 0
	if ok {
		value = o.organizationId
	}
	return
}

// HostedOidcConfigListKind is the name of the type used to represent list of objects of
// type 'hosted_oidc_config'.
const HostedOidcConfigListKind = "HostedOidcConfigList"

// HostedOidcConfigListLinkKind is the name of the type used to represent links to list
// of objects of type 'hosted_oidc_config'.
const HostedOidcConfigListLinkKind = "HostedOidcConfigListLink"

// HostedOidcConfigNilKind is the name of the type used to nil lists of objects of
// type 'hosted_oidc_config'.
const HostedOidcConfigListNilKind = "HostedOidcConfigListNil"

// HostedOidcConfigList is a list of values of the 'hosted_oidc_config' type.
type HostedOidcConfigList struct {
	href  string
	link  bool
	items []*HostedOidcConfig
}

// Len returns the length of the list.
func (l *HostedOidcConfigList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Empty returns true if the list is empty.
func (l *HostedOidcConfigList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *HostedOidcConfigList) Get(i int) *HostedOidcConfig {
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
func (l *HostedOidcConfigList) Slice() []*HostedOidcConfig {
	var slice []*HostedOidcConfig
	if l == nil {
		slice = make([]*HostedOidcConfig, 0)
	} else {
		slice = make([]*HostedOidcConfig, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *HostedOidcConfigList) Each(f func(item *HostedOidcConfig) bool) {
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
func (l *HostedOidcConfigList) Range(f func(index int, item *HostedOidcConfig) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
