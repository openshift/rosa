package cobra_mcp

import "github.com/spf13/cobra"

// ServerConfig holds configuration for the MCP server
type ServerConfig struct {
	// Name is the server name (default: "{cli-name}-mcp-server")
	Name string
	// Version is the server version (default: "1.0.0")
	Version string
	// ToolPrefix is the tool name prefix (default: CLI name)
	ToolPrefix string
	// EnableResources enables resource registry (default: true)
	EnableResources bool
	// CustomActions are custom action names to recognize
	CustomActions []string
	// StandaloneCmds are commands that don't need resources
	StandaloneCmds []string
	// DangerousCommands is a list of dangerous commands that require confirmation
	// Format: "action" or "action resource" (e.g., "delete", "delete cluster", "destroy")
	DangerousCommands []string
	// ExecutionMode determines how commands are executed:
	//   - "in-process" (default): Execute commands directly in-process (fast, but vulnerable to os.Exit())
	//   - "sub-process": Execute all commands in a sub-process (safer, but slower)
	//   - "auto": Auto-detect - use sub-process for commands with Run: (no RunE:), in-process for others
	ExecutionMode string
}

// ChatConfig holds configuration for the chat client
type ChatConfig struct {
	// APIKey is the OpenAI API key (can also use OPENAI_API_KEY env var)
	APIKey string
	// APIURL is optional custom API URL
	APIURL string
	// Model is the model to use (default: "gpt-4")
	Model string
	// Debug enables debug logging
	Debug bool
	// SystemMessage is an optional custom system message
	SystemMessage string
	// SystemMessageFile is an optional file path for system message
	SystemMessageFile string
}

// SystemMessageConfig holds configuration for system message generation
type SystemMessageConfig struct {
	// CLIName is the name of the CLI
	CLIName string
	// CLIDescription is the description of the CLI
	CLIDescription string
	// CLIHelp is the help text from the root CLI command
	CLIHelp string
	// ToolPrefix is the tool name prefix
	ToolPrefix string
	// AvailableActions are the available actions
	AvailableActions []string
	// AvailableResources maps action -> resources
	AvailableResources map[string][]string
	// CommandHelp maps action -> resource -> help text (or action -> "" -> help text for standalone)
	CommandHelp map[string]map[string]string
	// CommonPatterns are common usage patterns to include
	CommonPatterns []string
	// SafetyRequirements are safety requirements to include
	SafetyRequirements []string
	// DangerousCommands is a list of dangerous commands that require confirmation
	// Format: "action" or "action resource" (e.g., "delete", "delete cluster", "destroy")
	DangerousCommands []string
	// CustomInstructions are custom instructions to include
	CustomInstructions []string
}

// getDefaultActions returns the default list of actions
func getDefaultActions() []string {
	return []string{
		"create", "list", "describe", "delete", "edit", "upgrade",
		"grant", "revoke", "verify",
		"login", "logout", "whoami", "version", "help",
	}
}

// getDefaultStandaloneCommands returns the default list of standalone commands
func getDefaultStandaloneCommands() []string {
	return []string{
		"version", "help", "whoami", "login", "logout",
	}
}

// normalizeServerConfig applies defaults to ServerConfig
func normalizeServerConfig(rootCmd *cobra.Command, config *ServerConfig) *ServerConfig {
	if config == nil {
		config = &ServerConfig{}
	}

	if config.Name == "" {
		config.Name = rootCmd.Name() + "-mcp-server"
	}

	if config.Version == "" {
		config.Version = "1.0.0"
	}

	if config.ToolPrefix == "" {
		config.ToolPrefix = rootCmd.Name()
	}

	// Default execution mode to "in-process" for backward compatibility
	if config.ExecutionMode == "" {
		config.ExecutionMode = "in-process"
	}

	// Validate execution mode
	if config.ExecutionMode != "in-process" && config.ExecutionMode != "sub-process" && config.ExecutionMode != "auto" {
		// Invalid mode, default to in-process
		config.ExecutionMode = "in-process"
	}

	// CustomActions and StandaloneCmds are left as nil/empty for auto-detection
	// If explicitly set, they will be used as whitelists

	return config
}
