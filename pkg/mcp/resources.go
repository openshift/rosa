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
	"fmt"
	"strings"
)

// ResourceRegistry manages MCP resources for ROSA entities
type ResourceRegistry struct {
	executor *CommandExecutor
}

// NewResourceRegistry creates a new resource registry
func NewResourceRegistry(executor *CommandExecutor) *ResourceRegistry {
	return &ResourceRegistry{
		executor: executor,
	}
}

// ResourceDefinition represents an MCP resource
type ResourceDefinition struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

// GetResources returns all available resources
func (rr *ResourceRegistry) GetResources() []ResourceDefinition {
	return []ResourceDefinition{
		{
			URI:         "rosa://clusters",
			Name:        "ROSA Clusters",
			Description: "List all ROSA clusters",
			MimeType:    "application/json",
		},
		{
			URI:         "rosa://cluster",
			Name:        "ROSA Cluster",
			Description: "Get specific ROSA cluster details",
			MimeType:    "application/json",
		},
		{
			URI:         "rosa://account-roles",
			Name:        "Account Roles",
			Description: "List account roles",
			MimeType:    "application/json",
		},
		{
			URI:         "rosa://operator-roles",
			Name:        "Operator Roles",
			Description: "List operator roles",
			MimeType:    "application/json",
		},
		{
			URI:         "rosa://machinepools",
			Name:        "Machine Pools",
			Description: "List machine pools for a cluster",
			MimeType:    "application/json",
		},
		{
			URI:         "rosa://idps",
			Name:        "Identity Providers",
			Description: "List identity providers for a cluster",
			MimeType:    "application/json",
		},
		{
			URI:         "rosa://versions",
			Name:        "Available Versions",
			Description: "List available OpenShift versions",
			MimeType:    "application/json",
		},
		{
			URI:         "rosa://regions",
			Name:        "Regions",
			Description: "List available regions",
			MimeType:    "application/json",
		},
	}
}

// ReadResource reads a resource by URI
func (rr *ResourceRegistry) ReadResource(uri string) (string, string, error) {
	// Parse URI: rosa://<resource-type>[/<id>]
	if !strings.HasPrefix(uri, "rosa://") {
		return "", "", fmt.Errorf("invalid resource URI: %s", uri)
	}

	path := strings.TrimPrefix(uri, "rosa://")
	parts := strings.Split(path, "/")
	resourceType := parts[0]

	// Map resource types to commands
	switch resourceType {
	case "clusters":
		return rr.executeListCommand([]string{"list", "clusters"}, "--output", "json")
	case "cluster":
		if len(parts) < 2 {
			return "", "", fmt.Errorf("cluster resource requires cluster ID: rosa://cluster/<id>")
		}
		clusterID := parts[1]
		flags := map[string]string{"output": "json"}
		result, err := rr.executor.Execute([]string{"describe", "cluster", clusterID}, flags)
		if err != nil {
			return "", "", err
		}
		return result.Stdout, result.Stderr, nil
	case "account-roles":
		return rr.executeListCommand([]string{"list", "account-roles"}, "--output", "json")
	case "operator-roles":
		return rr.executeListCommand([]string{"list", "operator-roles"}, "--output", "json")
	case "machinepools":
		if len(parts) < 2 {
			return "", "", fmt.Errorf("machinepools resource requires cluster ID: rosa://machinepools/<cluster-id>")
		}
		clusterID := parts[1]
		flags := map[string]string{"cluster": clusterID, "output": "json"}
		result, err := rr.executor.Execute([]string{"list", "machinepools"}, flags)
		if err != nil {
			return "", "", err
		}
		return result.Stdout, result.Stderr, nil
	case "idps":
		if len(parts) < 2 {
			return "", "", fmt.Errorf("idps resource requires cluster ID: rosa://idps/<cluster-id>")
		}
		clusterID := parts[1]
		flags := map[string]string{"cluster": clusterID, "output": "json"}
		result, err := rr.executor.Execute([]string{"list", "idps"}, flags)
		if err != nil {
			return "", "", err
		}
		return result.Stdout, result.Stderr, nil
	case "versions":
		return rr.executeListCommand([]string{"list", "versions"}, "--output", "json")
	case "regions":
		return rr.executeListCommand([]string{"list", "regions"}, "--output", "json")
	default:
		return "", "", fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// executeListCommand executes a list command and returns the output
func (rr *ResourceRegistry) executeListCommand(cmdPath []string, outputFlag ...string) (string, string, error) {
	// Build flags map
	flags := make(map[string]string)
	for i := 0; i < len(outputFlag); i += 2 {
		if i+1 < len(outputFlag) {
			flags[outputFlag[i]] = outputFlag[i+1]
		}
	}

	result, err := rr.executor.Execute(cmdPath, flags)
	if err != nil {
		return "", "", err
	}

	return result.Stdout, result.Stderr, nil
}
