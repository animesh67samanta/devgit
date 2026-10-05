package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCmd creates and returns the 'completion' command.
func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell autocompletion scripts for DevGit commands.

Installation Instructions:

  Bash:
    $ source <(devgit completion bash)
    # To load completions for every session:
    $ devgit completion bash > /etc/bash_completion.d/devgit

  Zsh:
    # If shell completion is not already enabled:
    $ echo "autoload -U compinit; compinit" >> ~/.zshrc
    # To load completions:
    $ devgit completion zsh > "${fpath[1]}/_devgit"

  Fish:
    $ devgit completion fish | source
    # To load completions for every session:
    $ devgit completion fish > ~/.config/fish/completions/devgit.fish

  PowerShell:
    PS> devgit completion powershell | Out-String | Invoke-Expression
    # To load completions for every session, add the output to your profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			out := cmd.OutOrStdout()

			switch shell {
			case "bash":
				return cmd.Root().GenBashCompletion(out)
			case "zsh":
				return cmd.Root().GenZshCompletion(out)
			case "fish":
				return cmd.Root().GenFishCompletion(out, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(out)
			default:
				fmt.Fprintf(os.Stderr, "❌ Unsupported shell: %s. Supported shells: bash, zsh, fish, powershell\n", shell)
				return errors.New("unsupported shell")
			}
		},
	}

	return cmd
}
