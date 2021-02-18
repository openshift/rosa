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

// ClusterMetricBuilder contains the data and logic needed to build 'cluster_metric' objects.
//
// Metric describing the total and used amount of some resource (like RAM, CPU and storage) in
// a cluster.
type ClusterMetricBuilder struct {
	bitmap_          uint32
	total            *ValueBuilder
	updatedTimestamp time.Time
	used             *ValueBuilder
}

// NewClusterMetric creates a new builder of 'cluster_metric' objects.
func NewClusterMetric() *ClusterMetricBuilder {
	return &ClusterMetricBuilder{}
}

// Total sets the value of the 'total' attribute to the given value.
//
// Numeric value and the unit used to measure it.
//
// Units are not mandatory, and they're not specified for some resources. For
// resources that use bytes, the accepted units are:
//
// - 1 B = 1 byte
// - 1 KB = 10^3 bytes
// - 1 MB = 10^6 bytes
// - 1 GB = 10^9 bytes
// - 1 TB = 10^12 bytes
// - 1 PB = 10^15 bytes
//
// - 1 B = 1 byte
// - 1 KiB = 2^10 bytes
// - 1 MiB = 2^20 bytes
// - 1 GiB = 2^30 bytes
// - 1 TiB = 2^40 bytes
// - 1 PiB = 2^50 bytes
func (b *ClusterMetricBuilder) Total(value *ValueBuilder) *ClusterMetricBuilder {
	b.total = value
	if value != nil {
		b.bitmap_ |= 1
	} else {
		b.bitmap_ &^= 1
	}
	return b
}

// UpdatedTimestamp sets the value of the 'updated_timestamp' attribute to the given value.
//
//
func (b *ClusterMetricBuilder) UpdatedTimestamp(value time.Time) *ClusterMetricBuilder {
	b.updatedTimestamp = value
	b.bitmap_ |= 2
	return b
}

// Used sets the value of the 'used' attribute to the given value.
//
// Numeric value and the unit used to measure it.
//
// Units are not mandatory, and they're not specified for some resources. For
// resources that use bytes, the accepted units are:
//
// - 1 B = 1 byte
// - 1 KB = 10^3 bytes
// - 1 MB = 10^6 bytes
// - 1 GB = 10^9 bytes
// - 1 TB = 10^12 bytes
// - 1 PB = 10^15 bytes
//
// - 1 B = 1 byte
// - 1 KiB = 2^10 bytes
// - 1 MiB = 2^20 bytes
// - 1 GiB = 2^30 bytes
// - 1 TiB = 2^40 bytes
// - 1 PiB = 2^50 bytes
func (b *ClusterMetricBuilder) Used(value *ValueBuilder) *ClusterMetricBuilder {
	b.used = value
	if value != nil {
		b.bitmap_ |= 4
	} else {
		b.bitmap_ &^= 4
	}
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *ClusterMetricBuilder) Copy(object *ClusterMetric) *ClusterMetricBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	if object.total != nil {
		b.total = NewValue().Copy(object.total)
	} else {
		b.total = nil
	}
	b.updatedTimestamp = object.updatedTimestamp
	if object.used != nil {
		b.used = NewValue().Copy(object.used)
	} else {
		b.used = nil
	}
	return b
}

// Build creates a 'cluster_metric' object using the configuration stored in the builder.
func (b *ClusterMetricBuilder) Build() (object *ClusterMetric, err error) {
	object = new(ClusterMetric)
	object.bitmap_ = b.bitmap_
	if b.total != nil {
		object.total, err = b.total.Build()
		if err != nil {
			return
		}
	}
	object.updatedTimestamp = b.updatedTimestamp
	if b.used != nil {
		object.used, err = b.used.Build()
		if err != nil {
			return
		}
	}
	return
}
