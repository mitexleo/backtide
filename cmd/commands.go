package cmd

import (
	"fmt"

	"github.com/mitexleo/backtide/internal/commands"
	"github.com/spf13/cobra"
)

// commandsCmd represents the commands command
var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "List all available commands",
	Long: `List all available commands in Backtide.

This command demonstrates the centralized command registry and shows
all available commands with their descriptions.`,
	Run: runCommands,
}

func init() {
}

func runCommands(cmd *cobra.Command, args []string) {
	fmt.Println("📋 Available Backtide Commands")
	fmt.Println("================================")
	fmt.Println()

	// Get all registered commands
	allCommands := commands.GetAllCommands()
	commandNames := commands.GetCommandNames()

	if len(allCommands) == 0 {
		fmt.Println("No commands registered.")
		return
	}

	fmt.Printf("Total commands: %d\n\n", len(allCommands))

	for _, name := range commandNames {
		command, exists := allCommands[name]
		if !exists {
			continue
		}

		fmt.Printf("🔹 %s\n", name)
		fmt.Printf("   %s\n", command.Short)

		if command.Long != "" && command.Long != command.Short {
			// Show first line of long description if different from short
			lines := splitLines(command.Long)
			if len(lines) > 0 && lines[0] != command.Short {
				fmt.Printf("   %s\n", lines[0])
			}
		}

		// Show subcommands if any
		if len(command.Commands()) > 0 {
			fmt.Printf("   Subcommands: ")
			subcommands := command.Commands()
			for i, subcmd := range subcommands {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", subcmd.Name())
			}
			fmt.Println()
		}

		fmt.Println()
	}

	fmt.Println("💡 Use 'backtide <command> --help' for detailed information about each command.")
}

// splitLines splits a string into lines, handling both Unix and Windows line endings
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
