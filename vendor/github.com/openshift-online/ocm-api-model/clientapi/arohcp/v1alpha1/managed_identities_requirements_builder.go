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

// ManagedIdentitiesRequirementsBuilder contains the data and logic needed to build 'managed_identities_requirements' objects.
//
// Representation of managed identities requirements.
// When creating ARO-HCP Clusters, the end-users will need to pre-create the set of Managed Identities
// required by the clusters.
// The set of Managed Identities that the end-users need to precreate is not static and depends on
// several factors:
// (1) The OpenShift version of the cluster being created.
// (2) The functionalities that are being enabled for the cluster. Some Managed Identities are not
// always required but become required if a given functionality is enabled.
// Additionally, the Managed Identities that the end-users will need to precreate will have to have a
// set of required permissions assigned to them which also have to be returned to the end users.
type ManagedIdentitiesRequirementsBuilder struct {
	bitmap_                         uint32
	id                              string
	href                            string
	controlPlaneOperatorsIdentities []*ControlPlaneOperatorIdentityRequirementBuilder
	dataPlaneOperatorsIdentities    []*DataPlaneOperatorIdentityRequirementBuilder
}

// NewManagedIdentitiesRequirements creates a new builder of 'managed_identities_requirements' objects.
func NewManagedIdentitiesRequirements() *ManagedIdentitiesRequirementsBuilder {
	return &ManagedIdentitiesRequirementsBuilder{}
}

// Link sets the flag that indicates if this is a link.
func (b *ManagedIdentitiesRequirementsBuilder) Link(value bool) *ManagedIdentitiesRequirementsBuilder {
	b.bitmap_ |= 1
	return b
}

// ID sets the identifier of the object.
func (b *ManagedIdentitiesRequirementsBuilder) ID(value string) *ManagedIdentitiesRequirementsBuilder {
	b.id = value
	b.bitmap_ |= 2
	return b
}

// HREF sets the link to the object.
func (b *ManagedIdentitiesRequirementsBuilder) HREF(value string) *ManagedIdentitiesRequirementsBuilder {
	b.href = value
	b.bitmap_ |= 4
	return b
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *ManagedIdentitiesRequirementsBuilder) Empty() bool {
	return b == nil || b.bitmap_&^1 == 0
}

// ControlPlaneOperatorsIdentities sets the value of the 'control_plane_operators_identities' attribute to the given values.
func (b *ManagedIdentitiesRequirementsBuilder) ControlPlaneOperatorsIdentities(values ...*ControlPlaneOperatorIdentityRequirementBuilder) *ManagedIdentitiesRequirementsBuilder {
	b.controlPlaneOperatorsIdentities = make([]*ControlPlaneOperatorIdentityRequirementBuilder, len(values))
	copy(b.controlPlaneOperatorsIdentities, values)
	b.bitmap_ |= 8
	return b
}

// DataPlaneOperatorsIdentities sets the value of the 'data_plane_operators_identities' attribute to the given values.
func (b *ManagedIdentitiesRequirementsBuilder) DataPlaneOperatorsIdentities(values ...*DataPlaneOperatorIdentityRequirementBuilder) *ManagedIdentitiesRequirementsBuilder {
	b.dataPlaneOperatorsIdentities = make([]*DataPlaneOperatorIdentityRequirementBuilder, len(values))
	copy(b.dataPlaneOperatorsIdentities, values)
	b.bitmap_ |= 16
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *ManagedIdentitiesRequirementsBuilder) Copy(object *ManagedIdentitiesRequirements) *ManagedIdentitiesRequirementsBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.id = object.id
	b.href = object.href
	if object.controlPlaneOperatorsIdentities != nil {
		b.controlPlaneOperatorsIdentities = make([]*ControlPlaneOperatorIdentityRequirementBuilder, len(object.controlPlaneOperatorsIdentities))
		for i, v := range object.controlPlaneOperatorsIdentities {
			b.controlPlaneOperatorsIdentities[i] = NewControlPlaneOperatorIdentityRequirement().Copy(v)
		}
	} else {
		b.controlPlaneOperatorsIdentities = nil
	}
	if object.dataPlaneOperatorsIdentities != nil {
		b.dataPlaneOperatorsIdentities = make([]*DataPlaneOperatorIdentityRequirementBuilder, len(object.dataPlaneOperatorsIdentities))
		for i, v := range object.dataPlaneOperatorsIdentities {
			b.dataPlaneOperatorsIdentities[i] = NewDataPlaneOperatorIdentityRequirement().Copy(v)
		}
	} else {
		b.dataPlaneOperatorsIdentities = nil
	}
	return b
}

// Build creates a 'managed_identities_requirements' object using the configuration stored in the builder.
func (b *ManagedIdentitiesRequirementsBuilder) Build() (object *ManagedIdentitiesRequirements, err error) {
	object = new(ManagedIdentitiesRequirements)
	object.id = b.id
	object.href = b.href
	object.bitmap_ = b.bitmap_
	if b.controlPlaneOperatorsIdentities != nil {
		object.controlPlaneOperatorsIdentities = make([]*ControlPlaneOperatorIdentityRequirement, len(b.controlPlaneOperatorsIdentities))
		for i, v := range b.controlPlaneOperatorsIdentities {
			object.controlPlaneOperatorsIdentities[i], err = v.Build()
			if err != nil {
				return
			}
		}
	}
	if b.dataPlaneOperatorsIdentities != nil {
		object.dataPlaneOperatorsIdentities = make([]*DataPlaneOperatorIdentityRequirement, len(b.dataPlaneOperatorsIdentities))
		for i, v := range b.dataPlaneOperatorsIdentities {
			object.dataPlaneOperatorsIdentities[i], err = v.Build()
			if err != nil {
				return
			}
		}
	}
	return
}
