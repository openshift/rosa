package machinepool

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

func newListMPMocks(ctrl *gomock.Controller) (
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

var _ = Describe("runHyperfleetList", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	captureStdout := func(f func()) string {
		r, w, _ := os.Pipe()
		orig := os.Stdout
		os.Stdout = w
		f()
		w.Close()
		os.Stdout = orig
		out, _ := io.ReadAll(r)
		return string(out)
	}

	It("lists node pools when they exist", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newListMPMocks(ctrl)

		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{{
				ObjectMeta: metav1.ObjectMeta{Name: "np1", UID: types.UID("np-uid-1")},
				Spec: v1alpha1.NodePoolSpec{
					Labels:     map[string]string{"team": "test", "empty": ""},
					AutoRepair: ptr.To(false),
					NodePool: v1alpha1.NodePoolSpecPassthrough{
						Replicas:   ptr.To(int32(3)),
						Release:    hypershiftv1beta1.Release{Image: "quay.io/example/release:5.0.0-ec.6"},
						Management: hypershiftv1beta1.NodePoolManagement{AutoRepair: true},
						Taints:     []hypershiftv1beta1.Taint{{Key: "dedicated", Effect: "NoSchedule"}},
						Platform: v1alpha1.NodePoolPlatform{
							AWS: &hypershiftv1beta1.AWSNodePoolPlatform{
								RootVolume: &hypershiftv1beta1.Volume{Size: 75},
							},
						},
					},
				},
				Status: v1alpha1.NodePoolStatus{Phase: v1alpha1.NodePoolPhaseReady},
			}}}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		stdout := captureStdout(func() { runHyperfleetList(t.RosaRuntime) })

		Expect(stdout).To(ContainSubstring("LABELS"))
		Expect(stdout).To(ContainSubstring("TAINTS"))
		Expect(stdout).To(ContainSubstring("empty=, team=test"))
		Expect(stdout).To(ContainSubstring("dedicated=:NoSchedule"))
		Expect(stdout).To(ContainSubstring("DISK SIZE"))
		Expect(stdout).To(ContainSubstring("75 GiB"))
		Expect(stdout).To(ContainSubstring("np1"))
		Expect(stdout).To(ContainSubstring("np-uid-1"))
		// Exercise the same parser that 56782 and 56778 use, including blank columns.
		client := rosacli.NewClient()
		pools, err := client.MachinePool.ReflectNodePoolList(*bytes.NewBufferString(stdout))
		Expect(err).NotTo(HaveOccurred())
		pool := pools.Nodepool("np1")
		Expect(pool).NotTo(BeNil())
		Expect(pool.AutoScaling).To(Equal("No"))
		Expect(pool.AutoRepair).To(Equal("No"))
		Expect(pool.Replicas).To(Equal("-/3"))
		Expect(pool.Version).To(Equal("quay.io/example/release:5.0.0-ec.6"))
		Expect(pool.Labels).To(Equal("empty=, team=test"))
		Expect(pool.Taints).To(Equal("dedicated=:NoSchedule"))
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
