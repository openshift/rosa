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

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"
)

// ChatClient wraps the OpenAI client and MCP server for chat functionality
type ChatClient struct {
	client       openai.Client
	model        shared.ChatModel
	toolRegistry *ToolRegistry
	messages     []openai.ChatCompletionMessageParamUnion
	debug        bool
}

// GetDefaultSystemMessage returns the default system message used for the chat assistant
func GetDefaultSystemMessage() string {
	return `You are a helpful assistant for managing Red Hat OpenShift Service on AWS (ROSA) clusters.

TOOL USAGE:
- ROSA tools use a hierarchical structure. Hierarchical tools (rosa_list, rosa_describe, rosa_create, etc.) REQUIRE a 'resource' parameter.
- ALWAYS use the 'flags' parameter with flag names (without '--' prefix) for command options.
- Cluster identifiers: You can use cluster name OR cluster ID - both work. Cluster names are more user-friendly.
- For cluster-related operations, ALWAYS include flags={'cluster': 'cluster-name-or-id'}.
- Standalone tools (rosa_whoami, rosa_version, rosa_login, rosa_logout) don't need a 'resource' parameter.

COMMON PATTERNS:
- List clusters: rosa_list with resource='clusters'
- Describe cluster: rosa_describe with resource='cluster' and flags={'cluster': 'name'}
- List machine pools: rosa_list with resource='machinepools' and flags={'cluster': 'name'}
- List IDPs: rosa_list with resource='idps' and flags={'cluster': 'name'}
- Check identity: rosa_whoami (no parameters needed)

OUTPUT FORMAT:
- All commands return JSON output automatically. Parse and present results clearly to the user.
- When listing resources, summarize key information (name, ID, status) in a readable format.
- For detailed operations, show relevant details but keep output concise.

ERROR HANDLING:
- If unsure about command syntax or available options, use rosa_help with the command path (e.g., 'create cluster').
- If a tool call fails, check the error message and suggest using rosa_help if needed.
- Always provide actionable guidance when errors occur.

DESTRUCTIVE OPERATIONS - CRITICAL SAFETY REQUIREMENT:
- Destructive commands include: rosa_delete, rosa_revoke, rosa_uninstall, rosa_logout (clears credentials).
- NEVER execute destructive commands without explicit user confirmation.
- Before executing ANY destructive command, you MUST:
  1. Clearly explain what will be deleted/revoked/uninstalled.
  2. Show the specific resource(s) that will be affected (name, ID, etc.).
  3. Ask the user to explicitly confirm: "Are you sure you want to [action] [resource]?"
  4. Wait for explicit confirmation (e.g., "yes", "confirm", "proceed") before calling the tool.
- If the user's intent is unclear, ask for clarification rather than assuming.
- When confirming, restate what will happen: "I will now [action] [resource]. Proceeding..."

BEST PRACTICES:
- Before operating on a cluster, you may want to verify it exists by listing clusters first.
- When users mention a cluster by name, use that name directly in flags - no need to look up the ID.
- Be proactive: if listing something, also check and report the count/summary.
- Always explain what you're doing and why, especially for destructive operations.
- Provide clear, concise responses that help users understand their ROSA clusters and resources.`
}

// NewChatClient creates a new chat client
func NewChatClient(server *Server, apiKey string, apiURL string, model string, debug bool, customSystemMessage string) *ChatClient {
	opts := []option.RequestOption{}

	// Determine API key: flag takes precedence, then environment variable
	finalAPIKey := apiKey
	if finalAPIKey == "" {
		finalAPIKey = os.Getenv("OPENAI_API_KEY")
	}

	// Validate API key is provided (should be validated in cmd.go, but be defensive)
	if finalAPIKey == "" {
		// This shouldn't happen if cmd.go validation works, but guard against it
		panic("API key must be provided to NewChatClient")
	}

	// Always set API key explicitly
	// This ensures it's set even if DefaultClientOptions has issues
	opts = append(opts, option.WithAPIKey(finalAPIKey))

	// Set custom base URL if provided
	if apiURL != "" {
		opts = append(opts, option.WithBaseURL(apiURL))
	}

	// NewClient will also call DefaultClientOptions which reads OPENAI_API_KEY from env
	// but since we're explicitly setting it above, our option will be used
	client := openai.NewClient(opts...)

	// Initialize with system message (custom or default)
	var systemMessageText string
	if customSystemMessage != "" {
		systemMessageText = customSystemMessage
	} else {
		systemMessageText = GetDefaultSystemMessage()
	}
	systemMessage := openai.SystemMessage(systemMessageText)

	return &ChatClient{
		client:       client,
		model:        shared.ChatModel(model),
		toolRegistry: server.toolRegistry,
		messages:     []openai.ChatCompletionMessageParamUnion{systemMessage},
		debug:        debug,
	}
}

