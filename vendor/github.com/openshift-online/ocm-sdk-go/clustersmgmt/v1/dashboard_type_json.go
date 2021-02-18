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

// MarshalDashboard writes a value of the 'dashboard' type to the given writer.
func MarshalDashboard(object *Dashboard, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeDashboard(object, stream)
	stream.Flush()
	return stream.Error
}

// writeDashboard writes a value of the 'dashboard' type to the given stream.
func writeDashboard(object *Dashboard, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	stream.WriteObjectField("kind")
	if object.bitmap_&1 != 0 {
		stream.WriteString(DashboardLinkKind)
	} else {
		stream.WriteString(DashboardKind)
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
	present_ = object.bitmap_&8 != 0 && object.metrics != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("metrics")
		writeMetricList(object.metrics, stream)
		count++
	}
	present_ = object.bitmap_&16 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("name")
		stream.WriteString(object.name)
		count++
	}
	stream.WriteObjectEnd()
}

// UnmarshalDashboard reads a value of the 'dashboard' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalDashboard(source interface{}) (object *Dashboard, err error) {
	if source == http.NoBody {
		return
	}
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readDashboard(iterator)
	err = iterator.Error
	return
}

// readDashboard reads a value of the 'dashboard' type from the given iterator.
func readDashboard(iterator *jsoniter.Iterator) *Dashboard {
	object := &Dashboard{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "kind":
			value := iterator.ReadString()
			if value == DashboardLinkKind {
				object.bitmap_ |= 1
			}
		case "id":
			object.id = iterator.ReadString()
			object.bitmap_ |= 2
		case "href":
			object.href = iterator.ReadString()
			object.bitmap_ |= 4
		case "metrics":
			value := readMetricList(iterator)
			object.metrics = value
			object.bitmap_ |= 8
		case "name":
			value := iterator.ReadString()
			object.name = value
			object.bitmap_ |= 16
		default:
			iterator.ReadAny()
		}
	}
	return object
}
