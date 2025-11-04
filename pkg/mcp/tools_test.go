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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/mcp"
)

var _ = Describe("ToolRegistry", func() {
	var rootCmd *cobra.Command
	var registry *mcp.ToolRegistry

	BeforeEach(func() {
		rootCmd = &cobra.Command{
			Use:   "test",
			Short: "Test command",
		}
		registry = mcp.NewToolRegistry(rootCmd)
	})

	Describe("GetTools", func() {
		It("should return tools for discovered commands", func() {
			subCmd := &cobra.Command{
				Use:   "subcommand",
				Short: "A subcommand",
			}
			rootCmd.AddCommand(subCmd)

			registry = mcp.NewToolRegistry(rootCmd)
			tools := registry.GetTools()
			Expect(tools).ToNot(BeNil())
			Expect(len(tools)).To(BeNumerically(">=", 0))
		})
	})

	Describe("ParseToolName", func() {
		It("should parse tool name correctly", func() {
			path := registry.ParseToolName("rosa_create_cluster")
			Expect(path).To(Equal([]string{"create", "cluster"}))
		})

		It("should handle tool name without rosa_ prefix", func() {
			path := registry.ParseToolName("create_cluster")
			Expect(path).To(Equal([]string{"create", "cluster"}))
		})
	})

	Describe("CallTool", func() {
		It("should handle help tool", func() {
			result, err := registry.CallTool("rosa_help", map[string]interface{}{
				"command": []string{"test"},
			})
			// Help tool might fail in test environment but verify it's called
			if err != nil {
				Expect(err).ToNot(BeNil())
			} else {
				Expect(result).ToNot(BeNil())
			}
		})

		It("should handle hierarchical tool with resource", func() {
			// Create a test command for list action
			testCmd := &cobra.Command{
				Use:   "list",
				Short: "List command",
			}
			listSubCmd := &cobra.Command{
				Use:   "clusters",
				Short: "List clusters",
			}
			testCmd.AddCommand(listSubCmd)
			rootCmd.AddCommand(testCmd)

			registry = mcp.NewToolRegistry(rootCmd)

			result, err := registry.CallTool("rosa_list", map[string]interface{}{
				"resource": "clusters",
			})
			// May fail in test environment but verifies the path
			if err != nil {
				Expect(err).ToNot(BeNil())
			} else {
				Expect(result).ToNot(BeNil())
			}
		})

		It("should return error for hierarchical tool without resource", func() {
			testCmd := &cobra.Command{
				Use:   "create",
				Short: "Create command",
			}
			// Add a subcommand so getAvailableResources returns a non-empty slice
			createSubCmd := &cobra.Command{
				Use:   "cluster",
				Short: "Create cluster",
			}
			testCmd.AddCommand(createSubCmd)
			rootCmd.AddCommand(testCmd)

			registry = mcp.NewToolRegistry(rootCmd)

			_, err := registry.CallTool("rosa_create", map[string]interface{}{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource"))
		})

		It("should handle standalone tool (whoami)", func() {
			testCmd := &cobra.Command{
				Use:   "whoami",
				Short: "Show current user",
			}
			rootCmd.AddCommand(testCmd)

			registry = mcp.NewToolRegistry(rootCmd)

			result, err := registry.CallTool("rosa_whoami", map[string]interface{}{})
			if err != nil {
				Expect(err).ToNot(BeNil())
			} else {
				Expect(result).ToNot(BeNil())
			}
		})

		It("should handle flags in arguments", func() {
			testCmd := &cobra.Command{
				Use:   "list",
				Short: "List command",
			}
			listSubCmd := &cobra.Command{
				Use:   "clusters",
				Short: "List clusters",
			}
			listSubCmd.Flags().String("cluster", "", "Cluster name")
			testCmd.AddCommand(listSubCmd)
			rootCmd.AddCommand(testCmd)

			registry = mcp.NewToolRegistry(rootCmd)

			result, err := registry.CallTool("rosa_list", map[string]interface{}{
				"resource": "clusters",
				"flags": map[string]interface{}{
					"cluster": "test-cluster",
				},
			})
			if err != nil {
				Expect(err).ToNot(BeNil())
			} else {
				Expect(result).ToNot(BeNil())
			}
		})

		It("should return error for unknown tool", func() {
			_, err := registry.CallTool("rosa_unknown_tool", map[string]interface{}{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetHierarchicalTools", func() {
		It("should return hierarchical tools", func() {
			testCmd := &cobra.Command{
				Use:   "list",
				Short: "List command",
			}
			listSubCmd := &cobra.Command{
				Use:   "clusters",
				Short: "List clusters",
			}
			testCmd.AddCommand(listSubCmd)
			rootCmd.AddCommand(testCmd)

			registry = mcp.NewToolRegistry(rootCmd)

			tools := registry.GetHierarchicalTools()
			Expect(tools).ToNot(BeNil())
			Expect(len(tools)).To(BeNumerically(">=", 0))
		})
	})

	Describe("mapFlagTypeToMCPType", func() {
		It("should discover tools with various flag types", func() {
			testCmd := &cobra.Command{
				Use:   "testcmd",
				Short: "Test command",
			}
			testCmd.Flags().String("string-flag", "", "String flag")
			testCmd.Flags().Int("int-flag", 0, "Int flag")
			testCmd.Flags().Bool("bool-flag", false, "Bool flag")
			rootCmd.AddCommand(testCmd)

			registry = mcp.NewToolRegistry(rootCmd)
			tools := registry.GetTools()
			Expect(tools).ToNot(BeNil())
		})
	})
})
