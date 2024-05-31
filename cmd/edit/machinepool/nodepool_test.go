package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/rosa"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Nodepool", func() {
	Context("editAutoscaling", func() {
		It("editAutoscaling should equal nil if nothing is changed", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(2).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 1, 2)
			Expect(builder).To(BeNil())
		})

		It("editAutoscaling should equal the exepcted output", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(2).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 2, 3)
			asBuilder := cmv1.NewNodePoolAutoscaling().MaxReplica(3).MinReplica(2)
			Expect(builder).To(Equal(asBuilder))
		})

		It("editAutoscaling should equal the exepcted output with no min replica value", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(2).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 0, 3)
			asBuilder := cmv1.NewNodePoolAutoscaling().MaxReplica(3).MinReplica(1)
			Expect(builder).To(Equal(asBuilder))
		})

		It("editAutoscaling should equal the exepcted output with no max replica value", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(4).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 2, 0)
			asBuilder := cmv1.NewNodePoolAutoscaling().MaxReplica(4).MinReplica(2)
			Expect(builder).To(Equal(asBuilder))
		})
	})

	Context("Prompt For NodePoolNodeRecreate", func() {

		var t *TestingRuntime
		BeforeEach(func() {
			t = NewTestRuntime()
		})

		It("Prompts when the user has deleted a kubelet-config", func() {

			invoked := false

			f := func(r *rosa.Runtime) bool {
				invoked = true
				return invoked
			}

			original := MockNodePool(func(n *cmv1.NodePoolBuilder) {
				n.KubeletConfigs("test")
			})

			update := MockNodePool(func(n *cmv1.NodePoolBuilder) {
				n.KubeletConfigs("")
			})

			Expect(promptForNodePoolNodeRecreate(original, update, f, t.RosaRuntime)).To(BeTrue())
			Expect(invoked).To(BeTrue())
		})

		It("Prompts when the user has changed a kubelet-config", func() {

			invoked := false

			f := func(r *rosa.Runtime) bool {
				invoked = true
				return invoked
			}

			original := MockNodePool(func(n *cmv1.NodePoolBuilder) {
				n.KubeletConfigs("test")
			})

			update := MockNodePool(func(n *cmv1.NodePoolBuilder) {
				n.KubeletConfigs("bar")
			})

			Expect(promptForNodePoolNodeRecreate(original, update, f, t.RosaRuntime)).To(BeTrue())
			Expect(invoked).To(BeTrue())
		})

		It("Does not prompts when the user has not changed a kubelet-config", func() {

			invoked := false

			f := func(r *rosa.Runtime) bool {
				invoked = true
				return invoked
			}

			original := MockNodePool(func(n *cmv1.NodePoolBuilder) {
				n.KubeletConfigs("test")
			})

			update := MockNodePool(func(n *cmv1.NodePoolBuilder) {
				n.KubeletConfigs("test")
			})

			Expect(promptForNodePoolNodeRecreate(original, update, f, t.RosaRuntime)).To(BeTrue())
			Expect(invoked).To(BeFalse())
		})
	})

	Context("getNodePoolReplicas", func() {
		nodePoolId := "test-nodepool"

		It("KO: Fails if autoscaling is not set", func() {
			Cmd.Flags().Set("min-replicas", "2")
			_, _, _, _, err := getNodePoolReplicas(
				Cmd, rosa.NewRuntime().Reporter, nodePoolId, 2, nil, true)
			Expect(err).Error().Should(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"Autoscaling is not enabled on machine pool 'test-nodepool'. can't set min or max replicas"))
		})

		It("KO: Fails to set replicas if autoscaling is enabled", func() {
			autoScaling := &cmv1.NodePoolAutoscaling{}
			Cmd.Flags().Set("enable-autoscaling", "true")
			Cmd.Flags().Set("replicas", "3")
			_, _, _, _, err := getNodePoolReplicas(
				Cmd, rosa.NewRuntime().Reporter, nodePoolId, 2, autoScaling, true)
			Expect(err).Error().Should(HaveOccurred())
			Expect(err.Error()).To(Equal("Autoscaling enabled on machine pool 'test-nodepool'. can't set replicas"))
		})

	})

	Context("fillAutoScalingAndReplicas", func() {
		var npBuilder *cmv1.NodePoolBuilder
		existingNodepool, err := cmv1.NewNodePool().
			Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(4).MinReplica(1)).
			Build()
		Expect(err).To(BeNil())
		It("Autoscaling set", func() {
			npBuilder = cmv1.NewNodePool()
			fillAutoScalingAndReplicas(npBuilder, true, existingNodepool, 1, 3, 2)
			npPatch, err := npBuilder.Build()
			Expect(err).To(BeNil())
			Expect(npPatch.Autoscaling()).ToNot(BeNil())
			// Default (zero) value
			Expect(npPatch.Replicas()).To(Equal(0))
		})
		It("Replicas set", func() {
			npBuilder = cmv1.NewNodePool()
			fillAutoScalingAndReplicas(npBuilder, false, existingNodepool, 0, 0, 2)
			npPatch, err := npBuilder.Build()
			Expect(err).To(BeNil())
			Expect(npPatch.Autoscaling()).To(BeNil())
			Expect(npPatch.Replicas()).To(Equal(2))
		})

	})
})
