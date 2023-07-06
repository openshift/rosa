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

// MarshalClusterStsSupportRole writes a value of the 'cluster_sts_support_role' type to the given writer.
func MarshalClusterStsSupportRole(object *ClusterStsSupportRole, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeClusterStsSupportRole(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// writeClusterStsSupportRole writes a value of the 'cluster_sts_support_role' type to the given stream.
func writeClusterStsSupportRole(object *ClusterStsSupportRole, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = object.bitmap_&1 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("role_arn")
		stream.WriteString(object.roleArn)
	}
	stream.WriteObjectEnd()
}

// UnmarshalClusterStsSupportRole reads a value of the 'cluster_sts_support_role' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalClusterStsSupportRole(source interface{}) (object *ClusterStsSupportRole, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readClusterStsSupportRole(iterator)
	err = iterator.Error
	return
}

// readClusterStsSupportRole reads a value of the 'cluster_sts_support_role' type from the given iterator.
func readClusterStsSupportRole(iterator *jsoniter.Iterator) *ClusterStsSupportRole {
	object := &ClusterStsSupportRole{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "role_arn":
			value := iterator.ReadString()
			object.roleArn = value
			object.bitmap_ |= 1
		default:
			iterator.ReadAny()
		}
	}
	return object
}
