package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
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
	})
})
