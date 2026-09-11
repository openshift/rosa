package hyperfleet

import (
	"fmt"
	"strings"

	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

const (
	SuffixIngress                = "ingress"
	SuffixCloudControllerManager = "cloud-controller-manager"
	SuffixEBSCSI                 = "ebs-csi"
	SuffixImageRegistry          = "image-registry"
	SuffixNetworkConfig          = "network-config"
	SuffixControlPlaneOperator   = "control-plane-operator"
	SuffixNodePoolManagement     = "node-pool-management"
	SuffixWorkerRole             = "ROSA-Worker-Role"
)

var operatorRoleSuffixes = []string{
	SuffixIngress,
	SuffixCloudControllerManager,
	SuffixEBSCSI,
	SuffixImageRegistry,
	SuffixNetworkConfig,
	SuffixControlPlaneOperator,
	SuffixNodePoolManagement,
	SuffixWorkerRole,
}

// OperatorRoleSuffixes returns IAM role name suffixes (after "<prefix>-") for hyperfleet v2 HCP.
func OperatorRoleSuffixes() []string {
	return operatorRoleSuffixes
}

// ComputeRolesRef builds the AWSRolesRef from an operator roles prefix, AWS account ID, and
// partition. Role names follow the convention established by the Platform API IAM CloudFormation
// template.
func ComputeRolesRef(prefix, accountID, partition string) hypershiftv1beta1.AWSRolesRef {
	arn := func(suffix string) string {
		return fmt.Sprintf("arn:%s:iam::%s:role/%s-%s", partition, accountID, prefix, suffix)
	}
	return hypershiftv1beta1.AWSRolesRef{
		IngressARN:              arn(SuffixIngress),
		KubeCloudControllerARN:  arn(SuffixCloudControllerManager),
		StorageARN:              arn(SuffixEBSCSI),
		ImageRegistryARN:        arn(SuffixImageRegistry),
		NetworkARN:              arn(SuffixNetworkConfig),
		ControlPlaneOperatorARN: arn(SuffixControlPlaneOperator),
		NodePoolManagementARN:   arn(SuffixNodePoolManagement),
	}
}

// ComputeInstanceProfile returns the worker node instance profile name for a given prefix.
func ComputeInstanceProfile(prefix string) string {
	return prefix + "-" + SuffixWorkerRole
}

// OperatorRoleNames returns IAM role names created by hyperfleet v2 operator-roles create.
func OperatorRoleNames(prefix string) []string {
	names := make([]string, len(operatorRoleSuffixes))
	for i, suffix := range operatorRoleSuffixes {
		names[i] = prefix + "-" + suffix
	}
	return names
}

// InstanceProfileFromRolesRef derives the worker instance profile name from a cluster's
// RolesRef by extracting the operator roles prefix from the NodePoolManagementARN.
// Returns an empty string if the ARN is not set or cannot be parsed.
func InstanceProfileFromRolesRef(rolesRef hypershiftv1beta1.AWSRolesRef) string {
	arn := rolesRef.NodePoolManagementARN
	slash := strings.LastIndex(arn, "/")
	if slash < 0 {
		return ""
	}
	roleName := arn[slash+1:]
	nodePoolSuffix := "-" + SuffixNodePoolManagement
	prefix, found := strings.CutSuffix(roleName, nodePoolSuffix)
	if !found {
		return ""
	}
	return ComputeInstanceProfile(prefix)
}
