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

// Represents an application that can be configured for log forwarding.
type LogForwarderApplicationBuilder struct {
	fieldSet_ []bool
	id        string
	state     string
}

// NewLogForwarderApplication creates a new builder of 'log_forwarder_application' objects.
func NewLogForwarderApplication() *LogForwarderApplicationBuilder {
	return &LogForwarderApplicationBuilder{
		fieldSet_: make([]bool, 2),
	}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *LogForwarderApplicationBuilder) Empty() bool {
	if b == nil || len(b.fieldSet_) == 0 {
		return true
	}
	for _, set := range b.fieldSet_ {
		if set {
			return false
		}
	}
	return true
}

// ID sets the value of the 'ID' attribute to the given value.
func (b *LogForwarderApplicationBuilder) ID(value string) *LogForwarderApplicationBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 2)
	}
	b.id = value
	b.fieldSet_[0] = true
	return b
}

// State sets the value of the 'state' attribute to the given value.
func (b *LogForwarderApplicationBuilder) State(value string) *LogForwarderApplicationBuilder {
	if len(b.fieldSet_) == 0 {
		b.fieldSet_ = make([]bool, 2)
	}
	b.state = value
	b.fieldSet_[1] = true
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *LogForwarderApplicationBuilder) Copy(object *LogForwarderApplication) *LogForwarderApplicationBuilder {
	if object == nil {
		return b
	}
	if len(object.fieldSet_) > 0 {
		b.fieldSet_ = make([]bool, len(object.fieldSet_))
		copy(b.fieldSet_, object.fieldSet_)
	}
	b.id = object.id
	b.state = object.state
	return b
}

// Build creates a 'log_forwarder_application' object using the configuration stored in the builder.
func (b *LogForwarderApplicationBuilder) Build() (object *LogForwarderApplication, err error) {
	object = new(LogForwarderApplication)
	if len(b.fieldSet_) > 0 {
		object.fieldSet_ = make([]bool, len(b.fieldSet_))
		copy(object.fieldSet_, b.fieldSet_)
	}
	object.id = b.id
	object.state = b.state
	return
}
