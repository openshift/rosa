package cobra_mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandExecutor discovers and executes Cobra commands directly in-process or in sub-process
type CommandExecutor struct {
	rootCmd       *cobra.Command
	executionMode string // "in-process", "sub-process", or "auto"
}

// CommandInfo represents information about a discovered command
type CommandInfo struct {
	Path        []string // Command path: ["create", "cluster"]
	Description string   // Short description
	Use         string   // Usage string
	Long        string   // Long description
	Flags       []FlagInfo
}

// FlagInfo represents information about a command flag
type FlagInfo struct {
	Name        string
	Shorthand   string
	Description string
	Type        string // "string", "bool", "int", etc.
	Required    bool
}

// ExecuteResult represents the result of command execution
type ExecuteResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// NewCommandExecutor creates a new CommandExecutor with default in-process execution mode
func NewCommandExecutor(rootCmd *cobra.Command) *CommandExecutor {
	return NewCommandExecutorWithMode(rootCmd, "in-process")
}

// NewCommandExecutorWithMode creates a new CommandExecutor with the specified execution mode
func NewCommandExecutorWithMode(rootCmd *cobra.Command, executionMode string) *CommandExecutor {
	// Validate and default execution mode
	if executionMode == "" {
		executionMode = "in-process"
	}
	if executionMode != "in-process" && executionMode != "sub-process" && executionMode != "auto" {
		executionMode = "in-process"
	}
	return &CommandExecutor{
		rootCmd:       rootCmd,
		executionMode: executionMode,
	}
}

// GetAllCommands discovers all commands in the Cobra command tree
func (e *CommandExecutor) GetAllCommands() []CommandInfo {
	commands := []CommandInfo{}
	e.traverseCommands(e.rootCmd, []string{}, &commands)
	return commands
}

// traverseCommands recursively traverses the command tree
func (e *CommandExecutor) traverseCommands(cmd *cobra.Command, path []string, commands *[]CommandInfo) {
	// Skip hidden commands and root
	if cmd.Hidden {
		// Still traverse children
		for _, subCmd := range cmd.Commands() {
			e.traverseCommands(subCmd, path, commands)
		}
		return
	}

	// If this is the root command, just traverse children
	if cmd == e.rootCmd {
		for _, subCmd := range cmd.Commands() {
			e.traverseCommands(subCmd, []string{}, commands)
		}
		return
	}

	// Add current command
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

// extractFlags extracts flag information from a Cobra command
func (e *CommandExecutor) extractFlags(cmd *cobra.Command) []FlagInfo {
	flags := []FlagInfo{}

	// Extract from local flags
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flags = append(flags, e.flagToInfo(flag, cmd))
	})

	// Extract from persistent flags
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		// Check if we already have this flag (avoid duplicates)
		found := false
		for _, f := range flags {
			if f.Name == flag.Name {
				found = true
				break
			}
		}
		if !found {
			flags = append(flags, e.flagToInfo(flag, cmd))
		}
	})

	return flags
}

// flagToInfo converts a pflag.Flag to FlagInfo
func (e *CommandExecutor) flagToInfo(flag *pflag.Flag, cmd *cobra.Command) FlagInfo {
	info := FlagInfo{
		Name:        flag.Name,
		Shorthand:   flag.Shorthand,
		Description: flag.Usage,
		Type:        flag.Value.Type(),
		Required:    false,
	}

	// Check if flag is required using annotations
	if cmd.Annotations != nil {
		if _, ok := cmd.Annotations[cobra.BashCompOneRequiredFlag]; ok {
			// This is a simplified check - in practice, you'd need to check
			// if this specific flag is in the required list
			info.Required = strings.Contains(cmd.Annotations[cobra.BashCompOneRequiredFlag], flag.Name)
		}
	}

	// Also check if flag has "required" in its usage
	if strings.Contains(strings.ToLower(flag.Usage), "required") {
		info.Required = true
	}

	return info
}

