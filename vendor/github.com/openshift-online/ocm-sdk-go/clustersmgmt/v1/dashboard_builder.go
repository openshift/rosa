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

// DashboardBuilder contains the data and logic needed to build 'dashboard' objects.
//
// Collection of metrics intended to render a graphical dashboard.
type DashboardBuilder struct {
	bitmap_ uint32
	id      string
	href    string
	metrics []*MetricBuilder
	name    string
}

// NewDashboard creates a new builder of 'dashboard' objects.
func NewDashboard() *DashboardBuilder {
	return &DashboardBuilder{}
}

// Link sets the flag that indicates if this is a link.
func (b *DashboardBuilder) Link(value bool) *DashboardBuilder {
	b.bitmap_ |= 1
	return b
}

// ID sets the identifier of the object.
func (b *DashboardBuilder) ID(value string) *DashboardBuilder {
	b.id = value
	b.bitmap_ |= 2
	return b
}

// HREF sets the link to the object.
func (b *DashboardBuilder) HREF(value string) *DashboardBuilder {
	b.href = value
	b.bitmap_ |= 4
	return b
}

// Metrics sets the value of the 'metrics' attribute to the given values.
//
//
func (b *DashboardBuilder) Metrics(values ...*MetricBuilder) *DashboardBuilder {
	b.metrics = make([]*MetricBuilder, len(values))
	copy(b.metrics, values)
	b.bitmap_ |= 8
	return b
}

// Name sets the value of the 'name' attribute to the given value.
//
//
func (b *DashboardBuilder) Name(value string) *DashboardBuilder {
	b.name = value
	b.bitmap_ |= 16
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *DashboardBuilder) Copy(object *Dashboard) *DashboardBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.id = object.id
	b.href = object.href
	if object.metrics != nil {
		b.metrics = make([]*MetricBuilder, len(object.metrics))
		for i, v := range object.metrics {
			b.metrics[i] = NewMetric().Copy(v)
		}
	} else {
		b.metrics = nil
	}
	b.name = object.name
	return b
}

// Build creates a 'dashboard' object using the configuration stored in the builder.
func (b *DashboardBuilder) Build() (object *Dashboard, err error) {
	object = new(Dashboard)
	object.id = b.id
	object.href = b.href
	object.bitmap_ = b.bitmap_
	if b.metrics != nil {
		object.metrics = make([]*Metric, len(b.metrics))
		for i, v := range b.metrics {
			object.metrics[i], err = v.Build()
			if err != nil {
				return
			}
		}
	}
	object.name = b.name
	return
}