// convertMCPToolsToOpenAIFunctions converts MCP tool definitions to OpenAI function format
// With hierarchical tools, we have far fewer tools (12-15 vs 136), so no need for prioritization or limits
func (cc *ChatClient) convertMCPToolsToOpenAIFunctions() ([]openai.ChatCompletionToolUnionParam, error) {
	// Use hierarchical tools - already much smaller set (12-15 tools)
	tools := cc.toolRegistry.GetHierarchicalTools()

	functions := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))

	for _, toolDef := range tools {
		name, _ := toolDef["name"].(string)
		description, _ := toolDef["description"].(string)
		inputSchemaRaw, _ := toolDef["inputSchema"].(map[string]interface{})

		// Convert inputSchema to OpenAI function schema
		parameters := make(map[string]interface{})
		if inputSchemaRaw != nil {
			// Copy and fix properties - ensure arrays have items
			if props, ok := inputSchemaRaw["properties"].(map[string]interface{}); ok {
				fixedProps := make(map[string]interface{})
				for propName, propValue := range props {
					if propMap, ok := propValue.(map[string]interface{}); ok {
						// Copy the property
						fixedProp := make(map[string]interface{})
						for k, v := range propMap {
							fixedProp[k] = v
						}

						// If it's an array type, ensure it has items
						if propType, ok := fixedProp["type"].(string); ok && propType == "array" {
							if _, hasItems := fixedProp["items"]; !hasItems {
								// Default to string array for args
								fixedProp["items"] = map[string]interface{}{
									"type": "string",
								}
							}
						}

						fixedProps[propName] = fixedProp
					} else {
						// Not a map, copy as-is
						fixedProps[propName] = propValue
					}
				}
				parameters["properties"] = fixedProps
			}
			// Set type
			parameters["type"] = "object"
			// Copy required fields
			if req, ok := inputSchemaRaw["required"].([]interface{}); ok {
				parameters["required"] = req
			}
		} else {
			parameters["type"] = "object"
			parameters["properties"] = make(map[string]interface{})
		}

		functionDef := shared.FunctionDefinitionParam{
			Name:        name,
			Description: param.NewOpt(description),
			Parameters:  parameters,
		}

		functionTool := openai.ChatCompletionFunctionToolParam{
			Function: functionDef,
			Type:     constant.ValueOf[constant.Function](),
		}

		functions = append(functions, openai.ChatCompletionToolUnionParam{
			OfFunction: &functionTool,
		})
	}

	return functions, nil
}

// executeToolCall executes a tool call via MCP ToolRegistry
func (cc *ChatClient) executeToolCall(toolCall openai.ChatCompletionMessageToolCallUnion) (string, error) {
	// Extract function call from tool call
	toolCallAny := toolCall.AsAny()
	if toolCallAny == nil {
		return "", fmt.Errorf("tool call is empty")
	}

	functionCall, ok := toolCallAny.(openai.ChatCompletionMessageFunctionToolCall)
	if !ok {
		return "", fmt.Errorf("tool call is not a function call")
	}

	if functionCall.Function.Name == "" {
		return "", fmt.Errorf("tool call missing function name")
	}

	functionName := functionCall.Function.Name
	argumentsJSON := functionCall.Function.Arguments

	// Parse arguments JSON
	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsJSON), &arguments); err != nil {
		return "", fmt.Errorf("failed to unmarshal tool arguments: %w", err)
	}

	// Debug: Print tool call details
	if cc.debug {
		fmt.Printf("  [DEBUG] Tool: %s\n", functionName)
		fmt.Printf("  [DEBUG] Arguments JSON: %s\n", argumentsJSON)
		if argsPretty, err := json.MarshalIndent(arguments, "  [DEBUG] ", "  "); err == nil {
			fmt.Printf("  [DEBUG] Parsed arguments:\n%s\n", string(argsPretty))
		}
	}

	// Execute the tool via MCP ToolRegistry
	result, err := cc.toolRegistry.CallTool(functionName, arguments)
	if err != nil {
		errorMsg := fmt.Sprintf("Error executing tool %s: %v", functionName, err)
		if cc.debug {
			fmt.Printf("  [DEBUG] Error: %s\n", errorMsg)

			// Try to provide helpful suggestions for hierarchical tools
			if strings.HasPrefix(functionName, "rosa_") {
				action := strings.TrimPrefix(functionName, "rosa_")
				if !strings.Contains(action, "_") {
					// This is a hierarchical tool - check if resource is missing
					if resource, hasResource := arguments["resource"]; !hasResource || resource == nil || resource == "" {
						fmt.Printf("  [DEBUG] Hint: This tool requires a 'resource' parameter. Use rosa_help to see available resources.\n")
					}
				}
			}
		}

		return errorMsg, err
	}

	// Debug: Print result summary
	if cc.debug {
		if isError, _ := result["isError"].(bool); isError {
			fmt.Printf("  [DEBUG] Tool returned an error flag\n")
		}
		if content, ok := result["content"].([]map[string]interface{}); ok && len(content) > 0 {
			if text, ok := content[0]["text"].(string); ok {
				// Show first 200 chars of result
				preview := text
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				fmt.Printf("  [DEBUG] Result preview: %s\n", preview)
			}
		}
	}

	// Format result as JSON string for the AI
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("Error formatting tool result: %v", err), err
	}

	return string(resultJSON), nil
}

