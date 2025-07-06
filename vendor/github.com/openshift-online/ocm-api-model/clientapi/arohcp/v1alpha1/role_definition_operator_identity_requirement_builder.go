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

package v1alpha1 // github.com/openshift-online/ocm-api-model/clientapi/arohcp/v1alpha1

// RoleDefinitionOperatorIdentityRequirementBuilder contains the data and logic needed to build 'role_definition_operator_identity_requirement' objects.
type RoleDefinitionOperatorIdentityRequirementBuilder struct {
	bitmap_    uint32
	name       string
	resourceId string
}

// NewRoleDefinitionOperatorIdentityRequirement creates a new builder of 'role_definition_operator_identity_requirement' objects.
func NewRoleDefinitionOperatorIdentityRequirement() *RoleDefinitionOperatorIdentityRequirementBuilder {
	return &RoleDefinitionOperatorIdentityRequirementBuilder{}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *RoleDefinitionOperatorIdentityRequirementBuilder) Empty() bool {
	return b == nil || b.bitmap_ == 0
}

// Name sets the value of the 'name' attribute to the given value.
func (b *RoleDefinitionOperatorIdentityRequirementBuilder) Name(value string) *RoleDefinitionOperatorIdentityRequirementBuilder {
	b.name = value
	b.bitmap_ |= 1
	return b
}

// ResourceId sets the value of the 'resource_id' attribute to the given value.
func (b *RoleDefinitionOperatorIdentityRequirementBuilder) ResourceId(value string) *RoleDefinitionOperatorIdentityRequirementBuilder {
	b.resourceId = value
	b.bitmap_ |= 2
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *RoleDefinitionOperatorIdentityRequirementBuilder) Copy(object *RoleDefinitionOperatorIdentityRequirement) *RoleDefinitionOperatorIdentityRequirementBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.name = object.name
	b.resourceId = object.resourceId
	return b
}

// Build creates a 'role_definition_operator_identity_requirement' object using the configuration stored in the builder.
func (b *RoleDefinitionOperatorIdentityRequirementBuilder) Build() (object *RoleDefinitionOperatorIdentityRequirement, err error) {
	object = new(RoleDefinitionOperatorIdentityRequirement)
	object.bitmap_ = b.bitmap_
	object.name = b.name
	object.resourceId = b.resourceId
	return
}
