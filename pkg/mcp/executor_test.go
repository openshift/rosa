/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mcp_test

import (
	"github.com/spf13/cobra"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/mcp"
)

var _ = Describe("CommandExecutor", func() {
	var rootCmd *cobra.Command
	var executor *mcp.CommandExecutor

	BeforeEach(func() {
		rootCmd = &cobra.Command{
			Use:   "test",
			Short: "Test command",
		}
		executor = mcp.NewCommandExecutor(rootCmd)
	})

	Describe("GetAllCommands", func() {
		It("should return commands for root command", func() {
			commands := executor.GetAllCommands()
			Expect(commands).ToNot(BeNil())
			// Empty root command should return empty list (not nil)
			Expect(commands).To(BeEmpty())
		})

		It("should discover subcommands", func() {
			subCmd := &cobra.Command{
				Use:   "subcommand",
				Short: "A subcommand",
			}
			rootCmd.AddCommand(subCmd)

			commands := executor.GetAllCommands()
			found := false
			for _, cmd := range commands {
				if len(cmd.Path) > 0 && cmd.Path[0] == "subcommand" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Describe("FindCommand", func() {
		It("should find a command in the tree", func() {
			subCmd := &cobra.Command{
				Use:   "subcommand",
				Short: "A subcommand",
			}
			rootCmd.AddCommand(subCmd)

			cmd, remainingArgs, err := executor.FindCommand([]string{"subcommand"})
			Expect(err).ToNot(HaveOccurred())
			Expect(cmd).ToNot(BeNil())
			Expect(cmd.Use).To(Equal("subcommand"))
			Expect(remainingArgs).To(BeEmpty())
		})

		It("should return error for non-existent command", func() {
			_, _, err := executor.FindCommand([]string{"nonexistent"})
			Expect(err).To(HaveOccurred())
		})
	})
})
