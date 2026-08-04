package machinepool

import (
	"fmt"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/hyperfleet-operator/api/v1alpha1"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
)

func newListMPMocks(ctrl *gomock.Controller) (
	*hfmocks.MockInterface,
	*hfmocks.MockClusterInterface,
	*hfmocks.MockNodePoolInterface,
) {
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1Interface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	nodePools := hfmocks.NewMockNodePoolInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters(gomock.Any()).Return(clusters).AnyTimes()
	v1.EXPECT().NodePools(gomock.Any()).Return(nodePools).AnyTimes()
	return hf, clusters, nodePools
}

var _ = Describe("runHyperfleetList", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("lists node pools when they exist", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newListMPMocks(ctrl)

		replicas := int32(2)
		np := v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: hypershiftv1beta1.NodePoolSpec{
					Replicas: &replicas,
					Platform: hypershiftv1beta1.NodePoolPlatform{
						AWS: &hypershiftv1beta1.AWSNodePoolPlatform{InstanceType: "m5.xlarge"},
					},
				},
			},
			Status: v1alpha1.NodePoolStatus{Phase: v1alpha1.NodePoolPhaseReady},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{np}}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetList(t.RosaRuntime)
	})

	It("prints a message when there are no node pools", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newListMPMocks(ctrl)

		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.NodePoolList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetList(t.RosaRuntime)
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newListMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetList(t.RosaRuntime) }).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newListMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetList(t.RosaRuntime) }).To(Panic())
	})

	It("fails when listing node pools returns an error", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newListMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("list failed"))

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetList(t.RosaRuntime) }).To(Panic())
	})
})
