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
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// ToolRegistry manages MCP tools for ROSA CLI commands
type ToolRegistry struct {
	executor *CommandExecutor
	tools    map[string]*ToolDefinition
}

// ToolDefinition represents an MCP tool
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]ParameterInfo
}

// ParameterInfo describes a tool parameter
type ParameterInfo struct {
	Type        string
	Description string
	Required    bool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(rootCmd *cobra.Command) *ToolRegistry {
	executor := NewCommandExecutor(rootCmd)
	registry := &ToolRegistry{
		executor: executor,
		tools:    make(map[string]*ToolDefinition),
	}
	registry.discoverTools()
	return registry
}

// discoverTools discovers all ROSA CLI commands and registers them as MCP tools
func (tr *ToolRegistry) discoverTools() {
	commands := tr.executor.GetAllCommands()

	for _, cmdInfo := range commands {
		toolName := tr.getToolName(cmdInfo.Path)
		tool := &ToolDefinition{
			Name:        toolName,
			Description: tr.getDescription(cmdInfo),
			Parameters:  tr.extractParameters(cmdInfo),
		}
		tr.tools[toolName] = tool
	}
}

// getToolName converts a command path to an MCP tool name
func (tr *ToolRegistry) getToolName(path []string) string {
	// Convert path like ["create", "cluster"] to "rosa_create_cluster"
	return "rosa_" + strings.Join(path, "_")
}

// getDescription creates a description for the tool
func (tr *ToolRegistry) getDescription(cmdInfo CommandInfo) string {
	if cmdInfo.Long != "" {
		return cmdInfo.Long
	}
	if cmdInfo.Description != "" {
		return cmdInfo.Description
	}
	return fmt.Sprintf("Execute rosa %s command", strings.Join(cmdInfo.Path, " "))
}

// extractParameters converts command flags to MCP tool parameters
func (tr *ToolRegistry) extractParameters(cmdInfo CommandInfo) map[string]ParameterInfo {
	params := make(map[string]ParameterInfo)

	// Add positional arguments if the command accepts them
	if len(cmdInfo.Path) > 0 {
		// Check if last command accepts arguments
		lastCmd := cmdInfo.Path[len(cmdInfo.Path)-1]
		params["args"] = ParameterInfo{
			Type:        "array",
			Description: fmt.Sprintf("Positional arguments for %s command", lastCmd),
			Required:    false,
		}
	}

	// Add flag parameters
	for _, flag := range cmdInfo.Flags {
		paramType := tr.mapFlagTypeToMCPType(flag.Type)
		params[flag.Name] = ParameterInfo{
			Type:        paramType,
			Description: flag.Description,
			Required:    flag.Required,
		}
	}

	return params
}

// mapFlagTypeToMCPType converts cobra flag types to MCP parameter types
func (tr *ToolRegistry) mapFlagTypeToMCPType(flagType string) string {
	switch flagType {
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "string", "stringSlice":
		return "string"
	default:
		return "string"
	}
}

// GetTools returns all registered tools as MCP tool definitions in the legacy flat format.
// Each command is exposed as a separate tool (e.g., rosa_create_cluster, rosa_list_machinepools).
// For MCP server usage, prefer GetHierarchicalTools() which groups tools by action.
func (tr *ToolRegistry) GetTools() []map[string]interface{} {
	var tools []map[string]interface{}

	for _, tool := range tr.tools {
		toolDef := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
		}

		// Build input schema
		properties := make(map[string]interface{})
		required := []string{}

		for paramName, paramInfo := range tool.Parameters {
			paramSchema := map[string]interface{}{
				"type":        paramInfo.Type,
				"description": paramInfo.Description,
			}
			properties[paramName] = paramSchema

			if paramInfo.Required {
				required = append(required, paramName)
			}
		}

		inputSchema := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			inputSchema["required"] = required
		}

		toolDef["inputSchema"] = inputSchema
		tools = append(tools, toolDef)
	}

	return tools
}

