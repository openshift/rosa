package cobra_mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ToolRegistry converts discovered Cobra commands into hierarchical MCP tool definitions
type ToolRegistry struct {
	executor       *CommandExecutor
	rootCmd        *cobra.Command
	toolPrefix     string
	customActions  []string
	standaloneCmds []string
	commands       []CommandInfo
	actionMap      map[string][]CommandInfo // action -> commands
}

// ToolDefinition represents an MCP tool definition
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{} // JSON Schema
}

// NewToolRegistry creates a new ToolRegistry
func NewToolRegistry(rootCmd *cobra.Command, config *ServerConfig) *ToolRegistry {
	config = normalizeServerConfig(rootCmd, config)
	executor := NewCommandExecutorWithMode(rootCmd, config.ExecutionMode)
	commands := executor.GetAllCommands()

	tr := &ToolRegistry{
		executor:       executor,
		rootCmd:        rootCmd,
		toolPrefix:     config.ToolPrefix,
		customActions:  config.CustomActions,
		standaloneCmds: config.StandaloneCmds,
		commands:       commands,
		actionMap:      make(map[string][]CommandInfo),
	}

	tr.buildActionMap()
	return tr
}

// buildActionMap groups commands by action
func (tr *ToolRegistry) buildActionMap() {
	for _, cmd := range tr.commands {
		if len(cmd.Path) == 0 {
			continue
		}

		action := tr.detectAction(cmd.Path)
		if action != "" {
			tr.actionMap[action] = append(tr.actionMap[action], cmd)
		}
	}
}

// detectAction detects the action from a command path
func (tr *ToolRegistry) detectAction(path []string) string {
	if len(path) == 0 {
		return ""
	}

	first := strings.ToLower(path[0])

	// If CustomActions is explicitly set (not nil/empty), use it as a whitelist
	if len(tr.customActions) > 0 {
		for _, action := range tr.customActions {
			if strings.ToLower(action) == first {
				return strings.ToLower(action)
			}
		}
		// Not in whitelist, return empty
		return ""
	}

	// Auto-detect: any first-level command is an action
	return first
}

// isStandaloneCommand checks if a command is standalone (doesn't need resources)
func (tr *ToolRegistry) isStandaloneCommand(action string) bool {
	// If StandaloneCmds is explicitly set (not nil/empty), use it as a whitelist
	if len(tr.standaloneCmds) > 0 {
		for _, cmd := range tr.standaloneCmds {
			if strings.EqualFold(cmd, action) {
				return true
			}
		}
		return false
	}

	// Auto-detect: a command is standalone if it has no subcommands (is a leaf command)
	// Check if there are any commands with this action that have subcommands
	commands, ok := tr.actionMap[action]
	if !ok {
		// Action not found, check if it's a direct command
		for _, cmd := range tr.commands {
			if len(cmd.Path) == 1 && strings.EqualFold(cmd.Path[0], action) {
				// Check if this command has subcommands by looking for commands with longer paths
				hasSubcommands := false
				for _, otherCmd := range tr.commands {
					if len(otherCmd.Path) > 1 && strings.EqualFold(otherCmd.Path[0], action) {
						hasSubcommands = true
						break
					}
				}
				return !hasSubcommands
			}
		}
		return false
	}

	// Check if all commands with this action are leaf commands (path length == 1)
	// If any command has path length > 1, it means there are subcommands
	for _, cmd := range commands {
		if len(cmd.Path) > 1 {
			return false // Has subcommands, not standalone
		}
	}

	// All commands with this action are leaf commands (path length == 1)
	return true
}

// getAvailableResources returns available resources for an action
func (tr *ToolRegistry) getAvailableResources(action string) []string {
	commands, ok := tr.actionMap[action]
	if !ok {
		return []string{}
	}

	resources := make(map[string]bool)
	for _, cmd := range commands {
		if len(cmd.Path) > 1 {
			// Resource is typically the second element
			resource := strings.ToLower(cmd.Path[1])
			// Don't add the action itself as a resource
			if resource != strings.ToLower(action) {
				resources[resource] = true
			}
		}
		// Skip single-level commands (len == 1) - these are the action commands themselves,
		// not resources. Resources must be subcommands (len > 1).
	}

	result := []string{}
	for resource := range resources {
		result = append(result, resource)
	}

	return result
}

