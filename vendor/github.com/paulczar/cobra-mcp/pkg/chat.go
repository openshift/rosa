package cobra_mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

// ChatClient provides OpenAI-compatible chat interface with tool calling
type ChatClient struct {
	client        openai.Client
	model         shared.ChatModel
	toolRegistry  *ToolRegistry
	messages      []openai.ChatCompletionMessageParamUnion
	debug         bool
	systemMessage string
}

// NewChatClient creates a new ChatClient
func NewChatClient(server *Server, config *ChatConfig) (*ChatClient, error) {
	// Get API key from config or environment
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key not provided (use --api-key flag or OPENAI_API_KEY env var)")
	}

	// Set model
	modelStr := config.Model
	if modelStr == "" {
		modelStr = "gpt-4"
	}
	model := shared.ChatModel(modelStr)

	// Create OpenAI client
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if config.APIURL != "" {
		opts = append(opts, option.WithBaseURL(config.APIURL))
	}

	client := openai.NewClient(opts...)

	// Get system message
	systemMessage := config.SystemMessage
	if systemMessage == "" && config.SystemMessageFile != "" {
		content, err := os.ReadFile(config.SystemMessageFile)
		if err != nil {
			return nil, fmt.Errorf("error reading system message file: %w", err)
		}
		systemMessage = string(content)
	}

	// Generate system message if not provided
	if systemMessage == "" {
		systemMessageConfig := &SystemMessageConfig{
			CLIName:           server.rootCmd.Name(),
			CLIDescription:    server.rootCmd.Short,
			ToolPrefix:        server.config.ToolPrefix,
			DangerousCommands: server.config.DangerousCommands,
		}
		systemMessage = GenerateSystemMessageFromRegistry(server.toolRegistry, server.rootCmd, systemMessageConfig)
	}

	// Initialize messages with system message
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemMessage),
	}

	return &ChatClient{
		client:        client,
		model:         model,
		toolRegistry:  server.toolRegistry,
		messages:      messages,
		debug:         config.Debug,
		systemMessage: systemMessage,
	}, nil
}

// RunChatLoop runs an interactive chat loop
func (cc *ChatClient) RunChatLoop() error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Chat client started. Type 'exit' or 'quit' to exit.")
	if cc.debug {
		fmt.Printf("Debug: Using model: %s\n", cc.model)
	}
	fmt.Println()

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		if userInput == "exit" || userInput == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if err := cc.ProcessMessage(userInput); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
	}

	return scanner.Err()
}

// ProcessMessage processes a single user message
func (cc *ChatClient) ProcessMessage(userInput string) error {
	// Add user message
	cc.messages = append(cc.messages, openai.UserMessage(userInput))

	// Get tools
	tools := cc.getTools()

	// Call OpenAI API
	ctx := context.Background()
	maxIterations := 10
	iteration := 0

	for iteration < maxIterations {
		req := openai.ChatCompletionNewParams{
			Model:    cc.model,
			Messages: cc.messages,
		}

		if len(tools) > 0 {
			req.Tools = tools
		}

		if cc.debug {
			fmt.Printf("Debug: Calling API with model: %s\n", cc.model)
		}

		resp, err := cc.client.Chat.Completions.New(ctx, req)
		if err != nil {
			// Check for rate limit errors and provide helpful message
			errStr := err.Error()
			if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "quota") {
				return fmt.Errorf("OpenAI API rate limit/quota exceeded: %w\nPlease check your OpenAI account billing and quota limits at https://platform.openai.com/account/billing", err)
			}
			return fmt.Errorf("OpenAI API error: %w", err)
		}

		if len(resp.Choices) == 0 {
			return fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]
		message := choice.Message

		// Convert message to param and add to messages
		assistantMsg := message.ToParam()
		cc.messages = append(cc.messages, assistantMsg)

		// Check if there are tool calls
		if len(message.ToolCalls) == 0 {
			// No tool calls, display response
			if message.Content != "" {
				fmt.Printf("Assistant: %s\n\n", message.Content)
			}
			return nil
		}

		// Handle tool calls
		for _, toolCall := range message.ToolCalls {
			// Parse arguments
			var arguments map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
				return fmt.Errorf("error parsing tool arguments: %w", err)
			}

			// Show debug information if debug flag is enabled
			if cc.debug {
				cc.showDebugInfo(toolCall.Function.Name, arguments)
			}

			// Call tool
			result, err := cc.toolRegistry.CallTool(toolCall.Function.Name, arguments)
			if err != nil {
				if cc.debug {
					fmt.Printf("Debug: Tool call failed: %v\n", err)
				}
				return fmt.Errorf("error calling tool: %w", err)
			}

			if cc.debug {
				resultJSON, _ := json.MarshalIndent(result, "", "  ")
				fmt.Printf("Debug: Tool result:\n%s\n\n", string(resultJSON))
			}

			// Convert result to JSON string
			resultJSON, err := json.Marshal(result)
			if err != nil {
				return fmt.Errorf("error marshaling tool result: %w", err)
			}

			// Add tool result message
			cc.messages = append(cc.messages, openai.ToolMessage(string(resultJSON), toolCall.ID))
		}

		iteration++
	}

	return fmt.Errorf("max iterations reached")
}

