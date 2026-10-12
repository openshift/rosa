package machinepool

import (
	"context"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/test"
)

func upgradeEditCmd(opts *EditMachinepoolUserOptions, values map[string]string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringVar(&opts.maxSurge, "max-surge", "", "")
	cmd.Flags().StringVar(&opts.maxUnavailable, "max-unavailable", "", "")
	for flag, value := range values {
		Expect(cmd.Flags().Set(flag, value)).To(Succeed())
	}
	return cmd
}

// nodePoolWithRollingUpdate builds a node pool carrying the rolling-update
// settings the Platform API renders for it.
func nodePoolWithRollingUpdate(maxSurge, maxUnavailable string) *v1alpha1.NodePool {
	surge := intstr.Parse(maxSurge)
	unavailable := intstr.Parse(maxUnavailable)
	return &v1alpha1.NodePool{
		ObjectMeta: metav1.ObjectMeta{Name: "workers", UID: "pool-uid"},
		Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
			Management: hypershiftv1beta1.NodePoolManagement{
				UpgradeType: hypershiftv1beta1.UpgradeTypeReplace,
				Replace: &hypershiftv1beta1.ReplaceUpgrade{
					Strategy: hypershiftv1beta1.UpgradeStrategyRollingUpdate,
					RollingUpdate: &hypershiftv1beta1.RollingUpdate{
						MaxSurge: &surge, MaxUnavailable: &unavailable,
					},
				},
			},
		}},
	}
}

var _ = Describe("HyperFleet machine pool rolling update editing", func() {
	// editUpgrade runs the handler the way runHyperfleetEdit does, against a
	// node pool whose current rolling update is maxSurge 1 / maxUnavailable 0.
	editUpgrade := func(flags map[string]string) (*hyperfleet.NodePoolUpgrade, error) {
		t := test.NewTestRuntime()
		ctrl := gomock.NewController(GinkgoT())
		hf, _, pools := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		pools.EXPECT().Get(gomock.Any(), "pool-uid", gomock.Any()).
			Return(nodePoolWithRollingUpdate("1", "0"), nil).AnyTimes()

		opts := &EditMachinepoolUserOptions{}
		cmd := upgradeEditCmd(opts, flags)
		handler := &hyperfleetNodePoolUpdate{
			clusterUID: "cluster-uid", nodePoolUID: "pool-uid", nodePoolKey: "workers",
			userOptions: opts, cmd: cmd, upgrade: nodePoolUpgradeFromFlags(cmd, opts),
		}
		input := &hfpathbind.NodePoolUpdateInput{}
		if err := handler.PreRequest(context.Background(), t.RosaRuntime, input); err != nil {
			return nil, err
		}
		err := handler.PostExpand(context.Background(), t.RosaRuntime, input, &v1alpha1.NodePool{})
		return handler.upgrade, err
	}

	It("accepts an edit that only changes the rolling update", func() {
		upgrade, err := editUpgrade(map[string]string{"max-unavailable": "2"})
		Expect(err).NotTo(HaveOccurred())
		Expect(upgrade).To(Equal(&hyperfleet.NodePoolUpgrade{MaxSurge: "1", MaxUnavailable: "2"}))
	})

	It("carries the value the user did not set forward from the node pool", func() {
		upgrade, err := editUpgrade(map[string]string{"max-surge": "3"})
		Expect(err).NotTo(HaveOccurred())
		Expect(upgrade).To(Equal(&hyperfleet.NodePoolUpgrade{MaxSurge: "3", MaxUnavailable: "0"}))
	})

	It("sends both values when the user set both", func() {
		upgrade, err := editUpgrade(map[string]string{"max-surge": "20%", "max-unavailable": "10%"})
		Expect(err).NotTo(HaveOccurred())
		Expect(upgrade).To(Equal(&hyperfleet.NodePoolUpgrade{MaxSurge: "20%", MaxUnavailable: "10%"}))
	})

	It("rejects a rolling update that can never make progress", func() {
		// The node pool's current maxUnavailable is 0, so zeroing max surge
		// leaves a pair that cannot replace any node.
		_, err := editUpgrade(map[string]string{"max-surge": "0"})
		Expect(err).To(MatchError("only one of --max-surge and --max-unavailable can be zero"))
	})

	It("rejects an invalid value", func() {
		_, err := editUpgrade(map[string]string{"max-surge": "101%"})
		Expect(err).To(MatchError(ContainSubstring("invalid --max-surge")))
	})

	It("requires at least one supported flag", func() {
		opts := &EditMachinepoolUserOptions{}
		cmd := upgradeEditCmd(opts, nil)
		handler := &hyperfleetNodePoolUpdate{
			userOptions: opts, cmd: cmd, upgrade: nodePoolUpgradeFromFlags(cmd, opts),
		}
		Expect(handler.upgrade).To(BeNil())
		Expect(handler.PreRequest(context.Background(), test.NewTestRuntime().RosaRuntime,
			&hfpathbind.NodePoolUpdateInput{})).To(MatchError(ContainSubstring("--max-surge, --max-unavailable")))
	})

	It("leaves the rolling update untouched for edits that do not mention it", func() {
		t := test.NewTestRuntime()
		ctrl := gomock.NewController(GinkgoT())
		hf, _, pools := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		current := nodePoolWithRollingUpdate("1", "0")
		pools.EXPECT().Get(gomock.Any(), "pool-uid", gomock.Any()).Return(current, nil)

		opts := &EditMachinepoolUserOptions{}
		cmd := upgradeEditCmd(opts, nil)
		handler := &hyperfleetNodePoolUpdate{
			clusterUID: "cluster-uid", nodePoolUID: "pool-uid", nodePoolKey: "workers",
			userOptions: opts, cmd: cmd, upgrade: nodePoolUpgradeFromFlags(cmd, opts),
		}
		updated := &v1alpha1.NodePool{}
		Expect(handler.PostExpand(context.Background(), t.RosaRuntime,
			&hfpathbind.NodePoolUpdateInput{}, updated)).To(Succeed())
		Expect(handler.upgrade).To(BeNil())
		// Service-set management never travels back on a PUT.
		Expect(updated.Spec.NodePool.Management).To(Equal(hypershiftv1beta1.NodePoolManagement{}))
	})
})
