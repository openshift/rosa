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

	Describe("NewCreateMachinepoolUserOptions", func() {
		It("should create default user options", func() {
			Expect(userOptions.InstanceType).To(Equal("m5.xlarge"))
			Expect(userOptions.AutoscalingEnabled).To(BeFalse())
			Expect(userOptions.MultiAvailabilityZone).To(BeTrue())
			Expect(userOptions.Autorepair).To(BeTrue())
		})
	})

	Describe("NewCreateMachinepoolOptions", func() {
		It("should create default machine pool options", func() {
			Expect(machinepoolOptions.reporter).To(BeAssignableToTypeOf(&reporter.Object{}))
			Expect(machinepoolOptions.args).To(BeAssignableToTypeOf(&mpOpts.CreateMachinepoolUserOptions{}))
		})
	})

	Describe("Machinepool", func() {
		It("should return the args field", func() {
			machinepoolOptions.args = userOptions
			Expect(machinepoolOptions.Machinepool()).To(Equal(userOptions))
		})
	})

	Describe("Bind", func() {
		It("should bind the passed arguments", func() {
			argv := []string{"test-pool"}
			Expect(machinepoolOptions.Bind(userOptions, argv)).To(Succeed())
			Expect(machinepoolOptions.args).To(Equal(userOptions))
			Expect(machinepoolOptions.args.Name).To(Equal("test-pool"))
		})

		It("should not modify args.Name if no arguments passed", func() {
			initialName := userOptions.Name
			Expect(machinepoolOptions.Bind(userOptions, []string{})).To(Succeed())
			Expect(machinepoolOptions.args.Name).To(Equal(initialName))
		})
	})
})