// GetHierarchicalTools generates hierarchical MCP tool definitions
func (tr *ToolRegistry) GetHierarchicalTools() []map[string]interface{} {
	tools := []map[string]interface{}{}

	// Get all unique actions
	actions := make(map[string]bool)
	for action := range tr.actionMap {
		actions[action] = true
	}

	// Also check standalone commands
	for _, cmd := range tr.commands {
		if len(cmd.Path) > 0 {
			action := tr.detectAction(cmd.Path)
			if action != "" && tr.isStandaloneCommand(action) {
				actions[action] = true
			}
		}
	}

	// Generate tools for each action
	for action := range actions {
		resources := tr.getAvailableResources(action)

		if len(resources) == 0 {
			// Standalone command
			if tr.isStandaloneCommand(action) {
				tool := tr.createStandaloneTool(action)
				if tool != nil {
					tools = append(tools, tool)
				}
			}
			continue
		}

		// Hierarchical tool
		tool := tr.createHierarchicalTool(action, resources)
		if tool != nil {
			tools = append(tools, tool)
		}
	}

	// Always add help tool
	helpTool := tr.createHelpTool()
	if helpTool != nil {
		tools = append(tools, helpTool)
	}

	return tools
}

// createHierarchicalTool creates a hierarchical tool definition
func (tr *ToolRegistry) createHierarchicalTool(action string, resources []string) map[string]interface{} {
	toolName := fmt.Sprintf("%s_%s", tr.toolPrefix, action)

	// Build description
	description := fmt.Sprintf("%s a resource. Available resources: %s",
		titleCase(action), strings.Join(resources, ", "))

	// Build examples
	examples := []string{}
	for _, resource := range resources[:min(3, len(resources))] {
		examples = append(examples, fmt.Sprintf("%s_%s with resource='%s'", tr.toolPrefix, action, resource))
	}

	// Build flag properties for the schema
	flagProperties := map[string]interface{}{}
	flagRequired := []string{}

	// Collect flags from all resources for this action
	for _, resource := range resources {
		commands, ok := tr.actionMap[action]
		if !ok {
			continue
		}

		// Find command for this resource
		var cmdInfo *CommandInfo
		for _, cmd := range commands {
			if len(cmd.Path) > 1 && strings.EqualFold(cmd.Path[1], resource) {
				cmdInfo = &cmd
				break
			}
		}

		if cmdInfo == nil {
			continue
		}

		// Add each flag as a property
		for _, flag := range cmdInfo.Flags {
			// Skip help flags
			if flag.Name == "help" || flag.Name == "h" {
				continue
			}

			// Only add if not already added (flags might be shared across resources)
			if _, exists := flagProperties[flag.Name]; !exists {
				// Convert Go/pflag type to JSON Schema type
				jsonType := tr.convertToJSONSchemaType(flag.Type)

				flagSchema := map[string]interface{}{
					"type":        jsonType,
					"description": flag.Description,
				}

				// Add enum values if we can extract them from description
				if enumValues := tr.extractEnumFromDescription(flag.Description); len(enumValues) > 0 {
					flagSchema["enum"] = enumValues
				}

				flagProperties[flag.Name] = flagSchema

				// Track required flags
				if flag.Required {
					flagRequired = append(flagRequired, flag.Name)
				}
			}
		}
	}

	// Create flags property with properties and additionalProperties
	flagsProperty := map[string]interface{}{
		"type":                 "object",
		"description":          "Optional: Command flags as key-value pairs (flag names without '--' prefix). Provide an empty object {} if no flags are needed.",
		"additionalProperties": true,
	}

	if len(flagProperties) > 0 {
		flagsProperty["properties"] = flagProperties
		if len(flagRequired) > 0 {
			flagsProperty["required"] = flagRequired
		}
	}

	// Create JSON Schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"resource": map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("REQUIRED: The resource type to %s. Must be one of: %s", action, strings.Join(resources, ", ")),
				"enum":        resources,
			},
			"flags": flagsProperty,
			"args": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Optional positional arguments (rarely used)",
			},
		},
		"required": []string{"resource"},
	}

	return map[string]interface{}{
		"name":        toolName,
		"description": description,
		"inputSchema": schema,
	}
}

