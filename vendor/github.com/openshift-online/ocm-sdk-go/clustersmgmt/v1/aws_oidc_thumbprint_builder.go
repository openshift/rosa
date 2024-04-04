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

// AwsOidcThumbprintBuilder contains the data and logic needed to build 'aws_oidc_thumbprint' objects.
//
// Thumbprint of the cluster's OpenID Connect identity provider
type AwsOidcThumbprintBuilder struct {
	bitmap_    uint32
	issuerUrl  string
	thumbprint string
}

// NewAwsOidcThumbprint creates a new builder of 'aws_oidc_thumbprint' objects.
func NewAwsOidcThumbprint() *AwsOidcThumbprintBuilder {
	return &AwsOidcThumbprintBuilder{}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *AwsOidcThumbprintBuilder) Empty() bool {
	return b == nil || b.bitmap_ == 0
}

// IssuerUrl sets the value of the 'issuer_url' attribute to the given value.
func (b *AwsOidcThumbprintBuilder) IssuerUrl(value string) *AwsOidcThumbprintBuilder {
	b.issuerUrl = value
	b.bitmap_ |= 1
	return b
}

// Thumbprint sets the value of the 'thumbprint' attribute to the given value.
func (b *AwsOidcThumbprintBuilder) Thumbprint(value string) *AwsOidcThumbprintBuilder {
	b.thumbprint = value
	b.bitmap_ |= 2
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *AwsOidcThumbprintBuilder) Copy(object *AwsOidcThumbprint) *AwsOidcThumbprintBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.issuerUrl = object.issuerUrl
	b.thumbprint = object.thumbprint
	return b
}

// Build creates a 'aws_oidc_thumbprint' object using the configuration stored in the builder.
func (b *AwsOidcThumbprintBuilder) Build() (object *AwsOidcThumbprint, err error) {
	object = new(AwsOidcThumbprint)
	object.bitmap_ = b.bitmap_
	object.issuerUrl = b.issuerUrl
	object.thumbprint = b.thumbprint
	return
}
