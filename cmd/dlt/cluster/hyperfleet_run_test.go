package cluster

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

func newDltClusterMocks(ctrl *gomock.Controller) (*hfmocks.MockInterface, *hfmocks.MockClusterInterface) {
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	return hf, clusters
}

var _ = Describe("runHyperfleetDelete (cluster)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		origConfirm := confirmFn
		confirmFn = func(string, ...interface{}) bool { return true }
		args.watch = false
		DeferCleanup(func() {
			confirmFn = origConfirm
			args.watch = false
		})
	})

	It("deletes a cluster successfully", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		clusters.EXPECT().Delete(gomock.Any(), "cluster-uid", gomock.Any()).Return(nil)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetDelete(t.RosaRuntime, testCmd())
	})

	It("skips delete when user declines confirmation", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		// Delete must NOT be called — omitting the expectation enforces this via gomock.

		confirmFn = func(string, ...interface{}) bool { return false }
		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetDelete(t.RosaRuntime, testCmd())
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		hf, _ := newDltClusterMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetDelete(t.RosaRuntime, testCmd()) }).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetDelete(t.RosaRuntime, testCmd()) }).To(Panic())
	})

	It("fails when cluster delete fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		clusters.EXPECT().Delete(gomock.Any(), "cluster-uid", gomock.Any()).Return(fmt.Errorf("delete failed"))

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetDelete(t.RosaRuntime, testCmd()) }).To(Panic())
	})

	It("watches until the cluster is gone when --watch is set", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		clusters.EXPECT().Delete(gomock.Any(), "cluster-uid", gomock.Any()).Return(nil)
		clusters.EXPECT().WaitUntil(gomock.Any(), "cluster-uid", gomock.Any(), hfWatchInterval, hfWatchTimeout).Return(nil)

		args.watch = true
		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetDelete(t.RosaRuntime, testCmd())
	})

	It("fails when --watch polling fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		clusters.EXPECT().Delete(gomock.Any(), "cluster-uid", gomock.Any()).Return(nil)
		clusters.EXPECT().WaitUntil(gomock.Any(), "cluster-uid", gomock.Any(), hfWatchInterval, hfWatchTimeout).
			Return(fmt.Errorf("timeout"))

		args.watch = true
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetDelete(t.RosaRuntime, testCmd()) }).To(Panic())
	})
})
