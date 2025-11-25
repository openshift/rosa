package cobra_mcp

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// getHelpText gets concise help text for a Cobra command
func getHelpText(cmd *cobra.Command) string {
	var parts []string

	// Add description
	if cmd.Short != "" {
		parts = append(parts, cmd.Short)
	}
	if cmd.Long != "" && cmd.Long != cmd.Short {
		parts = append(parts, cmd.Long)
	}

	// Add usage
	if cmd.Use != "" {
		parts = append(parts, fmt.Sprintf("Usage: %s", cmd.Use))
	}

	// Add example if available
	if cmd.Example != "" {
		parts = append(parts, fmt.Sprintf("Example: %s", cmd.Example))
	}

	return strings.Join(parts, ". ")
}

// getFullHelpText gets the full help text for a Cobra command (for root CLI)
func getFullHelpText(cmd *cobra.Command) string {
	// Save original output
	originalOut := cmd.OutOrStdout()
	originalErr := cmd.ErrOrStderr()

	// Capture help output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Generate help text
	cmd.Help()

	// Restore original output
	cmd.SetOut(originalOut)
	cmd.SetErr(originalErr)

	return buf.String()
}

// GenerateSystemMessage generates a comprehensive system message for the chat client
func GenerateSystemMessage(config *SystemMessageConfig) string {
	var parts []string

	// Header with CLI help information
	if config.CLIDescription != "" {
		parts = append(parts, fmt.Sprintf("You are a helpful assistant for managing %s resources. %s", config.CLIName, config.CLIDescription))
	} else {
		parts = append(parts, fmt.Sprintf("You are a helpful assistant for managing %s resources.", config.CLIName))
	}

	// Add CLI help text if available
	if config.CLIHelp != "" {
		parts = append(parts, "")
		parts = append(parts, "CLI OVERVIEW:")
		parts = append(parts, config.CLIHelp)
		parts = append(parts, "")
	}

	parts = append(parts, "")

	// Tool Usage
	parts = append(parts, "TOOL USAGE:")
	parts = append(parts, fmt.Sprintf("- %s tools use a hierarchical structure. Hierarchical tools (%s_list, %s_describe, %s_create, etc.) REQUIRE a 'resource' parameter.",
		config.CLIName, config.ToolPrefix, config.ToolPrefix, config.ToolPrefix))
	parts = append(parts, "- CRITICAL: ALWAYS pass command parameters as 'flags' in the flags object, NOT as positional arguments in 'args'.")
	parts = append(parts, "- Example: To create a cluster with name='bacon', region='us-east-1', size='Small':")
	parts = append(parts, "  Use: flags={'name': 'bacon', 'region': 'us-east-1', 'size': 'Small'}, NOT args=['bacon', 'us-east-1', 'Small']")
	parts = append(parts, "- Flag names in the flags object should NOT include the '--' prefix (e.g., use 'name' not '--name').")
	parts = append(parts, "- PARAMETER DISCOVERY: When a user asks you to perform an action, analyze their request to extract parameter values.")
	parts = append(parts, "  - Look for explicit mentions of parameter values (e.g., 'create a cluster named my-cluster' → flags={'name': 'my-cluster'})")
	parts = append(parts, "  - Infer values from context when reasonable (e.g., 'create a large cluster' → flags={'size': 'Large'})")
	parts = append(parts, "  - Use help tool to discover available parameters and their requirements before making tool calls")
	parts = append(parts, "- MISSING PARAMETERS: If required parameters are missing or unclear, DO NOT proceed with the tool call.")
	parts = append(parts, "  Instead, present a numbered list of required parameters and ask the user for each value:")
	parts = append(parts, "  Example: 'To create a cluster, I need the following information:'")
	parts = append(parts, "  1. Cluster name (what would you like to name it?)")
	parts = append(parts, "  2. Region (which region should it be created in?)")
	parts = append(parts, "  3. Size (Small, Medium, or Large - which size do you need?)")
	parts = append(parts, "- When the user provides numbered responses, map them to the correct flag names and put them in the 'flags' object.")
	parts = append(parts, "- Only proceed with tool calls once you have all required parameters or the user has explicitly confirmed values.")

	// Add resource-specific guidance if available
	if len(config.AvailableResources) > 0 {
		parts = append(parts, "- For resource-related operations, ALWAYS include the appropriate resource identifier in flags.")
	}

	parts = append(parts, fmt.Sprintf("- Standalone tools (%s_whoami, %s_version, etc.) don't need a 'resource' parameter.", config.ToolPrefix, config.ToolPrefix))
	parts = append(parts, "")

	// Available Actions with help text
	if len(config.AvailableActions) > 0 {
		parts = append(parts, "AVAILABLE ACTIONS:")
		for _, action := range config.AvailableActions {
			resources := config.AvailableResources[action]
			actionHelp := config.CommandHelp[action]

			if len(resources) > 0 {
				line := fmt.Sprintf("- %s_%s: Available resources: %s",
					config.ToolPrefix, action, strings.Join(resources, ", "))
				parts = append(parts, line)

				// Add help text for each resource under this action
				if len(actionHelp) > 0 {
					for resource, help := range actionHelp {
						if help != "" && resource != "" {
							// Clean up help text - take first line or first sentence
							helpLines := strings.Split(help, "\n")
							helpSummary := strings.TrimSpace(helpLines[0])
							if len(helpSummary) > 200 {
								// Truncate long help text
								helpSummary = helpSummary[:200] + "..."
							}
							parts = append(parts, fmt.Sprintf("  - %s_%s with resource='%s': %s",
								config.ToolPrefix, action, resource, helpSummary))
						}
					}
				}
			} else {
				line := fmt.Sprintf("- %s_%s: Standalone command", config.ToolPrefix, action)
				parts = append(parts, line)

				// Add help text for standalone command
				if len(actionHelp) > 0 {
					// For standalone commands, use the first (and likely only) help entry
					for _, help := range actionHelp {
						if help != "" {
							// Clean up help text - take first line or first sentence
							helpLines := strings.Split(help, "\n")
							helpSummary := strings.TrimSpace(helpLines[0])
							if len(helpSummary) > 200 {
								// Truncate long help text
								helpSummary = helpSummary[:200] + "..."
							}
							parts = append(parts, fmt.Sprintf("  %s", helpSummary))
						}
						break
					}
				}
			}
		}
		parts = append(parts, "")
	}

	// Common Patterns
	if len(config.CommonPatterns) > 0 {
		parts = append(parts, "COMMON PATTERNS:")
		for _, pattern := range config.CommonPatterns {
			parts = append(parts, fmt.Sprintf("- %s", pattern))
		}
		parts = append(parts, "")
	} else {
		// Generate default patterns from available resources
		parts = append(parts, "COMMON PATTERNS:")
		for action, resources := range config.AvailableResources {
			if len(resources) > 0 {
				exampleResource := resources[0]
				switch action {
				case "list":
					parts = append(parts, fmt.Sprintf("- List %s: %s_list with resource='%s'",
						exampleResource, config.ToolPrefix, exampleResource))
				case "describe":
					parts = append(parts, fmt.Sprintf("- Describe %s: %s_describe with resource='%s' and flags={'name': 'resource-name'}",
						exampleResource, config.ToolPrefix, exampleResource))
				case "create":
					parts = append(parts, fmt.Sprintf("- Create %s: %s_create with resource='%s' and flags={'name': 'resource-name', ...}",
						exampleResource, config.ToolPrefix, exampleResource))
				case "delete":
					parts = append(parts, fmt.Sprintf("- Delete %s: %s_delete with resource='%s' and flags={'name': 'resource-name'}",
						exampleResource, config.ToolPrefix, exampleResource))
				}
			}
		}
		parts = append(parts, "")
	}

	// Output Format
	parts = append(parts, "OUTPUT FORMAT:")
	parts = append(parts, "- All commands return JSON output automatically. Parse and present results clearly to the user.")
	parts = append(parts, "- When listing resources, summarize key information (name, ID, status) in a readable format.")
	parts = append(parts, "- When describing resources, present all relevant details in an organized manner.")
	parts = append(parts, "")

	// Error Handling
	parts = append(parts, "ERROR HANDLING:")
	parts = append(parts, fmt.Sprintf("- If unsure about command syntax or available options, use %s_help with the command path.", config.ToolPrefix))
	parts = append(parts, "- When using help, interpret the help response and explain it to the user in plain, conversational language. Don't just copy the help text verbatim - use it to provide clear, actionable guidance.")
	parts = append(parts, "- Help responses contain structured information about commands, resources, and examples. Use this information to guide users on what they can do and how to do it.")
	parts = append(parts, "- IMPORTANT: When explaining how to use tools, tell users what they can ask YOU (the chat assistant) in natural language. NEVER show users the internal JSON tool call structure. Instead, explain what they can say to you, like 'create a cluster' or 'list all clusters'.")
	parts = append(parts, "- Example: Instead of showing JSON, say 'You can create a cluster by asking me to create one. Just tell me what you'd like to create and any options you need.'")
	parts = append(parts, "- If a tool call fails, check the error message and suggest using help if needed.")
	parts = append(parts, "- Always validate required parameters before making tool calls.")
	parts = append(parts, "")

	// Debug Mode
	parts = append(parts, "DEBUG MODE:")
	parts = append(parts, "- If the user asks for 'debug mode', 'debugging', 'show debug info', or similar, enable debug mode.")
	parts = append(parts, "- When debug mode is enabled, you MUST show debug information BEFORE EVERY tool call.")
	parts = append(parts, "- Debug output MUST include the following information in a clearly formatted way:")
	parts = append(parts, "  === DEBUG: Tool Call ===")
	parts = append(parts, "  Tool: <tool_name>")
	parts = append(parts, "  Expected Parameters:")
	parts = append(parts, "    - <parameter_name> (<type>) [REQUIRED/optional]: <description>")
	parts = append(parts, "  Actual Parameters Being Passed:")
	parts = append(parts, "    {")
	parts = append(parts, "      \"parameter\": \"value\",")
	parts = append(parts, "      ...")
	parts = append(parts, "    }")
	parts = append(parts, "  Missing Required Parameters: [list any missing]")
	parts = append(parts, "  ======================")
	parts = append(parts, "- To get expected parameters, use the help tool (e.g., help list clusters) before making the tool call.")
	parts = append(parts, "- Debug mode persists until the user explicitly disables it or asks you to stop showing debug info.")
	parts = append(parts, "- This is especially useful when troubleshooting failures - it helps identify parameter mismatches or missing required fields.")
	parts = append(parts, "- After showing debug information, proceed with the tool call if all required parameters are present.")
	parts = append(parts, "")

	// Dangerous Commands
	if len(config.DangerousCommands) > 0 {
		parts = append(parts, "DANGEROUS COMMANDS:")
		parts = append(parts, "The following commands are marked as potentially dangerous and require explicit confirmation:")
		for _, cmd := range config.DangerousCommands {
			parts = append(parts, fmt.Sprintf("  - %s", cmd))
		}
		parts = append(parts, "")
		parts = append(parts, "CRITICAL SAFETY PROTOCOL:")
		parts = append(parts, "- Before executing ANY command that matches the dangerous commands list above, you MUST:")
		parts = append(parts, "  1. Restate the exact action you are about to perform")
		parts = append(parts, "  2. Clearly identify what resource(s) will be affected")
		parts = append(parts, "  3. Explicitly ask the user for confirmation (e.g., 'Are you sure you want to delete cluster X?')")
		parts = append(parts, "  4. DO NOT proceed with the tool call until the user explicitly confirms")
		parts = append(parts, "- Example: If user asks to 'delete cluster my-cluster', respond with:")
		parts = append(parts, "  'I'm about to delete the cluster named 'my-cluster'. This action is destructive and cannot be undone. Are you sure you want to proceed?'")
		parts = append(parts, "  Wait for explicit confirmation before calling the delete tool.")
		parts = append(parts, "")
	}

	// General Dangerous Command Detection
	parts = append(parts, "DANGEROUS COMMAND DETECTION:")
	parts = append(parts, "- Even if a command is NOT in the dangerous commands list, you should evaluate it for dangerousness.")
	parts = append(parts, "- Commands that are typically dangerous include:")
	parts = append(parts, "  - Actions: delete, destroy, remove, drop, kill, terminate, force")
	parts = append(parts, "  - Operations that modify or remove data, resources, or configurations")
	parts = append(parts, "  - Operations that cannot be easily undone")
	parts = append(parts, "- When you detect a potentially dangerous command:")
	parts = append(parts, "  1. Restate what you understand the user wants to do")
	parts = append(parts, "  2. Explain the potential consequences")
	parts = append(parts, "  3. Ask for explicit confirmation before proceeding")
	parts = append(parts, "- When in doubt, err on the side of caution and ask for confirmation.")
	parts = append(parts, "")

	// Safety Requirements
	if len(config.SafetyRequirements) > 0 {
		parts = append(parts, "SAFETY REQUIREMENTS:")
		for _, req := range config.SafetyRequirements {
			parts = append(parts, fmt.Sprintf("- %s", req))
		}
		parts = append(parts, "")
	}

	// Best Practices
	parts = append(parts, "BEST PRACTICES:")
	parts = append(parts, "- Always confirm before executing destructive operations (delete, remove, etc.).")
	parts = append(parts, "- Use list commands to verify resource existence before operations.")
	parts = append(parts, "- Provide clear, concise responses to user queries.")
	parts = append(parts, "- When showing command results, format them in a user-friendly way.")

	if len(config.CustomInstructions) > 0 {
		for _, instruction := range config.CustomInstructions {
			parts = append(parts, fmt.Sprintf("- %s", instruction))
		}
	}
	parts = append(parts, "")

	return strings.Join(parts, "\n")
}

