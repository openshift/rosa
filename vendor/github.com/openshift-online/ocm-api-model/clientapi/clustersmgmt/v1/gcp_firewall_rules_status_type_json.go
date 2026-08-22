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

// MarshalGcpFirewallRulesStatus writes a value of the 'gcp_firewall_rules_status' type to the given writer.
func MarshalGcpFirewallRulesStatus(object *GcpFirewallRulesStatus, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	WriteGcpFirewallRulesStatus(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// WriteGcpFirewallRulesStatus writes a value of the 'gcp_firewall_rules_status' type to the given stream.
func WriteGcpFirewallRulesStatus(object *GcpFirewallRulesStatus, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = len(object.fieldSet_) > 0 && object.fieldSet_[0]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("description")
		stream.WriteString(object.description)
		count++
	}
	present_ = len(object.fieldSet_) > 1 && object.fieldSet_[1]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("kind")
		stream.WriteString(object.kind)
		count++
	}
	present_ = len(object.fieldSet_) > 2 && object.fieldSet_[2] && object.rules != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("rules")
		WriteGcpFirewallRulesStatusEntryList(object.rules, stream)
		count++
	}
	present_ = len(object.fieldSet_) > 3 && object.fieldSet_[3]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("state")
		stream.WriteString(object.state)
	}
	stream.WriteObjectEnd()
}

// UnmarshalGcpFirewallRulesStatus reads a value of the 'gcp_firewall_rules_status' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalGcpFirewallRulesStatus(source interface{}) (object *GcpFirewallRulesStatus, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = ReadGcpFirewallRulesStatus(iterator)
	err = iterator.Error
	return
}

// ReadGcpFirewallRulesStatus reads a value of the 'gcp_firewall_rules_status' type from the given iterator.
func ReadGcpFirewallRulesStatus(iterator *jsoniter.Iterator) *GcpFirewallRulesStatus {
	object := &GcpFirewallRulesStatus{
		fieldSet_: make([]bool, 4),
	}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "description":
			value := iterator.ReadString()
			object.description = value
			object.fieldSet_[0] = true
		case "kind":
			value := iterator.ReadString()
			object.kind = value
			object.fieldSet_[1] = true
		case "rules":
			value := ReadGcpFirewallRulesStatusEntryList(iterator)
			object.rules = value
			object.fieldSet_[2] = true
		case "state":
			value := iterator.ReadString()
			object.state = value
			object.fieldSet_[3] = true
		default:
			iterator.ReadAny()
		}
	}
	return object
}
