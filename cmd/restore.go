package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/commands"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	restoreJobName    string
	restoreForce      bool
	restorePath       string
	restoreTargetPath string
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore [backup-id]",
	Short: "Restore a backup",
	Long: `Restore a backup to its original locations or custom target path.

This command supports multiple restoration modes:

1. Configuration-based restore (recommended for same-server restoration):
   backtide restore backup-20241201-143000
   backtide restore --job daily-backup backup-20241201-143000

2. Path-based restore (for new servers or custom locations):
   backtide restore --path /mnt/backups/backup-20241201-143000
   backtide restore --path /mnt/backups/backup-20241201-143000 --target /new/location

3. S3-based restore (after mounting S3 bucket):
   backtide restore backup-20241201-143000  # automatically discovers from mounted S3

Features:
- Restore files and directories with preserved permissions
- Restore to original paths or custom target locations
- Support for both local and S3 storage
- Graceful handling of missing files and directories
- Validation of backup integrity before restoration`,
	Args: cobra.MaximumNArgs(1),
	Run:  runRestore,
}

func init() {
	restoreCmd.Flags().StringVarP(&restoreJobName, "job", "j", "", "restore backup for specific job")
	restoreCmd.Flags().BoolVarP(&restoreForce, "force", "f", false, "skip confirmation prompts")
	restoreCmd.Flags().StringVarP(&restorePath, "path", "p", "", "restore from specific backup path (bypasses config)")
	restoreCmd.Flags().StringVarP(&restoreTargetPath, "target", "t", "", "restore to custom target path instead of original locations")

	// Register with command registry
	commands.RegisterCommand("restore", restoreCmd)
}

func runRestore(cmd *cobra.Command, args []string) {
	// Validate arguments
	if len(args) == 0 && restorePath == "" {
		fmt.Println("Error: Either backup ID or --path must be specified")
		fmt.Println("Usage: backtide restore [backup-id] OR backtide restore --path /path/to/backup")
		os.Exit(1)
	}

	if len(args) > 0 && restorePath != "" {
		fmt.Println("Error: Cannot specify both backup ID and --path")
		fmt.Println("Use either: backtide restore [backup-id] OR backtide restore --path /path/to/backup")
		os.Exit(1)
	}

	// Determine restoration mode
	if restorePath != "" {
		// Mode 1: Path-based restoration (config-independent)
		runPathBasedRestore()
	} else {
		// Mode 2: Configuration-based restoration
		backupID := args[0]
		runConfigBasedRestore(backupID)
	}
}