// convertToJSONSchemaType converts Go/pflag type names to JSON Schema types
func (tr *ToolRegistry) convertToJSONSchemaType(goType string) string {
	switch goType {
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "string":
		return "string"
	case "[]string", "[]int", "[]bool":
		return "array"
	default:
		// Default to string for unknown types
		return "string"
	}
}

// extractEnumFromDescription extracts enum values from flag descriptions
// Looks for patterns like "Small, Medium, or Large" or "option1|option2|option3"
func (tr *ToolRegistry) extractEnumFromDescription(description string) []string {
	if description == "" {
		return nil
	}

	// Pattern 1: "value1, value2, or value3" or "value1, value2, value3"
	// Common in Cobra flag descriptions
	if strings.Contains(description, ",") {
		// Try to find a list pattern after colon (e.g., "Size: Small, Medium, or Large")
		parts := strings.Split(description, ":")
		if len(parts) > 1 {
			enumPart := strings.TrimSpace(parts[len(parts)-1])
			// Remove parenthetical notes first (e.g., "(required)")
			if idx := strings.Index(enumPart, "("); idx > 0 {
				enumPart = strings.TrimSpace(enumPart[:idx])
			}
			// Replace " or " with ", " before splitting
			enumPart = strings.ReplaceAll(enumPart, " or ", ", ")
			enumPart = strings.ReplaceAll(enumPart, " and ", ", ")

			values := []string{}
			for _, val := range strings.Split(enumPart, ",") {
				val = strings.TrimSpace(val)
				// Skip empty values and values that are too long
				if len(val) > 0 && len(val) < 50 {
					// Remove trailing punctuation
					val = strings.TrimRight(val, ".)")
					val = strings.TrimSpace(val)
					// Skip if it contains "required" or other metadata
					if !strings.Contains(strings.ToLower(val), "required") && val != "" {
						values = append(values, val)
					}
				}
			}
			if len(values) > 0 && len(values) <= 10 {
				return values
			}
		}
	}

	// Pattern 2: "option1|option2|option3"
	if strings.Contains(description, "|") {
		values := []string{}
		for _, val := range strings.Split(description, "|") {
			val = strings.TrimSpace(val)
			if len(val) > 0 && len(val) < 50 {
				values = append(values, val)
			}
		}
		if len(values) > 0 && len(values) <= 10 {
			return values
		}
	}

	return nil
}

// createStandaloneTool creates a standalone tool definition
func (tr *ToolRegistry) createStandaloneTool(action string) map[string]interface{} {
	toolName := fmt.Sprintf("%s_%s", tr.toolPrefix, action)

	// Find command info for description
	var cmdInfo *CommandInfo
	for _, cmd := range tr.commands {
		if len(cmd.Path) > 0 && strings.EqualFold(cmd.Path[0], action) {
			cmdInfo = &cmd
			break
		}
	}

	description := fmt.Sprintf("%s command", titleCase(action))
	if cmdInfo != nil && cmdInfo.Description != "" {
		description = cmdInfo.Description
	}

	// Create JSON Schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"flags": map[string]interface{}{
				"type":                 "object",
				"description":          "Optional command flags as key-value pairs (flag names without '--' prefix)",
				"additionalProperties": true,
			},
			"args": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Optional positional arguments",
			},
		},
	}

	return map[string]interface{}{
		"name":        toolName,
		"description": description,
		"inputSchema": schema,
	}
}

// createHelpTool creates a help tool definition
func (tr *ToolRegistry) createHelpTool() map[string]interface{} {
	toolName := fmt.Sprintf("%s_help", tr.toolPrefix)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Command path to get help for (e.g., 'create cluster')",
			},
			"resource": map[string]interface{}{
				"type":        "string",
				"description": "Resource type to get help for",
			},
		},
	}

	return map[string]interface{}{
		"name":        toolName,
		"description": "Get help for a command or resource",
		"inputSchema": schema,
	}
}

