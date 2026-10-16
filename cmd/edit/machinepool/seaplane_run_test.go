// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

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
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	seaplanemocks "github.com/openshift/rosa/pkg/seaplane/mocks"
	"github.com/openshift/rosa/pkg/test"
)

func newEditMPMocks(ctrl *gomock.Controller) (
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
	v1.EXPECT().RESTClient().Return(nil).AnyTimes()
	return seaplaneClient, clusters, nodePools
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

var _ = Describe("runSeaplaneEdit (machinepool)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("updates replicas on the success path", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: &replicas,
					Platform: v1alpha1.NodePoolPlatform{Type: hypershiftv1beta1.AWSPlatform},
					Management: hypershiftv1beta1.NodePoolManagement{
						UpgradeType: hypershiftv1beta1.UpgradeTypeReplace,
					},
				},
			},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		nodePools.EXPECT().Get(gomock.Any(), "cluster1.my-np", gomock.Any()).Return(np, nil).Times(2)
		nodePools.EXPECT().Patch(gomock.Any(), "cluster1.my-np", types.MergePatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ platform.PatchOptions) (*v1alpha1.NodePool, error) {
				Expect(string(data)).To(MatchJSON(`{"spec":{"nodePool":{"replicas":5}}}`))
				return np, nil
			})

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np", replicas: 5},
			makeEditCmd("5"), nil)
	})

	DescribeTable("forwards explicit autorepair edits", func(enabled, autoscaling bool) {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, pools := newEditMPMocks(ctrl)
		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.workers", UID: "pool-uid"},
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
		pools.EXPECT().Get(gomock.Any(), "cluster1.workers", gomock.Any()).Return(current, nil).Times(2)
		expectedPatch := fmt.Sprintf(`{"spec":{"autoRepair":%t}}`, enabled)
		if autoscaling {
			expectedPatch = fmt.Sprintf(`{"spec":{"autoRepair":%t,"nodePool":{"autoScaling":{"max":6,"min":3}}}}`, enabled)
		}
		pools.EXPECT().Patch(gomock.Any(), "cluster1.workers", types.MergePatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ platform.PatchOptions) (*v1alpha1.NodePool, error) {
				Expect(string(data)).To(MatchJSON(expectedPatch))
				return current, nil
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
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneEdit(t.RosaRuntime, opts, cmd, nil)
		Expect(current).To(Equal(original))
	},
		Entry("enable only", true, false),
		Entry("disable only", false, false),
		Entry("enable together with autoscaling (56778)", true, true),
	)

	It("resolves node pool name from argv when machinepool option is empty", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: &replicas,
					Platform: v1alpha1.NodePoolPlatform{Type: hypershiftv1beta1.AWSPlatform},
					Management: hypershiftv1beta1.NodePoolManagement{
						UpgradeType: hypershiftv1beta1.UpgradeTypeReplace,
					},
				},
			},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		nodePools.EXPECT().Get(gomock.Any(), "cluster1.my-np", gomock.Any()).Return(np, nil).Times(2)
		nodePools.EXPECT().Patch(gomock.Any(), "cluster1.my-np", types.MergePatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ platform.PatchOptions) (*v1alpha1.NodePool, error) {
				Expect(string(data)).To(MatchJSON(`{"spec":{"nodePool":{"replicas":5}}}`))
				return np, nil
			})

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneEdit(t.RosaRuntime, &EditMachinepoolUserOptions{replicas: 5},
			makeEditCmd("5"), []string{"my-np"})
	})

	It("fails when machinepool name is not specified", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, _ := newEditMPMocks(ctrl)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneEdit(t.RosaRuntime, &EditMachinepoolUserOptions{}, makeEditCmd("5"), nil)
		}).To(Panic())
	})

	It("fails when no supported flags are changed", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, nodePools := newEditMPMocks(ctrl)
		// Cluster and node pool names are resolved before PreRequest validates flags.
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.my-np", UID: types.UID("np-uid-1")},
		}}}, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np"},
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
		seaplaneClient, _, _ := newEditMPMocks(ctrl)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		cmd := NewEditMachinePoolCommand()
		if err := cmd.Flags().Set("replicas", "5"); err != nil {
			panic(err)
		}
		Expect(func() {
			runSeaplaneEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np"}, cmd, nil)
		}).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, _ := newEditMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np"},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})

	// setupListOnlyMocks sets up cluster/nodepool List mocks but NOT Get.
	// Use for tests that fail in PreRequest (before PostExpand calls Get).
	setupListOnlyMocks := func(ctrl *gomock.Controller) {
		seaplaneClient, clusters, nodePools := newEditMPMocks(ctrl)
		np := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "cluster1.my-np", UID: types.UID("np-uid-1")}}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
	}

	It("rejects negative replica count", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		// PreRequest fails before PostExpand — Get is never called.
		setupListOnlyMocks(ctrl)

		Expect(func() {
			runSeaplaneEdit(t.RosaRuntime,
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
			runSeaplaneEdit(t.RosaRuntime,
				&EditMachinepoolUserOptions{machinepool: "my-np", replicas: math.MaxInt32 + 1},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})

	It("accepts math.MaxInt32 as a valid replica count", func() {
		ctrl := gomock.NewController(GinkgoT())
		_, nodePools, np := func() (*seaplanemocks.MockInterface, *seaplanemocks.MockNodePoolInterface, *v1alpha1.NodePool) {
			seaplaneClient, clusters, nps := newEditMPMocks(ctrl)
			replicas := int32(3)
			np := &v1alpha1.NodePool{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1.my-np", UID: types.UID("np-uid-1")},
				Spec: v1alpha1.NodePoolSpec{
					NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
				},
			}
			clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			}}}, nil)
			nps.EXPECT().List(gomock.Any(), gomock.Any()).Return(
				&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
			nps.EXPECT().Get(gomock.Any(), "cluster1.my-np", gomock.Any()).Return(np, nil).Times(2)
			nps.EXPECT().Patch(gomock.Any(), "cluster1.my-np", types.MergePatchType, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ platform.PatchOptions) (*v1alpha1.NodePool, error) {
					Expect(string(data)).To(MatchJSON(`{"spec":{"nodePool":{"replicas":2147483647}}}`))
					return np, nil
				})
			t.RosaRuntime.SeaplaneClient = seaplaneClient
			return seaplaneClient, nps, np
		}()
		_ = nodePools
		_ = np

		runSeaplaneEdit(t.RosaRuntime,
			&EditMachinepoolUserOptions{machinepool: "my-np", replicas: math.MaxInt32},
			makeEditCmd("5"), nil)
	})

	It("fails when node pool update fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
			},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		nodePools.EXPECT().Get(gomock.Any(), "cluster1.my-np", gomock.Any()).Return(np, nil).Times(2)
		nodePools.EXPECT().Patch(gomock.Any(), "cluster1.my-np", types.MergePatchType, gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("update failed"))

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() {
			runSeaplaneEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np", replicas: 5},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})

	Describe("Spot max price updates", func() {
		spotNodePool := func(maxPrice string) *v1alpha1.NodePool {
			return &v1alpha1.NodePool{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1.my-np", UID: types.UID("np-uid-1")},
				Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
					Platform: v1alpha1.NodePoolPlatform{
						AWS: &v1alpha1.AWSNodePoolPlatform{
							Placement: &hypershiftv1beta1.PlacementOptions{
								MarketType: hypershiftv1beta1.MarketTypeSpot,
								Spot:       hypershiftv1beta1.SpotOptions{MaxPrice: maxPrice},
							},
						},
					},
				}},
			}
		}

		It("updates the Spot max price on an existing Spot node pool", func() {
			cmd := makeEditCmd("")
			Expect(cmd.Flags().Set("spot-max-price", "0.75")).To(Succeed())

			ctrl := gomock.NewController(GinkgoT())
			seaplaneClient, _, nodePools := newEditMPMocks(ctrl)
			existing := spotNodePool("0.50")
			nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(existing, nil)

			handler := &seaplaneNodePoolUpdate{
				nodePoolKey:  "np-uid",
				nodePoolName: "np-uid",
				cmd:          cmd,
				userOptions:  &EditMachinepoolUserOptions{spotMaxPrice: "0.75"},
			}
			updated := &v1alpha1.NodePool{}
			t.RosaRuntime.SeaplaneClient = seaplaneClient

			Expect(handler.PostExpand(context.Background(), t.RosaRuntime, nil, updated)).To(Succeed())
			Expect(updated.Spec.NodePool.Platform.AWS.Placement.MarketType).
				To(Equal(hypershiftv1beta1.MarketTypeSpot))
			Expect(updated.Spec.NodePool.Platform.AWS.Placement.Spot.MaxPrice).To(Equal("0.75"))
		})

		It("maps the on-demand Spot price sentinel to an empty max price", func() {
			cmd := makeEditCmd("")
			Expect(cmd.Flags().Set("spot-max-price", mpOpts.SpotPriceOnDemand)).To(Succeed())

			ctrl := gomock.NewController(GinkgoT())
			seaplaneClient, _, nodePools := newEditMPMocks(ctrl)
			existing := spotNodePool("0.50")
			nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(existing, nil)

			handler := &seaplaneNodePoolUpdate{
				nodePoolKey:  "np-uid",
				nodePoolName: "np-uid",
				cmd:          cmd,
				userOptions:  &EditMachinepoolUserOptions{spotMaxPrice: mpOpts.SpotPriceOnDemand},
			}
			updated := &v1alpha1.NodePool{}
			t.RosaRuntime.SeaplaneClient = seaplaneClient

			Expect(handler.PostExpand(context.Background(), t.RosaRuntime, nil, updated)).To(Succeed())
			Expect(updated.Spec.NodePool.Platform.AWS.Placement.Spot.MaxPrice).To(BeEmpty())
		})
	})
})

