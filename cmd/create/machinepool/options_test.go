package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/reporter"
)

var _ = Describe("CreateMachinepoolOptions", func() {
	var (
		machinepoolOptions *CreateMachinepoolOptions
		userOptions        *mpOpts.CreateMachinepoolUserOptions
	)

	BeforeEach(func() {
		machinepoolOptions = NewCreateMachinepoolOptions()
		userOptions = NewCreateMachinepoolUserOptions()
	})

	Context("NewCreateMachinepoolUserOptions", func() {
		It("should create default user options", func() {
			Expect(userOptions.InstanceType).To(Equal(instanceType))
			Expect(userOptions.AutoscalingEnabled).To(BeFalse())
			Expect(userOptions.MultiAvailabilityZone).To(BeTrue())
			Expect(userOptions.Autorepair).To(BeTrue())
		})
	})

	Context("NewCreateMachinepoolOptions", func() {
		It("should create default machine pool options", func() {
			Expect(machinepoolOptions.reporter).To(BeAssignableToTypeOf(&reporter.Object{}))
			Expect(machinepoolOptions.args).To(BeAssignableToTypeOf(&mpOpts.CreateMachinepoolUserOptions{}))
		})
	})

	Context("Machinepool", func() {
		It("should return the args field", func() {
			machinepoolOptions.args = userOptions
			Expect(machinepoolOptions.Machinepool()).To(Equal(userOptions))
		})
	})
})
