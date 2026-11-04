package iam

import (
	"fmt"
	"strings"
)

// RoleName builds the full IAM role name for a prefix and suffix.
func RoleName(prefix, suffix string) string {
	return prefix + "-" + suffix
}

// RoleARN builds the IAM role ARN for a prefix and suffix. A pure
// formatter: partition and accountID are expected from a trusted source
// (e.g. STS-derived caller identity); malformed values yield an ARN that
// AWS rejects. JSON-interpolating paths are validated in
// BuildWebIdentityTrustPolicy instead.
func RoleARN(partition, accountID, prefix, suffix string) string {
	return fmt.Sprintf("arn:%s:iam::%s:role/%s", partition, accountID, RoleName(prefix, suffix))
}

// RoleNames returns the IAM role names created by seaplane V2 operator-role
// creation for the given prefix, in definition order.
func RoleNames(prefix string) []string {
	return DefaultDefinitionSet().RoleNames(prefix)
}

// RoleTags returns the discovery tags applied to every operator role
// created from the embedded definitions, plus the applied definition
// version tag for drift detection (D6 reconciliation).
func RoleTags(prefix string) map[string]string {
	return map[string]string{
		TagRedHatManaged:      "true",
		TagHypershiftPolicies: "true",
		TagRolePrefix:         prefix,
		TagManagedPolicies:    "true",
		TagDefinitionVersion:  DefinitionVersion(),
	}
}

// ComputeRolesRef builds the RolesRef (operator role ARNs referenced by a
// hosted cluster) from a prefix, AWS account ID, and partition.
func ComputeRolesRef(prefix, accountID, partition string) RolesRef {
	arn := func(suffix string) string {
		return RoleARN(partition, accountID, prefix, suffix)
	}
	return RolesRef{
		IngressARN:              arn(SuffixIngress),
		KubeCloudControllerARN:  arn(SuffixCloudControllerManager),
		StorageARN:              arn(SuffixEBSCSI),
		ImageRegistryARN:        arn(SuffixImageRegistry),
		NetworkARN:              arn(SuffixNetworkConfig),
		ControlPlaneOperatorARN: arn(SuffixControlPlaneOperator),
		NodePoolManagementARN:   arn(SuffixNodePoolManagement),
	}
}

// ComputeInstanceProfile returns the worker node instance profile name for
// a given prefix.
func ComputeInstanceProfile(prefix string) string {
	return RoleName(prefix, SuffixWorkerRole)
}

// WorkerRoleName returns the worker node role name for a given prefix.
func WorkerRoleName(prefix string) string {
	return RoleName(prefix, SuffixWorkerRole)
}

// OperatorRolesPrefixFromRolesRef extracts the operator-roles prefix from a
// cluster's RolesRef (from the NodePoolManagementARN role name). Returns ""
// if unset or unparseable.
func OperatorRolesPrefixFromRolesRef(rolesRef RolesRef) string {
	arn := rolesRef.NodePoolManagementARN
	slash := strings.LastIndex(arn, "/")
	if slash < 0 {
		return ""
	}
	roleName := arn[slash+1:]
	prefix, found := strings.CutSuffix(roleName, "-"+SuffixNodePoolManagement)
	if !found {
		return ""
	}
	return prefix
}

// InstanceProfileFromRolesRef derives the worker instance profile name from
// a cluster's RolesRef. Returns "" if the ARN is not set or cannot be parsed.
func InstanceProfileFromRolesRef(rolesRef RolesRef) string {
	prefix := OperatorRolesPrefixFromRolesRef(rolesRef)
	if prefix == "" {
		return ""
	}
	return ComputeInstanceProfile(prefix)
}
