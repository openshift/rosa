package ocm

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/fedramp"
)

var _ = Describe("ManagedService", func() {
	var client Client

	BeforeEach(func() {
		fedramp.Enable()
	})

	AfterEach(func() {
		fedramp.Disable()
	})

	When("FedRAMP is true", func() {
		It("Can't create managedService", func() {
			fedramp.Enable() // Simulate logging into fedramp
			service, err := client.CreateManagedService(CreateManagedServiceArgs{})
			Expect(service).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fedrampError.Error()))
		})
		It("Can't update managedService", func() {
			fedramp.Enable() // Simulate logging into fedramp
			err := client.UpdateManagedService(UpdateManagedServiceArgs{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fedrampError.Error()))
		})
		It("Can't list managedService", func() {
			fedramp.Enable() // Simulate logging into fedramp
			output, err := client.ListManagedServices(1000)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fedrampError.Error()))
			Expect(output).To(BeNil())
		})
		It("Can't get managedService", func() {
			fedramp.Enable() // Simulate logging into fedramp
			output, err := client.GetManagedService(DescribeManagedServiceArgs{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fedrampError.Error()))
			Expect(output).To(BeNil())
		})
		It("Can't delete managedService", func() {
			fedramp.Enable() // Simulate logging into fedramp
			output, err := client.DeleteManagedService(DeleteManagedServiceArgs{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fedrampError.Error()))
			Expect(output).To(BeNil())
		})
	})
})
