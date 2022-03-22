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

package v1 // github.com/openshift-online/ocm-sdk-go/servicemgmt/v1

// ClusterBuilder contains the data and logic needed to build 'cluster' objects.
//
// This represents the parameters needed by Managed Service to create a cluster.
type ClusterBuilder struct {
	bitmap_     uint32
	aws         *AWSBuilder
	displayName string
	href        string
	id          string
	name        string
	properties  map[string]string
	region      *CloudRegionBuilder
	state       string
}

// NewCluster creates a new builder of 'cluster' objects.
func NewCluster() *ClusterBuilder {
	return &ClusterBuilder{}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *ClusterBuilder) Empty() bool {
	return b == nil || b.bitmap_ == 0
}

// AWS sets the value of the 'AWS' attribute to the given value.
//
// _Amazon Web Services_ specific settings of a cluster.
func (b *ClusterBuilder) AWS(value *AWSBuilder) *ClusterBuilder {
	b.aws = value
	if value != nil {
		b.bitmap_ |= 1
	} else {
		b.bitmap_ &^= 1
	}
	return b
}

// DisplayName sets the value of the 'display_name' attribute to the given value.
//
//
func (b *ClusterBuilder) DisplayName(value string) *ClusterBuilder {
	b.displayName = value
	b.bitmap_ |= 2
	return b
}

// Href sets the value of the 'href' attribute to the given value.
//
//
func (b *ClusterBuilder) Href(value string) *ClusterBuilder {
	b.href = value
	b.bitmap_ |= 4
	return b
}

// Id sets the value of the 'id' attribute to the given value.
//
//
func (b *ClusterBuilder) Id(value string) *ClusterBuilder {
	b.id = value
	b.bitmap_ |= 8
	return b
}

// Name sets the value of the 'name' attribute to the given value.
//
//
func (b *ClusterBuilder) Name(value string) *ClusterBuilder {
	b.name = value
	b.bitmap_ |= 16
	return b
}

// Properties sets the value of the 'properties' attribute to the given value.
//
//
func (b *ClusterBuilder) Properties(value map[string]string) *ClusterBuilder {
	b.properties = value
	if value != nil {
		b.bitmap_ |= 32
	} else {
		b.bitmap_ &^= 32
	}
	return b
}

// Region sets the value of the 'region' attribute to the given value.
//
// Description of a region of a cloud provider.
func (b *ClusterBuilder) Region(value *CloudRegionBuilder) *ClusterBuilder {
	b.region = value
	if value != nil {
		b.bitmap_ |= 64
	} else {
		b.bitmap_ &^= 64
	}
	return b
}

// State sets the value of the 'state' attribute to the given value.
//
//
func (b *ClusterBuilder) State(value string) *ClusterBuilder {
	b.state = value
	b.bitmap_ |= 128
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *ClusterBuilder) Copy(object *Cluster) *ClusterBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	if object.aws != nil {
		b.aws = NewAWS().Copy(object.aws)
	} else {
		b.aws = nil
	}
	b.displayName = object.displayName
	b.href = object.href
	b.id = object.id
	b.name = object.name
	if len(object.properties) > 0 {
		b.properties = map[string]string{}
		for k, v := range object.properties {
			b.properties[k] = v
		}
	} else {
		b.properties = nil
	}
	if object.region != nil {
		b.region = NewCloudRegion().Copy(object.region)
	} else {
		b.region = nil
	}
	b.state = object.state
	return b
}

// Build creates a 'cluster' object using the configuration stored in the builder.
func (b *ClusterBuilder) Build() (object *Cluster, err error) {
	object = new(Cluster)
	object.bitmap_ = b.bitmap_
	if b.aws != nil {
		object.aws, err = b.aws.Build()
		if err != nil {
			return
		}
	}
	object.displayName = b.displayName
	object.href = b.href
	object.id = b.id
	object.name = b.name
	if b.properties != nil {
		object.properties = make(map[string]string)
		for k, v := range b.properties {
			object.properties[k] = v
		}
	}
	if b.region != nil {
		object.region, err = b.region.Build()
		if err != nil {
			return
		}
	}
	object.state = b.state
	return
}
