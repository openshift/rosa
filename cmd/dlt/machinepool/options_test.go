package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test delete machinepool options", func() {
	var args *DeleteMachinepoolUserOptions
	var options *DeleteMachinepoolOptions
	BeforeEach(func() {
		options = NewDeleteMachinepoolOptions()
	})
	Context("Delete Machinepool User Options", func() {
		It("Creates default options", func() {
			args = NewDeleteMachinepoolUserOptions()
			Expect(args.machinepool).To(Equal(""))
		})
	})
	Context("Delete Machinepool Options", func() {
		It("Create args from options using Bind (also tests MachinePool func)", func() {
			// Set value then bind
			testMachinepool := "test"
			args.machinepool = testMachinepool
			err := options.Bind(args, []string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(options.Machinepool()).To(Equal(testMachinepool))
		})
		It("Fail to bind args due to empty machinepool name", func() {
			args.machinepool = ""
			err := options.Bind(args, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("You need to specify a machine pool name"))
		})
		It("Fail to bind args due to invalid machinepool name", func() {
			args.machinepool = "%asd"
			err := options.Bind(args, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected a valid identifier for the machine pool"))
			args.machinepool = "1asd"
			err = options.Bind(args, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected a valid identifier for the machine pool"))
			args.machinepool = "asd123$"
			err = options.Bind(args, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected a valid identifier for the machine pool"))
		})
		It("Test Bind with argv instead of normal args (single arg, no flag for machinepool)", func() {
			argv := []string{"test-id"}
			args.machinepool = ""
			options.Bind(args, argv)
			Expect(options.Machinepool()).To(Equal(argv[0]))
		})
	})
})