// CallTool executes a tool call
func (tr *ToolRegistry) CallTool(toolName string, arguments map[string]interface{}) (map[string]interface{}, error) {
	// Parse tool name to extract action
	if !strings.HasPrefix(toolName, tr.toolPrefix+"_") {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}

	action := strings.TrimPrefix(toolName, tr.toolPrefix+"_")

	// Handle help tool
	if action == "help" {
		return tr.handleHelpTool(arguments)
	}

	// Check if it's a standalone command
	if tr.isStandaloneCommand(action) {
		return tr.handleStandaloneTool(action, arguments)
	}

	// Handle hierarchical tool
	return tr.handleHierarchicalTool(action, arguments)
}

// handleHierarchicalTool handles a hierarchical tool call
func (tr *ToolRegistry) handleHierarchicalTool(action string, arguments map[string]interface{}) (map[string]interface{}, error) {
	// Extract resource
	resource, ok := arguments["resource"].(string)
	if !ok {
		return nil, fmt.Errorf("missing required parameter: resource")
	}

	// Build command path
	commandPath := []string{action, resource}

	// Extract flags
	flags := map[string]interface{}{}
	if flagsVal, ok := arguments["flags"].(map[string]interface{}); ok {
		flags = flagsVal
	}

	// Extract args
	args := []string{}
	if argsVal, ok := arguments["args"].([]interface{}); ok {
		for _, arg := range argsVal {
			if str, ok := arg.(string); ok {
				args = append(args, str)
			}
		}
	}

	// Execute command
	result, err := tr.executor.Execute(commandPath, flags)
	if err != nil {
		return map[string]interface{}{
			"isError": true,
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Error executing command: %v", err),
				},
			},
		}, nil
	}

	// Build response
	response := map[string]interface{}{
		"isError": result.ExitCode != 0,
		"content": []map[string]interface{}{},
	}

	// Always include output, even if empty, to ensure we return something
	if result.Stderr != "" {
		response["content"] = append(response["content"].([]map[string]interface{}), map[string]interface{}{
			"type": "text",
			"text": fmt.Sprintf("Error: %s", result.Stderr),
		})
	}

	if result.Stdout != "" {
		// Add stdout content
		response["content"] = append(response["content"].([]map[string]interface{}), map[string]interface{}{
			"type": "text",
			"text": result.Stdout,
		})
	} else if result.Stderr == "" {
		// If both stdout and stderr are empty, add an empty message to ensure we return content
		response["content"] = append(response["content"].([]map[string]interface{}), map[string]interface{}{
			"type": "text",
			"text": "",
		})
	}

	return response, nil
}

// handleStandaloneTool handles a standalone tool call
func (tr *ToolRegistry) handleStandaloneTool(action string, arguments map[string]interface{}) (map[string]interface{}, error) {
	// Build command path
	commandPath := []string{action}

	// Extract flags
	flags := map[string]interface{}{}
	if flagsVal, ok := arguments["flags"].(map[string]interface{}); ok {
		flags = flagsVal
	}

	// Extract args
	args := []string{}
	if argsVal, ok := arguments["args"].([]interface{}); ok {
		for _, arg := range argsVal {
			if str, ok := arg.(string); ok {
				args = append(args, str)
			}
		}
	}

	// Execute command
	result, err := tr.executor.Execute(commandPath, flags)
	if err != nil {
		return map[string]interface{}{
			"isError": true,
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Error executing command: %v", err),
				},
			},
		}, nil
	}

	// Build response
	response := map[string]interface{}{
		"isError": result.ExitCode != 0,
		"content": []map[string]interface{}{},
	}

	if result.Stderr != "" {
		response["content"] = append(response["content"].([]map[string]interface{}), map[string]interface{}{
			"type": "text",
			"text": fmt.Sprintf("Error: %s", result.Stderr),
		})
	}

	if result.Stdout != "" {
		response["content"] = append(response["content"].([]map[string]interface{}), map[string]interface{}{
			"type": "text",
			"text": result.Stdout,
		})
	}

	return response, nil
}

