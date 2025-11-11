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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/mcp"
)

var _ = Describe("ChatClient", func() {
	var rootCmd *cobra.Command
	var server *mcp.Server
	var originalAPIKey string
	var hasAPIKey bool

	BeforeEach(func() {
		rootCmd = &cobra.Command{
			Use:   "test",
			Short: "Test command",
		}
		server = mcp.NewServer(rootCmd)

		// Save original API key if it exists
		originalAPIKey, hasAPIKey = os.LookupEnv("OPENAI_API_KEY")
	})

	AfterEach(func() {
		// Restore original API key or unset it
		if hasAPIKey {
			os.Setenv("OPENAI_API_KEY", originalAPIKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	})

	Describe("GetDefaultSystemMessage", func() {
		It("should return a non-empty system message", func() {
			message := mcp.GetDefaultSystemMessage()
			Expect(message).ToNot(BeEmpty())
			Expect(len(message)).To(BeNumerically(">", 100))
		})

		It("should contain expected keywords", func() {
			message := mcp.GetDefaultSystemMessage()
			Expect(message).To(ContainSubstring("ROSA"))
			Expect(message).To(ContainSubstring("tool"))
			Expect(message).To(ContainSubstring("cluster"))
		})
	})

	Describe("NewChatClient", func() {
		It("should create a client with API key from flag", func() {
			os.Unsetenv("OPENAI_API_KEY")
			client := mcp.NewChatClient(server, "test-api-key-123", "", "gpt-4o", false, "")
			Expect(client).ToNot(BeNil())
		})

		It("should create a client with API key from environment", func() {
			os.Setenv("OPENAI_API_KEY", "env-api-key-456")
			client := mcp.NewChatClient(server, "", "", "gpt-4o", false, "")
			Expect(client).ToNot(BeNil())
		})

		It("should prioritize flag API key over environment variable", func() {
			os.Setenv("OPENAI_API_KEY", "env-api-key")
			client := mcp.NewChatClient(server, "flag-api-key", "", "gpt-4o", false, "")
			Expect(client).ToNot(BeNil())
		})

		It("should use custom system message when provided", func() {
			customMessage := "Custom system message for testing"
			client := mcp.NewChatClient(server, "test-api-key", "", "gpt-4o", false, customMessage)
			Expect(client).ToNot(BeNil())
		})

		It("should use default system message when custom message is empty", func() {
			client := mcp.NewChatClient(server, "test-api-key", "", "gpt-4o", false, "")
			Expect(client).ToNot(BeNil())
		})

		It("should create client with custom API URL", func() {
			client := mcp.NewChatClient(server, "test-api-key", "https://custom.api.com/v1", "gpt-4o", false, "")
			Expect(client).ToNot(BeNil())
		})

		It("should create client with debug mode enabled", func() {
			client := mcp.NewChatClient(server, "test-api-key", "", "gpt-4o", true, "")
			Expect(client).ToNot(BeNil())
		})

		It("should panic when no API key is provided", func() {
			os.Unsetenv("OPENAI_API_KEY")
			Expect(func() {
				mcp.NewChatClient(server, "", "", "gpt-4o", false, "")
			}).To(Panic())
		})
	})

	Describe("ChatClient initialization", func() {
		It("should initialize with tool registry from server", func() {
			os.Setenv("OPENAI_API_KEY", "test-api-key")
			client := mcp.NewChatClient(server, "", "", "gpt-4o", false, "")
			Expect(client).ToNot(BeNil())
		})

		It("should initialize with different models", func() {
			os.Setenv("OPENAI_API_KEY", "test-api-key")
			models := []string{"gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"}
			for _, model := range models {
				client := mcp.NewChatClient(server, "", "", model, false, "")
				Expect(client).ToNot(BeNil())
			}
		})
	})
})
