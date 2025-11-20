package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/mitexleo/backtide/internal/commands"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

// autoUpdateCmd represents the auto-update command
var autoUpdateCmd = &cobra.Command{
	Use:   "auto-update",
	Short: "Manage automatic update settings",
	Long: `Manage automatic update checking for Backtide.

This command allows you to enable or disable automatic update checking
in the daemon. When enabled, the daemon will check for new versions
every 24 hours and notify you when updates are available.

Examples:
  backtide auto-update enable    # Enable auto-update checking
  backtide auto-update disable   # Disable auto-update checking
  backtide auto-update status    # Show current auto-update status
  backtide auto-update interval 6h  # Set check interval to 6 hours`,
}

var (
	enableAutoUpdateCmd = &cobra.Command{
		Use:   "enable",
		Short: "Enable automatic update checking",
		Long: `Enable automatic update checking in the daemon.

When enabled, the daemon will check for new versions every 24 hours
(default) and notify you when updates are available. This helps you
stay up-to-date with the latest features and security fixes.

The daemon will only notify you about available updates - it will not
automatically install them. You still need to run 'backtide update'
to install the new version.`,
		Run: runEnableAutoUpdate,
	}

	disableAutoUpdateCmd = &cobra.Command{
		Use:   "disable",
		Short: "Disable automatic update checking",
		Long: `Disable automatic update checking in the daemon.

This will stop the daemon from checking for new versions. You can
still manually check for updates using 'backtide update'.`,
		Run: runDisableAutoUpdate,
	}

	statusAutoUpdateCmd = &cobra.Command{
		Use:   "status",
		Short: "Show current auto-update status",
		Long: `Show the current automatic update checking configuration.

This command displays whether auto-update is enabled and the current
check interval.`,
		Run: runStatusAutoUpdate,
	}

	intervalAutoUpdateCmd = &cobra.Command{
		Use:   "interval [duration]",
		Short: "Set auto-update check interval",
		Long: `Set how often the daemon should check for updates.

The interval can be specified in various formats:
  - 24h (24 hours)
  - 6h (6 hours)
  - 1h30m (1 hour 30 minutes)
  - 30m (30 minutes)

Examples:
  backtide auto-update interval 24h    # Check once per day
  backtide auto-update interval 6h     # Check every 6 hours
  backtide auto-update interval 1h30m  # Check every 1.5 hours`,
		Args: cobra.ExactArgs(1),
		Run:  runIntervalAutoUpdate,
	}
)

func init() {
	// Add subcommands to auto-update command
	autoUpdateCmd.AddCommand(enableAutoUpdateCmd)
	autoUpdateCmd.AddCommand(disableAutoUpdateCmd)
	autoUpdateCmd.AddCommand(statusAutoUpdateCmd)
	autoUpdateCmd.AddCommand(intervalAutoUpdateCmd)

	// Register with command registry
	commands.RegisterCommand("auto-update", autoUpdateCmd)
}

func runEnableAutoUpdate(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if cfg.AutoUpdate.Enabled {
		fmt.Println("✅ Auto-update is already enabled")
		fmt.Printf("📅 Check interval: %v\n", cfg.AutoUpdate.CheckInterval)
		return
	}

	cfg.AutoUpdate.Enabled = true

	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("❌ Failed to save configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Auto-update enabled successfully!")
	fmt.Printf("📅 The daemon will check for updates every %v\n", cfg.AutoUpdate.CheckInterval)
	fmt.Println("💡 Restart the daemon for this change to take effect")
	fmt.Println("   backtide daemon")
}

func runDisableAutoUpdate(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if !cfg.AutoUpdate.Enabled {
		fmt.Println("✅ Auto-update is already disabled")
		return
	}

	cfg.AutoUpdate.Enabled = false

	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("❌ Failed to save configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Auto-update disabled successfully!")
	fmt.Println("💡 Restart the daemon for this change to take effect")
	fmt.Println("   backtide daemon")
}

func runStatusAutoUpdate(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("📋 Auto-update Status")
	fmt.Println("====================")

	if cfg.AutoUpdate.Enabled {
		fmt.Println("✅ Status: Enabled")
		fmt.Printf("📅 Check interval: %v\n", cfg.AutoUpdate.CheckInterval)
		fmt.Println("💡 The daemon will notify you when updates are available")
	} else {
		fmt.Println("❌ Status: Disabled")
		fmt.Println("💡 Enable with: backtide auto-update enable")
	}

	fmt.Println()
	fmt.Println("📝 Next steps:")
	if cfg.AutoUpdate.Enabled {
		fmt.Println("   - Run 'backtide daemon' to start with auto-update")
		fmt.Println("   - Use 'backtide auto-update interval' to change frequency")
		fmt.Println("   - Use 'backtide auto-update disable' to turn it off")
	} else {
		fmt.Println("   - Run 'backtide auto-update enable' to enable")
		fmt.Println("   - Then run 'backtide daemon' to start")
	}
}

func runIntervalAutoUpdate(cmd *cobra.Command, args []string) {
	intervalStr := args[0]

	// Parse the duration
	duration, err := time.ParseDuration(intervalStr)
	if err != nil {
		fmt.Printf("❌ Invalid duration format: %v\n", err)
		fmt.Println("💡 Valid examples: 24h, 6h, 1h30m, 30m")
		os.Exit(1)
	}

	// Validate minimum interval (5 minutes)
	if duration < 5*time.Minute {
		fmt.Println("❌ Check interval must be at least 5 minutes")
		fmt.Println("💡 Use a longer interval like 1h, 6h, or 24h")
		os.Exit(1)
	}

	// Validate maximum interval (30 days)
	if duration > 30*24*time.Hour {
		fmt.Println("❌ Check interval cannot be longer than 30 days")
		fmt.Println("💡 Use a shorter interval like 24h or 7d")
		os.Exit(1)
	}

	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	oldInterval := cfg.AutoUpdate.CheckInterval
	cfg.AutoUpdate.CheckInterval = duration

	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("❌ Failed to save configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Auto-update check interval updated!\n")
	fmt.Printf("📅 Changed from %v to %v\n", oldInterval, duration)
	fmt.Println("💡 Restart the daemon for this change to take effect")
	fmt.Println("   backtide daemon")
}
