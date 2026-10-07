package machinepool

import (
	"context"
	"fmt"
	"math"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
)

func newEditMPMocks(ctrl *gomock.Controller) (
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

func makeEditCmd(replicasStr string) *cobra.Command {
	cmd := NewEditMachinePoolCommand()
	if err := cmd.Flag("cluster").Value.Set("cluster1"); err != nil {
		panic(err)
	}
	if replicasStr != "" {
		if err := cmd.Flags().Set("replicas", replicasStr); err != nil {
			panic(err)
		}
	}
	return cmd
}

var _ = Describe("runHyperfleetEdit (machinepool)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("updates replicas on the success path", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
			},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(np, nil)
		nodePools.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(np, nil)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np", replicas: 5},
			makeEditCmd("5"), nil)
	})

	DescribeTable("forwards explicit autorepair edits", func(enabled, autoscaling bool) {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, pools := newEditMPMocks(ctrl)
		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "workers", UID: "pool-uid"},
			Spec: v1alpha1.NodePoolSpec{
				AutoRepair: ptr.To(!enabled), Labels: map[string]string{"team": "test"},
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas:   ptr.To(int32(3)),
					Management: hypershiftv1beta1.NodePoolManagement{AutoRepair: !enabled},
				},
			},
		}
		original := current.DeepCopy()
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: "cluster-uid"},
		}}}, nil)
		pools.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.NodePoolList{
			Items: []v1alpha1.NodePool{*current},
		}, nil)
		pools.EXPECT().Get(gomock.Any(), "pool-uid", gomock.Any()).Return(current, nil)
		pools.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, updated *v1alpha1.NodePool, _ platform.UpdateOptions) (*v1alpha1.NodePool, error) {
				Expect(updated.Spec.AutoRepair).To(Equal(ptr.To(enabled)))
				Expect(updated.Spec.Labels).To(Equal(original.Spec.Labels))
				Expect(updated.Spec.NodePool.Management).To(Equal(hypershiftv1beta1.NodePoolManagement{}))
				if autoscaling {
					Expect(updated.Spec.NodePool.Replicas).To(BeNil())
					Expect(updated.Spec.NodePool.AutoScaling).To(Equal(&hypershiftv1beta1.NodePoolAutoScaling{
						Min: ptr.To(int32(3)), Max: 6,
					}))
				} else {
					Expect(updated.Spec.NodePool.Replicas).To(Equal(original.Spec.NodePool.Replicas))
				}
				return updated, nil
			})
		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("autorepair", fmt.Sprint(enabled))).To(Succeed())
		opts := &EditMachinepoolUserOptions{machinepool: "workers", autorepair: enabled}
		if autoscaling {
			Expect(cmd.Flags().Set("enable-autoscaling", "true")).To(Succeed())
			Expect(cmd.Flags().Set("min-replicas", "3")).To(Succeed())
			Expect(cmd.Flags().Set("max-replicas", "6")).To(Succeed())
			opts.autoscalingEnabled, opts.minReplicas, opts.maxReplicas = true, 3, 6
		}
		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetEdit(t.RosaRuntime, opts, cmd, nil)
		Expect(current).To(Equal(original))
	},
		Entry("enable only", true, false),
		Entry("disable only", false, false),
		Entry("enable together with autoscaling (56778)", true, true),
	)

	It("resolves node pool name from argv when machinepool option is empty", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
			},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(np, nil)
		nodePools.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(np, nil)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{replicas: 5},
			makeEditCmd("5"), []string{"my-np"})
	})

	It("fails when machinepool name is not specified", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{}, makeEditCmd("5"), nil)
		}).To(Panic())
	})

	It("fails when no supported flags are changed", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)
		// ResolveClusterUID and ResolveNodePoolUID are called before PreRequest validates flags.
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
		}}}, nil)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np"},
				makeEditCmd(""), nil)
		}).To(Panic())
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		cmd := NewEditMachinePoolCommand()
		if err := cmd.Flags().Set("replicas", "5"); err != nil {
			panic(err)
		}
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np"}, cmd, nil)
		}).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newEditMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np"},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})

	// setupListOnlyMocks sets up cluster/nodepool List mocks but NOT Get.
	// Use for tests that fail in PreRequest (before PostExpand calls Get).
	setupListOnlyMocks := func(ctrl *gomock.Controller) {
		hf, clusters, nodePools := newEditMPMocks(ctrl)
		np := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")}}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		t.RosaRuntime.HyperFleetClient = hf
	}

	It("rejects negative replica count", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		// PreRequest fails before PostExpand — Get is never called.
		setupListOnlyMocks(ctrl)

		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime,
				&EditMachinepoolUserOptions{machinepool: "my-np", replicas: -1},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})

	It("rejects replica count above math.MaxInt32", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		// PreRequest fails before PostExpand — Get is never called.
		setupListOnlyMocks(ctrl)

		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime,
				&EditMachinepoolUserOptions{machinepool: "my-np", replicas: math.MaxInt32 + 1},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})

	It("accepts math.MaxInt32 as a valid replica count", func() {
		ctrl := gomock.NewController(GinkgoT())
		_, nodePools, np := func() (*hfmocks.MockInterface, *hfmocks.MockNodePoolInterface, *v1alpha1.NodePool) {
			hf, clusters, nps := newEditMPMocks(ctrl)
			replicas := int32(3)
			np := &v1alpha1.NodePool{
				ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
				Spec: v1alpha1.NodePoolSpec{
					NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
				},
			}
			clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			}}}, nil)
			nps.EXPECT().List(gomock.Any(), gomock.Any()).Return(
				&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
			nps.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(np, nil)
			nps.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(np, nil)
			t.RosaRuntime.HyperFleetClient = hf
			return hf, nps, np
		}()
		_ = nodePools
		_ = np

		runHyperfleetEdit(t.RosaRuntime,
			&EditMachinepoolUserOptions{machinepool: "my-np", replicas: math.MaxInt32},
			makeEditCmd("5"), nil)
	})

	It("fails when node pool update fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
			},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(np, nil)
		nodePools.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("update failed"))

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np", replicas: 5},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})
})