var _ = Describe("PostExpand (edit machinepool security groups)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("updates security groups when flag is set", func() {
		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("additional-security-group-ids", "sg-new1,sg-new2")).To(Succeed())

		h := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "my-np",
			nodePoolName: "np-uid",
			userOptions:  &EditMachinepoolUserOptions{},
			cmd:          cmd,
		}

		// Current node pool with existing security groups
		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: ptr.To(int32(3)),
					Platform: v1alpha1.NodePoolPlatform{AWS: &v1alpha1.AWSNodePoolPlatform{
						InstanceType: "m5.xlarge",
						SecurityGroups: []hypershiftv1beta1.AWSResourceReference{
							{ID: ptr.To("sg-old1")},
							{ID: ptr.To("sg-old2")},
						},
					}},
				},
			},
		}

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newEditMPMocks(ctrl)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(current, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient

		obj := &v1alpha1.NodePool{}
		err := h.PostExpand(context.Background(), t.RosaRuntime, nil, obj)
		Expect(err).NotTo(HaveOccurred())

		// Verify new security groups are set
		Expect(obj.Spec.NodePool.Platform.AWS.SecurityGroups).To(HaveLen(2))
		Expect(*obj.Spec.NodePool.Platform.AWS.SecurityGroups[0].ID).To(Equal("sg-new1"))
		Expect(*obj.Spec.NodePool.Platform.AWS.SecurityGroups[1].ID).To(Equal("sg-new2"))
	})

	It("clears security groups when flag is set to empty string", func() {
		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("additional-security-group-ids", "")).To(Succeed())

		h := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "my-np",
			nodePoolName: "np-uid",
			userOptions:  &EditMachinepoolUserOptions{},
			cmd:          cmd,
		}

		// Current node pool with existing security groups
		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: ptr.To(int32(3)),
					Platform: v1alpha1.NodePoolPlatform{AWS: &v1alpha1.AWSNodePoolPlatform{
						InstanceType: "m5.xlarge",
						SecurityGroups: []hypershiftv1beta1.AWSResourceReference{
							{ID: ptr.To("sg-1")},
						},
					}},
				},
			},
		}

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newEditMPMocks(ctrl)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(current, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient

		obj := &v1alpha1.NodePool{}
		err := h.PostExpand(context.Background(), t.RosaRuntime, nil, obj)
		Expect(err).NotTo(HaveOccurred())

		// Verify security groups are cleared
		Expect(obj.Spec.NodePool.Platform.AWS.SecurityGroups).To(BeNil())
	})

	It("leaves security groups unchanged when flag is not set", func() {
		cmd := makeEditCmd("")
		// Don't set the additional-security-group-ids flag

		h := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "my-np",
			nodePoolName: "np-uid",
			userOptions:  &EditMachinepoolUserOptions{},
			cmd:          cmd,
		}

		// Current node pool with existing security groups
		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: ptr.To(int32(3)),
					Platform: v1alpha1.NodePoolPlatform{AWS: &v1alpha1.AWSNodePoolPlatform{
						InstanceType: "m5.xlarge",
						SecurityGroups: []hypershiftv1beta1.AWSResourceReference{
							{ID: ptr.To("sg-existing")},
						},
					}},
				},
			},
		}

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newEditMPMocks(ctrl)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(current, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient

		obj := &v1alpha1.NodePool{}
		err := h.PostExpand(context.Background(), t.RosaRuntime, nil, obj)
		Expect(err).NotTo(HaveOccurred())

		// Verify security groups remain the same
		Expect(obj.Spec.NodePool.Platform.AWS.SecurityGroups).To(HaveLen(1))
		Expect(*obj.Spec.NodePool.Platform.AWS.SecurityGroups[0].ID).To(Equal("sg-existing"))
	})

	It("filters out empty entries from security group list", func() {
		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("additional-security-group-ids", "sg-1,,sg-2, ,sg-3")).To(Succeed())

		h := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "my-np",
			nodePoolName: "np-uid",
			userOptions:  &EditMachinepoolUserOptions{},
			cmd:          cmd,
		}

		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: ptr.To(int32(3)),
					Platform: v1alpha1.NodePoolPlatform{AWS: &v1alpha1.AWSNodePoolPlatform{
						InstanceType: "m5.xlarge",
					}},
				},
			},
		}

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newEditMPMocks(ctrl)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(current, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient

		obj := &v1alpha1.NodePool{}
		err := h.PostExpand(context.Background(), t.RosaRuntime, nil, obj)
		Expect(err).NotTo(HaveOccurred())

		// Verify empty entries are filtered out, only valid IDs remain
		Expect(obj.Spec.NodePool.Platform.AWS.SecurityGroups).To(HaveLen(3))
		Expect(*obj.Spec.NodePool.Platform.AWS.SecurityGroups[0].ID).To(Equal("sg-1"))
		Expect(*obj.Spec.NodePool.Platform.AWS.SecurityGroups[1].ID).To(Equal("sg-2"))
		Expect(*obj.Spec.NodePool.Platform.AWS.SecurityGroups[2].ID).To(Equal("sg-3"))
	})

	It("clears security groups when all entries are empty/whitespace", func() {
		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("additional-security-group-ids", "  ,  ,  ")).To(Succeed())

		h := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "my-np",
			nodePoolName: "np-uid",
			userOptions:  &EditMachinepoolUserOptions{},
			cmd:          cmd,
		}

		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: ptr.To(int32(3)),
					Platform: v1alpha1.NodePoolPlatform{AWS: &v1alpha1.AWSNodePoolPlatform{
						InstanceType: "m5.xlarge",
						SecurityGroups: []hypershiftv1beta1.AWSResourceReference{
							{ID: ptr.To("sg-old")},
						},
					}},
				},
			},
		}

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newEditMPMocks(ctrl)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(current, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient

		obj := &v1alpha1.NodePool{}
		err := h.PostExpand(context.Background(), t.RosaRuntime, nil, obj)
		Expect(err).NotTo(HaveOccurred())

		// Verify security groups are cleared when all entries are empty/whitespace
		Expect(obj.Spec.NodePool.Platform.AWS.SecurityGroups).To(BeNil())
	})

	It("fails when trying to update security groups on a non-AWS node pool", func() {
		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("additional-security-group-ids", "sg-1")).To(Succeed())

		h := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "my-np",
			nodePoolName: "np-uid",
			userOptions:  &EditMachinepoolUserOptions{},
			cmd:          cmd,
		}

		// Current node pool WITHOUT AWS platform (e.g., GCP or Azure)
		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: ptr.To(int32(3)),
					Platform: v1alpha1.NodePoolPlatform{AWS: nil}, // Non-AWS
				},
			},
		}

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, nodePools := newEditMPMocks(ctrl)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(current, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient

		obj := &v1alpha1.NodePool{}
		err := h.PostExpand(context.Background(), t.RosaRuntime, nil, obj)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot update security groups on a non-AWS node pool"))
	})
})
