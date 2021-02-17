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

// ClusterMetricsBuilder contains the data and logic needed to build 'cluster_metrics' objects.
//
// Cluster metrics received via telemetry.
type ClusterMetricsBuilder struct {
	bitmap_                   uint32
	cpu                       *ClusterMetricBuilder
	computeNodesCPU           *ClusterMetricBuilder
	computeNodesMemory        *ClusterMetricBuilder
	computeNodesSockets       *ClusterMetricBuilder
	criticalAlertsFiring      int
	memory                    *ClusterMetricBuilder
	nodes                     *ClusterNodesBuilder
	operatorsConditionFailing int
	sockets                   *ClusterMetricBuilder
	storage                   *ClusterMetricBuilder
}

// NewClusterMetrics creates a new builder of 'cluster_metrics' objects.
func NewClusterMetrics() *ClusterMetricsBuilder {
	return &ClusterMetricsBuilder{}
}

// CPU sets the value of the 'CPU' attribute to the given value.
//
// Metric describing the total and used amount of some resource (like RAM, CPU and storage) in
// a cluster.
func (b *ClusterMetricsBuilder) CPU(value *ClusterMetricBuilder) *ClusterMetricsBuilder {
	b.cpu = value
	if value != nil {
		b.bitmap_ |= 1
	} else {
		b.bitmap_ &^= 1
	}
	return b
}

// ComputeNodesCPU sets the value of the 'compute_nodes_CPU' attribute to the given value.
//
// Metric describing the total and used amount of some resource (like RAM, CPU and storage) in
// a cluster.
func (b *ClusterMetricsBuilder) ComputeNodesCPU(value *ClusterMetricBuilder) *ClusterMetricsBuilder {
	b.computeNodesCPU = value
	if value != nil {
		b.bitmap_ |= 2
	} else {
		b.bitmap_ &^= 2
	}
	return b
}

// ComputeNodesMemory sets the value of the 'compute_nodes_memory' attribute to the given value.
//
// Metric describing the total and used amount of some resource (like RAM, CPU and storage) in
// a cluster.
func (b *ClusterMetricsBuilder) ComputeNodesMemory(value *ClusterMetricBuilder) *ClusterMetricsBuilder {
	b.computeNodesMemory = value
	if value != nil {
		b.bitmap_ |= 4
	} else {
		b.bitmap_ &^= 4
	}
	return b
}

// ComputeNodesSockets sets the value of the 'compute_nodes_sockets' attribute to the given value.
//
// Metric describing the total and used amount of some resource (like RAM, CPU and storage) in
// a cluster.
func (b *ClusterMetricsBuilder) ComputeNodesSockets(value *ClusterMetricBuilder) *ClusterMetricsBuilder {
	b.computeNodesSockets = value
	if value != nil {
		b.bitmap_ |= 8
	} else {
		b.bitmap_ &^= 8
	}
	return b
}

// CriticalAlertsFiring sets the value of the 'critical_alerts_firing' attribute to the given value.
//
//
func (b *ClusterMetricsBuilder) CriticalAlertsFiring(value int) *ClusterMetricsBuilder {
	b.criticalAlertsFiring = value
	b.bitmap_ |= 16
	return b
}

// Memory sets the value of the 'memory' attribute to the given value.
//
// Metric describing the total and used amount of some resource (like RAM, CPU and storage) in
// a cluster.
func (b *ClusterMetricsBuilder) Memory(value *ClusterMetricBuilder) *ClusterMetricsBuilder {
	b.memory = value
	if value != nil {
		b.bitmap_ |= 32
	} else {
		b.bitmap_ &^= 32
	}
	return b
}

// Nodes sets the value of the 'nodes' attribute to the given value.
//
// Counts of different classes of nodes inside a cluster.
func (b *ClusterMetricsBuilder) Nodes(value *ClusterNodesBuilder) *ClusterMetricsBuilder {
	b.nodes = value
	if value != nil {
		b.bitmap_ |= 64
	} else {
		b.bitmap_ &^= 64
	}
	return b
}

