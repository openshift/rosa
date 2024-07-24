package test

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type rosaCommandArg struct {
	Name string `json:"name"`
}

type ArgVerifier struct {
	Args    map[string]*rosaCommandArg
	Command *cobra.Command
}

const (
	CommandArgFileName      = "command_args.yml"
	CommandArgDirectoryName = "command_args"
)

func (a *ArgVerifier) AssertCommandArgs() {
	for k := range a.Args {
		flag := a.Command.Flags().Lookup(k)
		Expect(flag).NotTo(
			BeNil(),
			"Flag with name '%s' does not exist on command '%s'. Have you removed a flag?",
			k, a.Command.CommandPath())
	}

	a.Command.Flags().VisitAll(func(flag *pflag.Flag) {
		Expect(a.Args[flag.Name]).NotTo(
			BeNil(),
			"Unexpected flag '%s' on command '%s'. Have you added a new flag?",
			flag.Name, a.Command.CommandPath())
	})
}

func NewArgVerifier(commandStructureDir string, command *cobra.Command) *ArgVerifier {
	_, err := os.Stat(filepath.Join(commandStructureDir, CommandArgDirectoryName))
	Expect(err).NotTo(HaveOccurred())

	contents, err := os.ReadFile(GetCommandArgsFile(commandStructureDir, command))
	Expect(err).NotTo(HaveOccurred(), "Failed to open arg file '%s'", commandStructureDir)

	var args []*rosaCommandArg
	err = yaml.Unmarshal(contents, &args)
	Expect(err).NotTo(HaveOccurred(),
		"Failed to unmarshall arg file '%s' into YAML", commandStructureDir)

	argMap := make(map[string]*rosaCommandArg)
	for _, arg := range args {
		argMap[arg.Name] = arg
	}

	return &ArgVerifier{Args: argMap, Command: command}
}

func GetCommandArgsFile(commandStructureDir string, command *cobra.Command) string {
	commandDir := filepath.Join(strings.Split(command.CommandPath(), " ")...)
	return filepath.Join(commandStructureDir, CommandArgDirectoryName, commandDir, CommandArgFileName)
}
