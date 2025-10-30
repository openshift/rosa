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

var _ = Describe("ResourceRegistry", func() {
	var rootCmd *cobra.Command
	var executor *mcp.CommandExecutor
	var registry *mcp.ResourceRegistry

	BeforeEach(func() {
		rootCmd = &cobra.Command{
			Use:   "test",
			Short: "Test command",
		}
		executor = mcp.NewCommandExecutor(rootCmd)
		registry = mcp.NewResourceRegistry(executor)
	})

	Describe("GetResources", func() {
		It("should return list of resources", func() {
			resources := registry.GetResources()
			Expect(resources).ToNot(BeNil())
			Expect(len(resources)).To(BeNumerically(">", 0))
		})

		It("should include expected resource types", func() {
			resources := registry.GetResources()
			resourceURIs := make(map[string]bool)
			for _, res := range resources {
				resourceURIs[res.URI] = true
			}

			Expect(resourceURIs["rosa://clusters"]).To(BeTrue())
			Expect(resourceURIs["rosa://account-roles"]).To(BeTrue())
			Expect(resourceURIs["rosa://operator-roles"]).To(BeTrue())
		})
	})

	Describe("ReadResource", func() {
		It("should return error for invalid URI", func() {
			_, _, err := registry.ReadResource("invalid://uri")
			Expect(err).To(HaveOccurred())
		})

		It("should return error for missing cluster ID when required", func() {
			_, _, err := registry.ReadResource("rosa://cluster")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster ID"))
		})
	})
})