// OperatorsConditionFailing sets the value of the 'operators_condition_failing' attribute to the given value.
//
//
func (b *ClusterMetricsBuilder) OperatorsConditionFailing(value int) *ClusterMetricsBuilder {
	b.operatorsConditionFailing = value
	b.bitmap_ |= 128
	return b
}

// Sockets sets the value of the 'sockets' attribute to the given value.
//
// Metric describing the total and used amount of some resource (like RAM, CPU and storage) in
// a cluster.
func (b *ClusterMetricsBuilder) Sockets(value *ClusterMetricBuilder) *ClusterMetricsBuilder {
	b.sockets = value
	if value != nil {
		b.bitmap_ |= 256
	} else {
		b.bitmap_ &^= 256
	}
	return b
}

// Storage sets the value of the 'storage' attribute to the given value.
//
// Metric describing the total and used amount of some resource (like RAM, CPU and storage) in
// a cluster.
func (b *ClusterMetricsBuilder) Storage(value *ClusterMetricBuilder) *ClusterMetricsBuilder {
	b.storage = value
	if value != nil {
		b.bitmap_ |= 512
	} else {
		b.bitmap_ &^= 512
	}
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *ClusterMetricsBuilder) Copy(object *ClusterMetrics) *ClusterMetricsBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	if object.cpu != nil {
		b.cpu = NewClusterMetric().Copy(object.cpu)
	} else {
		b.cpu = nil
	}
	if object.computeNodesCPU != nil {
		b.computeNodesCPU = NewClusterMetric().Copy(object.computeNodesCPU)
	} else {
		b.computeNodesCPU = nil
	}
	if object.computeNodesMemory != nil {
		b.computeNodesMemory = NewClusterMetric().Copy(object.computeNodesMemory)
	} else {
		b.computeNodesMemory = nil
	}
	if object.computeNodesSockets != nil {
		b.computeNodesSockets = NewClusterMetric().Copy(object.computeNodesSockets)
	} else {
		b.computeNodesSockets = nil
	}
	b.criticalAlertsFiring = object.criticalAlertsFiring
	if object.memory != nil {
		b.memory = NewClusterMetric().Copy(object.memory)
	} else {
		b.memory = nil
	}
	if object.nodes != nil {
		b.nodes = NewClusterNodes().Copy(object.nodes)
	} else {
		b.nodes = nil
	}
	b.operatorsConditionFailing = object.operatorsConditionFailing
	if object.sockets != nil {
		b.sockets = NewClusterMetric().Copy(object.sockets)
	} else {
		b.sockets = nil
	}
	if object.storage != nil {
		b.storage = NewClusterMetric().Copy(object.storage)
	} else {
		b.storage = nil
	}
	return b
}

// Build creates a 'cluster_metrics' object using the configuration stored in the builder.
func (b *ClusterMetricsBuilder) Build() (object *ClusterMetrics, err error) {
	object = new(ClusterMetrics)
	object.bitmap_ = b.bitmap_
	if b.cpu != nil {
		object.cpu, err = b.cpu.Build()
		if err != nil {
			return
		}
	}
	if b.computeNodesCPU != nil {
		object.computeNodesCPU, err = b.computeNodesCPU.Build()
		if err != nil {
			return
		}
	}
	if b.computeNodesMemory != nil {
		object.computeNodesMemory, err = b.computeNodesMemory.Build()
		if err != nil {
			return
		}
	}
	if b.computeNodesSockets != nil {
		object.computeNodesSockets, err = b.computeNodesSockets.Build()
		if err != nil {
			return
		}
	}
	object.criticalAlertsFiring = b.criticalAlertsFiring
	if b.memory != nil {
		object.memory, err = b.memory.Build()
		if err != nil {
			return
		}
	}
	if b.nodes != nil {
		object.nodes, err = b.nodes.Build()
		if err != nil {
			return
		}
	}
	object.operatorsConditionFailing = b.operatorsConditionFailing
	if b.sockets != nil {
		object.sockets, err = b.sockets.Build()
		if err != nil {
			return
		}
	}
	if b.storage != nil {
		object.storage, err = b.storage.Build()
		if err != nil {
			return
		}
	}
	return
}