// handleHelpTool handles the help tool call with enhanced, structured help
func (tr *ToolRegistry) handleHelpTool(arguments map[string]interface{}) (map[string]interface{}, error) {
	var commandPath []string

	if cmd, ok := arguments["command"].(string); ok && cmd != "" {
		commandPath = strings.Fields(cmd)
	} else if resource, ok := arguments["resource"].(string); ok && resource != "" {
		// Try to find a command for this resource
		commandPath = []string{resource}
	} else {
		return tr.buildGeneralHelp(), nil
	}

	if len(commandPath) == 0 {
		return tr.buildGeneralHelp(), nil
	}

	// Detect what type of help is requested
	first := strings.ToLower(commandPath[0])
	action := tr.detectAction(commandPath)

	var helpText strings.Builder

	if len(commandPath) == 1 {
		// Single word - could be action or resource
		if action != "" {
			// It's an action
			helpText.WriteString(tr.buildActionHelp(action))
		} else {
			// Might be a resource - find which actions support it
			helpText.WriteString(tr.buildResourceHelp(first))
		}
	} else if len(commandPath) == 2 && action != "" {
		// Two words with action - specific command (action + resource)
		resource := strings.ToLower(commandPath[1])
		helpText.WriteString(tr.buildCommandHelp(action, resource))
	} else {
		// More complex path - fall back to CLI help
		helpText.WriteString(tr.buildCLIHelp(commandPath))
	}

	return map[string]interface{}{
		"isError": false,
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": helpText.String(),
			},
		},
	}, nil
}

// buildGeneralHelp builds general help when no specific command is requested
func (tr *ToolRegistry) buildGeneralHelp() map[string]interface{} {
	var help strings.Builder
	help.WriteString(fmt.Sprintf("Available MCP Tools for %s:\n\n", tr.toolPrefix))

	// Get all actions
	actions := make(map[string]bool)
	for action := range tr.actionMap {
		actions[action] = true
	}
	for _, cmd := range tr.commands {
		if len(cmd.Path) > 0 {
			action := tr.detectAction(cmd.Path)
			if action != "" && tr.isStandaloneCommand(action) {
				actions[action] = true
			}
		}
	}

	// List hierarchical tools
	for action := range actions {
		resources := tr.getAvailableResources(action)
		if len(resources) > 0 {
			toolName := fmt.Sprintf("%s_%s", tr.toolPrefix, action)
			help.WriteString(fmt.Sprintf("- %s: Available resources: %s\n", toolName, strings.Join(resources, ", ")))
		} else if tr.isStandaloneCommand(action) {
			toolName := fmt.Sprintf("%s_%s", tr.toolPrefix, action)
			help.WriteString(fmt.Sprintf("- %s: Standalone command\n", toolName))
		}
	}

	help.WriteString(fmt.Sprintf("\n- %s_help: Get help for commands or resources\n", tr.toolPrefix))
	help.WriteString("\nUse 'help <command>' or 'help <action>' for detailed information.\n")

	return map[string]interface{}{
		"isError": false,
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": help.String(),
			},
		},
	}
}

