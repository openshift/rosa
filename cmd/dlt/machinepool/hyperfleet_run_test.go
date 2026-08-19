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
	"github.com/spf13/cobra"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
)

func testCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

func newDltMPMocks(ctrl *gomock.Controller) (
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

func clusterList(name, uid string) *v1alpha1.ClusterList {
	return &v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(uid)},
	}}}
}

func npList(name, uid string) *v1alpha1.NodePoolList {
	return &v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(uid)},
	}}}
}

var _ = Describe("runHyperfleetDelete (machinepool)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		origConfirm := confirmFn
		confirmFn = func(string, ...interface{}) bool { return true }
		DeferCleanup(func() { confirmFn = origConfirm })
	})

	It("deletes a node pool successfully", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(npList("my-np", "np-uid"), nil)
		nodePools.EXPECT().Delete(gomock.Any(), "np-uid", gomock.Any()).Return(nil)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
	})

	It("resolves node pool name from argv when machinepool option is empty", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(npList("my-np", "np-uid"), nil)
		nodePools.EXPECT().Delete(gomock.Any(), "np-uid", gomock.Any()).Return(nil)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{}, []string{"my-np"})
	})

	It("skips delete when user declines confirmation", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(npList("my-np", "np-uid"), nil)
		// Delete must NOT be called — omitting the expectation enforces this via gomock.

		confirmFn = func(string, ...interface{}) bool { return false }
		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
	})

	It("fails when machinepool name is not specified", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newDltMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{}, nil)
		}).To(Panic())
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newDltMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
		}).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
		}).To(Panic())
	})

	It("fails when node pool cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.NodePoolList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
		}).To(Panic())
	})

	It("fails when node pool delete fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(npList("my-np", "np-uid"), nil)
		nodePools.EXPECT().Delete(gomock.Any(), "np-uid", gomock.Any()).Return(fmt.Errorf("delete failed"))

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
		}).To(Panic())
	})
})
