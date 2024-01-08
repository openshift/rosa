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

package v1 // github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1

import (
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// MarshalNotificationDetailsResponse writes a value of the 'notification_details_response' type to the given writer.
func MarshalNotificationDetailsResponse(object *NotificationDetailsResponse, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeNotificationDetailsResponse(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// writeNotificationDetailsResponse writes a value of the 'notification_details_response' type to the given stream.
func writeNotificationDetailsResponse(object *NotificationDetailsResponse, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = object.bitmap_&1 != 0 && object.associates != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("associates")
		writeStringList(object.associates, stream)
		count++
	}
	present_ = object.bitmap_&2 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("external_org_id")
		stream.WriteString(object.externalOrgID)
		count++
	}
	present_ = object.bitmap_&4 != 0 && object.recipients != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("recipients")
		writeStringList(object.recipients, stream)
	}
	stream.WriteObjectEnd()
}

// UnmarshalNotificationDetailsResponse reads a value of the 'notification_details_response' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalNotificationDetailsResponse(source interface{}) (object *NotificationDetailsResponse, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readNotificationDetailsResponse(iterator)
	err = iterator.Error
	return
}

// readNotificationDetailsResponse reads a value of the 'notification_details_response' type from the given iterator.
func readNotificationDetailsResponse(iterator *jsoniter.Iterator) *NotificationDetailsResponse {
	object := &NotificationDetailsResponse{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "associates":
			value := readStringList(iterator)
			object.associates = value
			object.bitmap_ |= 1
		case "external_org_id":
			value := iterator.ReadString()
			object.externalOrgID = value
			object.bitmap_ |= 2
		case "recipients":
			value := readStringList(iterator)
			object.recipients = value
			object.bitmap_ |= 4
		default:
			iterator.ReadAny()
		}
	}
	return object
}