// getTools converts ToolRegistry tools to OpenAI tool format
func (cc *ChatClient) getTools() []openai.ChatCompletionToolParam {
	toolDefs := cc.toolRegistry.GetHierarchicalTools()
	tools := make([]openai.ChatCompletionToolParam, 0, len(toolDefs))

	for _, toolDef := range toolDefs {
		toolName := toolDef["name"].(string)
		description := toolDef["description"].(string)
		inputSchema := toolDef["inputSchema"].(map[string]interface{})

		tool := openai.ChatCompletionToolParam{
			Type: "function",
			Function: shared.FunctionDefinitionParam{
				Name:        toolName,
				Description: param.NewOpt(description),
				Parameters:  inputSchema,
			},
		}

		tools = append(tools, tool)
	}

	return tools
}

// ProcessStdin reads messages from stdin and processes them
func (cc *ChatClient) ProcessStdin() error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			if err := cc.ProcessMessage(line); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

// showDebugInfo displays debug information about a tool call
func (cc *ChatClient) showDebugInfo(toolName string, arguments map[string]interface{}) {
	fmt.Println("\n=== DEBUG: Tool Call ===")
	fmt.Printf("Tool: %s\n", toolName)

	// Get expected parameters from tool schema
	toolDefs := cc.toolRegistry.GetHierarchicalTools()
	var expectedSchema map[string]interface{}
	for _, toolDef := range toolDefs {
		if name, ok := toolDef["name"].(string); ok && name == toolName {
			if schema, ok := toolDef["inputSchema"].(map[string]interface{}); ok {
				expectedSchema = schema
				break
			}
		}
	}

	if expectedSchema != nil {
		fmt.Println("\nExpected Parameters:")
		if properties, ok := expectedSchema["properties"].(map[string]interface{}); ok {
			requiredSet := make(map[string]bool)
			if required, ok := expectedSchema["required"].([]interface{}); ok {
				for _, r := range required {
					if reqName, ok := r.(string); ok {
						requiredSet[reqName] = true
					}
				}
			}
			for paramName, paramDef := range properties {
				paramMap := paramDef.(map[string]interface{})
				paramType := paramMap["type"]
				paramDesc := ""
				if desc, ok := paramMap["description"].(string); ok {
					paramDesc = desc
				}
				required := ""
				if requiredSet[paramName] {
					required = " [REQUIRED]"
				}
				fmt.Printf("  - %s (%v)%s: %s\n", paramName, paramType, required, paramDesc)
			}
		}
		if required, ok := expectedSchema["required"].([]interface{}); ok && len(required) > 0 {
			fmt.Printf("\nRequired parameters: %v\n", required)
		}
	}

	fmt.Println("\nActual Parameters Being Passed:")
	if len(arguments) == 0 {
		fmt.Println("  (none)")
	} else {
		argsJSON, _ := json.MarshalIndent(arguments, "  ", "  ")
		fmt.Println(string(argsJSON))
	}

	// Check for missing required parameters
	if expectedSchema != nil {
		if required, ok := expectedSchema["required"].([]interface{}); ok {
			missing := []string{}
			for _, req := range required {
				if reqName, ok := req.(string); ok {
					if _, found := arguments[reqName]; !found {
						missing = append(missing, reqName)
					}
				}
			}
			if len(missing) > 0 {
				fmt.Printf("\n⚠️  Missing required parameters: %v\n", missing)
			} else {
				fmt.Println("\n✓ All required parameters provided")
			}
		}
	}

	fmt.Println("======================")
}
