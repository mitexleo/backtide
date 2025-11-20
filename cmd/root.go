/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/mitexleo/backtide/internal/commands"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	dryRun  bool
	force   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "backtide",
	Short: "A comprehensive backup utility for Docker applications with S3 integration",
	Long: `Backtide is a powerful backup utility designed for Docker-based applications.

Features:
- Backup multiple directories with compression
- Stop and restart Docker containers during backup
- S3FS integration for cloud storage
- Metadata and permission preservation
- Retention policy management
- Systemd and cron integration
- Automatic update checking

Example usage:
  backtide backup --config /etc/backtide/config.toml
  backtide restore backup-2024-01-15-10-30-00
  backtide list
  backtide cleanup
  backtide auto-update enable    # Enable automatic update checking
  backtide auto-update disable   # Disable automatic update checking`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Register all commands with the centralized registry
	registerCommands()

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.backtide.toml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "force operation, skip confirmation prompts")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// registerCommands registers all commands with the centralized registry
func registerCommands() {
	// Register all top-level commands with the registry
	commands.RegisterCommand("auto-update", autoUpdateCmd)
	commands.RegisterCommand("backup", backupCmd)
	commands.RegisterCommand("cleanup", cleanupCmd)
	commands.RegisterCommand("cron", cronCmd)
	commands.RegisterCommand("daemon", daemonCmd)
	commands.RegisterCommand("init", initCmd)
	commands.RegisterCommand("jobs", jobsCmd)
	commands.RegisterCommand("list", listCmd)
	commands.RegisterCommand("restore", restoreCmd)
	commands.RegisterCommand("s3", s3Cmd)
	commands.RegisterCommand("systemd", systemdCmd)
	commands.RegisterCommand("update", updateCmd)
	commands.RegisterCommand("version", versionCmd)

	// Register all commands with the root command
	commands.RegisterAllWithRoot(rootCmd)
}
