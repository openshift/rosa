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

// MarshalGcpFirewallRuleSpec writes a value of the 'gcp_firewall_rule_spec' type to the given writer.
func MarshalGcpFirewallRuleSpec(object *GcpFirewallRuleSpec, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	WriteGcpFirewallRuleSpec(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// WriteGcpFirewallRuleSpec writes a value of the 'gcp_firewall_rule_spec' type to the given stream.
func WriteGcpFirewallRuleSpec(object *GcpFirewallRuleSpec, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = len(object.fieldSet_) > 0 && object.fieldSet_[0] && object.allowed != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("allowed")
		WriteGcpFirewallRuleAllowedList(object.allowed, stream)
		count++
	}
	present_ = len(object.fieldSet_) > 1 && object.fieldSet_[1]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("direction")
		stream.WriteString(object.direction)
		count++
	}
	present_ = len(object.fieldSet_) > 2 && object.fieldSet_[2]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("name")
		stream.WriteString(object.name)
		count++
	}
	present_ = len(object.fieldSet_) > 3 && object.fieldSet_[3]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("network")
		stream.WriteString(object.network)
		count++
	}
	present_ = len(object.fieldSet_) > 4 && object.fieldSet_[4]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("priority")
		stream.WriteInt(object.priority)
		count++
	}
	present_ = len(object.fieldSet_) > 5 && object.fieldSet_[5] && object.sourceRanges != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("source_ranges")
		WriteStringList(object.sourceRanges, stream)
		count++
	}
	present_ = len(object.fieldSet_) > 6 && object.fieldSet_[6] && object.sourceServiceAccounts != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("source_service_accounts")
		WriteStringList(object.sourceServiceAccounts, stream)
		count++
	}
	present_ = len(object.fieldSet_) > 7 && object.fieldSet_[7] && object.targetServiceAccounts != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("target_service_accounts")
		WriteStringList(object.targetServiceAccounts, stream)
	}
	stream.WriteObjectEnd()
}

// UnmarshalGcpFirewallRuleSpec reads a value of the 'gcp_firewall_rule_spec' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalGcpFirewallRuleSpec(source interface{}) (object *GcpFirewallRuleSpec, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = ReadGcpFirewallRuleSpec(iterator)
	err = iterator.Error
	return
}

// ReadGcpFirewallRuleSpec reads a value of the 'gcp_firewall_rule_spec' type from the given iterator.
func ReadGcpFirewallRuleSpec(iterator *jsoniter.Iterator) *GcpFirewallRuleSpec {
	object := &GcpFirewallRuleSpec{
		fieldSet_: make([]bool, 8),
	}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "allowed":
			value := ReadGcpFirewallRuleAllowedList(iterator)
			object.allowed = value
			object.fieldSet_[0] = true
		case "direction":
			value := iterator.ReadString()
			object.direction = value
			object.fieldSet_[1] = true
		case "name":
			value := iterator.ReadString()
			object.name = value
			object.fieldSet_[2] = true
		case "network":
			value := iterator.ReadString()
			object.network = value
			object.fieldSet_[3] = true
		case "priority":
			value := iterator.ReadInt()
			object.priority = value
			object.fieldSet_[4] = true
		case "source_ranges":
			value := ReadStringList(iterator)
			object.sourceRanges = value
			object.fieldSet_[5] = true
		case "source_service_accounts":
			value := ReadStringList(iterator)
			object.sourceServiceAccounts = value
			object.fieldSet_[6] = true
		case "target_service_accounts":
			value := ReadStringList(iterator)
			object.targetServiceAccounts = value
			object.fieldSet_[7] = true
		default:
			iterator.ReadAny()
		}
	}
	return object
}
