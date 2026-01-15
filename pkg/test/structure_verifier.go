package test

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const (
	commandStructureFileName = "command_structure.yml"
)

type StructureVerifier struct {
	command     *cobra.Command
	rosaCommand *rosaCommand
}

type rosaCommand struct {
	Name      string         `json:"name"`
	Children  []*rosaCommand `json:"children,omitempty"`
	Generated bool           `json:"generated,omitempty"`
}

func (c *rosaCommand) ChildCount() int {
	return len(c.Children)
}

func (s *StructureVerifier) toChildrenMap(cmd *cobra.Command) map[string]*cobra.Command {
	childrenMap := make(map[string]*cobra.Command)
	for _, c := range cmd.Commands() {
		childrenMap[c.Name()] = c
	}
	return childrenMap
}

func (s *StructureVerifier) AssertCommandStructure() {
	s.assertCommand(s.rosaCommand, s.command)
}

func (s *StructureVerifier) assertCommand(rosaCommand *rosaCommand, command *cobra.Command) {
	Expect(command).NotTo(BeNil(),
		"Command with name '%s' does not exist in cobra setup for ROSA", rosaCommand.Name)
	Expect(rosaCommand.Name).To(Equal(command.Name()))
	Expect(rosaCommand.ChildCount()).
		To(Equal(len(command.Commands())),
			"Unexpected child command on command '%s'", command.CommandPath())
	if rosaCommand.ChildCount() != 0 {
		childrenMap := s.toChildrenMap(command)
		for _, rc := range rosaCommand.Children {
			if !rc.Generated {
				cc := childrenMap[rc.Name]
				s.assertCommand(rc, cc)
			}
		}
	}
}

func NewStructureVerifier(commandStructureDirectory string, rootCommand *cobra.Command) *StructureVerifier {
	contents, err := os.ReadFile(filepath.Join(commandStructureDirectory, commandStructureFileName))
	Expect(err).NotTo(HaveOccurred())

	var cmd rosaCommand

	err = yaml.Unmarshal(contents, &cmd)
	Expect(err).NotTo(HaveOccurred())
	return &StructureVerifier{
		command:     rootCommand,
		rosaCommand: &cmd,
	}
}