// runPathBasedRestore handles restoration from a specific backup path
func runPathBasedRestore() {
	// Validate backup path
	if _, err := os.Stat(restorePath); os.IsNotExist(err) {
		fmt.Printf("Error: Backup path does not exist: %s\n", restorePath)
		os.Exit(1)
	}

	// Check if it's a valid backup directory
	metadataPath := filepath.Join(restorePath, "metadata.toml")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		fmt.Printf("Error: Invalid backup directory - metadata file not found: %s\n", metadataPath)
		fmt.Println("Please ensure the path points to a valid Backtide backup directory")
		os.Exit(1)
	}

	// Load metadata
	metadata, err := config.LoadBackupMetadata(metadataPath)
	if err != nil {
		fmt.Printf("Error: Failed to load backup metadata: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Restoring backup from path: %s\n", restorePath)
	fmt.Printf("Backup ID: %s\n", metadata.ID)
	fmt.Printf("Backup date: %s\n", metadata.Timestamp.Format("2006-01-02 15:04:05"))

	// Create a minimal backup config for the restore operation
	backupConfig := config.BackupConfig{
		BackupPath: filepath.Dir(restorePath), // Use parent directory as backup path
		TempPath:   "/tmp/backtide",
	}

	backupManager := backup.NewBackupManager(backupConfig)

	// Confirm restore operation
	if !restoreForce && !force {
		fmt.Printf("\nWARNING: This will restore backup '%s'\n", metadata.ID)
		fmt.Printf("Source: %s\n", restorePath)

		if restoreTargetPath != "" {
			fmt.Printf("Target: %s (custom location)\n", restoreTargetPath)
			fmt.Printf("Original paths will be mapped to: %s/{directory-name}\n", restoreTargetPath)
		} else {
			fmt.Printf("Target: Original locations\n")
			for _, dir := range metadata.Directories {
				fmt.Printf("  - %s -> %s\n", dir.Name, dir.Path)
			}
		}

		fmt.Print("\nAre you sure you want to continue? (yes/no): ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			fmt.Println("Restore cancelled")
			return
		}
	}

	if dryRun {
		fmt.Println("DRY RUN: Would restore backup (no changes made)")
		return
	}

	// Perform the restore with custom target path if specified
	if restoreTargetPath != "" {
		fmt.Printf("Restoring to custom target: %s\n", restoreTargetPath)
		if err := backupManager.RestoreBackupToPath(metadata.ID, restoreTargetPath); err != nil {
			fmt.Printf("Error restoring backup: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Restore to original locations
		if err := backupManager.RestoreBackup(metadata.ID); err != nil {
			fmt.Printf("Error restoring backup: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("✅ Backup restored successfully: %s\n", metadata.ID)
}

// runConfigBasedRestore handles restoration using configuration file
func runConfigBasedRestore(backupID string) {
	configPath := getConfigPath()
	if configPath == "" {
		fmt.Println("Error: No configuration file found for config-based restore")
		fmt.Println("Use one of the following options:")
		fmt.Println("  1. Create a configuration with 'backtide init'")
		fmt.Println("  2. Use path-based restore: backtide restore --path /path/to/backup")
		fmt.Println("  3. Specify config file: backtide restore --config /path/to/config.toml")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Check if we have any jobs configured
	if len(cfg.Jobs) == 0 {
		fmt.Println("No backup jobs configured.")
		fmt.Println("Use 'backtide jobs add' to create backup jobs.")
		return
	}

	// Determine which job to use for restore
	var job *config.BackupJob
	if restoreJobName != "" {
		for i, j := range cfg.Jobs {
			if j.Name == restoreJobName {
				job = &cfg.Jobs[i]
				break
			}
		}
		if job == nil {
			fmt.Printf("Error: Job '%s' not found\n", restoreJobName)
			fmt.Println("Use 'backtide jobs list' to see available jobs.")
			os.Exit(1)
		}
	} else {
		// Use the first enabled job, or first job if none enabled
		for i, j := range cfg.Jobs {
			if j.Enabled {
				job = &cfg.Jobs[i]
				break
			}
		}
		if job == nil && len(cfg.Jobs) > 0 {
			job = &cfg.Jobs[0]
		}
	}

	if job == nil {
		fmt.Println("No backup job found to use for restore.")
		fmt.Println("Use 'backtide jobs add' to create backup jobs.")
		return
	}

	// Find the bucket configuration for this job
	var bucketConfig *config.BucketConfig
	for _, bucket := range cfg.Buckets {
		if bucket.ID == job.BucketID {
			bucketConfig = &bucket
			break
		}
	}

	// Use S3 mount point as backup path if S3 storage is enabled
	backupPath := cfg.BackupPath
	if job.Storage.S3 && bucketConfig != nil {
		backupPath = bucketConfig.MountPoint
		fmt.Printf("Using S3 mount point for restore: %s\n", backupPath)
	}

	// Create job-specific backup config
	jobBackupConfig := config.BackupConfig{
		Jobs:       []config.BackupJob{*job},
		Buckets:    cfg.Buckets,
		BackupPath: backupPath,
		TempPath:   cfg.TempPath,
	}

	backupManager := backup.NewBackupManager(jobBackupConfig)

	// Confirm restore operation
	if !restoreForce && !force {
		fmt.Printf("WARNING: This will restore backup '%s' for job '%s'\n", backupID, job.Name)

		if restoreTargetPath != "" {
			fmt.Printf("Target: %s (custom location)\n", restoreTargetPath)
			fmt.Printf("Original paths will be mapped to: %s/{directory-name}\n", restoreTargetPath)
		} else {
			fmt.Printf("Target: Original locations\n")
			// Show original paths from the backup (if we can load the metadata)
			backupDir := filepath.Join(backupPath, backupID)
			metadataPath := filepath.Join(backupDir, "metadata.toml")
			if metadata, err := config.LoadBackupMetadata(metadataPath); err == nil {
				for _, dir := range metadata.Directories {
					fmt.Printf("  - %s -> %s\n", dir.Name, dir.Path)
				}
			}
		}

		fmt.Printf("This will overwrite existing files in the target directories.\n")
		fmt.Print("Are you sure you want to continue? (yes/no): ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			fmt.Println("Restore cancelled")
			return
		}
	}

	if dryRun {
		fmt.Println("DRY RUN: Would restore backup (no changes made)")
		return
	}

	// Perform the restore with custom target path if specified
	if restoreTargetPath != "" {
		fmt.Printf("Restoring to custom target: %s\n", restoreTargetPath)
		if err := backupManager.RestoreBackupToPath(backupID, restoreTargetPath); err != nil {
			fmt.Printf("Error restoring backup: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Restore to original locations
		if err := backupManager.RestoreBackup(backupID); err != nil {
			fmt.Printf("Error restoring backup: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("✅ Backup restored successfully: %s\n", backupID)
}
