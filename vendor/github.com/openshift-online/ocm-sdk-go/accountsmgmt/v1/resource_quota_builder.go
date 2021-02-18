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

import (
	time "time"
)

// ResourceQuotaBuilder contains the data and logic needed to build 'resource_quota' objects.
//
//
type ResourceQuotaBuilder struct {
	bitmap_              uint32
	id                   string
	href                 string
	sku                  string
	allowed              int
	availabilityZoneType string
	createdAt            time.Time
	organizationID       string
	resourceName         string
	resourceType         string
	skuCount             int
	type_                string
	updatedAt            time.Time
	byoc                 bool
}

// NewResourceQuota creates a new builder of 'resource_quota' objects.
func NewResourceQuota() *ResourceQuotaBuilder {
	return &ResourceQuotaBuilder{}
}

// Link sets the flag that indicates if this is a link.
func (b *ResourceQuotaBuilder) Link(value bool) *ResourceQuotaBuilder {
	b.bitmap_ |= 1
	return b
}

// ID sets the identifier of the object.
func (b *ResourceQuotaBuilder) ID(value string) *ResourceQuotaBuilder {
	b.id = value
	b.bitmap_ |= 2
	return b
}

// HREF sets the link to the object.
func (b *ResourceQuotaBuilder) HREF(value string) *ResourceQuotaBuilder {
	b.href = value
	b.bitmap_ |= 4
	return b
}

// BYOC sets the value of the 'BYOC' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) BYOC(value bool) *ResourceQuotaBuilder {
	b.byoc = value
	b.bitmap_ |= 8
	return b
}

// SKU sets the value of the 'SKU' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) SKU(value string) *ResourceQuotaBuilder {
	b.sku = value
	b.bitmap_ |= 16
	return b
}

// Allowed sets the value of the 'allowed' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) Allowed(value int) *ResourceQuotaBuilder {
	b.allowed = value
	b.bitmap_ |= 32
	return b
}

// AvailabilityZoneType sets the value of the 'availability_zone_type' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) AvailabilityZoneType(value string) *ResourceQuotaBuilder {
	b.availabilityZoneType = value
	b.bitmap_ |= 64
	return b
}

// CreatedAt sets the value of the 'created_at' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) CreatedAt(value time.Time) *ResourceQuotaBuilder {
	b.createdAt = value
	b.bitmap_ |= 128
	return b
}

// OrganizationID sets the value of the 'organization_ID' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) OrganizationID(value string) *ResourceQuotaBuilder {
	b.organizationID = value
	b.bitmap_ |= 256
	return b
}

// ResourceName sets the value of the 'resource_name' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) ResourceName(value string) *ResourceQuotaBuilder {
	b.resourceName = value
	b.bitmap_ |= 512
	return b
}

// ResourceType sets the value of the 'resource_type' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) ResourceType(value string) *ResourceQuotaBuilder {
	b.resourceType = value
	b.bitmap_ |= 1024
	return b
}

// SkuCount sets the value of the 'sku_count' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) SkuCount(value int) *ResourceQuotaBuilder {
	b.skuCount = value
	b.bitmap_ |= 2048
	return b
}

// Type sets the value of the 'type' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) Type(value string) *ResourceQuotaBuilder {
	b.type_ = value
	b.bitmap_ |= 4096
	return b
}

// UpdatedAt sets the value of the 'updated_at' attribute to the given value.
//
//
func (b *ResourceQuotaBuilder) UpdatedAt(value time.Time) *ResourceQuotaBuilder {
	b.updatedAt = value
	b.bitmap_ |= 8192
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *ResourceQuotaBuilder) Copy(object *ResourceQuota) *ResourceQuotaBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.id = object.id
	b.href = object.href
	b.byoc = object.byoc
	b.sku = object.sku
	b.allowed = object.allowed
	b.availabilityZoneType = object.availabilityZoneType
	b.createdAt = object.createdAt
	b.organizationID = object.organizationID
	b.resourceName = object.resourceName
	b.resourceType = object.resourceType
	b.skuCount = object.skuCount
	b.type_ = object.type_
	b.updatedAt = object.updatedAt
	return b
}

// Build creates a 'resource_quota' object using the configuration stored in the builder.
func (b *ResourceQuotaBuilder) Build() (object *ResourceQuota, err error) {
	object = new(ResourceQuota)
	object.id = b.id
	object.href = b.href
	object.bitmap_ = b.bitmap_
	object.byoc = b.byoc
	object.sku = b.sku
	object.allowed = b.allowed
	object.availabilityZoneType = b.availabilityZoneType
	object.createdAt = b.createdAt
	object.organizationID = b.organizationID
	object.resourceName = b.resourceName
	object.resourceType = b.resourceType
	object.skuCount = b.skuCount
	object.type_ = b.type_
	object.updatedAt = b.updatedAt
	return
}
