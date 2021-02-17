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

// MarshalMetric writes a value of the 'metric' type to the given writer.
func MarshalMetric(object *Metric, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeMetric(object, stream)
	stream.Flush()
	return stream.Error
}

// writeMetric writes a value of the 'metric' type to the given stream.
func writeMetric(object *Metric, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = object.bitmap_&1 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("name")
		stream.WriteString(object.name)
		count++
	}
	present_ = object.bitmap_&2 != 0 && object.vector != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("vector")
		writeSampleList(object.vector, stream)
		count++
	}
	stream.WriteObjectEnd()
}

// UnmarshalMetric reads a value of the 'metric' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalMetric(source interface{}) (object *Metric, err error) {
	if source == http.NoBody {
		return
	}
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readMetric(iterator)
	err = iterator.Error
	return
}

// readMetric reads a value of the 'metric' type from the given iterator.
func readMetric(iterator *jsoniter.Iterator) *Metric {
	object := &Metric{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "name":
			value := iterator.ReadString()
			object.name = value
			object.bitmap_ |= 1
		case "vector":
			value := readSampleList(iterator)
			object.vector = value
			object.bitmap_ |= 2
		default:
			iterator.ReadAny()
		}
	}
	return object
}
