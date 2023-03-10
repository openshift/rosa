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

// HostedOidcConfigBuilder contains the data and logic needed to build 'hosted_oidc_config' objects.
//
// Contains the necessary attributes to support oidc configuration hosting under Red Hat.
type HostedOidcConfigBuilder struct {
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

// NewHostedOidcConfig creates a new builder of 'hosted_oidc_config' objects.
func NewHostedOidcConfig() *HostedOidcConfigBuilder {
	return &HostedOidcConfigBuilder{}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *HostedOidcConfigBuilder) Empty() bool {
	return b == nil || b.bitmap_ == 0
}

// HREF sets the value of the 'HREF' attribute to the given value.
func (b *HostedOidcConfigBuilder) HREF(value string) *HostedOidcConfigBuilder {
	b.href = value
	b.bitmap_ |= 1
	return b
}

// ID sets the value of the 'ID' attribute to the given value.
func (b *HostedOidcConfigBuilder) ID(value string) *HostedOidcConfigBuilder {
	b.id = value
	b.bitmap_ |= 2
	return b
}

// CreationTimestamp sets the value of the 'creation_timestamp' attribute to the given value.
func (b *HostedOidcConfigBuilder) CreationTimestamp(value time.Time) *HostedOidcConfigBuilder {
	b.creationTimestamp = value
	b.bitmap_ |= 4
	return b
}

// InstallerRoleArn sets the value of the 'installer_role_arn' attribute to the given value.
func (b *HostedOidcConfigBuilder) InstallerRoleArn(value string) *HostedOidcConfigBuilder {
	b.installerRoleArn = value
	b.bitmap_ |= 8
	return b
}

// OidcEndpointUrl sets the value of the 'oidc_endpoint_url' attribute to the given value.
func (b *HostedOidcConfigBuilder) OidcEndpointUrl(value string) *HostedOidcConfigBuilder {
	b.oidcEndpointUrl = value
	b.bitmap_ |= 16
	return b
}

// OidcFolderName sets the value of the 'oidc_folder_name' attribute to the given value.
func (b *HostedOidcConfigBuilder) OidcFolderName(value string) *HostedOidcConfigBuilder {
	b.oidcFolderName = value
	b.bitmap_ |= 32
	return b
}

// OidcPrivateKeySecretArn sets the value of the 'oidc_private_key_secret_arn' attribute to the given value.
func (b *HostedOidcConfigBuilder) OidcPrivateKeySecretArn(value string) *HostedOidcConfigBuilder {
	b.oidcPrivateKeySecretArn = value
	b.bitmap_ |= 64
	return b
}

// OrganizationId sets the value of the 'organization_id' attribute to the given value.
func (b *HostedOidcConfigBuilder) OrganizationId(value string) *HostedOidcConfigBuilder {
	b.organizationId = value
	b.bitmap_ |= 128
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *HostedOidcConfigBuilder) Copy(object *HostedOidcConfig) *HostedOidcConfigBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.href = object.href
	b.id = object.id
	b.creationTimestamp = object.creationTimestamp
	b.installerRoleArn = object.installerRoleArn
	b.oidcEndpointUrl = object.oidcEndpointUrl
	b.oidcFolderName = object.oidcFolderName
	b.oidcPrivateKeySecretArn = object.oidcPrivateKeySecretArn
	b.organizationId = object.organizationId
	return b
}

// Build creates a 'hosted_oidc_config' object using the configuration stored in the builder.
func (b *HostedOidcConfigBuilder) Build() (object *HostedOidcConfig, err error) {
	object = new(HostedOidcConfig)
	object.bitmap_ = b.bitmap_
	object.href = b.href
	object.id = b.id
	object.creationTimestamp = b.creationTimestamp
	object.installerRoleArn = b.installerRoleArn
	object.oidcEndpointUrl = b.oidcEndpointUrl
	object.oidcFolderName = b.oidcFolderName
	object.oidcPrivateKeySecretArn = b.oidcPrivateKeySecretArn
	object.organizationId = b.organizationId
	return
}