// buildActionHelp builds help for an action (e.g., "create")
func (tr *ToolRegistry) buildActionHelp(action string) string {
	var help strings.Builder
	toolName := fmt.Sprintf("%s_%s", tr.toolPrefix, action)

	// Get command info for description
	var actionDesc string
	commands, ok := tr.actionMap[action]
	if ok && len(commands) > 0 {
		actionDesc = commands[0].Description
	}
	if actionDesc == "" {
		actionDesc = fmt.Sprintf("Perform %s operations", action)
	}

	help.WriteString(fmt.Sprintf("Action: %s\n", titleCase(action)))
	help.WriteString(fmt.Sprintf("Description: %s\n", actionDesc))
	help.WriteString(fmt.Sprintf("MCP Tool: `%s`\n\n", toolName))

	resources := tr.getAvailableResources(action)
	if len(resources) > 0 {
		help.WriteString("Available Resources:\n")
		for _, resource := range resources {
			// Find command info for this resource
			var resourceDesc string
			for _, cmd := range commands {
				if len(cmd.Path) > 1 && strings.EqualFold(cmd.Path[1], resource) {
					resourceDesc = cmd.Description
					break
				}
			}
			if resourceDesc == "" {
				resourceDesc = fmt.Sprintf("%s %s", titleCase(action), resource)
			}
			help.WriteString(fmt.Sprintf("  - %s: %s\n", resource, resourceDesc))
		}
		help.WriteString("\n")

		// Example usage (for AI agent reference, not for user display)
		help.WriteString("Usage Examples (for AI agent reference):\n")
		for i, resource := range resources {
			if i >= 3 { // Limit to 3 examples
				break
			}
			help.WriteString(fmt.Sprintf("\n%d. To %s a %s, use tool '%s' with resource='%s'\n",
				i+1, strings.ToLower(action), resource, toolName, resource))
		}
		help.WriteString("\nNote: Explain to users in natural language what they can ask for, not the tool call structure.\n")
	} else if tr.isStandaloneCommand(action) {
		// Standalone command
		help.WriteString("This is a standalone command (no resources required).\n\n")
		help.WriteString(fmt.Sprintf("Usage: Use tool '%s' to execute this command.\n", toolName))
		help.WriteString("Note: Explain to users in natural language what they can ask for.\n")
	}

	help.WriteString(fmt.Sprintf("\nTo get detailed help for a specific resource, use: help %s <resource>\n", action))
	help.WriteString(fmt.Sprintf("For CLI help, use: %s %s --help\n", tr.toolPrefix, action))

	return help.String()
}

// buildCommandHelp builds help for a specific command (e.g., "create cluster")
func (tr *ToolRegistry) buildCommandHelp(action, resource string) string {
	var help strings.Builder
	toolName := fmt.Sprintf("%s_%s", tr.toolPrefix, action)

	// Find the command
	var cmdInfo *CommandInfo
	commands, ok := tr.actionMap[action]
	if ok {
		for _, cmd := range commands {
			if len(cmd.Path) > 1 && strings.EqualFold(cmd.Path[1], resource) {
				cmdInfo = &cmd
				break
			}
		}
	}

	if cmdInfo == nil {
		// Fall back to CLI help
		return tr.buildCLIHelp([]string{action, resource})
	}

	help.WriteString(fmt.Sprintf("Command: %s %s\n", action, resource))
	if cmdInfo.Description != "" {
		help.WriteString(fmt.Sprintf("Description: %s\n", cmdInfo.Description))
	}
	if cmdInfo.Long != "" && cmdInfo.Long != cmdInfo.Description {
		help.WriteString(fmt.Sprintf("\n%s\n", cmdInfo.Long))
	}
	help.WriteString(fmt.Sprintf("MCP Tool: `%s`\n\n", toolName))

	// Show flags
	if len(cmdInfo.Flags) > 0 {
		help.WriteString("Available Flags:\n")
		for _, flag := range cmdInfo.Flags {
			flagStr := fmt.Sprintf("  --%s", flag.Name)
			if flag.Shorthand != "" {
				flagStr += fmt.Sprintf(", -%s", flag.Shorthand)
			}
			if flag.Type != "" {
				flagStr += fmt.Sprintf(" (%s)", flag.Type)
			}
			if flag.Required {
				flagStr += " [REQUIRED]"
			}
			help.WriteString(fmt.Sprintf("%s\n", flagStr))
			if flag.Description != "" {
				help.WriteString(fmt.Sprintf("    %s\n", flag.Description))
			}
		}
		help.WriteString("\n")
	}

	// Usage information (for AI agent reference)
	help.WriteString("Usage:\n")
	help.WriteString(fmt.Sprintf("- Use tool '%s' with resource='%s'\n", toolName, resource))
	if len(cmdInfo.Flags) > 0 {
		requiredFlags := []FlagInfo{}
		optionalFlags := []FlagInfo{}
		for _, flag := range cmdInfo.Flags {
			if flag.Required {
				requiredFlags = append(requiredFlags, flag)
			} else {
				optionalFlags = append(optionalFlags, flag)
			}
		}

		if len(requiredFlags) > 0 {
			help.WriteString("\nREQUIRED PARAMETERS (must be provided in 'flags' object):\n")
			for i, flag := range requiredFlags {
				flagDesc := flag.Description
				if flagDesc == "" {
					flagDesc = fmt.Sprintf("%s parameter", flag.Type)
				}
				help.WriteString(fmt.Sprintf("  %d. %s (%s): %s\n", i+1, flag.Name, flag.Type, flagDesc))
			}
			help.WriteString("\nExample flags object:\n")
			exampleFlags := map[string]interface{}{}
			for _, flag := range requiredFlags {
				switch flag.Type {
				case "bool":
					exampleFlags[flag.Name] = true
				case "int":
					exampleFlags[flag.Name] = 42
				default:
					exampleFlags[flag.Name] = "value"
				}
			}
			exampleJSON, _ := json.MarshalIndent(exampleFlags, "  ", "  ")
			help.WriteString(fmt.Sprintf("  flags: %s\n", string(exampleJSON)))
		}

		if len(optionalFlags) > 0 {
			help.WriteString("\nOPTIONAL PARAMETERS:\n")
			for _, flag := range optionalFlags {
				flagDesc := flag.Description
				if flagDesc == "" {
					flagDesc = fmt.Sprintf("%s parameter", flag.Type)
				}
				help.WriteString(fmt.Sprintf("  - %s (%s): %s\n", flag.Name, flag.Type, flagDesc))
			}
		}
	}
	help.WriteString("\nCRITICAL: All parameters must be passed in the 'flags' object, NOT as positional arguments in 'args'.\n")
	help.WriteString("IMPORTANT: If required parameters are missing from the user's request, ask for them using a numbered list before proceeding.\n")
	help.WriteString("Note: Explain to users in natural language what they can ask for and what options are available.\n")

	return help.String()
}

