package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	. "github.com/openshift/rosa/pkg/test"
)

func TestCommandStructure(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa command structure")
}

const (
	structureTestDirectory = "./structure_test"
)

var _ = Describe("ROSA Commands", func() {
	It("Have all basic fields defined correctly", func() {
		assertCommand(root)
	})

	It("Are correctly registered in the command structure", func() {
		/*
			Validates the command structure of the CLI at build time. New commands should be
			added to command_structure.yml in order for this test to pass.
		*/
		structureVerifier := NewStructureVerifier(structureTestDirectory, root)
		structureVerifier.AssertCommandStructure()
	})

	It("Have all command flags correctly registered", func() {
		/*
			Validates all command flags are in command_args.yml for each command. New flags should
			be added to the correct command_args.yml file.
		*/
		assertCommandArgs(root)
	})

	XIt("Re-generates the command_arg directory structure and files", func() {
		/*
			This test can be used to regenerate the structure_test/command_args directory and files.
			It should remain skipped for CI.
		*/
		generateCommandArgsFiles(root)
	})
})

func assertCommandArgs(command *cobra.Command) {
	if len(command.Commands()) == 0 {
		verifier := NewArgVerifier(structureTestDirectory, command)
		verifier.AssertCommandArgs()
	} else {
		for _, c := range command.Commands() {
			assertCommandArgs(c)
		}
	}
}

func generateCommandArgsFiles(command *cobra.Command) {
	cmdPath := filepath.Join(strings.Split(command.CommandPath(), " ")...)
	dirPath := filepath.Join(structureTestDirectory, CommandArgDirectoryName, cmdPath)
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		Expect(os.MkdirAll(dirPath, 0600)).To(Succeed())
	}

	if len(command.Commands()) != 0 {
		for _, c := range command.Commands() {
			generateCommandArgsFiles(c)
		}
	} else {
		generator := NewArgGenerator(filepath.Join(dirPath, CommandArgFileName), command)
		generator.GenerateArgsFile()
	}
}

func assertCommand(command *cobra.Command) {
	Expect(command.Use).NotTo(BeNil(), "Use cannot be nil on command '%s'", command.CommandPath())
	Expect(command.Short).NotTo(
		BeNil(), "Short description is not set on command '%s'", command.CommandPath())
	Expect(command.Long).NotTo(
		BeNil(), "Long description is not set on command '%s'", command.CommandPath())
	Expect(command.Example).NotTo(
		BeNil(), "Example is not set on command '%s'", command.CommandPath())
	Expect(command.Args).NotTo(
		BeNil(), "command.Args function is not set on command '%s'", command.CommandPath())

	if len(command.Commands()) == 0 {
		Expect(command.Run).NotTo(
			BeNil(), "The run function is not defined on command '%s'", command.CommandPath())
	} else {
		for _, c := range command.Commands() {
			assertCommand(c)
		}
	}
}
