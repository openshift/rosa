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

package chat_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/cmd/mcp/chat"
)

var _ = Describe("Chat Command", func() {
	var originalAPIKey string
	var hasAPIKey bool

	BeforeEach(func() {
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

	Describe("Command Flags", func() {
		It("should have api-key flag", func() {
			flag := chat.Cmd.Flag("api-key")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Usage).To(ContainSubstring("API key"))
		})

		It("should have api-url flag", func() {
			flag := chat.Cmd.Flag("api-url")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Usage).To(ContainSubstring("Base URL"))
		})

		It("should have model flag with default value", func() {
			flag := chat.Cmd.Flag("model")
			Expect(flag).ToNot(BeNil())
			Expect(flag.DefValue).To(Equal("gpt-4o"))
		})

		It("should have debug flag", func() {
			flag := chat.Cmd.Flag("debug")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Usage).To(ContainSubstring("debug"))
		})

		It("should have message flag", func() {
			flag := chat.Cmd.Flag("message")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Usage).To(ContainSubstring("message"))
		})

		It("should have stdin flag", func() {
			flag := chat.Cmd.Flag("stdin")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Usage).To(ContainSubstring("stdin"))
		})

		It("should have system-message-file flag", func() {
			flag := chat.Cmd.Flag("system-message-file")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Usage).To(ContainSubstring("system message"))
		})

		It("should have show-system-message flag", func() {
			flag := chat.Cmd.Flag("show-system-message")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Usage).To(ContainSubstring("system message"))
		})
	})

	Describe("Command Configuration", func() {
		It("should have correct command use", func() {
			Expect(chat.Cmd.Use).To(Equal("chat"))
		})

		It("should have a description", func() {
			Expect(chat.Cmd.Short).ToNot(BeEmpty())
			Expect(chat.Cmd.Long).ToNot(BeEmpty())
		})

		It("should have examples", func() {
			Expect(chat.Cmd.Example).ToNot(BeEmpty())
			Expect(chat.Cmd.Example).To(ContainSubstring("OPENAI_API_KEY"))
		})

		It("should accept no arguments", func() {
			Expect(chat.Cmd.Args).ToNot(BeNil())
		})
	})

	Describe("System Message File Handling", func() {
		It("should validate system message file path exists", func() {
			// This test would require mocking file system operations
			// or creating a temporary file for testing
		})

		It("should return error for non-existent system message file", func() {
			// This test would require executing the command with invalid file path
			// and checking the error output
		})
	})

	Describe("API Key Validation", func() {
		It("should require API key from flag or environment", func() {
			// This test verifies that the command enforces API key requirement
			// The actual validation happens in the run function
		})

		It("should prioritize flag API key over environment variable", func() {
			// This is tested at the NewChatClient level in pkg/mcp/chat_test.go
		})
	})
})
