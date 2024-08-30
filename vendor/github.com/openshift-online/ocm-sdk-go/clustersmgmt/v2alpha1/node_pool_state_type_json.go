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

package v2alpha1 // github.com/openshift-online/ocm-sdk-go/clustersmgmt/v2alpha1

import (
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// MarshalNodePoolState writes a value of the 'node_pool_state' type to the given writer.
func MarshalNodePoolState(object *NodePoolState, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeNodePoolState(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// writeNodePoolState writes a value of the 'node_pool_state' type to the given stream.
func writeNodePoolState(object *NodePoolState, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	stream.WriteObjectField("kind")
	if object.bitmap_&1 != 0 {
		stream.WriteString(NodePoolStateLinkKind)
	} else {
		stream.WriteString(NodePoolStateKind)
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
	present_ = object.bitmap_&8 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("details")
		stream.WriteString(object.details)
		count++
	}
	present_ = object.bitmap_&16 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("value")
		stream.WriteString(string(object.value))
	}
	stream.WriteObjectEnd()
}

// UnmarshalNodePoolState reads a value of the 'node_pool_state' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalNodePoolState(source interface{}) (object *NodePoolState, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readNodePoolState(iterator)
	err = iterator.Error
	return
}

// readNodePoolState reads a value of the 'node_pool_state' type from the given iterator.
func readNodePoolState(iterator *jsoniter.Iterator) *NodePoolState {
	object := &NodePoolState{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "kind":
			value := iterator.ReadString()
			if value == NodePoolStateLinkKind {
				object.bitmap_ |= 1
			}
		case "id":
			object.id = iterator.ReadString()
			object.bitmap_ |= 2
		case "href":
			object.href = iterator.ReadString()
			object.bitmap_ |= 4
		case "details":
			value := iterator.ReadString()
			object.details = value
			object.bitmap_ |= 8
		case "value":
			text := iterator.ReadString()
			value := NodePoolStateValues(text)
			object.value = value
			object.bitmap_ |= 16
		default:
			iterator.ReadAny()
		}
	}
	return object
}
