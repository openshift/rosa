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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandExecutor executes ROSA CLI commands and captures their output
type CommandExecutor struct {
	rootCmd *cobra.Command
}

// NewCommandExecutor creates a new command executor with the root command
func NewCommandExecutor(rootCmd *cobra.Command) *CommandExecutor {
	return &CommandExecutor{
		rootCmd: rootCmd,
	}
}

// ExecuteResult contains the output and error from command execution
type ExecuteResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// Execute runs a ROSA CLI command with the given arguments and captures output
func (e *CommandExecutor) Execute(commandPath []string, args map[string]string) (*ExecuteResult, error) {
	// Find the command in the tree
	cmd, remainingArgs, err := e.findCommand(commandPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find command: %w", err)
	}

	// Build command line arguments
	cmdArgs := e.buildCommandArgs(cmd, remainingArgs, args)

	// Execute command in a subprocess to capture output cleanly
	// This ensures we don't interfere with the current process's stdout/stderr
	return e.executeSubprocess(commandPath, cmdArgs)
}

// findCommand locates a command in the cobra command tree
func (e *CommandExecutor) findCommand(path []string) (*cobra.Command, []string, error) {
	if len(path) == 0 {
		return nil, nil, fmt.Errorf("command path cannot be empty")
	}

	cmd := e.rootCmd
	remainingArgs := []string{}

	for i, part := range path {
		foundCmd, _, err := cmd.Find([]string{part})
		if err != nil || foundCmd == nil || foundCmd == cmd {
			// If we can't find a subcommand, treat remaining parts as arguments
			remainingArgs = path[i:]
			break
		}
		cmd = foundCmd
		if i == len(path)-1 {
			// Last part is the command itself
			break
		}
	}

	if cmd == e.rootCmd && len(path) > 0 {
		return nil, nil, fmt.Errorf("command not found: %s", strings.Join(path, " "))
	}

	return cmd, remainingArgs, nil
}

// FindCommand is a public wrapper for findCommand
func (e *CommandExecutor) FindCommand(path []string) (*cobra.Command, []string, error) {
	return e.findCommand(path)
}

// buildCommandArgs converts map of arguments to command line flags
func (e *CommandExecutor) buildCommandArgs(cmd *cobra.Command, positionalArgs []string, flagArgs map[string]string) []string {
	args := []string{}

	// Add positional arguments first
	args = append(args, positionalArgs...)

	// Check if output flag is already specified
	outputSpecified := false
	if _, ok := flagArgs["output"]; ok {
		outputSpecified = true
	}
	if _, ok := flagArgs["o"]; ok {
		outputSpecified = true
	}

	// Check if command supports output flag
	supportsOutput := false
	if cmd.Flags().Lookup("output") != nil || cmd.Flags().Lookup("o") != nil {
		supportsOutput = true
	} else if cmd.PersistentFlags().Lookup("output") != nil || cmd.PersistentFlags().Lookup("o") != nil {
		supportsOutput = true
	}

	// Add flag arguments
	for key, value := range flagArgs {
		// Check if flag exists
		flag := cmd.Flags().Lookup(key)
		if flag == nil {
			// Try persistent flags
			flag = cmd.PersistentFlags().Lookup(key)
		}
		if flag == nil {
			// Skip unknown flags - let the command handle validation
			continue
		}

		flagName := "--" + key
		if flag.Shorthand != "" {
			// Prefer shorthand if available
			flagName = "-" + flag.Shorthand
		}

		if flag.Value.Type() == "bool" {
			// Boolean flags are just present/absent
			if value == "true" || value == "1" {
				args = append(args, flagName)
			}
		} else {
			args = append(args, flagName+"="+value)
		}
	}

	// Automatically add -o json if command supports output flag and it's not already specified
	if supportsOutput && !outputSpecified {
		// Prefer shorthand if available
		outputFlag := "-o"
		if cmd.Flags().Lookup("o") == nil && cmd.PersistentFlags().Lookup("o") == nil {
			outputFlag = "--output"
		}
		args = append(args, outputFlag+"=json")
	}

	return args
}