// CallTool executes a tool call - now handles both hierarchical and flat tools
func (tr *ToolRegistry) CallTool(toolName string, arguments map[string]interface{}) (map[string]interface{}, error) {
	// Handle help tool specially
	if toolName == "rosa_help" {
		return tr.callHelpTool(arguments)
	}

	// Check if this is a hierarchical tool (single action word after rosa_)
	parts := strings.Split(strings.TrimPrefix(toolName, "rosa_"), "_")

	var path []string
	if len(parts) == 1 {
		action := parts[0]

		// Check if this is a standalone command (no subcommands/resource needed)
		standaloneCommands := map[string]bool{
			"whoami":  true,
			"version": true,
			"login":   true,
			"logout":  true,
		}

		if standaloneCommands[action] {
			// Standalone command: rosa_whoami, rosa_version, etc.
			path = []string{action}

			// Handle flags from the flags object
			if flagsObj, ok := arguments["flags"].(map[string]interface{}); ok {
				for k, v := range flagsObj {
					arguments[k] = v
				}
			}
			// Remove the flags object as we've merged it
			delete(arguments, "flags")
		} else {
			// Hierarchical tool: rosa_create, rosa_list, etc.
			resource, ok := arguments["resource"].(string)
			if !ok {
				// Get available resources for better error message
				availableResources := tr.getAvailableResources(action)
				return nil, fmt.Errorf("hierarchical tool %s requires 'resource' parameter. Available resources: %s. Example: use resource='%s'", toolName, strings.Join(availableResources, ", "), availableResources[0])
			}
			path = []string{action, resource}

			// Handle flags from the flags object
			if flagsObj, ok := arguments["flags"].(map[string]interface{}); ok {
				for k, v := range flagsObj {
					arguments[k] = v
				}
			}
			// Remove the flags object as we've merged it
			delete(arguments, "flags")
		}
	} else {
		// Legacy flat tool format: rosa_create_cluster, etc.
		// Still supported for backward compatibility
		path = tr.ParseToolName(toolName)
	}

	if len(path) == 0 {
		return nil, fmt.Errorf("invalid tool name: %s", toolName)
	}

	// Convert arguments to command flags
	flagArgs := make(map[string]string)
	positionalArgs := []string{}

	// Handle positional arguments - try to convert to flags for common cases
	hasClusterFlag := false
	for key := range arguments {
		if key == "cluster" {
			hasClusterFlag = true
			break
		}
		// Also check if flags object contains cluster
		if key == "flags" {
			if flagsObj, ok := arguments["flags"].(map[string]interface{}); ok {
				if _, hasCluster := flagsObj["cluster"]; hasCluster {
					hasClusterFlag = true
					break
				}
			}
		}
	}

	for key, value := range arguments {
		if key == "args" {
			// Handle positional arguments
			if argsArray, ok := value.([]interface{}); ok {
				for _, arg := range argsArray {
					if strArg, ok := arg.(string); ok {
						// For cluster-related commands, if cluster flag is not set, convert first arg to cluster flag
						if !hasClusterFlag && len(positionalArgs) == 0 && (path[0] == "describe" || path[0] == "list" || path[0] == "edit" || path[0] == "delete") && path[1] == "cluster" {
							flagArgs["cluster"] = strArg
							hasClusterFlag = true
						} else {
							positionalArgs = append(positionalArgs, strArg)
						}
					}
				}
			}
		} else if key != "resource" && key != "flags" {
			// Convert value to string for flag
			flagArgs[key] = tr.valueToString(value)
		}
	}

	// Execute the command
	result, err := tr.executor.Execute(path, flagArgs)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}, err
	}

	// Prepare response
	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result.Stdout,
			},
		},
	}

	if result.Stderr != "" {
		response["content"] = append(response["content"].([]map[string]interface{}), map[string]interface{}{
			"type": "text",
			"text": "Error output: " + result.Stderr,
		})
	}

	if result.Error != nil {
		response["isError"] = true
	}

	return response, nil
}

// callHelpTool executes the help command
func (tr *ToolRegistry) callHelpTool(arguments map[string]interface{}) (map[string]interface{}, error) {
	var helpPath []string

	if cmd, ok := arguments["command"].(string); ok && cmd != "" {
		// User specified a command path
		helpPath = strings.Split(cmd, " ")
		helpPath = append([]string{"help"}, helpPath...)
	} else if resource, ok := arguments["resource"].(string); ok && resource != "" {
		// User specified just a resource - show general help
		helpPath = []string{"help", resource}
	} else {
		// General help
		helpPath = []string{"help"}
	}

	// Execute rosa help <command>
	result, err := tr.executor.Execute(helpPath, map[string]string{})
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}, err
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result.Stdout,
			},
		},
	}, nil
}

