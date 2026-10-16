package machinepool

import (
	"context"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("hyperfleet dispatch", func() {
	var origEnabled func() bool
	var origEdit func(*EditMachinepoolUserOptions, *cobra.Command, []string)

	BeforeEach(func() {
		origEnabled = hyperfleetEnabled
		origEdit = hfEditMachinePool
	})

	AfterEach(func() {
		hyperfleetEnabled = origEnabled
		hfEditMachinePool = origEdit
	})

	It("routes to hfEditMachinePool when hyperfleet is enabled", func() {
		called := false
		hyperfleetEnabled = func() bool { return true }
		hfEditMachinePool = func(_ *EditMachinepoolUserOptions, _ *cobra.Command, _ []string) { called = true }

		cmd := NewEditMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})

var _ = Describe("HyperFleet nodepool spot edit", func() {
	It("updates the Spot max price on an existing Spot node pool", func() {
		cmd := &cobra.Command{}
		var spotMaxPrice string
		cmd.Flags().StringVar(&spotMaxPrice, "spot-max-price", "", "")
		Expect(cmd.Flags().Set("spot-max-price", "0.75")).To(Succeed())

		ctrl := gomock.NewController(GinkgoT())
		hf := hfmocks.NewMockInterface(ctrl)
		v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
		nodePools := hfmocks.NewMockNodePoolInterface(ctrl)
		hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
		v1.EXPECT().NodePools().Return(nodePools).AnyTimes()
		existing := &v1alpha1.NodePool{
			Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
				Platform: v1alpha1.NodePoolPlatform{
					AWS: &v1alpha1.AWSNodePoolPlatform{
						Placement: &hypershiftv1beta1.PlacementOptions{
							MarketType: hypershiftv1beta1.MarketTypeSpot,
							Spot:       hypershiftv1beta1.SpotOptions{MaxPrice: "0.50"},
						},
					},
				},
			}},
		}
		nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(existing, nil)

		handler := &hyperfleetNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "np-uid",
			nodePoolName: "np-uid",
			cmd:          cmd,
			userOptions:  &EditMachinepoolUserOptions{spotMaxPrice: "0.75"},
		}
		updated := &v1alpha1.NodePool{}

		err := handler.PostExpand(context.Background(), &rosa.Runtime{HyperFleetClient: hf}, nil, updated)

		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Spec.NodePool.Platform.AWS.Placement.MarketType).To(Equal(hypershiftv1beta1.MarketTypeSpot))
		Expect(updated.Spec.NodePool.Platform.AWS.Placement.Spot.MaxPrice).To(Equal("0.75"))
	})

	It("maps the on-demand Spot price sentinel to an empty Platform API max price", func() {
		cmd := &cobra.Command{}
		var spotMaxPrice string
		cmd.Flags().StringVar(&spotMaxPrice, "spot-max-price", "", "")
		Expect(cmd.Flags().Set("spot-max-price", mpOpts.SpotPriceOnDemand)).To(Succeed())

		ctrl := gomock.NewController(GinkgoT())
		hf := hfmocks.NewMockInterface(ctrl)
		v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
		nodePools := hfmocks.NewMockNodePoolInterface(ctrl)
		hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
		v1.EXPECT().NodePools().Return(nodePools).AnyTimes()
		existing := &v1alpha1.NodePool{
			Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
				Platform: v1alpha1.NodePoolPlatform{
					AWS: &v1alpha1.AWSNodePoolPlatform{
						Placement: &hypershiftv1beta1.PlacementOptions{
							MarketType: hypershiftv1beta1.MarketTypeSpot,
							Spot:       hypershiftv1beta1.SpotOptions{MaxPrice: "0.50"},
						},
					},
				},
			}},
		}
		nodePools.EXPECT().Get(gomock.Any(), "np-uid", gomock.Any()).Return(existing, nil)

		handler := &hyperfleetNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "np-uid",
			nodePoolName: "np-uid",
			cmd:          cmd,
			userOptions:  &EditMachinepoolUserOptions{spotMaxPrice: mpOpts.SpotPriceOnDemand},
		}
		updated := &v1alpha1.NodePool{}

		err := handler.PostExpand(context.Background(), &rosa.Runtime{HyperFleetClient: hf}, nil, updated)

		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Spec.NodePool.Platform.AWS.Placement.Spot.MaxPrice).To(BeEmpty())
	})
})
