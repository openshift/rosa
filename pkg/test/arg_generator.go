package test

import (
	"os"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ArgGenerator struct {
	file    string
	command *cobra.Command
}

func (a *ArgGenerator) GenerateArgsFile() {
	var args []*rosaCommandArg

	a.command.Flags().VisitAll(func(flag *pflag.Flag) {
		args = append(args, &rosaCommandArg{Name: flag.Name})
	})

	output, err := yaml.Marshal(args)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(a.file, output, 0600)).To(Succeed())
}

func NewArgGenerator(argFile string, command *cobra.Command) *ArgGenerator {
	return &ArgGenerator{
		file:    argFile,
		command: command,
	}
}