// FindCommand finds a command by path in the command tree
func (e *CommandExecutor) FindCommand(path []string) (*cobra.Command, []string, error) {
	if len(path) == 0 {
		return e.rootCmd, []string{}, nil
	}

	// Use Cobra's Find method which handles the traversal correctly
	cmd, args, err := e.rootCmd.Find(path)
	if err != nil {
		return nil, nil, fmt.Errorf("command not found: %v", path)
	}
	if cmd == nil {
		return nil, nil, fmt.Errorf("command not found: %v", path)
	}

	// args returned by Find are the remaining positional arguments
	// We want to return the command and any positional args
	return cmd, args, nil
}

// FindCommandsUsingRun finds all commands that use Run: instead of RunE:
// These commands may call os.Exit() which will terminate the MCP/chat process
func (e *CommandExecutor) FindCommandsUsingRun() []CommandRunWarning {
	warnings := []CommandRunWarning{}
	e.traverseForRunWarnings(e.rootCmd, []string{}, &warnings)
	return warnings
}

// CommandRunWarning represents a warning about a command using Run: instead of RunE:
type CommandRunWarning struct {
	Path        []string // Command path: ["list", "clusters"]
	CommandName string   // Full command name for display
}

// traverseForRunWarnings recursively finds commands using Run: instead of RunE:
func (e *CommandExecutor) traverseForRunWarnings(cmd *cobra.Command, path []string, warnings *[]CommandRunWarning) {
	// Skip hidden commands and root
	if cmd.Hidden {
		// Still traverse children
		for _, subCmd := range cmd.Commands() {
			e.traverseForRunWarnings(subCmd, path, warnings)
		}
		return
	}

	// If this is the root command, just traverse children
	if cmd == e.rootCmd {
		for _, subCmd := range cmd.Commands() {
			e.traverseForRunWarnings(subCmd, []string{}, warnings)
		}
		return
	}

	// Check if command uses Run: instead of RunE:
	// Commands using Run: may call os.Exit() which terminates the MCP/chat process
	// Skip built-in Cobra commands like "help" which users can't control
	if cmd.Run != nil && cmd.RunE == nil {
		// Skip built-in help command (Cobra adds this automatically)
		if cmd.Name() == "help" && len(path) == 0 {
			// This is the built-in help command at root level - skip it
		} else {
			currentPath := append(path, cmd.Name())
			*warnings = append(*warnings, CommandRunWarning{
				Path:        currentPath,
				CommandName: strings.Join(currentPath, " "),
			})
		}
	}

	// Traverse subcommands
	currentPath := append(path, cmd.Name())
	for _, subCmd := range cmd.Commands() {
		e.traverseForRunWarnings(subCmd, currentPath, warnings)
	}
}

// shouldUseSubProcess determines if a command should be executed in a sub-process
func (e *CommandExecutor) shouldUseSubProcess(cmd *cobra.Command) bool {
	switch e.executionMode {
	case "sub-process":
		return true
	case "auto":
		// Auto-detect: use sub-process for commands with Run: (no RunE:)
		return cmd.Run != nil && cmd.RunE == nil
	case "in-process":
		fallthrough
	default:
		return false
	}
}

