package machinepool

import (
	"context"
	"fmt"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/ocm"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/test"
)

func newCreateMPMocks(ctrl *gomock.Controller) (
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

func makeCluster(name, uid string) *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(uid)},
		Spec: v1alpha1.ClusterSpec{
			HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
				Platform: v1alpha1.PlatformSpec{
					AWS: &hypershiftv1beta1.AWSPlatformSpec{
						RolesRef: hypershiftv1beta1.AWSRolesRef{
							NodePoolManagementARN: "arn:aws:iam::123456789:role/cluster1-node-pool-management",
						},
					},
				},
			},
		},
	}
}

var _ = Describe("runHyperfleetCreate (machinepool)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("creates a node pool on the success path", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newCreateMPMocks(ctrl)

		cluster := makeCluster("cluster1", "cluster-uid")
		created := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: "np-uid-new"}}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		nodePools.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, np *v1alpha1.NodePool, _ platform.CreateOptions) (*v1alpha1.NodePool, error) {
				Expect(np.Name).To(Equal("my-np"))
				Expect(np.Spec.NodePool.ClusterName).To(Equal("cluster1"))
				Expect(*np.Spec.NodePool.Replicas).To(Equal(int32(2)))
				Expect(np.Spec.NodePool.Platform.AWS.InstanceProfile).To(Equal("cluster1-ROSA-Worker-Role"))
				Expect(*np.Spec.NodePool.Platform.AWS.Subnet.ID).To(Equal("subnet-abc123"))
				return created, nil
			})

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
			Name:         "my-np",
			Replicas:     2,
			InstanceType: "m5.xlarge",
			Subnet:       "subnet-abc123",
		}, nil)
	})

	It("resolves name from argv when Name option is empty", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newCreateMPMocks(ctrl)

		cluster := makeCluster("cluster1", "cluster-uid")
		created := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: "np-uid-new"}}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		nodePools.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, np *v1alpha1.NodePool, _ platform.CreateOptions) (*v1alpha1.NodePool, error) {
				Expect(np.Name).To(Equal("my-np"))
				Expect(np.Spec.NodePool.ClusterName).To(Equal("cluster1"))
				Expect(*np.Spec.NodePool.Replicas).To(Equal(int32(2)))
				Expect(np.Spec.NodePool.Platform.AWS.InstanceProfile).To(Equal("cluster1-ROSA-Worker-Role"))
				Expect(*np.Spec.NodePool.Platform.AWS.Subnet.ID).To(Equal("subnet-abc123"))
				return created, nil
			})

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{Replicas: 2, Subnet: "subnet-abc123"}, []string{"my-np"})
	})

	It("fails when name is not specified", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newCreateMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{}, nil)
		}).To(Panic())
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newCreateMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{Name: "my-np"}, nil)
		}).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{Name: "my-np"}, nil)
		}).To(Panic())
	})

	It("fails when cluster get fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		// Get is called in PostExpand (after subnet validation passes).
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(nil, fmt.Errorf("get failed"))

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name: "my-np", Subnet: "subnet-abc123",
			}, nil)
		}).To(Panic())
	})

	It("fails when instance profile cannot be derived (no AWS platform)", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: "cluster-uid"},
			Spec:       v1alpha1.ClusterSpec{},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name: "my-np", Subnet: "subnet-abc123",
			}, nil)
		}).To(Panic())
	})

	It("fails when subnet is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		// Get is NOT called when subnet is missing: PreRequest fails before PostExpand.

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name: "my-np",
			}, nil)
		}).To(Panic())
	})
})
