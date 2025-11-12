package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	initForce    bool
	initExamples bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize backtide configuration",
	Long: `Initialize backtide with a default configuration file.

This command will:
1. Create a default configuration file
2. Set up necessary directories
3. Provide guidance for next steps

The configuration file will be created in the default location
(~/.backtide.yaml) unless specified otherwise.`,
	Run: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing configuration file")
	initCmd.Flags().BoolVar(&initExamples, "examples", false, "include example configurations")
}

func runInit(cmd *cobra.Command, args []string) {
	fmt.Println("Initializing backtide...")

	// Determine config file path
	configPath := cfgFile
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		configPath = filepath.Join(home, ".backtide.yaml")
	}

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil && !initForce {
		fmt.Printf("Configuration file already exists: %s\n", configPath)
		fmt.Println("Use --force to overwrite existing configuration")
		os.Exit(1)
	}

	// Create configuration directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating configuration directory: %v\n", err)
		os.Exit(1)
	}

	// Create default configuration
	var defaultConfig *config.BackupConfig
	if initExamples {
		defaultConfig = createExampleConfig()
	} else {
		defaultConfig = config.DefaultConfig()
	}

	// Save configuration
	if err := config.SaveConfig(defaultConfig, configPath); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	// Create necessary directories
	dirs := []string{
		"/var/lib/backtide",
		"/var/log/backtide",
		"/mnt/backup",
		"/mnt/s3backup",
		"/tmp/backtide",
	}

	fmt.Println("\nCreating necessary directories...")
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("  Warning: Could not create %s: %v\n", dir, err)
		} else {
			fmt.Printf("  Created: %s\n", dir)
		}
	}

	fmt.Printf("\nConfiguration created successfully: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file with your specific settings")
	fmt.Println("2. Set S3 credentials and bucket information")
	fmt.Println("3. Configure directories you want to backup")
	fmt.Println("4. Test the backup: backtide backup --dry-run")
	fmt.Println("5. Set up automated backups: backtide systemd install")
	fmt.Println("\nExample commands:")
	fmt.Println("  backtide backup                    # Run backup")
	fmt.Println("  backtide list                      # List backups")
	fmt.Println("  backtide systemd install           # Set up systemd service")
	fmt.Println("  backtide cron install              # Set up cron job")
}

func createExampleConfig() *config.BackupConfig {
	cfg := config.DefaultConfig()

	// Example S3 configuration
	cfg.S3Config.Bucket = "my-backup-bucket"
	cfg.S3Config.Region = "us-east-1"
	cfg.S3Config.AccessKey = "YOUR_ACCESS_KEY_HERE"
	cfg.S3Config.SecretKey = "YOUR_SECRET_KEY_HERE"
	cfg.S3Config.MountPoint = "/mnt/s3backup"
	cfg.S3Config.UsePathStyle = false

	// Example directories to backup
	cfg.Directories = []config.DirectoryConfig{}
	cfg.Directories = append(cfg.Directories, config.DirectoryConfig{
		Path:        "/var/lib/docker/volumes",
		Name:        "docker-volumes",
		Compression: true,
	})
	cfg.Directories = append(cfg.Directories, config.DirectoryConfig{
		Path:        "/opt/myapp/data",
		Name:        "app-data",
		Compression: true,
	})
	cfg.Directories = append(cfg.Directories, config.DirectoryConfig{
		Path:        "/home/user/documents",
		Name:        "user-documents",
		Compression: true,
	})
	cfg.Directories = append(cfg.Directories, config.DirectoryConfig{
		Path:        "/etc",
		Name:        "system-config",
		Compression: true,
	})

	// Example retention policy
	cfg.RetentionPolicy.KeepDays = 30
	cfg.RetentionPolicy.KeepCount = 10
	cfg.RetentionPolicy.KeepMonthly = 6

	// Backup paths
	cfg.BackupPath = "/mnt/backup"
	cfg.TempPath = "/tmp/backtide"

	return cfg
}
