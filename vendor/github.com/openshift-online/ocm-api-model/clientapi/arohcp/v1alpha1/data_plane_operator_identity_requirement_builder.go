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

// DataPlaneOperatorIdentityRequirementBuilder contains the data and logic needed to build 'data_plane_operator_identity_requirement' objects.
type DataPlaneOperatorIdentityRequirementBuilder struct {
	bitmap_             uint32
	maxOpenShiftVersion string
	minOpenShiftVersion string
	operatorName        string
	required            string
	roleDefinitions     []*RoleDefinitionOperatorIdentityRequirementBuilder
	serviceAccounts     []*K8sServiceAccountOperatorIdentityRequirementBuilder
}

// NewDataPlaneOperatorIdentityRequirement creates a new builder of 'data_plane_operator_identity_requirement' objects.
func NewDataPlaneOperatorIdentityRequirement() *DataPlaneOperatorIdentityRequirementBuilder {
	return &DataPlaneOperatorIdentityRequirementBuilder{}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *DataPlaneOperatorIdentityRequirementBuilder) Empty() bool {
	return b == nil || b.bitmap_ == 0
}

// MaxOpenShiftVersion sets the value of the 'max_open_shift_version' attribute to the given value.
func (b *DataPlaneOperatorIdentityRequirementBuilder) MaxOpenShiftVersion(value string) *DataPlaneOperatorIdentityRequirementBuilder {
	b.maxOpenShiftVersion = value
	b.bitmap_ |= 1
	return b
}

// MinOpenShiftVersion sets the value of the 'min_open_shift_version' attribute to the given value.
func (b *DataPlaneOperatorIdentityRequirementBuilder) MinOpenShiftVersion(value string) *DataPlaneOperatorIdentityRequirementBuilder {
	b.minOpenShiftVersion = value
	b.bitmap_ |= 2
	return b
}

// OperatorName sets the value of the 'operator_name' attribute to the given value.
func (b *DataPlaneOperatorIdentityRequirementBuilder) OperatorName(value string) *DataPlaneOperatorIdentityRequirementBuilder {
	b.operatorName = value
	b.bitmap_ |= 4
	return b
}

// Required sets the value of the 'required' attribute to the given value.
func (b *DataPlaneOperatorIdentityRequirementBuilder) Required(value string) *DataPlaneOperatorIdentityRequirementBuilder {
	b.required = value
	b.bitmap_ |= 8
	return b
}

// RoleDefinitions sets the value of the 'role_definitions' attribute to the given values.
func (b *DataPlaneOperatorIdentityRequirementBuilder) RoleDefinitions(values ...*RoleDefinitionOperatorIdentityRequirementBuilder) *DataPlaneOperatorIdentityRequirementBuilder {
	b.roleDefinitions = make([]*RoleDefinitionOperatorIdentityRequirementBuilder, len(values))
	copy(b.roleDefinitions, values)
	b.bitmap_ |= 16
	return b
}

// ServiceAccounts sets the value of the 'service_accounts' attribute to the given values.
func (b *DataPlaneOperatorIdentityRequirementBuilder) ServiceAccounts(values ...*K8sServiceAccountOperatorIdentityRequirementBuilder) *DataPlaneOperatorIdentityRequirementBuilder {
	b.serviceAccounts = make([]*K8sServiceAccountOperatorIdentityRequirementBuilder, len(values))
	copy(b.serviceAccounts, values)
	b.bitmap_ |= 32
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *DataPlaneOperatorIdentityRequirementBuilder) Copy(object *DataPlaneOperatorIdentityRequirement) *DataPlaneOperatorIdentityRequirementBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.maxOpenShiftVersion = object.maxOpenShiftVersion
	b.minOpenShiftVersion = object.minOpenShiftVersion
	b.operatorName = object.operatorName
	b.required = object.required
	if object.roleDefinitions != nil {
		b.roleDefinitions = make([]*RoleDefinitionOperatorIdentityRequirementBuilder, len(object.roleDefinitions))
		for i, v := range object.roleDefinitions {
			b.roleDefinitions[i] = NewRoleDefinitionOperatorIdentityRequirement().Copy(v)
		}
	} else {
		b.roleDefinitions = nil
	}
	if object.serviceAccounts != nil {
		b.serviceAccounts = make([]*K8sServiceAccountOperatorIdentityRequirementBuilder, len(object.serviceAccounts))
		for i, v := range object.serviceAccounts {
			b.serviceAccounts[i] = NewK8sServiceAccountOperatorIdentityRequirement().Copy(v)
		}
	} else {
		b.serviceAccounts = nil
	}
	return b
}

// Build creates a 'data_plane_operator_identity_requirement' object using the configuration stored in the builder.
func (b *DataPlaneOperatorIdentityRequirementBuilder) Build() (object *DataPlaneOperatorIdentityRequirement, err error) {
	object = new(DataPlaneOperatorIdentityRequirement)
	object.bitmap_ = b.bitmap_
	object.maxOpenShiftVersion = b.maxOpenShiftVersion
	object.minOpenShiftVersion = b.minOpenShiftVersion
	object.operatorName = b.operatorName
	object.required = b.required
	if b.roleDefinitions != nil {
		object.roleDefinitions = make([]*RoleDefinitionOperatorIdentityRequirement, len(b.roleDefinitions))
		for i, v := range b.roleDefinitions {
			object.roleDefinitions[i], err = v.Build()
			if err != nil {
				return
			}
		}
	}
	if b.serviceAccounts != nil {
		object.serviceAccounts = make([]*K8sServiceAccountOperatorIdentityRequirement, len(b.serviceAccounts))
		for i, v := range b.serviceAccounts {
			object.serviceAccounts[i], err = v.Build()
			if err != nil {
				return
			}
		}
	}
	return
}
