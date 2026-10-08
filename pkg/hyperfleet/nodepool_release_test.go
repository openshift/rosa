package hyperfleet

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

var _ = Describe("Node pool release image resolution", func() {
	DescribeTable("resolves only versions with a known release image",
		func(requestedVersion, clusterVersion, clusterImage, explicitImage, expectedError string) {
			cluster := &v1alpha1.Cluster{
				Spec: v1alpha1.ClusterSpec{HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
					Release: hypershiftv1beta1.Release{Image: clusterImage},
				}},
				Status: v1alpha1.ClusterStatus{Version: clusterVersion},
			}
			image, err := ResolveNodePoolReleaseImage(cluster, requestedVersion, explicitImage)
			if expectedError == "" {
				Expect(err).NotTo(HaveOccurred())
				Expect(image).To(Equal(clusterImage))
			} else {
				Expect(err).To(MatchError(ContainSubstring(expectedError)))
				Expect(image).To(BeEmpty())
			}
		},
		Entry("current version", "4.20.3", "4.20.3", "quay.io/release@sha256:abc", "", ""),
		Entry("prerelease version", "5.0.0-ec.6", "5.0.0-ec.6", "quay.io/release:5.0.0-ec.6", "", ""),
		Entry("unsupported version from 56786", "4.12.1", "5.0.0-ec.6", "", "", "expected a valid OpenShift version"),
		Entry("unknown newer version", "4.21.1", "4.20.3", "quay.io/release@sha256:abc", "", "cannot be resolved"),
		Entry("malformed version", "wrong", "4.20.3", "quay.io/release@sha256:abc", "", "expected a valid OpenShift version"),
		Entry("unreported cluster version", "4.20.3", "", "quay.io/release@sha256:abc", "", "cannot be resolved"),
		Entry("missing cluster image", "4.20.3", "4.20.3", "", "", "no configured release image"),
		Entry("conflicting flags", "4.20.3", "4.20.3", "quay.io/release@sha256:abc", "another-image", "--version and --image"),
	)
})
