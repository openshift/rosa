// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"context"
	"fmt"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
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

func newDltMPMocks(ctrl *gomock.Controller) (
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

var _ = Describe("runSeaplaneDelete (machinepool)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		origConfirm := confirmFn
		confirmFn = func(string, ...interface{}) bool { return true }
		DeferCleanup(func() { confirmFn = origConfirm })
	})

	It("deletes a node pool successfully", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Eq(platform.NodePoolListOptions{ClusterUID: "cluster-uid"})).
			Return(npList("cluster1.my-np", "np-uid"), nil)
		nodePools.EXPECT().Delete(gomock.Any(), "cluster1.my-np", gomock.Any()).Return(nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
	})

	It("resolves node pool name from argv when machinepool option is empty", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Eq(platform.NodePoolListOptions{ClusterUID: "cluster-uid"})).
			Return(npList("cluster1.my-np", "np-uid"), nil)
		nodePools.EXPECT().Delete(gomock.Any(), "cluster1.my-np", gomock.Any()).Return(nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{}, []string{"my-np"})
	})

	It("skips delete when user declines confirmation", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Eq(platform.NodePoolListOptions{ClusterUID: "cluster-uid"})).
			Return(npList("cluster1.my-np", "np-uid"), nil)
		// Delete must NOT be called — omitting the expectation enforces this via gomock.

		confirmFn = func(string, ...interface{}) bool { return false }
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
	})

	It("fails when machinepool name is not specified", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, _ := newDltMPMocks(ctrl)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{}, nil)
		}).To(Panic())
	})

	It("fails when machinepool name is invalid", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, _ := newDltMPMocks(ctrl)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneDelete(
				t.RosaRuntime,
				testCmd(),
				&DeleteMachinepoolUserOptions{machinepool: "anything%^"},
				nil,
			)
		}).To(Panic())
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, _ := newDltMPMocks(ctrl)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
		}).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, _ := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
		}).To(Panic())
	})

	It("fails when node pool cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.NodePoolList{}, nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
		}).To(Panic())
	})

	It("fails when node pool delete fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, nodePools := newDltMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(clusterList("cluster1", "cluster-uid"), nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Eq(platform.NodePoolListOptions{ClusterUID: "cluster-uid"})).
			Return(npList("cluster1.my-np", "np-uid"), nil)
		nodePools.EXPECT().Delete(gomock.Any(), "cluster1.my-np", gomock.Any()).Return(fmt.Errorf("delete failed"))

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneDelete(t.RosaRuntime, testCmd(), &DeleteMachinepoolUserOptions{machinepool: "my-np"}, nil)
		}).To(Panic())
	})
})
