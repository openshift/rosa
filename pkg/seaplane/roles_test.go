// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// sentinelRolesRef returns an AWSRolesRef with a distinct sentinel ARN per
// field so that a field-mapping mistake in the delegation layer cannot go
// unnoticed: each ARN must land on the field of the same name.
func sentinelRolesRef() hypershiftv1beta1.AWSRolesRef {
	return hypershiftv1beta1.AWSRolesRef{
		IngressARN:              "arn:aws:iam::123456789012:role/prefix-ingress",
		KubeCloudControllerARN:  "arn:aws:iam::123456789012:role/prefix-cloud-controller-manager",
		StorageARN:              "arn:aws:iam::123456789012:role/prefix-ebs-csi",
		ImageRegistryARN:        "arn:aws:iam::123456789012:role/prefix-image-registry",
		NetworkARN:              "arn:aws:iam::123456789012:role/prefix-network-config",
		ControlPlaneOperatorARN: "arn:aws:iam::123456789012:role/prefix-control-plane-operator",
		NodePoolManagementARN:   "arn:aws:iam::123456789012:role/prefix-node-pool-management",
	}
}

var _ = Describe("Seaplane roles delegation", func() {
	Context("ComputeRolesRef", func() {
		It("maps each iam module role to the HyperShift field of the same name", func() {
			rolesRef := ComputeRolesRef("prefix", "123456789012", "aws")
			suffixes := []string{
				"ingress", "cloud-controller-manager", "ebs-csi", "image-registry",
				"network-config", "control-plane-operator", "node-pool-management",
			}
			// Address each field via a helper closure per expected ARN so a
			// swap between any two roles fails the assertion below.
			arns := []string{
				rolesRef.IngressARN,
				rolesRef.KubeCloudControllerARN,
				rolesRef.StorageARN,
				rolesRef.ImageRegistryARN,
				rolesRef.NetworkARN,
				rolesRef.ControlPlaneOperatorARN,
				rolesRef.NodePoolManagementARN,
			}
			for i, suffix := range suffixes {
				Expect(arns[i]).To(Equal(
					fmt.Sprintf("arn:aws:iam::123456789012:role/prefix-%s", suffix)),
					"role %s mapped to the wrong field", suffix)
			}
		})
	})

	Context("toIAMRolesRef", func() {
		It("maps each HyperShift field to the iam module field of the same name", func() {
			in := sentinelRolesRef()
			out := toIAMRolesRef(in)
			Expect(out.IngressARN).To(Equal(in.IngressARN))
			Expect(out.KubeCloudControllerARN).To(Equal(in.KubeCloudControllerARN))
			Expect(out.StorageARN).To(Equal(in.StorageARN))
			Expect(out.ImageRegistryARN).To(Equal(in.ImageRegistryARN))
			Expect(out.NetworkARN).To(Equal(in.NetworkARN))
			Expect(out.ControlPlaneOperatorARN).To(Equal(in.ControlPlaneOperatorARN))
			Expect(out.NodePoolManagementARN).To(Equal(in.NodePoolManagementARN))
		})

		It("round-trips through the iam module", func() {
			in := sentinelRolesRef()
			Expect(InstanceProfileFromRolesRef(in)).To(Equal("prefix-ROSA-Worker-Role"))
			Expect(OperatorRolesPrefixFromRolesRef(in)).To(Equal("prefix"))
		})
	})
})
