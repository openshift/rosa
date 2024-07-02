package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
)

// Will un-comment and become the cmd test once the work on refactoring this command is finalized
var _ = Describe("Edit Machinepool", func() {

	/*Context("getNodePoolReplicas", func() {
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

	})*/
})