// ParseToolName extracts command path from tool name (public for testing)
func (tr *ToolRegistry) ParseToolName(toolName string) []string {
	// Remove "rosa_" prefix
	toolName = strings.TrimPrefix(toolName, "rosa_")

	// Split by underscore
	return strings.Split(toolName, "_")
}

// valueToString converts an interface{} value to string
func (tr *ToolRegistry) valueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case []interface{}:
		// For string slices, join with comma
		var parts []string
		for _, item := range v {
			parts = append(parts, tr.valueToString(item))
		}
		return strings.Join(parts, ",")
	default:
		// Try JSON marshaling
		if jsonBytes, err := json.Marshal(value); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", value)
	}
}

// getAvailableResources discovers available subcommands/resources for a given action
func (tr *ToolRegistry) getAvailableResources(action string) []string {
	var resources []string
	commands := tr.executor.GetAllCommands()

	for _, cmdInfo := range commands {
		if len(cmdInfo.Path) > 0 && cmdInfo.Path[0] == action {
			// Get the resource name (second part of path, or first if action is the only part)
			if len(cmdInfo.Path) > 1 {
				resources = append(resources, cmdInfo.Path[1])
			} else if len(cmdInfo.Path) == 1 && cmdInfo.Path[0] != action {
				// Handle single-level commands like "whoami", "version"
				resources = append(resources, cmdInfo.Path[0])
			}
		}
	}

	// Remove duplicates and sort
	seen := make(map[string]bool)
	var unique []string
	for _, r := range resources {
		if !seen[r] {
			seen[r] = true
			unique = append(unique, r)
		}
	}
	sort.Strings(unique)
	return unique
}

// GetHierarchicalTools returns tools organized by action (create, list, describe, etc.).
// This is the preferred format for MCP servers as it dramatically reduces tool count
// from ~136 individual tools to ~12-15 hierarchical tools. Each tool accepts a 'resource'
// parameter to specify what resource type to operate on.
func (tr *ToolRegistry) GetHierarchicalTools() []map[string]interface{} {
	var tools []map[string]interface{}

	// Define high-level actions
	actions := []struct {
		name        string
		description string
	}{
		{"create", "Create ROSA resources (clusters, machinepools, IDPs, etc.)"},
		{"list", "List ROSA resources"},
		{"describe", "Show details of ROSA resources"},
		{"delete", "Delete ROSA resources"},
		{"edit", "Edit ROSA resources"},
		{"upgrade", "Upgrade ROSA resources"},
		{"grant", "Grant permissions or access"},
		{"revoke", "Revoke permissions or access"},
		{"verify", "Verify ROSA resources or configurations"},
		{"login", "Log in to ROSA"},
		{"logout", "Log out from ROSA"},
		{"whoami", "Display information about your AWS and Red Hat accounts"},
		{"version", "Print the version number"},
		{"help", "Get help for ROSA commands and resources"},
	}

	for _, action := range actions {
		// Check if this is a standalone command (no subcommands/resource needed)
		standaloneCommands := map[string]bool{
			"whoami":  true,
			"version": true,
			"login":   true,
			"logout":  true,
		}

		if standaloneCommands[action.name] {
			// These are standalone commands, not hierarchical
			tool := tr.createStandaloneTool(action.name)
			if tool != nil {
				tools = append(tools, tool)
			}
			continue
		}

		if action.name == "help" {
			// Special help tool
			tool := tr.createHelpTool()
			tools = append(tools, tool)
			continue
		}

		// Create hierarchical tool for this action
		resources := tr.getAvailableResources(action.name)
		if len(resources) > 0 {
			tool := tr.createHierarchicalTool(action.name, action.description, resources)
			tools = append(tools, tool)
		}
	}

	return tools
}

