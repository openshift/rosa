package machinepool

import (
	"context"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/pathbind"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

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

// nodePoolWithUpgrade builds a node pool carrying the customer rolling-update
// settings the Platform API reports on spec.
func nodePoolWithUpgrade(maxSurge, maxUnavailable string) *v1alpha1.NodePool {
	surge := intstr.Parse(maxSurge)
	unavailable := intstr.Parse(maxUnavailable)
	return &v1alpha1.NodePool{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers", UID: "pool-uid"},
		Spec: v1alpha1.NodePoolSpec{
			MaxSurge:       &surge,
			MaxUnavailable: &unavailable,
			NodePool: v1alpha1.NodePoolSpecPassthrough{
				Management: hypershiftv1beta1.NodePoolManagement{
					UpgradeType: hypershiftv1beta1.UpgradeTypeReplace,
				},
			},
		},
	}
}

var _ = Describe("HyperFleet machine pool rolling update editing", func() {
	// editUpgrade runs the handler the way runHyperfleetEdit does, against a
	// node pool whose current rolling update is maxSurge 1 / maxUnavailable 0.
	editUpgrade := func(flags map[string]string) (*v1alpha1.NodePool, error) {
		t := test.NewTestRuntime()
		ctrl := gomock.NewController(GinkgoT())
		hf, _, pools := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		pools.EXPECT().Get(gomock.Any(), "my-cluster.workers", gomock.Any()).
			Return(nodePoolWithUpgrade("1", "0"), nil).AnyTimes()

		opts := &EditMachinepoolUserOptions{}
		cmd := upgradeEditCmd(opts, flags)
		handler := &hyperfleetNodePoolUpdate{
			clusterKey: "my-cluster", nodePoolKey: "workers", nodePoolName: "my-cluster.workers",
			userOptions: opts, cmd: cmd,
		}
		input := &hfpathbind.NodePoolUpdateInput{}
		if err := handler.PreRequest(context.Background(), t.RosaRuntime, input); err != nil {
			return nil, err
		}
		// Expand the input the way RunUpdateNodePool does, so the test decodes
		// the rolling-update values through the real int-or-string handling
		// rather than a shortcut that cannot fail the same way.
		obj := &v1alpha1.NodePool{}
		if err := pathbind.Expand(context.Background(), *input, obj); err != nil {
			return nil, err
		}
		err := handler.PostExpand(context.Background(), t.RosaRuntime, input, obj)
		return obj, err
	}

	It("accepts an edit that only changes the rolling update", func() {
		updated, err := editUpgrade(map[string]string{"max-unavailable": "2"})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Spec.MaxUnavailable.String()).To(Equal("2"))
	})

	It("carries the value the user did not set forward from the node pool", func() {
		updated, err := editUpgrade(map[string]string{"max-surge": "3"})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Spec.MaxSurge.String()).To(Equal("3"))
		Expect(updated.Spec.MaxUnavailable.String()).To(Equal("0"))
	})

	It("sends both values when the user set both", func() {
		updated, err := editUpgrade(map[string]string{"max-surge": "20%", "max-unavailable": "10%"})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Spec.MaxSurge.String()).To(Equal("20%"))
		Expect(updated.Spec.MaxUnavailable.String()).To(Equal("10%"))
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
		handler := &hyperfleetNodePoolUpdate{userOptions: opts, cmd: upgradeEditCmd(opts, nil)}
		Expect(handler.PreRequest(context.Background(), test.NewTestRuntime().RosaRuntime,
			&hfpathbind.NodePoolUpdateInput{})).To(MatchError(ContainSubstring("--max-surge, --max-unavailable")))
	})

	It("leaves the rolling update untouched for edits that do not mention it", func() {
		t := test.NewTestRuntime()
		ctrl := gomock.NewController(GinkgoT())
		hf, _, pools := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		pools.EXPECT().Get(gomock.Any(), "my-cluster.workers", gomock.Any()).
			Return(nodePoolWithUpgrade("3", "2"), nil)

		opts := &EditMachinepoolUserOptions{replicas: 5}
		cmd := upgradeEditCmd(opts, nil)
		cmd.Flags().Int("replicas", 0, "")
		Expect(cmd.Flags().Set("replicas", "5")).To(Succeed())
		handler := &hyperfleetNodePoolUpdate{
			clusterKey: "my-cluster", nodePoolKey: "workers", nodePoolName: "my-cluster.workers",
			userOptions: opts, cmd: cmd,
		}
		input := &hfpathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), t.RosaRuntime, input)).To(Succeed())
		Expect(input.MaxSurge).To(BeEmpty())
		Expect(input.MaxUnavailable).To(BeEmpty())

		updated := &v1alpha1.NodePool{}
		Expect(handler.PostExpand(context.Background(), t.RosaRuntime, input, updated)).To(Succeed())
		Expect(updated.Spec.MaxSurge.String()).To(Equal("3"))
		Expect(updated.Spec.MaxUnavailable.String()).To(Equal("2"))
	})
})
