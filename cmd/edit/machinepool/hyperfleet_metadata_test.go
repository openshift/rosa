package machinepool

import (
	"context"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/test"
)

func metadataEditCmd(opts *EditMachinepoolUserOptions, values map[string]string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringVar(&opts.labels, "labels", "", "")
	cmd.Flags().StringVar(&opts.taints, "taints", "", "")
	for flag, value := range values {
		Expect(cmd.Flags().Set(flag, value)).To(Succeed())
	}
	return cmd
}

var _ = Describe("HyperFleet label and taint editing", func() {
	DescribeTable("changes only requested fields", func(flags map[string]string, expectedLabels map[string]string, expectedTaints []hypershiftv1beta1.Taint) {
		t := test.NewTestRuntime()
		ctrl := gomock.NewController(GinkgoT())
		hf, _, pools := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		replicas, repair := int32(3), true
		current := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "workers", UID: "pool-uid"}, Spec: v1alpha1.NodePoolSpec{
			DisplayName: "Workers", AutoRepair: &repair, Labels: map[string]string{"old": "value"},
			NodePool: v1alpha1.NodePoolSpecPassthrough{
				Replicas: &replicas, NodeLabels: map[string]string{"operator": "managed"},
				Taints:   []hypershiftv1beta1.Taint{{Key: "old", Value: "value", Effect: "NoSchedule"}},
				Platform: v1alpha1.NodePoolPlatform{AWS: &hypershiftv1beta1.AWSNodePoolPlatform{InstanceType: "c5.xlarge", RootVolume: &hypershiftv1beta1.Volume{Size: 75}}},
			},
		}}
		original := current.DeepCopy()
		pools.EXPECT().Get(gomock.Any(), "pool-uid", gomock.Any()).Return(current, nil)
		opts := &EditMachinepoolUserOptions{}
		handler := &hyperfleetNodePoolUpdate{clusterUID: "cluster-uid", nodePoolUID: "pool-uid", nodePoolKey: "workers", userOptions: opts, cmd: metadataEditCmd(opts, flags)}
		input := &hfpathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), t.RosaRuntime, input)).To(Succeed())
		updated := &v1alpha1.NodePool{}
		Expect(handler.PostExpand(context.Background(), t.RosaRuntime, input, updated)).To(Succeed())
		expected := original.DeepCopy()
		expected.Spec.Labels = expectedLabels
		expected.Spec.NodePool.Taints = expectedTaints
		expected.Spec.NodePool.NodeLabels = nil
		expected.Spec.NodePool.Management = hypershiftv1beta1.NodePoolManagement{}
		Expect(updated).To(Equal(expected))
		Expect(current).To(Equal(original))
	},
		Entry("labels only", map[string]string{"labels": "new="}, map[string]string{"new": ""}, []hypershiftv1beta1.Taint{{Key: "old", Value: "value", Effect: "NoSchedule"}}),
		Entry("taints only", map[string]string{"taints": "new=:NoExecute"}, map[string]string{"old": "value"}, []hypershiftv1beta1.Taint{{Key: "new", Effect: "NoExecute"}}),
		Entry("both", map[string]string{"labels": "new=test", "taints": "new=test:PreferNoSchedule"}, map[string]string{"new": "test"}, []hypershiftv1beta1.Taint{{Key: "new", Value: "test", Effect: "PreferNoSchedule"}}),
		Entry("clear labels", map[string]string{"labels": ""}, map[string]string{}, []hypershiftv1beta1.Taint{{Key: "old", Value: "value", Effect: "NoSchedule"}}),
		Entry("clear taints", map[string]string{"taints": ""}, map[string]string{"old": "value"}, []hypershiftv1beta1.Taint(nil)),
		Entry("clear both", map[string]string{"labels": `""`, "taints": `""`}, map[string]string{}, []hypershiftv1beta1.Taint(nil)),
	)
	DescribeTable("rejects invalid values before updating", func(flags map[string]string, expectedError string) {
		opts := &EditMachinepoolUserOptions{}
		h := &hyperfleetNodePoolUpdate{userOptions: opts, cmd: metadataEditCmd(opts, flags)}
		Expect(h.PreRequest(context.Background(), test.NewTestRuntime().RosaRuntime, &hfpathbind.NodePoolUpdateInput{})).To(MatchError(ContainSubstring(expectedError)))
	},
		Entry("label format", map[string]string{"labels": "invalid"}, "expected key=value format"),
		Entry("duplicate labels", map[string]string{"labels": "key=a,key=b"}, "duplicated label key"),
		Entry("invalid taint effect", map[string]string{"taints": "key=value:Invalid"}, "invalid taint effect"),
		Entry("invalid taint format", map[string]string{"taints": "key=value"}, "expected key=value:scheduleType format"),
	)
})
