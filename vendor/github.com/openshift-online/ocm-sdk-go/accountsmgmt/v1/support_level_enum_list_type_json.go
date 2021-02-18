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

// MarshalSupportLevelEnumList writes a list of values of the 'support_level_enum' type to
// the given writer.
func MarshalSupportLevelEnumList(list []SupportLevelEnum, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeSupportLevelEnumList(list, stream)
	stream.Flush()
	return stream.Error
}

// writeSupportLevelEnumList writes a list of value of the 'support_level_enum' type to
// the given stream.
func writeSupportLevelEnumList(list []SupportLevelEnum, stream *jsoniter.Stream) {
	stream.WriteArrayStart()
	for i, value := range list {
		if i > 0 {
			stream.WriteMore()
		}
		stream.WriteString(string(value))
	}
	stream.WriteArrayEnd()
}

// UnmarshalSupportLevelEnumList reads a list of values of the 'support_level_enum' type
// from the given source, which can be a slice of bytes, a string or a reader.
func UnmarshalSupportLevelEnumList(source interface{}) (items []SupportLevelEnum, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	items = readSupportLevelEnumList(iterator)
	err = iterator.Error
	return
}

// readSupportLevelEnumList reads list of values of the ''support_level_enum' type from
// the given iterator.
func readSupportLevelEnumList(iterator *jsoniter.Iterator) []SupportLevelEnum {
	list := []SupportLevelEnum{}
	for iterator.ReadArray() {
		text := iterator.ReadString()
		item := SupportLevelEnum(text)
		list = append(list, item)
	}
	return list
}
