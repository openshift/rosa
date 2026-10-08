package hyperfleet

import (
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

var _ = Describe("HyperFleet nodepool metadata formatting", func() {
	It("formats labels deterministically and retains empty values", func() {
		Expect(FormatNodePoolLabels(map[string]string{"z": "last", "a": ""})).To(Equal("a=, z=last"))
		Expect(FormatNodePoolLabels(nil)).To(BeEmpty())
	})
	It("formats all supported taint effects and retains empty values", func() {
		taints := []hypershiftv1beta1.Taint{{Key: "a", Effect: "PreferNoSchedule"}, {Key: "b", Value: "value", Effect: "NoExecute"}, {Key: "c", Value: "v", Effect: "NoSchedule"}}
		Expect(FormatNodePoolTaints(taints)).To(Equal("a=:PreferNoSchedule, b=value:NoExecute, c=v:NoSchedule"))
		Expect(FormatNodePoolTaints(nil)).To(BeEmpty())
	})
	It("distinguishes unknown, zero and observed replicas", func() {
		np := &v1alpha1.NodePool{}
		np.Spec.NodePool.Replicas = ptr.To(int32(3))
		Expect(FormatNodePoolReplicas(np)).To(Equal("-/3"))
		np.Status.Replicas = ptr.To(int32(0))
		Expect(FormatNodePoolReplicas(np)).To(Equal("0/3"))
		np.Status.Replicas = ptr.To(int32(2))
		Expect(FormatNodePoolReplicas(np)).To(Equal("2/3"))
		np.Spec.NodePool.Replicas = nil
		Expect(FormatNodePoolReplicas(np)).To(Equal("2/0"))
	})
	It("shows autoscaling bounds instead of a fixed desired count", func() {
		np := &v1alpha1.NodePool{}
		Expect(FormatNodePoolAutoscaling(np)).To(Equal("No"))
		np.Spec.NodePool.AutoScaling = &hypershiftv1beta1.NodePoolAutoScaling{Min: ptr.To(int32(3)), Max: 6}
		Expect(FormatNodePoolAutoscaling(np)).To(Equal("Yes"))
		Expect(FormatNodePoolReplicas(np)).To(Equal("-/3-6"))
		np.Status.Replicas = ptr.To(int32(4))
		Expect(FormatNodePoolReplicas(np)).To(Equal("4/3-6"))
		np.Spec.NodePool.AutoScaling.Min = nil
		Expect(FormatNodePoolReplicas(np)).To(Equal("4/0-6"))
	})
	It("uses explicit autorepair settings before rendered management", func() {
		np := &v1alpha1.NodePool{}
		Expect(FormatNodePoolAutorepair(np)).To(Equal("No"))
		np.Spec.NodePool.Management.AutoRepair = true
		Expect(FormatNodePoolAutorepair(np)).To(Equal("Yes"))
		np.Spec.AutoRepair = ptr.To(false)
		Expect(FormatNodePoolAutorepair(np)).To(Equal("No"))
		np.Spec.NodePool.Management.AutoRepair = false
		np.Spec.AutoRepair = ptr.To(true)
		Expect(FormatNodePoolAutorepair(np)).To(Equal("Yes"))
	})

})
