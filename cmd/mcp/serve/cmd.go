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

package serve

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/color"
	"github.com/openshift/rosa/pkg/mcp"
)

var args struct {
	transport string
	port      int
}

var Cmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the Model Context Protocol server that exposes ROSA CLI commands
as MCP tools and resources. The server supports both stdio and HTTP transports.`,
	Example: `  # Start MCP server with stdio transport (default)
  rosa mcp serve

  # Start MCP server with HTTP transport on port 8080
  rosa mcp serve --transport=http --port=8080`,
	Run: run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.transport,
		"transport",
		"stdio",
		"Transport method for MCP server (stdio or http)",
	)
	flags.IntVar(
		&args.port,
		"port",
		8080,
		"Port number for HTTP transport (only used when transport=http)",
	)
}

func run(cmd *cobra.Command, _ []string) {
	transport := args.transport
	if transport != "stdio" && transport != "http" {
		fmt.Fprintf(os.Stderr, "Invalid transport: %s. Must be 'stdio' or 'http'\n", transport)
		os.Exit(1)
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
	registerCommands(rootCmd)

	var err error
	if transport == "stdio" {
		err = mcp.ServeStdio(rootCmd)
	} else {
		err = mcp.ServeHTTP(rootCmd, args.port)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start MCP server: %v\n", err)
		os.Exit(1)
	}
}

// registerCommandsForMCP registers commands - this avoids circular dependency
// We'll use a helper that doesn't cause import cycles
func registerCommandsForMCP(root *cobra.Command) {
	// Call RegisterCommands by importing it dynamically
	// We can't import it at package level due to circular dependency
	// So we'll import it in a separate file that won't be imported by commands
	registerCommands(root)
}
