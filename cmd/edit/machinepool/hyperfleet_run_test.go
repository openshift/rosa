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
	v1 := hfmocks.NewMockV1alpha1Interface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	nodePools := hfmocks.NewMockNodePoolInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters(gomock.Any()).Return(clusters).AnyTimes()
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
				NodePool: hypershiftv1beta1.NodePoolSpec{Replicas: &replicas},
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

	It("resolves node pool name from argv when machinepool option is empty", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: hypershiftv1beta1.NodePoolSpec{Replicas: &replicas},
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
		hf, _, _ := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		// Pass cmd with no flags set (replicas not Changed)
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
				NodePool: hypershiftv1beta1.NodePoolSpec{Replicas: &replicas},
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
