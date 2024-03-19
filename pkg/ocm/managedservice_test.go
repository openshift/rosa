package ocm

import (
	"fmt"

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
			Expect(err.Error()).To(Equal(
				fmt.Errorf("managed services are not supported for FedRAMP clusters").Error()))
		})
		It("Can't update managedService", func() {
			fedramp.Enable() // Simulate logging into fedramp
			err := client.UpdateManagedService(UpdateManagedServiceArgs{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				fmt.Errorf("managed services are not supported for FedRAMP clusters").Error()))
		})
	})
})
