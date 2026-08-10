package hyperfleet

import (
	"context"
	"errors"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
)

func newResolveMocks(ctrl *gomock.Controller) (
	*hfmocks.MockInterface,
	*hfmocks.MockClusterInterface,
	*hfmocks.MockNodePoolInterface,
) {
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	nodePools := hfmocks.NewMockNodePoolInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	v1.EXPECT().NodePools(gomock.Any()).Return(nodePools).AnyTimes()
	return hf, clusters, nodePools
}

var _ = Describe("ResolveClusterUID", func() {
	ctx := context.Background()

	It("returns the UID when a matching cluster is found", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newResolveMocks(ctrl)

		cluster := v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "my-cluster", UID: types.UID("cluster-uid-123")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{cluster}}, nil)

		uid, err := ResolveClusterUID(ctx, hf, "my-cluster")
		Expect(err).ToNot(HaveOccurred())
		Expect(uid).To(Equal("cluster-uid-123"))
	})

	It("returns an error when no cluster matches the name", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newResolveMocks(ctrl)

		cluster := v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "other-cluster", UID: types.UID("uid-other")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{cluster}}, nil)

		_, err := ResolveClusterUID(ctx, hf, "my-cluster")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})

	It("returns an error when the cluster list call fails", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newResolveMocks(ctrl)

		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("connection refused"))

		_, err := ResolveClusterUID(ctx, hf, "any")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to list clusters"))
	})
})

var _ = Describe("ResolveNodePoolUID", func() {
	ctx := context.Background()

	It("returns the UID when a matching node pool is found", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, _, nodePools := newResolveMocks(ctrl)

		np := v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-nodepool", UID: types.UID("np-uid-456")},
		}
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{np}}, nil)

		uid, err := ResolveNodePoolUID(ctx, hf, "cluster-uid", "my-nodepool")
		Expect(err).ToNot(HaveOccurred())
		Expect(uid).To(Equal("np-uid-456"))
	})

	It("returns an error when no node pool matches the name", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, _, nodePools := newResolveMocks(ctrl)

		np := v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "other-nodepool", UID: types.UID("uid-other")},
		}
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{np}}, nil)

		_, err := ResolveNodePoolUID(ctx, hf, "cluster-uid", "my-nodepool")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})

	It("returns an error when the node pool list call fails", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, _, nodePools := newResolveMocks(ctrl)

		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("timeout"))

		_, err := ResolveNodePoolUID(ctx, hf, "cluster-uid", "any")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to list node pools"))
	})
})