// createHierarchicalTool creates a tool for an action with resource parameter
func (tr *ToolRegistry) createHierarchicalTool(action, description string, resources []string) map[string]interface{} {
	// Create examples based on action type
	var exampleResource string
	var exampleFlags map[string]string
	switch action {
	case "list":
		exampleResource = "machinepools"
		exampleFlags = map[string]string{"cluster": "mycluster"}
	case "describe":
		exampleResource = "cluster"
		exampleFlags = map[string]string{"cluster": "mycluster"}
	case "create":
		exampleResource = "machinepool"
		exampleFlags = map[string]string{"cluster": "mycluster", "name": "worker-pool", "replicas": "3"}
	case "delete":
		exampleResource = "machinepool"
		exampleFlags = map[string]string{"cluster": "mycluster", "name": "worker-pool"}
	default:
		exampleResource = resources[0]
		exampleFlags = map[string]string{"cluster": "mycluster"}
	}

	properties := map[string]interface{}{
		"resource": map[string]interface{}{
			"type":        "string",
			"description": fmt.Sprintf("REQUIRED: The resource type to %s. Must be one of: %s. Examples: %s", action, strings.Join(resources, ", "), exampleResource),
			"enum":        resources,
		},
		"flags": map[string]interface{}{
			"type":                 "object",
			"description":          fmt.Sprintf("REQUIRED for most commands: Command flags as key-value pairs. Use flag names without '--' prefix. For cluster-related commands, ALWAYS use flags with 'cluster' key (e.g., {\"cluster\": \"cluster-name-or-id\"}). Example: %v", exampleFlags),
			"additionalProperties": true,
		},
		"args": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "string",
			},
			"description": "Optional positional arguments (rarely used). PREFER using flags instead. Only use args if the command specifically requires positional arguments.",
		},
	}

	// Create a more detailed description with examples
	detailedDescription := fmt.Sprintf("%s. IMPORTANT: You MUST provide the 'resource' parameter specifying what to %s. ALWAYS use the 'flags' parameter with appropriate flag names. For cluster-related operations, use flags={'cluster': 'cluster-name-or-id'}. Examples: To list machine pools, use resource='machinepools' with flags={'cluster': 'cluster-name'}. To describe a cluster, use resource='cluster' with flags={'cluster': 'cluster-name-or-id'}. DO NOT use 'args' for cluster identifiers - use flags['cluster'] instead.", description, action)

	return map[string]interface{}{
		"name":        "rosa_" + action,
		"description": detailedDescription,
		"inputSchema": map[string]interface{}{
			"type":       "object",
			"properties": properties,
			"required":   []string{"resource"},
		},
	}
}

// createStandaloneTool creates a tool for standalone commands like whoami, version
func (tr *ToolRegistry) createStandaloneTool(command string) map[string]interface{} {
	// Find the command to get its description
	commands := tr.executor.GetAllCommands()
	for _, cmdInfo := range commands {
		if len(cmdInfo.Path) == 1 && cmdInfo.Path[0] == command {
			properties := map[string]interface{}{
				"flags": map[string]interface{}{
					"type":                 "object",
					"description":          "Command flags as key-value pairs",
					"additionalProperties": true,
				},
			}

			return map[string]interface{}{
				"name":        "rosa_" + command,
				"description": tr.getDescription(cmdInfo),
				"inputSchema": map[string]interface{}{
					"type":       "object",
					"properties": properties,
				},
			}
		}
	}
	return nil
}

// createHelpTool creates the help tool
func (tr *ToolRegistry) createHelpTool() map[string]interface{} {
	// Get all available commands for help
	allCommands := tr.executor.GetAllCommands()
	var commandPaths []string
	for _, cmd := range allCommands {
		commandPaths = append(commandPaths, strings.Join(cmd.Path, " "))
	}
	sort.Strings(commandPaths)

	// Limit displayed commands in description to avoid overwhelming
	maxDisplay := 30
	displayCommands := commandPaths
	if len(displayCommands) > maxDisplay {
		displayCommands = displayCommands[:maxDisplay]
	}

	properties := map[string]interface{}{
		"command": map[string]interface{}{
			"type":        "string",
			"description": fmt.Sprintf("ROSA command path to get help for (e.g., 'create cluster', 'list machinepools'). Available commands include: %s (and %d more)", strings.Join(displayCommands, ", "), len(commandPaths)-maxDisplay),
		},
		"resource": map[string]interface{}{
			"type":        "string",
			"description": "Optional: Resource type to get help for (e.g., 'cluster', 'machinepool', 'idp')",
		},
	}

	return map[string]interface{}{
		"name":        "rosa_help",
		"description": "Get help information for ROSA CLI commands. Use this when you need to understand command syntax, available flags, or usage examples.",
		"inputSchema": map[string]interface{}{
			"type":       "object",
			"properties": properties,
		},
	}
}
