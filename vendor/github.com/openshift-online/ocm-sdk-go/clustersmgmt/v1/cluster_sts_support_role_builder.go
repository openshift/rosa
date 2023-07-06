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

// ClusterStsSupportRoleBuilder contains the data and logic needed to build 'cluster_sts_support_role' objects.
//
// Isolated STS support role created per organization.
type ClusterStsSupportRoleBuilder struct {
	bitmap_ uint32
	roleArn string
}

// NewClusterStsSupportRole creates a new builder of 'cluster_sts_support_role' objects.
func NewClusterStsSupportRole() *ClusterStsSupportRoleBuilder {
	return &ClusterStsSupportRoleBuilder{}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *ClusterStsSupportRoleBuilder) Empty() bool {
	return b == nil || b.bitmap_ == 0
}

// RoleArn sets the value of the 'role_arn' attribute to the given value.
func (b *ClusterStsSupportRoleBuilder) RoleArn(value string) *ClusterStsSupportRoleBuilder {
	b.roleArn = value
	b.bitmap_ |= 1
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *ClusterStsSupportRoleBuilder) Copy(object *ClusterStsSupportRole) *ClusterStsSupportRoleBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	b.roleArn = object.roleArn
	return b
}

// Build creates a 'cluster_sts_support_role' object using the configuration stored in the builder.
func (b *ClusterStsSupportRoleBuilder) Build() (object *ClusterStsSupportRole, err error) {
	object = new(ClusterStsSupportRole)
	object.bitmap_ = b.bitmap_
	object.roleArn = b.roleArn
	return
}