// GenerateSystemMessageFromRegistry generates a system message from a ToolRegistry
func GenerateSystemMessageFromRegistry(registry *ToolRegistry, rootCmd *cobra.Command, config *SystemMessageConfig) string {
	if config == nil {
		config = &SystemMessageConfig{}
	}

	// Fill in defaults from registry
	if config.CLIName == "" {
		config.CLIName = rootCmd.Name()
	}
	if config.CLIDescription == "" {
		config.CLIDescription = rootCmd.Short
	}
	if config.ToolPrefix == "" {
		config.ToolPrefix = registry.toolPrefix
	}

	// Get root command help
	if config.CLIHelp == "" {
		config.CLIHelp = getFullHelpText(rootCmd)
	}

	// Discover available actions and resources, and collect help text
	actions := make(map[string]bool)
	resources := make(map[string][]string)
	commandHelp := make(map[string]map[string]string)

	commands := registry.executor.GetAllCommands()
	for _, cmdInfo := range commands {
		if len(cmdInfo.Path) > 0 {
			action := registry.detectAction(cmdInfo.Path)
			if action != "" {
				actions[action] = true

				// Find the actual Cobra command to get help text
				// cmdInfo.Path contains the full path, so FindCommand should return the leaf command
				cobraCmd, _, err := registry.executor.FindCommand(cmdInfo.Path)
				if err == nil && cobraCmd != nil {
					// Get concise help text for this command (description + usage)
					helpText := getHelpText(cobraCmd)

					if len(cmdInfo.Path) > 1 {
						// Hierarchical command (action + resource)
						resource := strings.ToLower(cmdInfo.Path[1])
						resources[action] = append(resources[action], resource)

						// Store help text for this action+resource combination
						if commandHelp[action] == nil {
							commandHelp[action] = make(map[string]string)
						}
						// Always store/update help text for the resource
						commandHelp[action][resource] = helpText
					} else {
						// Standalone command (no resource)
						if commandHelp[action] == nil {
							commandHelp[action] = make(map[string]string)
						}
						commandHelp[action][""] = helpText
					}
				}
			}
		}
	}

	// Convert to slices
	config.AvailableActions = []string{}
	for action := range actions {
		config.AvailableActions = append(config.AvailableActions, action)
	}

	// Deduplicate resources
	for action := range resources {
		seen := make(map[string]bool)
		unique := []string{}
		for _, r := range resources[action] {
			if !seen[r] {
				seen[r] = true
				unique = append(unique, r)
			}
		}
		resources[action] = unique
	}

	if config.AvailableResources == nil {
		config.AvailableResources = resources
	} else {
		// Merge with existing
		for action, res := range resources {
			config.AvailableResources[action] = res
		}
	}

	// Set command help
	if config.CommandHelp == nil {
		config.CommandHelp = commandHelp
	} else {
		// Merge with existing
		for action, helpMap := range commandHelp {
			if config.CommandHelp[action] == nil {
				config.CommandHelp[action] = make(map[string]string)
			}
			for resource, help := range helpMap {
				config.CommandHelp[action][resource] = help
			}
		}
	}

	return GenerateSystemMessage(config)
}
