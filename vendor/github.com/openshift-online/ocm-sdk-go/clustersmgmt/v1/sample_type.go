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
	time "time"
)

// Sample represents the values of the 'sample' type.
//
// Sample of a metric.
type Sample struct {
	bitmap_ uint32
	time    time.Time
	value   float64
}

// Empty returns true if the object is empty, i.e. no attribute has a value.
func (o *Sample) Empty() bool {
	return o == nil || o.bitmap_ == 0
}

// Time returns the value of the 'time' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Time when the sample was obtained.
func (o *Sample) Time() time.Time {
	if o != nil && o.bitmap_&1 != 0 {
		return o.time
	}
	return time.Time{}
}

// GetTime returns the value of the 'time' attribute and
// a flag indicating if the attribute has a value.
//
// Time when the sample was obtained.
func (o *Sample) GetTime() (value time.Time, ok bool) {
	ok = o != nil && o.bitmap_&1 != 0
	if ok {
		value = o.time
	}
	return
}

// Value returns the value of the 'value' attribute, or
// the zero value of the type if the attribute doesn't have a value.
//
// Numeric value of the sample.
func (o *Sample) Value() float64 {
	if o != nil && o.bitmap_&2 != 0 {
		return o.value
	}
	return 0.0
}

// GetValue returns the value of the 'value' attribute and
// a flag indicating if the attribute has a value.
//
// Numeric value of the sample.
func (o *Sample) GetValue() (value float64, ok bool) {
	ok = o != nil && o.bitmap_&2 != 0
	if ok {
		value = o.value
	}
	return
}

// SampleListKind is the name of the type used to represent list of objects of
// type 'sample'.
const SampleListKind = "SampleList"

// SampleListLinkKind is the name of the type used to represent links to list
// of objects of type 'sample'.
const SampleListLinkKind = "SampleListLink"

// SampleNilKind is the name of the type used to nil lists of objects of
// type 'sample'.
const SampleListNilKind = "SampleListNil"

// SampleList is a list of values of the 'sample' type.
type SampleList struct {
	href  string
	link  bool
	items []*Sample
}

// Len returns the length of the list.
func (l *SampleList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Empty returns true if the list is empty.
func (l *SampleList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *SampleList) Get(i int) *Sample {
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
func (l *SampleList) Slice() []*Sample {
	var slice []*Sample
	if l == nil {
		slice = make([]*Sample, 0)
	} else {
		slice = make([]*Sample, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *SampleList) Each(f func(item *Sample) bool) {
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
func (l *SampleList) Range(f func(index int, item *Sample) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
