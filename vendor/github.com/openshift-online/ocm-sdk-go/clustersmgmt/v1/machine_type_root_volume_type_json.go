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

// MarshalMachineTypeRootVolume writes a value of the 'machine_type_root_volume' type to the given writer.
func MarshalMachineTypeRootVolume(object *MachineTypeRootVolume, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeMachineTypeRootVolume(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// writeMachineTypeRootVolume writes a value of the 'machine_type_root_volume' type to the given stream.
func writeMachineTypeRootVolume(object *MachineTypeRootVolume, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = object.bitmap_&1 != 0 && object.aws != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("aws")
		writeAWSVolume(object.aws, stream)
		count++
	}
	present_ = object.bitmap_&2 != 0 && object.gcp != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("gcp")
		writeGCPVolume(object.gcp, stream)
	}
	stream.WriteObjectEnd()
}

// UnmarshalMachineTypeRootVolume reads a value of the 'machine_type_root_volume' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalMachineTypeRootVolume(source interface{}) (object *MachineTypeRootVolume, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readMachineTypeRootVolume(iterator)
	err = iterator.Error
	return
}

// readMachineTypeRootVolume reads a value of the 'machine_type_root_volume' type from the given iterator.
func readMachineTypeRootVolume(iterator *jsoniter.Iterator) *MachineTypeRootVolume {
	object := &MachineTypeRootVolume{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "aws":
			value := readAWSVolume(iterator)
			object.aws = value
			object.bitmap_ |= 1
		case "gcp":
			value := readGCPVolume(iterator)
			object.gcp = value
			object.bitmap_ |= 2
		default:
			iterator.ReadAny()
		}
	}
	return object
}
