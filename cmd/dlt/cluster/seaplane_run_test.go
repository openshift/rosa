// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	seaplanemocks "github.com/openshift/rosa/pkg/seaplane/mocks"
	"github.com/openshift/rosa/pkg/test"
)

func testCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

func newDltClusterMocks(ctrl *gomock.Controller) (*seaplanemocks.MockInterface, *seaplanemocks.MockClusterInterface) {
	seaplaneClient := seaplanemocks.NewMockInterface(ctrl)
	v1 := seaplanemocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := seaplanemocks.NewMockClusterInterface(ctrl)
	seaplaneClient.EXPECT().SeaplaneV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	return seaplaneClient, clusters
}

var _ = Describe("runSeaplaneDelete (cluster)", func() {
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
		seaplaneClient, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster1", gomock.Any()).Return(&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}, nil)
		clusters.EXPECT().Delete(gomock.Any(), "cluster1", gomock.Any()).Return(nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneDelete(t.RosaRuntime, testCmd())
	})

	It("skips delete when user declines confirmation", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster1", gomock.Any()).Return(&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}, nil)
		// Delete must NOT be called — omitting the expectation enforces this via gomock.

		confirmFn = func(string, ...interface{}) bool { return false }
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneDelete(t.RosaRuntime, testCmd())
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _ := newDltClusterMocks(ctrl)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneDelete(t.RosaRuntime, testCmd()) }).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneDelete(t.RosaRuntime, testCmd()) }).To(Panic())
	})

	It("fails when cluster delete fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster1", gomock.Any()).Return(&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}, nil)
		clusters.EXPECT().Delete(gomock.Any(), "cluster1", gomock.Any()).Return(fmt.Errorf("delete failed"))

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneDelete(t.RosaRuntime, testCmd()) }).To(Panic())
	})

	It("fails when delete protection is enabled", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		enabled := true
		clusters.EXPECT().Get(gomock.Any(), "cluster1", gomock.Any()).Return(&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			Spec:       v1alpha1.ClusterSpec{DeleteProtection: &enabled},
		}, nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneDelete(t.RosaRuntime, testCmd()) }).To(Panic())
	})

	It("watches until the cluster is gone when --watch is set", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster1", gomock.Any()).Return(&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}, nil)
		clusters.EXPECT().Delete(gomock.Any(), "cluster1", gomock.Any()).Return(nil)
		clusters.EXPECT().WaitUntil(gomock.Any(), "cluster1", gomock.Any(), seaplaneWatchInterval, seaplaneWatchTimeout).Return(nil)

		args.watch = true
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneDelete(t.RosaRuntime, testCmd())
	})

	It("fails when --watch polling fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDltClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster1", gomock.Any()).Return(&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}, nil)
		clusters.EXPECT().Delete(gomock.Any(), "cluster1", gomock.Any()).Return(nil)
		clusters.EXPECT().WaitUntil(gomock.Any(), "cluster1", gomock.Any(), seaplaneWatchInterval, seaplaneWatchTimeout).
			Return(fmt.Errorf("timeout"))

		args.watch = true
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneDelete(t.RosaRuntime, testCmd()) }).To(Panic())
	})
})
