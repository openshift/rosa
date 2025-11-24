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

package main

import (
	"fmt"
	"os"
	"strings"

	cobra_mcp "github.com/paulczar/cobra-mcp/pkg"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/color"
	"github.com/openshift/rosa/pkg/commands"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
	versionUtils "github.com/openshift/rosa/pkg/version"
)

var root = &cobra.Command{
	Use:   "rosa",
	Short: "Command line tool for ROSA.",
	Long: "Command line tool for Red Hat OpenShift Service on AWS.\n" +
		"For further documentation visit " +
		"https://access.redhat.com/documentation/en-us/red_hat_openshift_service_on_aws\n",
	PersistentPreRun: versionCheck,
	Args:             cobra.NoArgs,
}

func init() {
	// Add the command line flags:
	fs := root.PersistentFlags()
	color.AddFlag(root)
	arguments.AddDebugFlag(fs)

	// Register the subcommands:
	commands.RegisterCommands(root)

	// Add MCP support with sub-process execution mode
	serverConfig := &cobra_mcp.ServerConfig{
		ToolPrefix:      "rosa",
		ExecutionMode:   "sub-process",
		EnableResources: true,
	}
	mcpCmd := cobra_mcp.NewMCPCommand(root, serverConfig)
	mcpCmd.Args = cobra.NoArgs
	// Set Args for mcp subcommands
	for _, subCmd := range mcpCmd.Commands() {
		subCmd.Args = cobra.NoArgs
		// Convert RunE to Run for test compatibility
		if subCmd.RunE != nil {
			runE := subCmd.RunE
			subCmd.RunE = nil
			subCmd.Run = func(cmd *cobra.Command, args []string) {
				if err := runE(cmd, args); err != nil {
					cmd.PrintErrln(err)
					os.Exit(1)
				}
			}
		}
	}
	root.AddCommand(mcpCmd)

	// Add Chat support
	chatConfig := &cobra_mcp.ChatConfig{
		Model: "gpt-5-mini",
		Debug: false,
	}
	chatCmd := cobra_mcp.NewChatCommand(root, chatConfig, serverConfig)
	chatCmd.Args = cobra.NoArgs
	// Set Args and Run for system-message subcommand
	if systemMsgCmd := chatCmd.Commands()[0]; systemMsgCmd != nil && systemMsgCmd.Name() == "system-message" {
		systemMsgCmd.Args = cobra.NoArgs
		// Convert RunE to Run for test compatibility
		if systemMsgCmd.RunE != nil {
			runE := systemMsgCmd.RunE
			systemMsgCmd.RunE = nil
			systemMsgCmd.Run = func(cmd *cobra.Command, args []string) {
				if err := runE(cmd, args); err != nil {
					cmd.PrintErrln(err)
					os.Exit(1)
				}
			}
		}
	}
	root.AddCommand(chatCmd)
}

func main() {
	// Execute the root command:
	root.SetArgs(os.Args[1:])
	err := root.Execute()
	if err != nil {
		if !strings.Contains(err.Error(), "Did you mean this?") {
			fmt.Fprintf(os.Stderr, "Failed to execute root command: %s\n", err)
		}
		os.Exit(1)
	}
}

func versionCheck(cmd *cobra.Command, _ []string) {
	if !versionUtils.ShouldRunCheck(cmd) {
		return
	}

	rprtr := reporter.CreateReporter()
	rosaVersion, err := versionUtils.NewRosaVersion()
	if err != nil {
		rprtr.Debugf("Could not verify the current version of ROSA: %v", err)
		rprtr.Debugf("You might be running on an outdated version. Make sure you are using the current version of ROSA.")
		return
	}
	latestVersionFromMirror, isLatest, err := rosaVersion.IsLatest(info.DefaultVersion)
	if err != nil {
		rprtr.Debugf("There was a problem retrieving the latest version of ROSA: %v", err)
		rprtr.Debugf("You might be running on an outdated version. Make sure you are using the current version of ROSA.")
		return
	}
	if !isLatest {
		rprtr.Warnf("The current version (%s) is not up to date with latest rosa cli released version (%s).",
			info.DefaultVersion,
			latestVersionFromMirror.Original(),
		)
		rprtr.Warnf("It is recommended that you update to the latest version.")
	}
}
