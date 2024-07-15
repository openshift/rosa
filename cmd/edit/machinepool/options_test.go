package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test edit machinepool options", func() {
	var args *EditMachinepoolUserOptions
	Context("Edit Machinepool User Options", func() {
		It("Creates default options", func() {
			args = NewEditMachinepoolUserOptions()
			Expect(args.machinepool).To(Equal(""))
		})
	})
	Context("Edit Machinepool Options", func() {
		var options EditMachinepoolOptions
		It("Create args from options using Bind (also tests MachinePool func)", func() {
			// Set value then bind
			testMachinepool := "test"
			args.machinepool = testMachinepool
			Expect(options.Bind(args, []string{})).To(Succeed())
			Expect(options.Machinepool()).To(Equal(testMachinepool))
		})
		It("Fail to bind args due to empty machinepool name", func() {
			args.machinepool = ""
			err := options.Bind(args, []string{})
			Expect(err).To(MatchError("you need to specify a machine pool name"))
		})
		It("Test Bind with argv instead of normal args (single arg, no flag for machinepool)", func() {
			argv := []string{"test-id"}
			args.machinepool = ""
			Expect(options.Bind(args, argv)).To(Succeed())
			Expect(options.Machinepool()).To(Equal(argv[0]))
		})
		It("Test labels with options (pass)", func() {
			testLabels := "test=true"
			testMachinepool := "test"
			args.labels = testLabels
			args.machinepool = testMachinepool
			Expect(options.Bind(args, []string{})).To(Succeed())
		})
		It("Test labels with options (fail)", func() {
			testLabels := "test:::::::123123123123::,,,"
			testMachinepool := "test"
			args.labels = testLabels
			args.machinepool = testMachinepool
			err := options.Bind(args, []string{})
			Expect(err).To(MatchError("Expected key=value format for labels"))
		})
	})
})
