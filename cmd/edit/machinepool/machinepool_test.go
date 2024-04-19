package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Machinepool", func() {
	Context("editMachinePoolAutoscaling", func() {
		It("editMachinePoolAutoscaling should equal nil if nothing is changed", func() {
			machinepool, err := cmv1.NewMachinePool().
				Autoscaling(cmv1.NewMachinePoolAutoscaling().MaxReplicas(2).MinReplicas(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editMachinePoolAutoscaling(machinepool, 1, 2)
			Expect(builder).To(BeNil())
		})

		It("editMachinePoolAutoscaling should equal the exepcted output", func() {
			machinePool, err := cmv1.NewMachinePool().
				Autoscaling(cmv1.NewMachinePoolAutoscaling().MaxReplicas(2).MinReplicas(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editMachinePoolAutoscaling(machinePool, 2, 3)
			asBuilder := cmv1.NewMachinePoolAutoscaling().MaxReplicas(3).MinReplicas(2)
			Expect(builder).To(Equal(asBuilder))
		})
	})

	Context("isMultiAZMachinePool", func() {
		It("isMultiAZMachinePool should return true", func() {
			machinePool, err := cmv1.NewMachinePool().Build()
			Expect(err).ToNot(HaveOccurred())
			boolean := isMultiAZMachinePool(machinePool)
			Expect(boolean).To(Equal(true))
		})

		It("isMultiAZMachinePool should return false", func() {
			machinePool, err := cmv1.NewMachinePool().AvailabilityZones("test").Build()
			Expect(err).ToNot(HaveOccurred())
			boolean := isMultiAZMachinePool(machinePool)
			Expect(boolean).To(Equal(false))
		})
	})
})
