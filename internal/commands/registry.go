package commands

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

// CommandRegistry maintains a centralized registry of all available commands
type CommandRegistry struct {
	commands map[string]*cobra.Command
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*cobra.Command),
	}
}

// Register adds a command to the registry
func (r *CommandRegistry) Register(name string, cmd *cobra.Command) error {
	if _, exists := r.commands[name]; exists {
		return fmt.Errorf("command '%s' is already registered", name)
	}
	r.commands[name] = cmd
	return nil
}

// Get retrieves a command by name
func (r *CommandRegistry) Get(name string) (*cobra.Command, bool) {
	cmd, exists := r.commands[name]
	return cmd, exists
}

// GetAll returns all registered commands
func (r *CommandRegistry) GetAll() map[string]*cobra.Command {
	return r.commands
}

// GetCommandNames returns a sorted list of all command names
func (r *CommandRegistry) GetCommandNames() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegisterWithRoot registers all commands with the root command
func (r *CommandRegistry) RegisterWithRoot(rootCmd *cobra.Command) error {
	for _, cmd := range r.commands {
		rootCmd.AddCommand(cmd)
	}
	return nil
}

// Global registry instance
var globalRegistry = NewCommandRegistry()

// RegisterCommand adds a command to the global registry
func RegisterCommand(name string, cmd *cobra.Command) error {
	return globalRegistry.Register(name, cmd)
}

// GetCommand retrieves a command from the global registry
func GetCommand(name string) (*cobra.Command, bool) {
	return globalRegistry.Get(name)
}

// GetAllCommands returns all commands from the global registry
func GetAllCommands() map[string]*cobra.Command {
	return globalRegistry.GetAll()
}

// GetCommandNames returns sorted command names from the global registry
func GetCommandNames() []string {
	return globalRegistry.GetCommandNames()
}

// RegisterAllWithRoot registers all commands with the root command
func RegisterAllWithRoot(rootCmd *cobra.Command) error {
	return globalRegistry.RegisterWithRoot(rootCmd)
}
