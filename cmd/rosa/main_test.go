package main

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

func TestCommandStructure(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa command structure")
}

var _ = Describe("ROSA Commands", func() {
	It("Have all basic fields defined correctly", func() {
		assertCommand(root)
	})
})

func assertCommand(command *cobra.Command) {
	Expect(command.Use).NotTo(BeNil(), fmt.Sprintf("Use cannot be nil on command '%s'", command.CommandPath()))
	Expect(command.Short).NotTo(
		BeNil(), fmt.Sprintf("Short description is not set on command '%s'", command.CommandPath()))
	Expect(command.Long).NotTo(
		BeNil(), fmt.Sprintf("Long description is not set on command '%s'", command.CommandPath()))
	Expect(command.Example).NotTo(
		BeNil(), fmt.Sprintf("Example is not set on command '%s'", command.CommandPath()))
	Expect(command.Args).NotTo(
		BeNil(), fmt.Sprintf("command.Args function is not set on command '%s'", command.CommandPath()))

	if len(command.Commands()) == 0 {
		Expect(command.Run).NotTo(
			BeNil(), fmt.Sprintf("The run function is not defined on command '%s'", command.CommandPath()))
	} else {
		for _, c := range command.Commands() {
			assertCommand(c)
		}
	}
}
