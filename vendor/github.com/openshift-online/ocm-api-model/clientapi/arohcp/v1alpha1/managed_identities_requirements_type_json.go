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

package v1alpha1 // github.com/openshift-online/ocm-api-model/clientapi/arohcp/v1alpha1

import (
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/openshift-online/ocm-api-model/clientapi/helpers"
)

// MarshalManagedIdentitiesRequirements writes a value of the 'managed_identities_requirements' type to the given writer.
func MarshalManagedIdentitiesRequirements(object *ManagedIdentitiesRequirements, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	WriteManagedIdentitiesRequirements(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// WriteManagedIdentitiesRequirements writes a value of the 'managed_identities_requirements' type to the given stream.
func WriteManagedIdentitiesRequirements(object *ManagedIdentitiesRequirements, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	stream.WriteObjectField("kind")
	if object.bitmap_&1 != 0 {
		stream.WriteString(ManagedIdentitiesRequirementsLinkKind)
	} else {
		stream.WriteString(ManagedIdentitiesRequirementsKind)
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
	present_ = object.bitmap_&8 != 0 && object.controlPlaneOperatorsIdentities != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("control_plane_operators_identities")
		WriteControlPlaneOperatorIdentityRequirementList(object.controlPlaneOperatorsIdentities, stream)
		count++
	}
	present_ = object.bitmap_&16 != 0 && object.dataPlaneOperatorsIdentities != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("data_plane_operators_identities")
		WriteDataPlaneOperatorIdentityRequirementList(object.dataPlaneOperatorsIdentities, stream)
	}
	stream.WriteObjectEnd()
}

// UnmarshalManagedIdentitiesRequirements reads a value of the 'managed_identities_requirements' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalManagedIdentitiesRequirements(source interface{}) (object *ManagedIdentitiesRequirements, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = ReadManagedIdentitiesRequirements(iterator)
	err = iterator.Error
	return
}

// ReadManagedIdentitiesRequirements reads a value of the 'managed_identities_requirements' type from the given iterator.
func ReadManagedIdentitiesRequirements(iterator *jsoniter.Iterator) *ManagedIdentitiesRequirements {
	object := &ManagedIdentitiesRequirements{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "kind":
			value := iterator.ReadString()
			if value == ManagedIdentitiesRequirementsLinkKind {
				object.bitmap_ |= 1
			}
		case "id":
			object.id = iterator.ReadString()
			object.bitmap_ |= 2
		case "href":
			object.href = iterator.ReadString()
			object.bitmap_ |= 4
		case "control_plane_operators_identities":
			value := ReadControlPlaneOperatorIdentityRequirementList(iterator)
			object.controlPlaneOperatorsIdentities = value
			object.bitmap_ |= 8
		case "data_plane_operators_identities":
			value := ReadDataPlaneOperatorIdentityRequirementList(iterator)
			object.dataPlaneOperatorsIdentities = value
			object.bitmap_ |= 16
		default:
			iterator.ReadAny()
		}
	}
	return object
}