// ProcessMessage processes a single message non-interactively and prints the response
func (cc *ChatClient) ProcessMessage(userInput string) error {
	// Convert MCP tools to OpenAI functions
	functions, err := cc.convertMCPToolsToOpenAIFunctions()
	if err != nil {
		return fmt.Errorf("failed to convert tools to functions: %w", err)
	}

	userInput = strings.TrimSpace(userInput)
	if userInput == "" {
		return fmt.Errorf("message cannot be empty")
	}

	// Add user message
	userMessage := openai.UserMessage(userInput)
	cc.messages = append(cc.messages, userMessage)

	// Process conversation until we get a final response
	for {
		// Call OpenAI API
		params := openai.ChatCompletionNewParams{
			Model:    cc.model,
			Messages: cc.messages,
			Tools:    functions,
		}

		completion, err := cc.client.Chat.Completions.New(context.Background(), params)
		if err != nil {
			return fmt.Errorf("failed to get chat completion: %w", err)
		}

		if len(completion.Choices) == 0 {
			return fmt.Errorf("no choices in completion response")
		}

		choice := completion.Choices[0]
		message := choice.Message

		// Convert response message to param format and add to conversation
		assistantMessage := message.ToParam()
		cc.messages = append(cc.messages, assistantMessage)

		// Check if we need to execute tool calls
		if len(message.ToolCalls) > 0 {
			// Execute all tool calls
			for i, toolCall := range message.ToolCalls {
				// Extract function name for display
				toolCallAny := toolCall.AsAny()
				functionName := ""
				if functionCall, ok := toolCallAny.(openai.ChatCompletionMessageFunctionToolCall); ok {
					functionName = functionCall.Function.Name
				}

				fmt.Printf("[Executing: %s] (call %d/%d)\n", functionName, i+1, len(message.ToolCalls))
				result, err := cc.executeToolCall(toolCall)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}

				// Add tool result message
				toolMessage := openai.ToolMessage(result, toolCall.ID)
				cc.messages = append(cc.messages, toolMessage)
			}

			// Continue loop to get final response after tool execution
			continue
		}

		// No tool calls, display the response and exit
		if message.Content != "" {
			fmt.Println(message.Content)
		}

		return nil
	}
}

// RunChatLoop runs the main REPL chat loop
func (cc *ChatClient) RunChatLoop() error {
	// Convert MCP tools to OpenAI functions
	functions, err := cc.convertMCPToolsToOpenAIFunctions()
	if err != nil {
		return fmt.Errorf("failed to convert tools to functions: %w", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("ROSA AI Chat Assistant")
	fmt.Println("Type 'exit' or 'quit' to end the session, or Ctrl+C to interrupt")
	fmt.Println()

	for {
		// Check for interrupt signal
		select {
		case <-sigChan:
			fmt.Println("\n\nGoodbye!")
			return nil
		default:
		}

		// Prompt for user input
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		// Check for exit commands
		if userInput == "exit" || userInput == "quit" {
			fmt.Println("Goodbye!")
			return nil
		}

		// Add user message
		userMessage := openai.UserMessage(userInput)
		cc.messages = append(cc.messages, userMessage)

		// Continue conversation loop until we get a final response
		for {
			// Call OpenAI API
			params := openai.ChatCompletionNewParams{
				Model:    cc.model,
				Messages: cc.messages,
				Tools:    functions,
			}

			completion, err := cc.client.Chat.Completions.New(context.Background(), params)
			if err != nil {
				return fmt.Errorf("failed to get chat completion: %w", err)
			}

			if len(completion.Choices) == 0 {
				return fmt.Errorf("no choices in completion response")
			}

			choice := completion.Choices[0]
			message := choice.Message

			// Convert response message to param format and add to conversation
			assistantMessage := message.ToParam()
			cc.messages = append(cc.messages, assistantMessage)

			// Check if we need to execute tool calls
			if len(message.ToolCalls) > 0 {
				// Execute all tool calls
				for i, toolCall := range message.ToolCalls {
					// Extract function name for display
					toolCallAny := toolCall.AsAny()
					functionName := ""
					if functionCall, ok := toolCallAny.(openai.ChatCompletionMessageFunctionToolCall); ok {
						functionName = functionCall.Function.Name
					}

					fmt.Printf("\n[Executing: %s] (call %d/%d)\n", functionName, i+1, len(message.ToolCalls))
					result, err := cc.executeToolCall(toolCall)
					if err != nil {
						result = fmt.Sprintf("Error: %v", err)
					}

					// Add tool result message
					toolMessage := openai.ToolMessage(result, toolCall.ID)
					cc.messages = append(cc.messages, toolMessage)
				}

				// Continue loop to get final response after tool execution
				continue
			}

			// No tool calls, display the response
			if message.Content != "" {
				fmt.Printf("\nAssistant: %s\n\n", message.Content)
			}

			// Break out of inner loop, ready for next user input
			break
		}
	}

	return nil
}
