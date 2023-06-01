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

// DNSDomainBuilder contains the data and logic needed to build 'DNS_domain' objects.
//
// Contains the properties of a DNS domain.
type DNSDomainBuilder struct {
	bitmap_          uint32
	id               string
	href             string
	clusterLink      *ClusterLinkBuilder
	organizationLink *OrganizationLinkBuilder
	reservedAt       time.Time
}

// NewDNSDomain creates a new builder of 'DNS_domain' objects.
func NewDNSDomain() *DNSDomainBuilder {
	return &DNSDomainBuilder{}
}

// Link sets the flag that indicates if this is a link.
func (b *DNSDomainBuilder) Link(value bool) *DNSDomainBuilder {
	b.bitmap_ |= 1
	return b
}

// ID sets the identifier of the object.
func (b *DNSDomainBuilder) ID(value string) *DNSDomainBuilder {
	b.id = value
	b.bitmap_ |= 2
	return b
}

// HREF sets the link to the object.
func (b *DNSDomainBuilder) HREF(value string) *DNSDomainBuilder {
	b.href = value
	b.bitmap_ |= 4
	return b
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *DNSDomainBuilder) Empty() bool {
	return b == nil || b.bitmap_&^1 == 0
}

// ClusterLink sets the value of the 'cluster_link' attribute to the given value.
//
// Definition of a cluster link.
func (b *DNSDomainBuilder) ClusterLink(value *ClusterLinkBuilder) *DNSDomainBuilder {
	b.clusterLink = value
	if value != nil {
		b.bitmap_ |= 8
	} else {
		b.bitmap_ &^= 8
	}
	return b
}

// OrganizationLink sets the value of the 'organization_link' attribute to the given value.
//
// Definition of an organization link.
func (b *DNSDomainBuilder) OrganizationLink(value *OrganizationLinkBuilder) *DNSDomainBuilder {
	b.organizationLink = value
	if value != nil {
		b.bitmap_ |= 16
	} else {
		b.bitmap_ &^= 16
	}
	return b
}

// ReservedAt sets the value of the 'reserved_at' attribute to the given value.
func (b *DNSDomainBuilder) ReservedAt(value time.Time) *DNSDomainBuilder {
	b.reservedAt = value
	b.bitmap_ |= 32
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *DNSDomainBuilder) Copy(object *DNSDomain) *DNSDomainBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.id = object.id
	b.href = object.href
	if object.clusterLink != nil {
		b.clusterLink = NewClusterLink().Copy(object.clusterLink)
	} else {
		b.clusterLink = nil
	}
	if object.organizationLink != nil {
		b.organizationLink = NewOrganizationLink().Copy(object.organizationLink)
	} else {
		b.organizationLink = nil
	}
	b.reservedAt = object.reservedAt
	return b
}

// Build creates a 'DNS_domain' object using the configuration stored in the builder.
func (b *DNSDomainBuilder) Build() (object *DNSDomain, err error) {
	object = new(DNSDomain)
	object.id = b.id
	object.href = b.href
	object.bitmap_ = b.bitmap_
	if b.clusterLink != nil {
		object.clusterLink, err = b.clusterLink.Build()
		if err != nil {
			return
		}
	}
	if b.organizationLink != nil {
		object.organizationLink, err = b.organizationLink.Build()
		if err != nil {
			return
		}
	}
	object.reservedAt = b.reservedAt
	return
}
