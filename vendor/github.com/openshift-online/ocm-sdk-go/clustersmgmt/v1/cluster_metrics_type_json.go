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
	"io"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// MarshalClusterMetrics writes a value of the 'cluster_metrics' type to the given writer.
func MarshalClusterMetrics(object *ClusterMetrics, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeClusterMetrics(object, stream)
	stream.Flush()
	return stream.Error
}

// writeClusterMetrics writes a value of the 'cluster_metrics' type to the given stream.
func writeClusterMetrics(object *ClusterMetrics, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = object.bitmap_&1 != 0 && object.cpu != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("cpu")
		writeClusterMetric(object.cpu, stream)
		count++
	}
	present_ = object.bitmap_&2 != 0 && object.computeNodesCPU != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("compute_nodes_cpu")
		writeClusterMetric(object.computeNodesCPU, stream)
		count++
	}
	present_ = object.bitmap_&4 != 0 && object.computeNodesMemory != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("compute_nodes_memory")
		writeClusterMetric(object.computeNodesMemory, stream)
		count++
	}
	present_ = object.bitmap_&8 != 0 && object.computeNodesSockets != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("compute_nodes_sockets")
		writeClusterMetric(object.computeNodesSockets, stream)
		count++
	}
	present_ = object.bitmap_&16 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("critical_alerts_firing")
		stream.WriteInt(object.criticalAlertsFiring)
		count++
	}
	present_ = object.bitmap_&32 != 0 && object.memory != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("memory")
		writeClusterMetric(object.memory, stream)
		count++
	}
	present_ = object.bitmap_&64 != 0 && object.nodes != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("nodes")
		writeClusterNodes(object.nodes, stream)
		count++
	}
	present_ = object.bitmap_&128 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("operators_condition_failing")
		stream.WriteInt(object.operatorsConditionFailing)
		count++
	}
	present_ = object.bitmap_&256 != 0 && object.sockets != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("sockets")
		writeClusterMetric(object.sockets, stream)
		count++
	}
	present_ = object.bitmap_&512 != 0 && object.storage != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("storage")
		writeClusterMetric(object.storage, stream)
		count++
	}
	stream.WriteObjectEnd()
}

// UnmarshalClusterMetrics reads a value of the 'cluster_metrics' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalClusterMetrics(source interface{}) (object *ClusterMetrics, err error) {
	if source == http.NoBody {
		return
	}
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readClusterMetrics(iterator)
	err = iterator.Error
	return
}

// readClusterMetrics reads a value of the 'cluster_metrics' type from the given iterator.
func readClusterMetrics(iterator *jsoniter.Iterator) *ClusterMetrics {
	object := &ClusterMetrics{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "cpu":
			value := readClusterMetric(iterator)
			object.cpu = value
			object.bitmap_ |= 1
		case "compute_nodes_cpu":
			value := readClusterMetric(iterator)
			object.computeNodesCPU = value
			object.bitmap_ |= 2
		case "compute_nodes_memory":
			value := readClusterMetric(iterator)
			object.computeNodesMemory = value
			object.bitmap_ |= 4
		case "compute_nodes_sockets":
			value := readClusterMetric(iterator)
			object.computeNodesSockets = value
			object.bitmap_ |= 8
		case "critical_alerts_firing":
			value := iterator.ReadInt()
			object.criticalAlertsFiring = value
			object.bitmap_ |= 16
		case "memory":
			value := readClusterMetric(iterator)
			object.memory = value
			object.bitmap_ |= 32
		case "nodes":
			value := readClusterNodes(iterator)
			object.nodes = value
			object.bitmap_ |= 64
		case "operators_condition_failing":
			value := iterator.ReadInt()
			object.operatorsConditionFailing = value
			object.bitmap_ |= 128
		case "sockets":
			value := readClusterMetric(iterator)
			object.sockets = value
			object.bitmap_ |= 256
		case "storage":
			value := readClusterMetric(iterator)
			object.storage = value
			object.bitmap_ |= 512
		default:
			iterator.ReadAny()
		}
	}
	return object
}