// ExecuteSubProcess executes a command in a sub-process
func (e *CommandExecutor) ExecuteSubProcess(commandPath []string, flags map[string]interface{}) (*ExecuteResult, error) {
	// Get the executable path (current running binary)
	executablePath, err := os.Executable()
	if err != nil {
		return &ExecuteResult{
			Stdout:   "",
			Stderr:   fmt.Sprintf("Error getting executable path: %v", err),
			ExitCode: 1,
			Error:    err,
		}, nil
	}

	// Find the command to get positional args and validate it exists
	cmd, args, err := e.FindCommand(commandPath)
	if err != nil {
		return &ExecuteResult{
			Stdout:   "",
			Stderr:   fmt.Sprintf("Command not found: %v", err),
			ExitCode: 1,
			Error:    err,
		}, nil
	}

	// Build command line arguments
	execArgs := make([]string, 0)
	execArgs = append(execArgs, commandPath...)

	// Add flags
	for name, value := range flags {
		if value == nil {
			continue
		}

		// Check if flag exists on the command (or its parents)
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			flag = cmd.PersistentFlags().Lookup(name)
		}
		if flag == nil {
			// Try shorthand
			if len(name) == 1 {
				flag = cmd.Flags().ShorthandLookup(name)
				if flag == nil {
					flag = cmd.PersistentFlags().ShorthandLookup(name)
				}
			}
		}

		if flag != nil {
			// Convert value to string for flag arg
			var flagValue string
			switch v := value.(type) {
			case string:
				flagValue = v
			case bool:
				if v {
					// Boolean flags: --flag (no value) for true
					if flag.Shorthand != "" && len(flag.Shorthand) == 1 {
						execArgs = append(execArgs, fmt.Sprintf("-%s", flag.Shorthand))
					} else {
						execArgs = append(execArgs, fmt.Sprintf("--%s", name))
					}
					continue // Boolean flags don't need a value
				} else {
					continue // Skip false boolean flags
				}
			case int, int8, int16, int32, int64:
				flagValue = fmt.Sprintf("%d", v)
			case uint, uint8, uint16, uint32, uint64:
				flagValue = fmt.Sprintf("%d", v)
			case float32, float64:
				flagValue = fmt.Sprintf("%g", v)
			default:
				// Try JSON encoding for complex types
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					flagValue = fmt.Sprintf("%v", v)
				} else {
					flagValue = string(jsonBytes)
				}
			}

			// Add flag to args: --name value or -n value
			if flag.Shorthand != "" && len(flag.Shorthand) == 1 {
				execArgs = append(execArgs, fmt.Sprintf("-%s", flag.Shorthand), flagValue)
			} else {
				execArgs = append(execArgs, fmt.Sprintf("--%s", name), flagValue)
			}
		}
	}

	// Check if command supports output flag and add JSON output (for consistency with in-process)
	hasOutputFlag := e.hasOutputFlag(cmd)
	if hasOutputFlag {
		// Check if output flag is already set in flags map
		if _, alreadySet := flags["output"]; !alreadySet {
			if _, alreadySet := flags["o"]; !alreadySet {
				// Add --output json flag
				execArgs = append(execArgs, "--output", "json")
			}
		}
	}

	// Add positional args
	if len(args) > 0 {
		execArgs = append(execArgs, args...)
	}

	// Create command context
	ctx := context.Background()
	execCmd := exec.CommandContext(ctx, executablePath, execArgs...)

	// Capture stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	execCmd.Stdout = &stdoutBuf
	execCmd.Stderr = &stderrBuf

	// Execute the command
	err = execCmd.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		// Check if it's an ExitError to get the actual exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Other error (e.g., process spawn failure)
			exitCode = 1
		}
	}

	result := &ExecuteResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
		Error:    err,
	}

	return result, nil
}

// Execute executes a command with the given path and flags
// It routes to either in-process or sub-process execution based on execution mode
func (e *CommandExecutor) Execute(commandPath []string, flags map[string]interface{}) (*ExecuteResult, error) {
	// Find the command to check if we should use sub-process
	cmd, _, err := e.FindCommand(commandPath)
	if err != nil {
		return nil, err
	}

	// Check if we should use sub-process execution
	if e.shouldUseSubProcess(cmd) {
		return e.ExecuteSubProcess(commandPath, flags)
	}

	// Otherwise, use in-process execution (existing logic)
	return e.executeInProcess(commandPath, flags)
}

