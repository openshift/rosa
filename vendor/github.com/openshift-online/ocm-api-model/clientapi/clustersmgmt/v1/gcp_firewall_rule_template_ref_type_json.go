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

// MarshalGcpFirewallRuleTemplateRef writes a value of the 'gcp_firewall_rule_template_ref' type to the given writer.
func MarshalGcpFirewallRuleTemplateRef(object *GcpFirewallRuleTemplateRef, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	WriteGcpFirewallRuleTemplateRef(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// WriteGcpFirewallRuleTemplateRef writes a value of the 'gcp_firewall_rule_template_ref' type to the given stream.
func WriteGcpFirewallRuleTemplateRef(object *GcpFirewallRuleTemplateRef, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = len(object.fieldSet_) > 0 && object.fieldSet_[0]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("href")
		stream.WriteString(object.href)
		count++
	}
	present_ = len(object.fieldSet_) > 1 && object.fieldSet_[1]
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("version")
		stream.WriteString(object.version)
	}
	stream.WriteObjectEnd()
}

// UnmarshalGcpFirewallRuleTemplateRef reads a value of the 'gcp_firewall_rule_template_ref' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalGcpFirewallRuleTemplateRef(source interface{}) (object *GcpFirewallRuleTemplateRef, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = ReadGcpFirewallRuleTemplateRef(iterator)
	err = iterator.Error
	return
}

// ReadGcpFirewallRuleTemplateRef reads a value of the 'gcp_firewall_rule_template_ref' type from the given iterator.
func ReadGcpFirewallRuleTemplateRef(iterator *jsoniter.Iterator) *GcpFirewallRuleTemplateRef {
	object := &GcpFirewallRuleTemplateRef{
		fieldSet_: make([]bool, 2),
	}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "href":
			value := iterator.ReadString()
			object.href = value
			object.fieldSet_[0] = true
		case "version":
			value := iterator.ReadString()
			object.version = value
			object.fieldSet_[1] = true
		default:
			iterator.ReadAny()
		}
	}
	return object
}
