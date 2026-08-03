package hyperfleet

import (
	"fmt"

	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// ComputeRolesRef builds the AWSRolesRef from an operator roles prefix and AWS account ID.
// Role names follow the convention established by the Platform API IAM CloudFormation template.
func ComputeRolesRef(prefix, accountID string) hypershiftv1beta1.AWSRolesRef {
	arn := func(suffix string) string {
		return fmt.Sprintf("arn:aws:iam::%s:role/%s%s", accountID, prefix, suffix)
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
