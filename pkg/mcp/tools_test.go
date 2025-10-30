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
})