// executeSubprocess executes the command in a subprocess to capture output
func (e *CommandExecutor) executeSubprocess(commandPath []string, cmdArgs []string) (*ExecuteResult, error) {
	// Build full command: rosa <command-path> <args>
	fullArgs := append(commandPath, cmdArgs...)

	// Get the executable path
	executable, err := os.Executable()
	if err != nil {
		// Fallback to "rosa" if we can't determine executable
		executable = "rosa"
	}

	// Check if we're running under "go run" by examining the executable path
	// When running via "go run", os.Executable() returns the go binary or a temp build path
	isGoRun := strings.Contains(executable, "/go") || strings.Contains(executable, "/tmp/") ||
		strings.Contains(executable, "go-build") || executable == "go" ||
		strings.HasSuffix(executable, "/go") || strings.Contains(executable, "go_")

	var cmd *exec.Cmd
	if isGoRun {
		// Running under go run, use "go run ./cmd/rosa" to execute subcommands
		// This ensures we use the same codebase and environment
		goRunArgs := []string{"run", "-mod=mod", "./cmd/rosa"}
		goRunArgs = append(goRunArgs, fullArgs...)
		cmd = exec.Command("go", goRunArgs...)
		// Set working directory to repo root (where go.mod is)
		if wd, err := os.Getwd(); err == nil {
			// Find repo root by looking for go.mod
			repoRoot := wd
			for {
				if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
					cmd.Dir = repoRoot
					break
				}
				parent := filepath.Dir(repoRoot)
				if parent == repoRoot {
					break // Reached root
				}
				repoRoot = parent
			}
		}
	} else {
		// Running as installed binary, use executable directly
		cmd = exec.Command(executable, fullArgs...)
	}
	cmd.Env = os.Environ()

	// Disable interactive mode
	// Note: Don't override OCM_CONFIG as it might point to the config file location
	cmd.Env = append(cmd.Env, "SURVEY_FORCE_NO_INTERACTIVE=1")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	result := &ExecuteResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
		result.Error = err
	} else {
		result.ExitCode = 0
	}

	return result, nil
}

// GetAllCommands returns all available commands by traversing the command tree
func (e *CommandExecutor) GetAllCommands() []CommandInfo {
	commands := make([]CommandInfo, 0)
	e.traverseCommands(e.rootCmd, []string{}, &commands)
	return commands
}

// CommandInfo represents information about a command
type CommandInfo struct {
	Path        []string
	Description string
	Use         string
	Long        string
	Flags       []FlagInfo
}

// FlagInfo represents information about a command flag
type FlagInfo struct {
	Name        string
	Shorthand   string
	Description string
	Type        string
	Required    bool
}

// traverseCommands recursively traverses the command tree to discover all commands
func (e *CommandExecutor) traverseCommands(cmd *cobra.Command, path []string, commands *[]CommandInfo) {
	// Skip hidden commands and the root command itself
	if cmd.Hidden || cmd == e.rootCmd {
		// Still traverse children of root
		if cmd == e.rootCmd {
			for _, subCmd := range cmd.Commands() {
				e.traverseCommands(subCmd, []string{}, commands)
			}
		}
		return
	}

	currentPath := append(path, cmd.Name())
	info := CommandInfo{
		Path:        currentPath,
		Description: cmd.Short,
		Use:         cmd.Use,
		Long:        cmd.Long,
		Flags:       e.extractFlags(cmd),
	}

	*commands = append(*commands, info)

	// Traverse subcommands
	for _, subCmd := range cmd.Commands() {
		e.traverseCommands(subCmd, currentPath, commands)
	}
}

// extractFlags extracts flag information from a command
func (e *CommandExecutor) extractFlags(cmd *cobra.Command) []FlagInfo {
	var flags []FlagInfo

	// Process local flags
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flags = append(flags, FlagInfo{
			Name:        flag.Name,
			Shorthand:   flag.Shorthand,
			Description: flag.Usage,
			Type:        flag.Value.Type(),
			Required:    flag.Annotations != nil && flag.Annotations[cobra.BashCompOneRequiredFlag] != nil,
		})
	})

	// Process persistent flags (avoid duplicates)
	seen := make(map[string]bool)
	for _, flag := range flags {
		seen[flag.Name] = true
	}

	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if !seen[flag.Name] {
			flags = append(flags, FlagInfo{
				Name:        flag.Name,
				Shorthand:   flag.Shorthand,
				Description: flag.Usage,
				Type:        flag.Value.Type(),
				Required:    flag.Annotations != nil && flag.Annotations[cobra.BashCompOneRequiredFlag] != nil,
			})
		}
	})

	return flags
}
