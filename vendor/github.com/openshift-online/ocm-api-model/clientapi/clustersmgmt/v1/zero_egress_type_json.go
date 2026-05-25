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

// MarshalZeroEgress writes a value of the 'zero_egress' type to the given writer.
func MarshalZeroEgress(object *ZeroEgress, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	WriteZeroEgress(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// WriteZeroEgress writes a value of the 'zero_egress' type to the given stream.
func WriteZeroEgress(object *ZeroEgress, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = len(object.fieldSet_) > 0 && object.fieldSet_[0]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("enabled")
		stream.WriteBool(object.enabled)
		count++
	}
	present_ = len(object.fieldSet_) > 1 && object.fieldSet_[1] && object.noProxyDefaultDomains != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("no_proxy_default_domains")
		WriteStringList(object.noProxyDefaultDomains, stream)
	}
	stream.WriteObjectEnd()
}

// UnmarshalZeroEgress reads a value of the 'zero_egress' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalZeroEgress(source interface{}) (object *ZeroEgress, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = ReadZeroEgress(iterator)
	err = iterator.Error
	return
}

// ReadZeroEgress reads a value of the 'zero_egress' type from the given iterator.
func ReadZeroEgress(iterator *jsoniter.Iterator) *ZeroEgress {
	object := &ZeroEgress{
		fieldSet_: make([]bool, 2),
	}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "enabled":
			value := iterator.ReadBool()
			object.enabled = value
			object.fieldSet_[0] = true
		case "no_proxy_default_domains":
			value := ReadStringList(iterator)
			object.noProxyDefaultDomains = value
			object.fieldSet_[1] = true
		default:
			iterator.ReadAny()
		}
	}
	return object
}
