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

package chat

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/color"
	"github.com/openshift/rosa/pkg/mcp"
	"github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	apiKey            string
	apiURL            string
	model             string
	debug             bool
	message           string
	stdin             bool
	systemMessageFile string
	showSystemMessage bool
}

var Cmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an AI chat interface with ROSA CLI tools",
	Long: `Start an interactive chat interface powered by an AI assistant.
The AI has access to all ROSA CLI commands through the MCP tool registry,
allowing you to interact with ROSA using natural language.

Requires an OpenAI-compatible API key. Set OPENAI_API_KEY environment variable
or use --api-key flag.`,
	Example: `  # Using default OpenAI API
  export OPENAI_API_KEY=sk-...
  rosa mcp chat

  # Using custom OpenAI-compatible endpoint
  rosa mcp chat --api-url https://api.anthropic.com/v1 --model claude-3-opus

  # Using localhost model server
  rosa mcp chat --api-url http://localhost:8080/v1 --model local-model

  # Non-interactive mode: send a single message
  rosa mcp chat --message "list all clusters"

  # Read message from stdin
  echo "who am I logged in as?" | rosa mcp chat --stdin

  # View the default system message
  rosa mcp chat --show-system-message

  # Use a custom system message from a file
  rosa mcp chat --system-message-file ./custom-system-message.txt`,
	RunE: runE,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.apiKey,
		"api-key",
		"",
		"API key for OpenAI-compatible service (defaults to OPENAI_API_KEY env var)",
	)
	flags.StringVar(
		&args.apiURL,
		"api-url",
		"",
		"Base URL for OpenAI-compatible API (defaults to https://api.openai.com/v1)",
	)
	flags.StringVar(
		&args.model,
		"model",
		"gpt-4o",
		"Model to use for chat completions (gpt-4o recommended for larger context, gpt-4-turbo also supports more tokens than gpt-4)",
	)
	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug output for troubleshooting",
	)
	flags.StringVar(
		&args.message,
		"message",
		"",
		"Non-interactive mode: send a single message and exit",
	)
	flags.BoolVar(
		&args.stdin,
		"stdin",
		false,
		"Read message from stdin instead of interactive mode",
	)
	flags.StringVar(
		&args.systemMessageFile,
		"system-message-file",
		"",
		"Path to a file containing a custom system message to override the default",
	)
	flags.BoolVar(
		&args.showSystemMessage,
		"show-system-message",
		false,
		"Display the default system message and exit",
	)
}

func runE(cmd *cobra.Command, _ []string) error {
	rprtr := reporter.CreateReporter()

	// Show system message if requested
	if args.showSystemMessage {
		fmt.Println(mcp.GetDefaultSystemMessage())
		return nil
	}

	// Get API key from flag or environment variable
	apiKey := args.apiKey

	if apiKey == "" {
		envKey := os.Getenv("OPENAI_API_KEY")
		if envKey == "" {
			if args.debug {
				// Debug: Check if variable exists but is empty, or doesn't exist at all
				allEnvVars := os.Environ()
				found := false
				for _, envVar := range allEnvVars {
					if len(envVar) >= 16 && envVar[:16] == "OPENAI_API_KEY=" {
						found = true
						if len(envVar) == 16 {
							rprtr.Debugf("OPENAI_API_KEY exists but is empty")
						} else {
							rprtr.Debugf("OPENAI_API_KEY exists with length %d (first 10 chars: %s...)", len(envVar)-16, envVar[16:26])
						}
						break
					}
				}
				if !found {
					rprtr.Debugf("OPENAI_API_KEY not found in environment variables")
					rprtr.Debugf("Total env vars: %d", len(allEnvVars))
				}
			}

			return rprtr.Errorf("API key required. Set OPENAI_API_KEY environment variable or use --api-key flag")
		}
		apiKey = envKey
	}

	// Validate API key is not empty
	if apiKey == "" {
		return rprtr.Errorf("API key is empty. Please provide a valid API key")
	}

	// Create root command with all commands registered
	// We need to do this here to avoid circular dependency
	rootCmd := &cobra.Command{
		Use:   "rosa",
		Short: "Command line tool for ROSA.",
	}

	// Initialize flags
	fs := rootCmd.PersistentFlags()
	color.AddFlag(rootCmd)
	arguments.AddDebugFlag(fs)

	// Register all commands using helper to avoid circular dependency
	registerCommandsForChat(rootCmd)

	// Create MCP server to get access to tool and resource registries
	mcpServer := mcp.NewServer(rootCmd)

	// Create chat client
	// API key should be validated above, but double-check
	if apiKey == "" {
		return rprtr.Errorf("API key is empty after validation. This should not happen")
	}

	// Read system message from file if provided, otherwise use default
	systemMessage := ""
	if args.systemMessageFile != "" {
		content, err := os.ReadFile(args.systemMessageFile)
		if err != nil {
			return rprtr.Errorf("Error reading system message file: %v", err)
		}
		systemMessage = string(content)
	}

	chatClient := mcp.NewChatClient(mcpServer, apiKey, args.apiURL, args.model, args.debug, systemMessage)

	// Determine input source for non-interactive mode
	var userInput string
	if args.message != "" {
		// Non-interactive mode with --message flag
		userInput = args.message
	} else if args.stdin {
		// Read from stdin
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return rprtr.Errorf("Error reading from stdin: %v", err)
		}
		userInput = strings.Join(lines, "\n")
	}

	if userInput != "" {
		// Non-interactive mode: process single message and exit
		if err := chatClient.ProcessMessage(userInput); err != nil {
			return rprtr.Errorf("Error processing message: %v", err)
		}
		return nil
	}

	// Interactive mode: start REPL loop
	if err := chatClient.RunChatLoop(); err != nil {
		return rprtr.Errorf("Error running chat: %v", err)
	}
	return nil
}
