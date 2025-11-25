package cobra_mcp

import (
	"fmt"
	"strings"
)

// ResourceRegistry exposes CLI data as MCP resources for read-only access
type ResourceRegistry struct {
	executor   *CommandExecutor
	toolPrefix string
}

// ResourceDefinition represents an MCP resource definition
type ResourceDefinition struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

// NewResourceRegistry creates a new ResourceRegistry
func NewResourceRegistry(executor *CommandExecutor, toolPrefix string) *ResourceRegistry {
	return &ResourceRegistry{
		executor:   executor,
		toolPrefix: toolPrefix,
	}
}

// GetResources discovers and returns available resources
func (rr *ResourceRegistry) GetResources() []ResourceDefinition {
	resources := []ResourceDefinition{}

	// Discover resources from commands
	commands := rr.executor.GetAllCommands()
	resourceTypes := make(map[string]bool)

	for _, cmd := range commands {
		if len(cmd.Path) >= 2 {
			// Resource is typically the second element in path
			resourceType := strings.ToLower(cmd.Path[1])
			resourceTypes[resourceType] = true
		} else if len(cmd.Path) == 1 {
			// Single-level command might be a resource itself
			// Skip if it's an action (has subcommands)
			resourceType := strings.ToLower(cmd.Path[0])
			hasSubcommands := false
			// Check if this command has subcommands by looking for commands with longer paths
			for _, otherCmd := range commands {
				if len(otherCmd.Path) > 1 && strings.ToLower(otherCmd.Path[0]) == resourceType {
					hasSubcommands = true
					break
				}
			}
			// Only treat as resource if it has no subcommands (not an action)
			if !hasSubcommands {
				resourceTypes[resourceType] = true
			}
		}
	}

	// Create resource definitions
	for resourceType := range resourceTypes {
		// Plural form for list resources
		plural := resourceType + "s"
		if !strings.HasSuffix(resourceType, "s") {
			plural = resourceType + "s"
		}

		resources = append(resources, ResourceDefinition{
			URI:         fmt.Sprintf("%s://%s", rr.toolPrefix, plural),
			Name:        fmt.Sprintf("%s List", strings.Title(resourceType)),
			Description: fmt.Sprintf("List of %s resources", resourceType),
			MimeType:    "application/json",
		})

		// Singular form for individual resources
		resources = append(resources, ResourceDefinition{
			URI:         fmt.Sprintf("%s://%s/{id}", rr.toolPrefix, resourceType),
			Name:        fmt.Sprintf("%s Resource", strings.Title(resourceType)),
			Description: fmt.Sprintf("Individual %s resource", resourceType),
			MimeType:    "application/json",
		})
	}

	return resources
}

// ReadResource reads a resource by URI
func (rr *ResourceRegistry) ReadResource(uri string) (string, string, error) {
	// Parse URI: {prefix}://{resource-type}[/{id}]
	if !strings.Contains(uri, "://") {
		return "", "", fmt.Errorf("invalid URI format: %s", uri)
	}

	parts := strings.SplitN(uri, "://", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid URI format: %s", uri)
	}

	scheme := parts[0]
	path := parts[1]

	if scheme != rr.toolPrefix {
		return "", "", fmt.Errorf("unknown URI scheme: %s", scheme)
	}

	// Parse path: {resource-type}[/{id}]
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return "", "", fmt.Errorf("invalid resource path: %s", path)
	}

	resourceType := pathParts[0]
	var resourceID string
	if len(pathParts) > 1 {
		resourceID = strings.Join(pathParts[1:], "/")
	}

	// Build command to list or describe resource
	var commandPath []string
	var flags map[string]interface{}

	if resourceID != "" {
		// Describe specific resource
		commandPath = []string{"describe", resourceType}
		flags = map[string]interface{}{
			// Try common ID flag names
			"id":      resourceID,
			"name":    resourceID,
			"cluster": resourceID,
		}
	} else {
		// List resources
		// Try plural form first
		plural := resourceType + "s"
		if !strings.HasSuffix(resourceType, "s") {
			plural = resourceType + "s"
		}
		commandPath = []string{"list", plural}
		flags = map[string]interface{}{}
	}

	// Execute command
	result, err := rr.executor.Execute(commandPath, flags)
	if err != nil {
		return "", "", fmt.Errorf("error executing command: %w", err)
	}

	if result.ExitCode != 0 {
		return "", "", fmt.Errorf("command failed: %s", result.Stderr)
	}

	// Return content and mime type
	content := result.Stdout
	if content == "" {
		content = "{}"
	}

	return content, "application/json", nil
}
