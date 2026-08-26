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

package v1 // github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1

import (
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/openshift-online/ocm-api-model/clientapi/helpers"
)

// MarshalAzureNodePoolImage writes a value of the 'azure_node_pool_image' type to the given writer.
func MarshalAzureNodePoolImage(object *AzureNodePoolImage, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	WriteAzureNodePoolImage(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// WriteAzureNodePoolImage writes a value of the 'azure_node_pool_image' type to the given stream.
func WriteAzureNodePoolImage(object *AzureNodePoolImage, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = len(object.fieldSet_) > 0 && object.fieldSet_[0]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("sku")
		stream.WriteString(object.sku)
		count++
	}
	present_ = len(object.fieldSet_) > 1 && object.fieldSet_[1]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("offer")
		stream.WriteString(object.offer)
		count++
	}
	present_ = len(object.fieldSet_) > 2 && object.fieldSet_[2]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("publisher")
		stream.WriteString(object.publisher)
		count++
	}
	present_ = len(object.fieldSet_) > 3 && object.fieldSet_[3]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("version")
		stream.WriteString(object.version)
	}
	stream.WriteObjectEnd()
}

// UnmarshalAzureNodePoolImage reads a value of the 'azure_node_pool_image' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalAzureNodePoolImage(source interface{}) (object *AzureNodePoolImage, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = ReadAzureNodePoolImage(iterator)
	err = iterator.Error
	return
}

// ReadAzureNodePoolImage reads a value of the 'azure_node_pool_image' type from the given iterator.
func ReadAzureNodePoolImage(iterator *jsoniter.Iterator) *AzureNodePoolImage {
	object := &AzureNodePoolImage{
		fieldSet_: make([]bool, 4),
	}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "sku":
			value := iterator.ReadString()
			object.sku = value
			object.fieldSet_[0] = true
		case "offer":
			value := iterator.ReadString()
			object.offer = value
			object.fieldSet_[1] = true
		case "publisher":
			value := iterator.ReadString()
			object.publisher = value
			object.fieldSet_[2] = true
		case "version":
			value := iterator.ReadString()
			object.version = value
			object.fieldSet_[3] = true
		default:
			iterator.ReadAny()
		}
	}
	return object
}
