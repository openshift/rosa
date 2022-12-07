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

	jsoniter "github.com/json-iterator/go"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// MarshalNodePool writes a value of the 'node_pool' type to the given writer.
func MarshalNodePool(object *NodePool, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeNodePool(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// writeNodePool writes a value of the 'node_pool' type to the given stream.
func writeNodePool(object *NodePool, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	stream.WriteObjectField("kind")
	if object.bitmap_&1 != 0 {
		stream.WriteString(NodePoolLinkKind)
	} else {
		stream.WriteString(NodePoolKind)
	}
	count++
	if object.bitmap_&2 != 0 {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("id")
		stream.WriteString(object.id)
		count++
	}
	if object.bitmap_&4 != 0 {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("href")
		stream.WriteString(object.href)
		count++
	}
	var present_ bool
	present_ = object.bitmap_&8 != 0 && object.awsNodePool != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("aws_node_pool")
		writeAWSNodePool(object.awsNodePool, stream)
		count++
	}
	present_ = object.bitmap_&16 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("auto_repair")
		stream.WriteBool(object.autoRepair)
		count++
	}
	present_ = object.bitmap_&32 != 0 && object.autoscaling != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("autoscaling")
		writeNodePoolAutoscaling(object.autoscaling, stream)
		count++
	}
	present_ = object.bitmap_&64 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("availability_zone")
		stream.WriteString(object.availabilityZone)
		count++
	}
	present_ = object.bitmap_&128 != 0 && object.cluster != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("cluster")
		writeCluster(object.cluster, stream)
		count++
	}
	present_ = object.bitmap_&256 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("replicas")
		stream.WriteInt(object.replicas)
		count++
	}
	present_ = object.bitmap_&512 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("subnet")
		stream.WriteString(object.subnet)
	}
	stream.WriteObjectEnd()
}

// UnmarshalNodePool reads a value of the 'node_pool' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalNodePool(source interface{}) (object *NodePool, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readNodePool(iterator)
	err = iterator.Error
	return
}

// readNodePool reads a value of the 'node_pool' type from the given iterator.
func readNodePool(iterator *jsoniter.Iterator) *NodePool {
	object := &NodePool{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "kind":
			value := iterator.ReadString()
			if value == NodePoolLinkKind {
				object.bitmap_ |= 1
			}
		case "id":
			object.id = iterator.ReadString()
			object.bitmap_ |= 2
		case "href":
			object.href = iterator.ReadString()
			object.bitmap_ |= 4
		case "aws_node_pool":
			value := readAWSNodePool(iterator)
			object.awsNodePool = value
			object.bitmap_ |= 8
		case "auto_repair":
			value := iterator.ReadBool()
			object.autoRepair = value
			object.bitmap_ |= 16
		case "autoscaling":
			value := readNodePoolAutoscaling(iterator)
			object.autoscaling = value
			object.bitmap_ |= 32
		case "availability_zone":
			value := iterator.ReadString()
			object.availabilityZone = value
			object.bitmap_ |= 64
		case "cluster":
			value := readCluster(iterator)
			object.cluster = value
			object.bitmap_ |= 128
		case "replicas":
			value := iterator.ReadInt()
			object.replicas = value
			object.bitmap_ |= 256
		case "subnet":
			value := iterator.ReadString()
			object.subnet = value
			object.bitmap_ |= 512
		default:
			iterator.ReadAny()
		}
	}
	return object
}
