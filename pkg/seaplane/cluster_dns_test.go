// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"context"
	"errors"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

var _ = Describe("ClusterIDsByDNSBaseDomain", func() {
	ctx := context.Background()

	It("maps in-use base domains to the first cluster UID", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, _ := newResolveMocks(ctrl)

		clusters.EXPECT().List(gomock.Any(), gomock.Eq(platform.ListOptions{})).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{
				clusterWithDNSDomain("cluster-one", "shared.example.com"),
				clusterWithDNSDomain("cluster-two", "shared.example.com"),
				clusterWithDNSDomain("cluster-no-dns", ""),
				clusterWithDNSDomain("cluster-three", "other.example.com"),
			}}, nil)

		clusterIDs, err := ClusterIDsByDNSBaseDomain(ctx, seaplaneClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(clusterIDs).To(Equal(map[string]string{
			"shared.example.com": "cluster-one",
			"other.example.com":  "cluster-three",
		}))
	})

	It("returns an error when the cluster list call fails", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, _ := newResolveMocks(ctrl)

		clusters.EXPECT().List(gomock.Any(), gomock.Eq(platform.ListOptions{})).Return(
			nil, errors.New("connection refused"))

		clusterIDs, err := ClusterIDsByDNSBaseDomain(ctx, seaplaneClient)
		Expect(clusterIDs).To(BeNil())
		Expect(err).To(MatchError("failed to list clusters: connection refused"))
	})
})

func clusterWithDNSDomain(uid, baseDomain string) v1alpha1.Cluster {
	return v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{UID: types.UID(uid)},
		Spec: v1alpha1.ClusterSpec{
			HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
				DNS: hypershiftv1beta1.DNSSpec{BaseDomain: baseDomain},
			},
		},
	}
}
