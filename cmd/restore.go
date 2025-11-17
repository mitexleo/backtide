package cmd

import (
	"fmt"
	"os"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/commands"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	restoreJobName string
	restoreForce   bool
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore [backup-id]",
	Short: "Restore a backup",
	Long: `Restore a backup to its original locations.

This command will:
- Restore files and directories from the specified backup
- Preserve original permissions and ownership
- Restore from the appropriate storage location (local or S3)

Examples:
  backtide restore backup-20241201-143000
  backtide restore --job daily-backup backup-20241201-143000
  backtide restore --force backup-20241201-143000`,
	Args: cobra.ExactArgs(1),
	Run:  runRestore,
}

func init() {
	restoreCmd.Flags().StringVarP(&restoreJobName, "job", "j", "", "restore backup for specific job")
	restoreCmd.Flags().BoolVarP(&restoreForce, "force", "f", false, "skip confirmation prompts")

	// Register with command registry
	commands.RegisterCommand("restore", restoreCmd)
}

func runRestore(cmd *cobra.Command, args []string) {
	backupID := args[0]
	configPath := getConfigPath()
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
		fmt.Printf("This will overwrite existing files in the target directories.\n")
		fmt.Print("Are you sure you want to continue? (yes/no): ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			fmt.Println("Restore cancelled")
			return
		}
	}

	fmt.Printf("Restoring backup: %s\n", backupID)
	fmt.Printf("Job: %s\n", job.Name)

	if dryRun {
		fmt.Println("DRY RUN: Would restore backup (no changes made)")
		return
	}

	// Perform the restore
	if err := backupManager.RestoreBackup(backupID); err != nil {
		fmt.Printf("Error restoring backup: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Backup restored successfully: %s\n", backupID)
}
