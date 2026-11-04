// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	iam "github.com/openshift-online/rosa-seaplane-api/iam"
)

// Role name suffixes are owned by the iam module (single source of truth for
// seaplane V2 operator role definitions). These aliases preserve the
// previously exported constants for existing callers.
const (
	SuffixIngress                = iam.SuffixIngress
	SuffixCloudControllerManager = iam.SuffixCloudControllerManager
	SuffixEBSCSI                 = iam.SuffixEBSCSI
	SuffixImageRegistry          = iam.SuffixImageRegistry
	SuffixNetworkConfig          = iam.SuffixNetworkConfig
	SuffixControlPlaneOperator   = iam.SuffixControlPlaneOperator
	SuffixNodePoolManagement     = iam.SuffixNodePoolManagement
	SuffixWorkerRole             = iam.SuffixWorkerRole
)

// OperatorRoleSuffixes returns IAM role name suffixes (after "<prefix>-") for seaplane v2 HCP.
func OperatorRoleSuffixes() []string {
	set := iam.DefaultDefinitionSet()
	suffixes := make([]string, len(set.Roles))
	for i, role := range set.Roles {
		suffixes[i] = role.Name
	}
	return suffixes
}

// OperatorRoleDefinitions returns the canonical V2 operator role definitions
// from the iam module (ROSAENG-75447).
func OperatorRoleDefinitions() []iam.OperatorRoleSpec {
	return iam.DefaultDefinitionSet().Roles
}

// ComputeRolesRef builds the AWSRolesRef from an operator roles prefix, AWS account ID, and
// partition. Role names follow the convention established by the Platform API IAM CloudFormation
// template.
func ComputeRolesRef(prefix, accountID, partition string) hypershiftv1beta1.AWSRolesRef {
	ref := iam.ComputeRolesRef(prefix, accountID, partition)
	return hypershiftv1beta1.AWSRolesRef{
		IngressARN:              ref.IngressARN,
		KubeCloudControllerARN:  ref.KubeCloudControllerARN,
		StorageARN:              ref.StorageARN,
		ImageRegistryARN:        ref.ImageRegistryARN,
		NetworkARN:              ref.NetworkARN,
		ControlPlaneOperatorARN: ref.ControlPlaneOperatorARN,
		NodePoolManagementARN:   ref.NodePoolManagementARN,
	}
}

// ComputeInstanceProfile returns the worker node instance profile name for a given prefix.
func ComputeInstanceProfile(prefix string) string {
	return iam.ComputeInstanceProfile(prefix)
}

// OperatorRoleNames returns IAM role names created by seaplane v2 operator-roles create.
func OperatorRoleNames(prefix string) []string {
	return iam.RoleNames(prefix)
}

// toIAMRolesRef converts hypershift's AWSRolesRef to the iam module's
// equivalent (identical fields; conversion is explicit because the structs
// declare fields in a different order).
func toIAMRolesRef(rolesRef hypershiftv1beta1.AWSRolesRef) iam.RolesRef {
	return iam.RolesRef{
		IngressARN:              rolesRef.IngressARN,
		KubeCloudControllerARN:  rolesRef.KubeCloudControllerARN,
		StorageARN:              rolesRef.StorageARN,
		ImageRegistryARN:        rolesRef.ImageRegistryARN,
		NetworkARN:              rolesRef.NetworkARN,
		ControlPlaneOperatorARN: rolesRef.ControlPlaneOperatorARN,
		NodePoolManagementARN:   rolesRef.NodePoolManagementARN,
	}
}

// InstanceProfileFromRolesRef derives the worker instance profile name from a cluster's
// RolesRef by extracting the operator roles prefix from the NodePoolManagementARN.
// Returns an empty string if the ARN is not set or cannot be parsed.
func InstanceProfileFromRolesRef(rolesRef hypershiftv1beta1.AWSRolesRef) string {
	return iam.InstanceProfileFromRolesRef(toIAMRolesRef(rolesRef))
}

// OperatorRolesPrefixFromRolesRef extracts the operator-roles prefix from a cluster's
// RolesRef (from the NodePoolManagementARN role name). Returns "" if unset/unparseable.
func OperatorRolesPrefixFromRolesRef(rolesRef hypershiftv1beta1.AWSRolesRef) string {
	return iam.OperatorRolesPrefixFromRolesRef(toIAMRolesRef(rolesRef))
}
