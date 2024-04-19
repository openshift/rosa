package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test describe machinepool options", func() {
	var args DescribeMachinepoolUserOptions
	Context("Describe Machinepool User Options", func() {
		It("Creates default options", func() {
			args = NewDescribeMachinepoolUserOptions()
			Expect(args.machinepool).To(Equal(""))
		})
	})
	Context("Describe Machinepool Options", func() {
		var options DescribeMachinepoolOptions
		It("Create args from options using Bind (also tests MachinePool func)", func() {
			// Set value then bind
			testMachinepool := "test"
			args.machinepool = testMachinepool
			err := options.Bind(args)
			Expect(err).ToNot(HaveOccurred())
			Expect(options.Machinepool()).To(Equal(testMachinepool))
		})
		It("Fail to bind args due to empty machinepool name", func() {
			args.machinepool = ""
			err := options.Bind(args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("you need to specify a machine pool name"))
		})
	})
})
