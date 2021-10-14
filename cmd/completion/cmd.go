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

package completion

import (
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "completion",
	Short: "Generates completion scripts",
	Long: `To load completions:

Bash:

  $ source <(rosa completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ rosa completion bash > /etc/bash_completion.d/rosa
  # macOS:
  $ rosa completion bash > /usr/local/etc/bash_completion.d/rosa

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ rosa completion zsh > "${fpath[1]}/_rosa"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ rosa completion fish | source

  # To load completions for each session, execute once:
  $ rosa completion fish > ~/.config/fish/completions/rosa.fish

PowerShell:

  PS> rosa completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> rosa completion powershell > rosa.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Default to bash for backwards compatibility
			cmd.Root().GenBashCompletion(os.Stdout)
			return
		}
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}
