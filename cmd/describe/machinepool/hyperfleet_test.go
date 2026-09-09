package machinepool

import (
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

func buildTestNodePool() *v1alpha1.NodePool {
	replicas := int32(3)
	return &v1alpha1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "my-nodepool",
			UID:               types.UID("np-uid-789"),
			CreationTimestamp: metav1.NewTime(time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)),
		},
		Spec: v1alpha1.NodePoolSpec{
			NodePool: v1alpha1.NodePoolSpecPassthrough{
				Replicas: &replicas,
				Release:  hypershiftv1beta1.Release{Image: "v4.17.0-ec.2"},
				Platform: v1alpha1.NodePoolPlatform{
					AWS: &hypershiftv1beta1.AWSNodePoolPlatform{
						InstanceType: "m5.xlarge",
					},
				},
			},
		},
		Status: v1alpha1.NodePoolStatus{
			Phase: v1alpha1.NodePoolPhaseReady,
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  "True",
					Reason:  "AsExpected",
					Message: "All nodes are ready",
				},
			},
		},
	}
}

var _ = Describe("hyperfleet dispatch", func() {
	var origEnabled func() bool
	var origDescribe func(*DescribeMachinepoolUserOptions, []string)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origDescribe = hfDescribeMachinePool
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfDescribeMachinePool = origDescribe
	})

	It("routes to hfDescribeMachinePool when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfDescribeMachinePool = func(_ *DescribeMachinepoolUserOptions, _ []string) { called = true }

		cmd := NewDescribeMachinePoolCommand()
		cmd.Run(cmd, nil)

		Expect(called).To(BeTrue())
	})
})

var _ = Describe("hfNodePoolToMap", func() {
	It("maps all core fields", func() {
		np := buildTestNodePool()
		m := hfNodePoolToMap(np)

		Expect(m["id"]).To(Equal("np-uid-789"))
		Expect(m["name"]).To(Equal("my-nodepool"))
		Expect(m["state"]).To(Equal("Ready"))
		Expect(m["replicas"]).To(Equal(int32(3)))
		Expect(m["instanceType"]).To(Equal("m5.xlarge"))
		Expect(m["version"]).To(Equal("v4.17.0-ec.2"))
		Expect(m["created_at"]).To(Equal("2026-06-15T10:30:00Z"))
	})

	It("maps conditions", func() {
		np := buildTestNodePool()
		m := hfNodePoolToMap(np)

		conds, ok := m["conditions"].([]map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(conds).To(HaveLen(1))
		Expect(conds[0]["type"]).To(Equal("Ready"))
		Expect(conds[0]["status"]).To(Equal("True"))
		Expect(conds[0]["reason"]).To(Equal("AsExpected"))
		Expect(conds[0]["message"]).To(Equal("All nodes are ready"))
	})

	It("handles nil replicas as zero", func() {
		np := buildTestNodePool()
		np.Spec.NodePool.Replicas = nil
		m := hfNodePoolToMap(np)
		Expect(m["replicas"]).To(Equal(int32(0)))
	})

	It("handles nil AWS platform", func() {
		np := buildTestNodePool()
		np.Spec.NodePool.Platform.AWS = nil
		m := hfNodePoolToMap(np)
		Expect(m["instanceType"]).To(Equal(""))
	})

	It("returns empty conditions slice when no conditions", func() {
		np := buildTestNodePool()
		np.Status.Conditions = nil
		m := hfNodePoolToMap(np)

		conds, ok := m["conditions"].([]map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(conds).To(BeEmpty())
	})
})

var _ = Describe("hfNodePoolToString", func() {
	It("includes all core fields in output", func() {
		np := buildTestNodePool()
		s := hfNodePoolToString(np, "my-cluster")

		Expect(s).To(ContainSubstring("my-nodepool"))
		Expect(s).To(ContainSubstring("np-uid-789"))
		Expect(s).To(ContainSubstring("my-cluster"))
		Expect(s).To(ContainSubstring("Ready"))
		Expect(s).To(ContainSubstring("3"))
		Expect(s).To(ContainSubstring("m5.xlarge"))
		Expect(s).To(ContainSubstring("v4.17.0-ec.2"))
		Expect(s).To(ContainSubstring("2026-06-15"))
	})

	It("includes conditions in output", func() {
		np := buildTestNodePool()
		s := hfNodePoolToString(np, "my-cluster")

		Expect(s).To(ContainSubstring("Conditions:"))
		Expect(s).To(ContainSubstring("Ready:"))
		Expect(s).To(ContainSubstring("True"))
	})

	It("omits conditions block when none present", func() {
		np := buildTestNodePool()
		np.Status.Conditions = nil
		s := hfNodePoolToString(np, "my-cluster")

		Expect(s).NotTo(ContainSubstring("Conditions:"))
	})

	It("handles nil replicas and nil AWS platform", func() {
		np := buildTestNodePool()
		np.Spec.NodePool.Replicas = nil
		np.Spec.NodePool.Platform.AWS = nil
		s := hfNodePoolToString(np, "my-cluster")

		Expect(strings.Contains(s, "0")).To(BeTrue())
	})
})

var _ = Describe("hfConditionSummary", func() {
	It("returns message when reason is empty", func() {
		Expect(hfConditionSummary("", "some message")).To(Equal("some message"))
	})

	It("returns reason when message is empty", func() {
		Expect(hfConditionSummary("MyReason", "")).To(Equal("MyReason"))
	})

	It("returns reason when message starts with reason", func() {
		Expect(hfConditionSummary("MyReason", "MyReason: some detail")).To(Equal("MyReason"))
	})

	It("combines reason and message when both are non-empty and distinct", func() {
		Expect(hfConditionSummary("MyReason", "some detail")).To(Equal("MyReason: some detail"))
	})
})