// buildResourceHelp builds help for a resource (showing which actions support it)
func (tr *ToolRegistry) buildResourceHelp(resource string) string {
	var help strings.Builder
	help.WriteString(fmt.Sprintf("Resource: %s\n\n", titleCase(resource)))

	// Find which actions support this resource
	supportingActions := []string{}
	for action := range tr.actionMap {
		resources := tr.getAvailableResources(action)
		for _, r := range resources {
			if strings.EqualFold(r, resource) {
				supportingActions = append(supportingActions, action)
				break
			}
		}
	}

	if len(supportingActions) == 0 {
		help.WriteString("This resource is not found or not supported by any actions.\n")
		help.WriteString("Falling back to CLI help...\n\n")
		return help.String() + tr.buildCLIHelp([]string{resource})
	}

	help.WriteString("Supported Actions:\n")
	for _, action := range supportingActions {
		toolName := fmt.Sprintf("%s_%s", tr.toolPrefix, action)
		help.WriteString(fmt.Sprintf("  - %s: Use tool `%s` with resource='%s'\n", action, toolName, resource))
	}
	help.WriteString("\n")

	// Usage examples (for AI agent reference)
	help.WriteString("Usage Examples:\n")
	for i, action := range supportingActions {
		if i >= 3 { // Limit to 3 examples
			break
		}
		toolName := fmt.Sprintf("%s_%s", tr.toolPrefix, action)
		help.WriteString(fmt.Sprintf("\n%d. To %s a %s, use tool '%s' with resource='%s'\n",
			i+1, strings.ToLower(action), resource, toolName, resource))
	}
	help.WriteString("\nNote: Explain to users in natural language what they can ask for.\n")

	return help.String()
}

// buildCLIHelp falls back to executing the CLI help command
func (tr *ToolRegistry) buildCLIHelp(commandPath []string) string {
	helpPath := append([]string{"help"}, commandPath...)
	result, err := tr.executor.Execute(helpPath, map[string]interface{}{})
	if err != nil {
		return fmt.Sprintf("Error getting help: %v", err)
	}
	return result.Stdout + result.Stderr
}

// titleCase capitalizes the first letter of a string
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
