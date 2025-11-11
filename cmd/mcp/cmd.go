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
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/mcp/chat"
	"github.com/openshift/rosa/cmd/mcp/serve"
)

var Cmd = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol server for ROSA CLI",
	Long:  "Host an MCP server that exposes ROSA CLI commands as tools and resources for AI assistants and other MCP clients.",
	Args:  cobra.NoArgs,
}

func init() {
	Cmd.AddCommand(serve.Cmd)
	Cmd.AddCommand(chat.Cmd)
}
