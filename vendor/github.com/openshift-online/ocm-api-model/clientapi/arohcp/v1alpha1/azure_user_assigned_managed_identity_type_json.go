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

// MarshalAzureUserAssignedManagedIdentity writes a value of the 'azure_user_assigned_managed_identity' type to the given writer.
func MarshalAzureUserAssignedManagedIdentity(object *AzureUserAssignedManagedIdentity, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	WriteAzureUserAssignedManagedIdentity(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// WriteAzureUserAssignedManagedIdentity writes a value of the 'azure_user_assigned_managed_identity' type to the given stream.
func WriteAzureUserAssignedManagedIdentity(object *AzureUserAssignedManagedIdentity, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = len(object.fieldSet_) > 0 && object.fieldSet_[0]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("resource_id")
		stream.WriteString(object.resourceID)
	}
	stream.WriteObjectEnd()
}

// UnmarshalAzureUserAssignedManagedIdentity reads a value of the 'azure_user_assigned_managed_identity' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalAzureUserAssignedManagedIdentity(source interface{}) (object *AzureUserAssignedManagedIdentity, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = ReadAzureUserAssignedManagedIdentity(iterator)
	err = iterator.Error
	return
}

// ReadAzureUserAssignedManagedIdentity reads a value of the 'azure_user_assigned_managed_identity' type from the given iterator.
func ReadAzureUserAssignedManagedIdentity(iterator *jsoniter.Iterator) *AzureUserAssignedManagedIdentity {
	object := &AzureUserAssignedManagedIdentity{
		fieldSet_: make([]bool, 1),
	}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "resource_id":
			value := iterator.ReadString()
			object.resourceID = value
			object.fieldSet_[0] = true
		default:
			iterator.ReadAny()
		}
	}
	return object
}
