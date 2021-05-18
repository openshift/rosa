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

// MarshalCustomIAMRoles writes a value of the 'custom_IAM_roles' type to the given writer.
func MarshalCustomIAMRoles(object *CustomIAMRoles, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeCustomIAMRoles(object, stream)
	stream.Flush()
	return stream.Error
}

// writeCustomIAMRoles writes a value of the 'custom_IAM_roles' type to the given stream.
func writeCustomIAMRoles(object *CustomIAMRoles, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = object.bitmap_&1 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("master_iam_role")
		stream.WriteString(object.masterIAMRole)
		count++
	}
	present_ = object.bitmap_&2 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("worker_iam_role")
		stream.WriteString(object.workerIAMRole)
		count++
	}
	stream.WriteObjectEnd()
}

// UnmarshalCustomIAMRoles reads a value of the 'custom_IAM_roles' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalCustomIAMRoles(source interface{}) (object *CustomIAMRoles, err error) {
	if source == http.NoBody {
		return
	}
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readCustomIAMRoles(iterator)
	err = iterator.Error
	return
}

// readCustomIAMRoles reads a value of the 'custom_IAM_roles' type from the given iterator.
func readCustomIAMRoles(iterator *jsoniter.Iterator) *CustomIAMRoles {
	object := &CustomIAMRoles{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "master_iam_role":
			value := iterator.ReadString()
			object.masterIAMRole = value
			object.bitmap_ |= 1
		case "worker_iam_role":
			value := iterator.ReadString()
			object.workerIAMRole = value
			object.bitmap_ |= 2
		default:
			iterator.ReadAny()
		}
	}
	return object
}