// executeInProcess executes a command with the given path and flags directly in-process
func (e *CommandExecutor) executeInProcess(commandPath []string, flags map[string]interface{}) (*ExecuteResult, error) {
	cmd, args, err := e.FindCommand(commandPath)
	if err != nil {
		return nil, err
	}

	// Set up output capture with buffers
	// CRITICAL: We must capture ALL output to buffers, not the real stdout/stderr
	// because the MCP server uses stdio for JSON-RPC communication
	var stdoutBuf, stderrBuf bytes.Buffer
	var directStdoutBuf bytes.Buffer // For direct os.Stdout writes

	// We need to redirect output on the root command since we execute from root
	// Save original writers
	originalRootOut := e.rootCmd.OutOrStdout()
	originalRootErr := e.rootCmd.ErrOrStderr()
	originalRootIn := e.rootCmd.InOrStdin()

	// Set output capture on root command
	e.rootCmd.SetOut(&stdoutBuf)
	e.rootCmd.SetErr(&stderrBuf)
	// Don't set stdin - let it use the original to avoid interfering with MCP's stdin
	// The command shouldn't need stdin anyway since we're calling it programmatically

	// CRITICAL: Redirect os.Stdout and os.Stderr temporarily to prevent any direct writes
	// Commands may use fmt.Println, os.Stdout.Write, etc. which bypass Cobra's SetOut
	// This would break the MCP stdio JSON-RPC protocol, so we must capture it
	// NOTE: While we capture direct writes for compatibility, commands should prefer
	// using cmd.Println() or cmd.Printf() as they respect output redirection and
	// follow Cobra best practices.
	originalOsStdout := os.Stdout
	originalOsStderr := os.Stderr

	// Redirect os.Stdout using a pipe to capture direct writes (fmt.Println, os.Stdout.Write, etc.)
	var stdoutR, stdoutW *os.File
	var stdoutDone chan struct{}
	stdoutR, stdoutW, err = os.Pipe()
	if err == nil {
		os.Stdout = stdoutW
		stdoutDone = make(chan struct{})
		go func() {
			defer close(stdoutDone)
			io.Copy(&directStdoutBuf, stdoutR)
			stdoutR.Close()
		}()
		defer func() {
			os.Stdout = originalOsStdout
		}()
	}

	// Redirect os.Stderr to /dev/null to prevent direct writes from breaking protocol
	discardFile, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if discardFile != nil {
		os.Stderr = discardFile
		defer func() {
			os.Stderr = originalOsStderr
			discardFile.Close()
		}()
	}

	// Restore original writers after execution
	defer func() {
		e.rootCmd.SetOut(originalRootOut)
		e.rootCmd.SetErr(originalRootErr)
		e.rootCmd.SetIn(originalRootIn)
		// Flush any buffered output to ensure clean state
		if w, ok := originalRootOut.(interface{ Flush() error }); ok {
			_ = w.Flush()
		}
	}()

	// Set environment variable to disable interactive prompts
	originalEnv := os.Getenv("SURVEY_FORCE_NO_INTERACTIVE")
	os.Setenv("SURVEY_FORCE_NO_INTERACTIVE", "1")
	defer func() {
		if originalEnv != "" {
			os.Setenv("SURVEY_FORCE_NO_INTERACTIVE", originalEnv)
		} else {
			os.Unsetenv("SURVEY_FORCE_NO_INTERACTIVE")
		}
	}()

	// Build flag args to include in command execution
	// We need to add flags to the args array so Cobra can parse them correctly
	flagArgs := []string{}
	for name, value := range flags {
		if value == nil {
			continue
		}

		// Check if flag exists on the command (or its parents)
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			flag = cmd.PersistentFlags().Lookup(name)
		}
		if flag == nil {
			// Try shorthand
			if len(name) == 1 {
				flag = cmd.Flags().ShorthandLookup(name)
				if flag == nil {
					flag = cmd.PersistentFlags().ShorthandLookup(name)
				}
			}
		}

		if flag != nil {
			// Convert value to string for flag arg
			var flagValue string
			switch v := value.(type) {
			case string:
				flagValue = v
			case bool:
				if v {
					flagArgs = append(flagArgs, fmt.Sprintf("--%s", name))
					continue // Boolean flags don't need a value
				} else {
					continue // Skip false boolean flags
				}
			case int, int8, int16, int32, int64:
				flagValue = fmt.Sprintf("%d", v)
			case uint, uint8, uint16, uint32, uint64:
				flagValue = fmt.Sprintf("%d", v)
			case float32, float64:
				flagValue = fmt.Sprintf("%g", v)
			default:
				// Try JSON encoding for complex types
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					flagValue = fmt.Sprintf("%v", v)
				} else {
					flagValue = string(jsonBytes)
				}
			}

			// Add flag to args: --name value or -n value
			if flag.Shorthand != "" && len(flag.Shorthand) == 1 {
				flagArgs = append(flagArgs, fmt.Sprintf("-%s", flag.Shorthand), flagValue)
			} else {
				flagArgs = append(flagArgs, fmt.Sprintf("--%s", name), flagValue)
			}
		}
	}

	// Check if command supports output flag and add JSON output
	hasOutputFlag := e.hasOutputFlag(cmd)
	if hasOutputFlag {
		// Check if output flag is already set
		outputFlag := cmd.Flags().Lookup("output")
		if outputFlag == nil {
			outputFlag = cmd.Flags().Lookup("o")
		}
		if outputFlag == nil {
			outputFlag = cmd.PersistentFlags().Lookup("output")
		}
		if outputFlag == nil {
			outputFlag = cmd.PersistentFlags().Lookup("o")
		}

		if outputFlag != nil && !outputFlag.Changed {
			// Set output to json
			if err := outputFlag.Value.Set("json"); err != nil {
				// Try setting via flag name
				_ = cmd.Flags().Set("output", "json")
			}
		}
	}

	// Execute the command
	// IMPORTANT: We need to execute from the root command with the full path,
	// not from the leaf command directly, because Cobra's ExecuteContext expects
	// to be called on the root with the command path as arguments
	ctx := context.Background()

	// If this is not the root command, we need to execute from root with the full path
	if cmd != e.rootCmd {
		// Build the full command path for execution
		fullPath := commandPath
		// Add flag args before positional args
		fullPath = append(fullPath, flagArgs...)
		// Add any positional args
		if len(args) > 0 {
			fullPath = append(fullPath, args...)
		}

		// Set args on root command and execute from root
		e.rootCmd.SetArgs(fullPath)
		err = e.rootCmd.ExecuteContext(ctx)
	} else {
		// This is the root command, set args and execute
		// Combine flag args and positional args
		allArgs := append(flagArgs, args...)
		cmd.SetArgs(allArgs)
		err = cmd.ExecuteContext(ctx)
	}

	// Determine exit code
	exitCode := 0
	if err != nil {
		// Cobra commands typically return errors, not exit codes
		// We'll use 1 for errors, 0 for success
		exitCode = 1
	}

	// Close the write end of the pipe to signal EOF to the reader
	// This ensures any buffered writes are flushed
	if stdoutW != nil {
		stdoutW.Close()
		// Wait for stdout capture goroutine to complete
		if stdoutDone != nil {
			<-stdoutDone
		}
		// Merge direct stdout writes with Cobra's output
		if directStdoutBuf.Len() > 0 {
			stdoutBuf.Write(directStdoutBuf.Bytes())
		}
	}

	// Read the buffer content after execution completes
	// Note: The buffer should already contain everything written via cmd.Println, etc.
	// because cmd.Println writes to cmd.OutOrStdout() which we redirected to stdoutBuf
	// Direct writes to os.Stdout (fmt.Println, etc.) are captured via the pipe and merged above
	stdoutOutput := stdoutBuf.String()
	stderrOutput := stderrBuf.String()

	result := &ExecuteResult{
		Stdout:   stdoutOutput,
		Stderr:   stderrOutput,
		ExitCode: exitCode,
		Error:    err,
	}

	return result, nil
}

// hasOutputFlag checks if a command supports output flags
func (e *CommandExecutor) hasOutputFlag(cmd *cobra.Command) bool {
	hasOutput := false
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "output" || flag.Name == "o" || flag.Shorthand == "o" {
			hasOutput = true
		}
	})
	if !hasOutput {
		cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
			if flag.Name == "output" || flag.Name == "o" || flag.Shorthand == "o" {
				hasOutput = true
			}
		})
	}
	return hasOutput
}
