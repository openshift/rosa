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

// GetTools returns all registered tools as MCP tool definitions
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

// CallTool executes a tool call
func (tr *ToolRegistry) CallTool(toolName string, arguments map[string]interface{}) (map[string]interface{}, error) {
	// Extract command path from tool name
	// toolName format: "rosa_create_cluster" -> ["create", "cluster"]
	path := tr.ParseToolName(toolName)
	if len(path) == 0 {
		return nil, fmt.Errorf("invalid tool name: %s", toolName)
	}

	// Convert arguments to command flags
	flagArgs := make(map[string]string)
	positionalArgs := []string{}

	for key, value := range arguments {
		if key == "args" {
			// Handle positional arguments
			if argsArray, ok := value.([]interface{}); ok {
				for _, arg := range argsArray {
					if strArg, ok := arg.(string); ok {
						positionalArgs = append(positionalArgs, strArg)
					}
				}
			}
		} else {
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

// ParseToolName extracts command path from tool name (public for testing)
func (tr *ToolRegistry) ParseToolName(toolName string) []string {
	// Remove "rosa_" prefix
	if strings.HasPrefix(toolName, "rosa_") {
		toolName = strings.TrimPrefix(toolName, "rosa_")
	}

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
