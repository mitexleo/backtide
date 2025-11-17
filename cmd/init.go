package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitexleo/backtide/internal/config"
	"github.com/mitexleo/backtide/internal/systemd"
	"github.com/spf13/cobra"
)

var (
	initForce bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Backtide system configuration",
	Long: `Initialize Backtide with system-wide configuration.

This command creates:
- Configuration file at /etc/backtide/config.toml
- Required system directories
- S3 credentials directory

Use this command once during initial setup.`,
	Run: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing configuration")
}

func runInit(cmd *cobra.Command, args []string) {
	fmt.Println("Initializing backtide...")

	// Use specified config file or default to system location
	configPath := cfgFile
	if configPath == "" {
		configPath = "/etc/backtide/config.toml"
	}

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		if !initForce {
			fmt.Printf("Configuration file already exists: %s\n", configPath)
			fmt.Println("Use --force to overwrite existing configuration")
			os.Exit(1)
		}
	}

	// Create configuration directory
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating configuration directory: %v\n", err)
		fmt.Println("💡 You may need to run with sudo for system configuration")
		fmt.Println("   Try: sudo backtide init")
		os.Exit(1)
	}

	// Create default configuration
	defaultConfig := config.DefaultConfig()

	// Save configuration to system location
	fmt.Printf("💾 Saving configuration to: %s\n", configPath)
	if err := config.SaveConfig(defaultConfig, configPath); err != nil {
		fmt.Printf("❌ Error saving configuration: %v\n", err)
		fmt.Println("💡 You may need to run with sudo for system configuration")
		fmt.Println("   Try: sudo backtide init")
		os.Exit(1)
	}

	// Create necessary system directories
	fmt.Println("📁 Creating system directories...")
	dirs := []string{
		"/etc/backtide",
		"/etc/backtide/s3-credentials",
		"/var/lib/backtide",
		"/var/log/backtide",
		"/tmp/backtide",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("  Warning: Could not create %s: %v\n", dir, err)
		} else {
			fmt.Printf("  Created: %s\n", dir)
		}
	}

	// Automatically set up systemd daemon if running as root
	if os.Geteuid() == 0 {
		fmt.Println("\n⏰ Setting up scheduling daemon...")

		// Create systemd service manager
		manager := systemd.NewServiceManager("backtide", "", configPath, "root")

		// Create systemd service directory if it doesn't exist
		systemdDir := "/etc/systemd/system"
		if err := os.MkdirAll(systemdDir, 0755); err != nil {
			fmt.Printf("  ⚠️  Warning: Could not create systemd directory: %v\n", err)
		} else {
			// Create service files for daemon
			if err := manager.UpdateServiceFiles(""); err != nil {
				fmt.Printf("  ⚠️  Warning: Could not set up scheduling daemon: %v\n", err)
			} else {
				// Enable and start service
				if err := manager.EnableService(); err != nil {
					fmt.Printf("  ⚠️  Warning: Could not enable daemon: %v\n", err)
				} else if err := manager.StartTimer(); err != nil {
					fmt.Printf("  ⚠️  Warning: Could not start daemon: %v\n", err)
				} else {
					fmt.Println("  ✅ Scheduling daemon enabled!")
					fmt.Println("     Daemon will manage ALL backup job schedules internally")
				}
			}
		}
	} else {
		fmt.Println("\n💡 To enable automated backups, run:")
		fmt.Println("   sudo backtide systemd install")
	}

	fmt.Printf("\n✅ Configuration created successfully: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file with your specific settings")
	fmt.Println("2. Add backup jobs: backtide jobs add")
	fmt.Println("3. Add S3 buckets: backtide s3 add")
	fmt.Println("4. Configure directories to backup")
	fmt.Println("5. Test the backup: backtide backup --dry-run")
	if os.Geteuid() != 0 {
		fmt.Println("6. Set up automated backups: sudo backtide systemd install")
	}
	fmt.Println("\nExample commands:")
	fmt.Println("  backtide jobs add                  # Add backup job")
	fmt.Println("  backtide s3 add                    # Add S3 bucket")
	fmt.Println("  backtide backup --dry-run          # Test backup")
	fmt.Println("  backtide systemd install           # Set up systemd service")
}
