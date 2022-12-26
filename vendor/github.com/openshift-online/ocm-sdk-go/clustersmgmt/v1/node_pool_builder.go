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

// NodePoolBuilder contains the data and logic needed to build 'node_pool' objects.
//
// Representation of a node pool in a cluster.
type NodePoolBuilder struct {
	bitmap_          uint32
	id               string
	href             string
	awsNodePool      *AWSNodePoolBuilder
	autoscaling      *NodePoolAutoscalingBuilder
	availabilityZone string
	cluster          *ClusterBuilder
	replicas         int
	subnet           string
	autoRepair       bool
}

// NewNodePool creates a new builder of 'node_pool' objects.
func NewNodePool() *NodePoolBuilder {
	return &NodePoolBuilder{}
}

// Link sets the flag that indicates if this is a link.
func (b *NodePoolBuilder) Link(value bool) *NodePoolBuilder {
	b.bitmap_ |= 1
	return b
}

// ID sets the identifier of the object.
func (b *NodePoolBuilder) ID(value string) *NodePoolBuilder {
	b.id = value
	b.bitmap_ |= 2
	return b
}

// HREF sets the link to the object.
func (b *NodePoolBuilder) HREF(value string) *NodePoolBuilder {
	b.href = value
	b.bitmap_ |= 4
	return b
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *NodePoolBuilder) Empty() bool {
	return b == nil || b.bitmap_&^1 == 0
}

// AWSNodePool sets the value of the 'AWS_node_pool' attribute to the given value.
//
// Representation of aws node pool specific parameters.
func (b *NodePoolBuilder) AWSNodePool(value *AWSNodePoolBuilder) *NodePoolBuilder {
	b.awsNodePool = value
	if value != nil {
		b.bitmap_ |= 8
	} else {
		b.bitmap_ &^= 8
	}
	return b
}

// AutoRepair sets the value of the 'auto_repair' attribute to the given value.
func (b *NodePoolBuilder) AutoRepair(value bool) *NodePoolBuilder {
	b.autoRepair = value
	b.bitmap_ |= 16
	return b
}

// Autoscaling sets the value of the 'autoscaling' attribute to the given value.
//
// Representation of a autoscaling in a node pool.
func (b *NodePoolBuilder) Autoscaling(value *NodePoolAutoscalingBuilder) *NodePoolBuilder {
	b.autoscaling = value
	if value != nil {
		b.bitmap_ |= 32
	} else {
		b.bitmap_ &^= 32
	}
	return b
}

// AvailabilityZone sets the value of the 'availability_zone' attribute to the given value.
func (b *NodePoolBuilder) AvailabilityZone(value string) *NodePoolBuilder {
	b.availabilityZone = value
	b.bitmap_ |= 64
	return b
}

// Cluster sets the value of the 'cluster' attribute to the given value.
//
// Definition of an _OpenShift_ cluster.
//
// The `cloud_provider` attribute is a reference to the cloud provider. When a
// cluster is retrieved it will be a link to the cloud provider, containing only
// the kind, id and href attributes:
//
// ```json
//
//	{
//	  "cloud_provider": {
//	    "kind": "CloudProviderLink",
//	    "id": "123",
//	    "href": "/api/clusters_mgmt/v1/cloud_providers/123"
//	  }
//	}
//
// ```
//
// When a cluster is created this is optional, and if used it should contain the
// identifier of the cloud provider to use:
//
// ```json
//
//	{
//	  "cloud_provider": {
//	    "id": "123",
//	  }
//	}
//
// ```
//
// If not included, then the cluster will be created using the default cloud
// provider, which is currently Amazon Web Services.
//
// The region attribute is mandatory when a cluster is created.
//
// The `aws.access_key_id`, `aws.secret_access_key` and `dns.base_domain`
// attributes are mandatory when creation a cluster with your own Amazon Web
// Services account.
func (b *NodePoolBuilder) Cluster(value *ClusterBuilder) *NodePoolBuilder {
	b.cluster = value
	if value != nil {
		b.bitmap_ |= 128
	} else {
		b.bitmap_ &^= 128
	}
	return b
}

// Replicas sets the value of the 'replicas' attribute to the given value.
func (b *NodePoolBuilder) Replicas(value int) *NodePoolBuilder {
	b.replicas = value
	b.bitmap_ |= 256
	return b
}

// Subnet sets the value of the 'subnet' attribute to the given value.
func (b *NodePoolBuilder) Subnet(value string) *NodePoolBuilder {
	b.subnet = value
	b.bitmap_ |= 512
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *NodePoolBuilder) Copy(object *NodePool) *NodePoolBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.id = object.id
	b.href = object.href
	if object.awsNodePool != nil {
		b.awsNodePool = NewAWSNodePool().Copy(object.awsNodePool)
	} else {
		b.awsNodePool = nil
	}
	b.autoRepair = object.autoRepair
	if object.autoscaling != nil {
		b.autoscaling = NewNodePoolAutoscaling().Copy(object.autoscaling)
	} else {
		b.autoscaling = nil
	}
	b.availabilityZone = object.availabilityZone
	if object.cluster != nil {
		b.cluster = NewCluster().Copy(object.cluster)
	} else {
		b.cluster = nil
	}
	b.replicas = object.replicas
	b.subnet = object.subnet
	return b
}

// Build creates a 'node_pool' object using the configuration stored in the builder.
func (b *NodePoolBuilder) Build() (object *NodePool, err error) {
	object = new(NodePool)
	object.id = b.id
	object.href = b.href
	object.bitmap_ = b.bitmap_
	if b.awsNodePool != nil {
		object.awsNodePool, err = b.awsNodePool.Build()
		if err != nil {
			return
		}
	}
	object.autoRepair = b.autoRepair
	if b.autoscaling != nil {
		object.autoscaling, err = b.autoscaling.Build()
		if err != nil {
			return
		}
	}
	object.availabilityZone = b.availabilityZone
	if b.cluster != nil {
		object.cluster, err = b.cluster.Build()
		if err != nil {
			return
		}
	}
	object.replicas = b.replicas
	object.subnet = b.subnet
	return
}
