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

// LogForwarderApplication represents the values of the 'log_forwarder_application' type.
//
// Represents an application that can be configured for log forwarding.
type LogForwarderApplication struct {
	fieldSet_ []bool
	id        string
	state     string
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *LogForwarderApplication) Empty() bool {
	if o == nil || len(o.fieldSet_) == 0 {
		return true
	}
	for _, set := range o.fieldSet_ {
		if set {
			return false
		}
	}
	return true
}

// ID returns the value of the 'ID' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// The identifier of the application.
func (o *LogForwarderApplication) ID() string {
	if o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0] {
		return o.id
	}
	return ""
}

// GetID returns the value of the 'ID' attribute and
// a flag indicating if the attribute has a value.
//
// The identifier of the application.
func (o *LogForwarderApplication) GetID() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0]
	if ok {
		value = o.id
	}
	return
}

// State returns the value of the 'state' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Indicates whether this application is available for use for log forwarding.
func (o *LogForwarderApplication) State() string {
	if o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1] {
		return o.state
	}
	return ""
}

// GetState returns the value of the 'state' attribute and
// a flag indicating if the attribute has a value.
//
// Indicates whether this application is available for use for log forwarding.
func (o *LogForwarderApplication) GetState() (value string, ok bool) {
	ok = o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1]
	if ok {
		value = o.state
	}
	return
}

// LogForwarderApplicationListKind is the name of the type used to represent list of objects of
// type 'log_forwarder_application'.
const LogForwarderApplicationListKind = "LogForwarderApplicationList"

// LogForwarderApplicationListLinkKind is the name of the type used to represent links to list
// of objects of type 'log_forwarder_application'.
const LogForwarderApplicationListLinkKind = "LogForwarderApplicationListLink"

// LogForwarderApplicationNilKind is the name of the type used to nil lists of objects of
// type 'log_forwarder_application'.
const LogForwarderApplicationListNilKind = "LogForwarderApplicationListNil"

// LogForwarderApplicationList is a list of values of the 'log_forwarder_application' type.
type LogForwarderApplicationList struct {
	href  string
	link  bool
	items []*LogForwarderApplication
}

// Len returns the length of the list.
func (l *LogForwarderApplicationList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Items sets the items of the list.
func (l *LogForwarderApplicationList) SetLink(link bool) {
	l.link = link
}

// Items sets the items of the list.
func (l *LogForwarderApplicationList) SetHREF(href string) {
	l.href = href
}

// Items sets the items of the list.
func (l *LogForwarderApplicationList) SetItems(items []*LogForwarderApplication) {
	l.items = items
}

// Items returns the items of the list.
func (l *LogForwarderApplicationList) Items() []*LogForwarderApplication {
	if l == nil {
		return nil
	}
	return l.items
}

// Empty returns true if the list is empty.
func (l *LogForwarderApplicationList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *LogForwarderApplicationList) Get(i int) *LogForwarderApplication {
	if l == nil || i < 0 || i >= len(l.items) {
		return nil
	}
	return l.items[i]
}

// Slice returns an slice containing the items of the list. The returned slice is a
// copy of the one used internally, so it can be modified without affecting the
// internal representation.
//
// If you don't need to modify the returned slice consider using the Each or Range
// functions, as they don't need to allocate a new slice.
func (l *LogForwarderApplicationList) Slice() []*LogForwarderApplication {
	var slice []*LogForwarderApplication
	if l == nil {
		slice = make([]*LogForwarderApplication, 0)
	} else {
		slice = make([]*LogForwarderApplication, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *LogForwarderApplicationList) Each(f func(item *LogForwarderApplication) bool) {
	if l == nil {
		return
	}
	for _, item := range l.items {
		if !f(item) {
			break
		}
	}
}

// Range runs the given function for each index and item of the list, in order. If
// the function returns false the iteration stops, otherwise it continues till all
// the elements of the list have been processed.
func (l *LogForwarderApplicationList) Range(f func(index int, item *LogForwarderApplication) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
