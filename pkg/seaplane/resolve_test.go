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

	seaplanemocks "github.com/openshift/rosa/pkg/seaplane/mocks"
)

func newResolveMocks(ctrl *gomock.Controller) (
	*seaplanemocks.MockInterface,
	*seaplanemocks.MockClusterInterface,
	*seaplanemocks.MockNodePoolInterface,
) {
	seaplaneClient := seaplanemocks.NewMockInterface(ctrl)
	v1 := seaplanemocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := seaplanemocks.NewMockClusterInterface(ctrl)
	nodePools := seaplanemocks.NewMockNodePoolInterface(ctrl)
	seaplaneClient.EXPECT().SeaplaneV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	v1.EXPECT().NodePools().Return(nodePools).AnyTimes()
	return seaplaneClient, clusters, nodePools
}

var _ = Describe("ResolveClusterUID", func() {
	ctx := context.Background()

	It("returns the UID when a matching cluster is found", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, _ := newResolveMocks(ctrl)

		cluster := v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "my-cluster", UID: types.UID("cluster-uid-123")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{cluster}}, nil)

		uid, err := ResolveClusterUID(ctx, seaplaneClient, "my-cluster")
		Expect(err).ToNot(HaveOccurred())
		Expect(uid).To(Equal("cluster-uid-123"))
	})

	It("returns the UID when the key is already a cluster UID", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, _ := newResolveMocks(ctrl)

		cluster := v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "my-cluster", UID: types.UID("cluster-uid-123")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{cluster}}, nil)

		uid, err := ResolveClusterUID(ctx, seaplaneClient, "cluster-uid-123")
		Expect(err).ToNot(HaveOccurred())
		Expect(uid).To(Equal("cluster-uid-123"))
	})

	It("returns an error when no cluster matches the name", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, _ := newResolveMocks(ctrl)

		cluster := v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "other-cluster", UID: types.UID("uid-other")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{cluster}}, nil)

		_, err := ResolveClusterUID(ctx, seaplaneClient, "my-cluster")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})

	It("returns an error when the cluster list call fails", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, _ := newResolveMocks(ctrl)

		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("connection refused"))

		_, err := ResolveClusterUID(ctx, seaplaneClient, "any")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to list clusters"))
	})
})

var _ = Describe("ResolveNodePoolUID", func() {
	ctx := context.Background()

	It("returns the UID when a matching node pool is found", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newResolveMocks(ctrl)

		np := v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.my-nodepool", UID: types.UID("np-uid-456")},
		}
		nodePools.EXPECT().List(gomock.Any(), gomock.Eq(platform.NodePoolListOptions{ClusterUID: "cluster-uid"})).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{np}}, nil)

		uid, err := ResolveNodePoolUID(ctx, seaplaneClient, "cluster-uid", "my-nodepool")
		Expect(err).ToNot(HaveOccurred())
		Expect(uid).To(Equal("np-uid-456"))
	})

	It("returns an error when no node pool matches the name", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newResolveMocks(ctrl)

		np := v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "other-nodepool", UID: types.UID("uid-other")},
		}
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{np}}, nil)

		_, err := ResolveNodePoolUID(ctx, seaplaneClient, "cluster-uid", "my-nodepool")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("machine pool 'my-nodepool' does not exist"))
	})

	It("returns an error when the node pool list call fails", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newResolveMocks(ctrl)

		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("timeout"))

		_, err := ResolveNodePoolUID(ctx, seaplaneClient, "cluster-uid", "any")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to list node pools"))
	})
})

var _ = Describe("ResolveNodePoolName", func() {
	ctx := context.Background()

	It("maps a CLI child name to the account-scoped resource name", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newResolveMocks(ctrl)

		nodePools.EXPECT().List(gomock.Any(), gomock.Eq(platform.NodePoolListOptions{ClusterUID: "cluster-uid"})).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{{
				ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers"},
			}}}, nil)

		name, err := ResolveNodePoolName(ctx, seaplaneClient, "cluster-uid", "workers")
		Expect(err).NotTo(HaveOccurred())
		Expect(name).To(Equal("my-cluster.workers"))
	})

	It("accepts an account-scoped resource name", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newResolveMocks(ctrl)

		nodePools.EXPECT().List(gomock.Any(), gomock.Eq(platform.NodePoolListOptions{ClusterUID: "cluster-uid"})).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{{
				ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers"},
			}}}, nil)

		name, err := ResolveNodePoolName(ctx, seaplaneClient, "cluster-uid", "my-cluster.workers")
		Expect(err).NotTo(HaveOccurred())
		Expect(name).To(Equal("my-cluster.workers"))
	})
})

var _ = Describe("NodePoolDisplayName", func() {
	It("strips the cluster prefix from account-scoped resource names", func() {
		Expect(NodePoolDisplayName("my-cluster", "my-cluster.workers")).To(Equal("workers"))
	})

	It("leaves names without the cluster prefix unchanged", func() {
		Expect(NodePoolDisplayName("my-cluster", "workers")).To(Equal("workers"))
	})
})
