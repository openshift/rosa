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

// CustomIAMRolesBuilder contains the data and logic needed to build 'custom_IAM_roles' objects.
//
// Contains the necessary attributes to support role-based authentication on AWS.
type CustomIAMRolesBuilder struct {
	bitmap_       uint32
	masterIAMRole string
	workerIAMRole string
}

// NewCustomIAMRoles creates a new builder of 'custom_IAM_roles' objects.
func NewCustomIAMRoles() *CustomIAMRolesBuilder {
	return &CustomIAMRolesBuilder{}
}

// MasterIAMRole sets the value of the 'master_IAM_role' attribute to the given value.
//
//
func (b *CustomIAMRolesBuilder) MasterIAMRole(value string) *CustomIAMRolesBuilder {
	b.masterIAMRole = value
	b.bitmap_ |= 1
	return b
}

// WorkerIAMRole sets the value of the 'worker_IAM_role' attribute to the given value.
//
//
func (b *CustomIAMRolesBuilder) WorkerIAMRole(value string) *CustomIAMRolesBuilder {
	b.workerIAMRole = value
	b.bitmap_ |= 2
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *CustomIAMRolesBuilder) Copy(object *CustomIAMRoles) *CustomIAMRolesBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.masterIAMRole = object.masterIAMRole
	b.workerIAMRole = object.workerIAMRole
	return b
}

// Build creates a 'custom_IAM_roles' object using the configuration stored in the builder.
func (b *CustomIAMRolesBuilder) Build() (object *CustomIAMRoles, err error) {
	object = new(CustomIAMRoles)
	object.bitmap_ = b.bitmap_
	object.masterIAMRole = b.masterIAMRole
	object.workerIAMRole = b.workerIAMRole
	return
}
