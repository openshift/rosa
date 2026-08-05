package hyperfleet

import (
	"fmt"
	"strings"

	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// ComputeRolesRef builds the AWSRolesRef from an operator roles prefix, AWS account ID, and
// partition. Role names follow the convention established by the Platform API IAM CloudFormation
// template.
func ComputeRolesRef(prefix, accountID, partition string) hypershiftv1beta1.AWSRolesRef {
	arn := func(suffix string) string {
		return fmt.Sprintf("arn:%s:iam::%s:role/%s%s", partition, accountID, prefix, suffix)
	}
	return hypershiftv1beta1.AWSRolesRef{
		IngressARN:              arn("-ingress"),
		KubeCloudControllerARN:  arn("-cloud-controller-manager"),
		StorageARN:              arn("-ebs-csi"),
		ImageRegistryARN:        arn("-image-registry"),
		NetworkARN:              arn("-network-config"),
		ControlPlaneOperatorARN: arn("-control-plane-operator"),
		NodePoolManagementARN:   arn("-node-pool-management"),
	}
}

// ComputeInstanceProfile returns the worker node instance profile name for a given prefix.
func ComputeInstanceProfile(prefix string) string {
	return prefix + "-ROSA-Worker-Role"
}

// InstanceProfileFromRolesRef derives the worker instance profile name from a cluster's
// RolesRef by extracting the operator roles prefix from the NodePoolManagementARN.
// Returns an empty string if the ARN is not set or cannot be parsed.
func InstanceProfileFromRolesRef(rolesRef hypershiftv1beta1.AWSRolesRef) string {
	arn := rolesRef.NodePoolManagementARN
	// ARN format: arn:aws:iam::<account>:role/<prefix>-node-pool-management
	slash := strings.LastIndex(arn, "/")
	if slash < 0 {
		return ""
	}
	roleName := arn[slash+1:]
	const suffix = "-node-pool-management"
	prefix, found := strings.CutSuffix(roleName, suffix)
	if !found {
		return ""
	}
	return ComputeInstanceProfile(prefix)
}
