package test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	//nolint:staticcheck
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

func (a *ArgVerifier) CommandArgMismatches() []string {
	var mismatches []string
	for k := range a.Args {
		if a.Command.Flags().Lookup(k) == nil {
			mismatches = append(mismatches, "Command '"+a.Command.CommandPath()+"': missing expected flag '"+k+"'")
		}
	}

	a.Command.Flags().VisitAll(func(flag *pflag.Flag) {
		if a.Args[flag.Name] == nil {
			mismatches = append(mismatches, "Command '"+a.Command.CommandPath()+"': unexpected flag '"+flag.Name+"'")
		}
	})
	sort.Strings(mismatches)
	return mismatches
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
