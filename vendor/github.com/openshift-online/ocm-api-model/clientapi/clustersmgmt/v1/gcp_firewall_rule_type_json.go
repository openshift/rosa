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

// MarshalGcpFirewallRule writes a value of the 'gcp_firewall_rule' type to the given writer.
func MarshalGcpFirewallRule(object *GcpFirewallRule, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	WriteGcpFirewallRule(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// WriteGcpFirewallRule writes a value of the 'gcp_firewall_rule' type to the given stream.
func WriteGcpFirewallRule(object *GcpFirewallRule, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	stream.WriteObjectField("kind")
	if len(object.fieldSet_) > 0 && object.fieldSet_[0] {
		stream.WriteString(GcpFirewallRuleLinkKind)
	} else {
		stream.WriteString(GcpFirewallRuleKind)
	}
	count++
	if len(object.fieldSet_) > 1 && object.fieldSet_[1] {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("id")
		stream.WriteString(object.id)
		count++
	}
	if len(object.fieldSet_) > 2 && object.fieldSet_[2] {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("href")
		stream.WriteString(object.href)
		count++
	}
	var present_ bool
	present_ = len(object.fieldSet_) > 3 && object.fieldSet_[3] && object.gcpNetwork != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("gcp_network")
		WriteGcpFirewallRuleGcpNetwork(object.gcpNetwork, stream)
		count++
	}
	present_ = len(object.fieldSet_) > 4 && object.fieldSet_[4]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("name")
		stream.WriteString(object.name)
		count++
	}
	present_ = len(object.fieldSet_) > 5 && object.fieldSet_[5] && object.organization != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("organization")
		WriteOrganizationLink(object.organization, stream)
		count++
	}
	present_ = len(object.fieldSet_) > 6 && object.fieldSet_[6]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("profile")
		stream.WriteString(object.profile)
		count++
	}
	present_ = len(object.fieldSet_) > 7 && object.fieldSet_[7] && object.status != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("status")
		WriteGcpFirewallRuleStatus(object.status, stream)
		count++
	}
	present_ = len(object.fieldSet_) > 8 && object.fieldSet_[8] && object.template != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("template")
		WriteGcpFirewallRuleTemplateRef(object.template, stream)
		count++
	}
	present_ = len(object.fieldSet_) > 9 && object.fieldSet_[9] && object.wifConfig != nil
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("wif_config")
		WriteWifConfig(object.wifConfig, stream)
	}
	stream.WriteObjectEnd()
}

// UnmarshalGcpFirewallRule reads a value of the 'gcp_firewall_rule' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalGcpFirewallRule(source interface{}) (object *GcpFirewallRule, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = ReadGcpFirewallRule(iterator)
	err = iterator.Error
	return
}

// ReadGcpFirewallRule reads a value of the 'gcp_firewall_rule' type from the given iterator.
func ReadGcpFirewallRule(iterator *jsoniter.Iterator) *GcpFirewallRule {
	object := &GcpFirewallRule{
		fieldSet_: make([]bool, 10),
	}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "kind":
			value := iterator.ReadString()
			if value == GcpFirewallRuleLinkKind {
				object.fieldSet_[0] = true
			}
		case "id":
			object.id = iterator.ReadString()
			object.fieldSet_[1] = true
		case "href":
			object.href = iterator.ReadString()
			object.fieldSet_[2] = true
		case "gcp_network":
			value := ReadGcpFirewallRuleGcpNetwork(iterator)
			object.gcpNetwork = value
			object.fieldSet_[3] = true
		case "name":
			value := iterator.ReadString()
			object.name = value
			object.fieldSet_[4] = true
		case "organization":
			value := ReadOrganizationLink(iterator)
			object.organization = value
			object.fieldSet_[5] = true
		case "profile":
			value := iterator.ReadString()
			object.profile = value
			object.fieldSet_[6] = true
		case "status":
			value := ReadGcpFirewallRuleStatus(iterator)
			object.status = value
			object.fieldSet_[7] = true
		case "template":
			value := ReadGcpFirewallRuleTemplateRef(iterator)
			object.template = value
			object.fieldSet_[8] = true
		case "wif_config":
			value := ReadWifConfig(iterator)
			object.wifConfig = value
			object.fieldSet_[9] = true
		default:
			iterator.ReadAny()
		}
	}
	return object
}
