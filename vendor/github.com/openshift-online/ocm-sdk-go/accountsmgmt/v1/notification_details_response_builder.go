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

// NotificationDetailsResponseBuilder contains the data and logic needed to build 'notification_details_response' objects.
//
// This struct is a request to get a templated email to a user related to this.
// subscription/cluster.
type NotificationDetailsResponseBuilder struct {
	bitmap_       uint32
	associates    []string
	externalOrgID string
	recipients    []string
}

// NewNotificationDetailsResponse creates a new builder of 'notification_details_response' objects.
func NewNotificationDetailsResponse() *NotificationDetailsResponseBuilder {
	return &NotificationDetailsResponseBuilder{}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *NotificationDetailsResponseBuilder) Empty() bool {
	return b == nil || b.bitmap_ == 0
}

// Associates sets the value of the 'associates' attribute to the given values.
func (b *NotificationDetailsResponseBuilder) Associates(values ...string) *NotificationDetailsResponseBuilder {
	b.associates = make([]string, len(values))
	copy(b.associates, values)
	b.bitmap_ |= 1
	return b
}

// ExternalOrgID sets the value of the 'external_org_ID' attribute to the given value.
func (b *NotificationDetailsResponseBuilder) ExternalOrgID(value string) *NotificationDetailsResponseBuilder {
	b.externalOrgID = value
	b.bitmap_ |= 2
	return b
}

// Recipients sets the value of the 'recipients' attribute to the given values.
func (b *NotificationDetailsResponseBuilder) Recipients(values ...string) *NotificationDetailsResponseBuilder {
	b.recipients = make([]string, len(values))
	copy(b.recipients, values)
	b.bitmap_ |= 4
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *NotificationDetailsResponseBuilder) Copy(object *NotificationDetailsResponse) *NotificationDetailsResponseBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	if object.associates != nil {
		b.associates = make([]string, len(object.associates))
		copy(b.associates, object.associates)
	} else {
		b.associates = nil
	}
	b.externalOrgID = object.externalOrgID
	if object.recipients != nil {
		b.recipients = make([]string, len(object.recipients))
		copy(b.recipients, object.recipients)
	} else {
		b.recipients = nil
	}
	return b
}

// Build creates a 'notification_details_response' object using the configuration stored in the builder.
func (b *NotificationDetailsResponseBuilder) Build() (object *NotificationDetailsResponse, err error) {
	object = new(NotificationDetailsResponse)
	object.bitmap_ = b.bitmap_
	if b.associates != nil {
		object.associates = make([]string, len(b.associates))
		copy(object.associates, b.associates)
	}
	object.externalOrgID = b.externalOrgID
	if b.recipients != nil {
		object.recipients = make([]string, len(b.recipients))
		copy(object.recipients, b.recipients)
	}
	return
}
