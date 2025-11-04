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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// Server wraps the MCP server and provides ROSA-specific functionality
type Server struct {
	rootCmd          *cobra.Command
	toolRegistry     *ToolRegistry
	resourceRegistry *ResourceRegistry
	mcpServer        *mcp.Server
}

// NewServer creates a new MCP server instance
func NewServer(rootCmd *cobra.Command) *Server {
	executor := NewCommandExecutor(rootCmd)
	toolRegistry := NewToolRegistry(rootCmd)
	resourceRegistry := NewResourceRegistry(executor)

	// Create MCP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "rosa-mcp-server",
		Version: "1.0.0",
	}, nil)

	// Register all tools dynamically using hierarchical structure
	tools := toolRegistry.GetHierarchicalTools()
	for _, toolDef := range tools {
		name, _ := toolDef["name"].(string)
		description, _ := toolDef["description"].(string)
		inputSchemaRaw, _ := toolDef["inputSchema"].(map[string]interface{})

		// Convert inputSchema to jsonschema.Schema
		inputSchema := convertToJSONSchema(inputSchemaRaw)

		// Create tool handler
		toolName := name
		handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse arguments from the request
			var arguments map[string]interface{}
			if req.Params.Arguments != nil {
				if err := json.Unmarshal(req.Params.Arguments, &arguments); err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							&mcp.TextContent{Text: fmt.Sprintf("Error parsing arguments: %v", err)},
						},
						IsError: true,
					}, nil
				}
			}

			// Execute the tool
			result, err := toolRegistry.CallTool(toolName, arguments)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)},
					},
					IsError: true,
				}, nil
			}

			// Convert result to MCP format
			content := []mcp.Content{}
			if contentData, ok := result["content"].([]map[string]interface{}); ok {
				for _, item := range contentData {
					if textType, ok := item["type"].(string); ok && textType == "text" {
						if text, ok := item["text"].(string); ok {
							content = append(content, &mcp.TextContent{Text: text})
						}
					}
				}
			}

			isError := false
			if errFlag, ok := result["isError"].(bool); ok {
				isError = errFlag
			}

			return &mcp.CallToolResult{
				Content: content,
				IsError: isError,
			}, nil
		}

		// Add tool to server
		mcpServer.AddTool(&mcp.Tool{
			Name:        name,
			Description: description,
			InputSchema: inputSchema,
		}, handler)
	}

	// Register all resources dynamically
	resources := resourceRegistry.GetResources()
	for _, res := range resources {
		resourceURI := res.URI
		handler := func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			content, stderr, err := resourceRegistry.ReadResource(resourceURI)
			if err != nil {
				return nil, err
			}

			contents := []*mcp.ResourceContents{
				{
					URI:      resourceURI,
					MIMEType: "application/json",
					Text:     content,
				},
			}

			if stderr != "" {
				contents = append(contents, &mcp.ResourceContents{
					URI:      resourceURI + "#stderr",
					MIMEType: "text/plain",
					Text:     stderr,
				})
			}

			return &mcp.ReadResourceResult{
				Contents: contents,
			}, nil
		}

		mcpServer.AddResource(&mcp.Resource{
			URI:         res.URI,
			Name:        res.Name,
			Description: res.Description,
			MIMEType:    res.MimeType,
		}, handler)
	}

	return &Server{
		rootCmd:          rootCmd,
		toolRegistry:     toolRegistry,
		resourceRegistry: resourceRegistry,
		mcpServer:        mcpServer,
	}
}

// ServeStdio starts the MCP server with stdio transport
// rootCmd should be a fully initialized root command with all subcommands registered
func ServeStdio(rootCmd *cobra.Command) error {
	server := NewServer(rootCmd)
	return server.mcpServer.Run(context.Background(), &mcp.StdioTransport{})
}

// ServeHTTP starts the MCP server with HTTP transport
// rootCmd should be a fully initialized root command with all subcommands registered
func ServeHTTP(rootCmd *cobra.Command, port int) error {
	// Create a single server instance that will be reused
	server := NewServer(rootCmd)

	// Create the streamable HTTP handler
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		// Return the server instance
		return server.mcpServer
	}, nil)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting MCP server on %s", addr)
	return http.ListenAndServe(addr, handler)
}

// convertToJSONSchema converts our JSON schema format to jsonschema.Schema
func convertToJSONSchema(schema map[string]interface{}) *jsonschema.Schema {
	jschema := &jsonschema.Schema{
		Type:       "object",
		Properties: make(map[string]*jsonschema.Schema),
	}

	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for name, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				propSchema := &jsonschema.Schema{}
				if propType, ok := propMap["type"].(string); ok {
					propSchema.Type = propType
				}
				if desc, ok := propMap["description"].(string); ok {
					propSchema.Description = desc
				}
				jschema.Properties[name] = propSchema
			}
		}
	}

	if required, ok := schema["required"].([]interface{}); ok {
		requiredStrs := make([]string, 0, len(required))
		for _, r := range required {
			if str, ok := r.(string); ok {
				requiredStrs = append(requiredStrs, str)
			}
		}
		jschema.Required = requiredStrs
	}

	return jschema
}
