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
	deleteBackupID string
	deleteForce    bool
	deleteAll      bool
	deleteDryRun   bool
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete [backup-id]",
	Short: "Delete specific backups",
	Long: `Delete specific backups or clean up according to retention policies.

This command provides multiple ways to manage backup deletion:

1. Delete specific backup by ID:
   backtide delete backup-20241201-143000

2. Delete all backups for a specific job:
   backtide delete --job daily-backup --all

3. Force cleanup according to retention policies:
   backtide delete --force

4. Dry run to see what would be deleted:
   backtide delete --dry-run

Features:
- Safe deletion with confirmation prompts
- Respects retention policies by default
- Can force cleanup beyond retention
- Dry run mode for safety
- Validation to prevent accidental deletion`,
	Args: cobra.MaximumNArgs(1),
	Run:  runDelete,
}

func init() {
	deleteCmd.Flags().StringVarP(&deleteBackupID, "job", "j", "", "delete backups for specific job")
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "force deletion beyond retention policies")
	deleteCmd.Flags().BoolVarP(&deleteAll, "all", "a", false, "delete all backups for specified job")
	deleteCmd.Flags().BoolVar(&deleteDryRun, "dry-run", false, "show what would be deleted without making changes")

	// Register with command registry
	commands.RegisterCommand("delete", deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) {
	// Validate arguments
	if len(args) == 0 && deleteBackupID == "" && !deleteForce {
		fmt.Println("Error: Must specify backup ID, job name, or use --force for retention cleanup")
		fmt.Println("Usage: backtide delete [backup-id] OR backtide delete --job [job-name] OR backtide delete --force")
		os.Exit(1)
	}

	if len(args) > 0 && deleteBackupID != "" {
		fmt.Println("Error: Cannot specify both backup ID and --job")
		fmt.Println("Use either: backtide delete [backup-id] OR backtide delete --job [job-name]")
		os.Exit(1)
	}

	if deleteAll && deleteBackupID == "" {
		fmt.Println("Error: --all requires --job to be specified")
		fmt.Println("Usage: backtide delete --job [job-name] --all")
		os.Exit(1)
	}

	// Determine deletion mode
	if len(args) > 0 {
		// Mode 1: Delete specific backup by ID
		backupID := args[0]
		deleteSpecificBackup(backupID)
	} else if deleteBackupID != "" {
		// Mode 2: Delete backups for specific job
		deleteJobBackups(deleteBackupID)
	} else if deleteForce {
		// Mode 3: Force cleanup according to retention
		forceCleanup()
	}
}

// deleteSpecificBackup deletes a specific backup by ID
func deleteSpecificBackup(backupID string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Find the backup across all jobs
	found := false
	var backupInfo *config.BackupMetadata
	var backupPath string

	backupRunner := backup.NewBackupRunner(*cfg)
	allBackups, err := backupRunner.ListBackups()
	if err != nil {
		fmt.Printf("Error listing backups: %v\n", err)
		os.Exit(1)
	}

	for _, b := range allBackups {
		if b.ID == backupID {
			backupInfo = &b
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("Error: Backup not found: %s\n", backupID)
		fmt.Println("Use 'backtide list --backups' to see available backups")
		os.Exit(1)
	}

	// Determine backup path
	for _, job := range cfg.Jobs {
		if job.Enabled {
			var bucketConfig *config.BucketConfig
			for _, bucket := range cfg.Buckets {
				if bucket.ID == job.BucketID {
					bucketConfig = &bucket
					break
				}
			}

			backupPath = cfg.BackupPath
			if job.Storage.S3 && bucketConfig != nil {
				backupPath = bucketConfig.MountPoint
			}

			backupDir := filepath.Join(backupPath, backupID)
			if _, err := os.Stat(backupDir); err == nil {
				break
			}
		}
	}

	if backupPath == "" {
		fmt.Printf("Error: Could not locate backup directory for: %s\n", backupID)
		os.Exit(1)
	}

	backupDir := filepath.Join(backupPath, backupID)

	// Confirm deletion
	if !deleteForce && !deleteDryRun {
		fmt.Printf("WARNING: This will permanently delete backup: %s\n", backupID)
		fmt.Printf("Backup date: %s\n", backupInfo.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("Location: %s\n", backupDir)
		fmt.Printf("Directories: %d\n", len(backupInfo.Directories))
		fmt.Printf("Total size: %d bytes\n", backupInfo.TotalSize)
		fmt.Print("\nAre you sure you want to delete this backup? (yes/no): ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			fmt.Println("Deletion cancelled")
			return
		}
	}

	if deleteDryRun {
		fmt.Printf("DRY RUN: Would delete backup: %s\n", backupID)
		fmt.Printf("Location: %s\n", backupDir)
		return
	}

	// Perform deletion
	fmt.Printf("Deleting backup: %s\n", backupID)
	if err := os.RemoveAll(backupDir); err != nil {
		fmt.Printf("Error deleting backup: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Backup deleted successfully: %s\n", backupID)
}

// deleteJobBackups deletes backups for a specific job
func deleteJobBackups(jobName string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Find the job
	var job *config.BackupJob
	for i, j := range cfg.Jobs {
		if j.Name == jobName {
			job = &cfg.Jobs[i]
			break
		}
	}

	if job == nil {
		fmt.Printf("Error: Job not found: %s\n", jobName)
		fmt.Println("Use 'backtide jobs list' to see available jobs")
		os.Exit(1)
	}

	// Find the bucket configuration for this job
	var bucketConfig *config.BucketConfig
	for _, bucket := range cfg.Buckets {
		if bucket.ID == job.BucketID {
			bucketConfig = &bucket
			break
		}
	}

	// Determine backup path for this job
	backupPath := cfg.BackupPath
	if job.Storage.S3 && bucketConfig != nil {
		backupPath = bucketConfig.MountPoint
	}

	// List backups for this job
	jobBackupConfig := config.BackupConfig{
		Jobs:       []config.BackupJob{*job},
		Buckets:    cfg.Buckets,
		BackupPath: backupPath,
		TempPath:   cfg.TempPath,
	}

	backupManager := backup.NewBackupManager(jobBackupConfig)
	backups, err := backupManager.ListBackups()
	if err != nil {
		fmt.Printf("Error listing backups for job %s: %v\n", jobName, err)
		os.Exit(1)
	}

	if len(backups) == 0 {
		fmt.Printf("No backups found for job: %s\n", jobName)
		return
	}

	if deleteAll {
		// Delete all backups for this job
		if !deleteForce && !deleteDryRun {
			fmt.Printf("WARNING: This will delete ALL %d backups for job: %s\n", len(backups), jobName)
			fmt.Print("Are you sure you want to continue? (yes/no): ")

			var response string
			fmt.Scanln(&response)
			if response != "yes" && response != "y" {
				fmt.Println("Deletion cancelled")
				return
			}
		}

		if deleteDryRun {
			fmt.Printf("DRY RUN: Would delete ALL %d backups for job: %s\n", len(backups), jobName)
			for _, b := range backups {
				fmt.Printf("  - %s (%s)\n", b.ID, b.Timestamp.Format("2006-01-02"))
			}
			return
		}

		fmt.Printf("Deleting ALL %d backups for job: %s\n", len(backups), jobName)
		deletedCount := 0

		for _, b := range backups {
			backupDir := filepath.Join(backupPath, b.ID)
			if err := os.RemoveAll(backupDir); err != nil {
				fmt.Printf("Warning: Failed to delete backup %s: %v\n", b.ID, err)
			} else {
				fmt.Printf("  ✅ Deleted: %s\n", b.ID)
				deletedCount++
			}
		}

		fmt.Printf("✅ Deleted %d out of %d backups for job: %s\n", deletedCount, len(backups), jobName)

	} else {
		// Show backups for this job and let user choose
		fmt.Printf("Backups for job: %s\n", jobName)
		for i, b := range backups {
			fmt.Printf("%d. %s - %s - %d bytes\n", i+1, b.ID, b.Timestamp.Format("2006-01-02 15:04:05"), b.TotalSize)
		}

		fmt.Print("\nSelect backup to delete (number) or 'all' for all backups: ")
		var choice string
		fmt.Scanln(&choice)

		if choice == "all" {
			deleteJobBackups(jobName) // Recursive call with deleteAll implied
			return
		}

		var backupIndex int
		if _, err := fmt.Sscanf(choice, "%d", &backupIndex); err == nil && backupIndex >= 1 && backupIndex <= len(backups) {
			backupID := backups[backupIndex-1].ID
			deleteSpecificBackup(backupID)
		} else {
			fmt.Println("Invalid selection")
		}
	}
}

// forceCleanup forces cleanup according to retention policies
func forceCleanup() {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Jobs) == 0 {
		fmt.Println("No backup jobs configured.")
		return
	}

	// Remove unused variable - cleanup is handled by individual job managers

	if deleteDryRun {
		fmt.Println("DRY RUN: Would force cleanup according to retention policies")
	} else {
		fmt.Println("Forcing cleanup according to retention policies...")
	}

	// Run cleanup for all jobs
	for _, job := range cfg.Jobs {
		if !job.Enabled {
			continue
		}

		fmt.Printf("\nCleaning up backups for job: %s\n", job.Name)

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
		}

		// Create job-specific backup config
		jobBackupConfig := config.BackupConfig{
			Jobs:       []config.BackupJob{job},
			Buckets:    cfg.Buckets,
			BackupPath: backupPath,
			TempPath:   cfg.TempPath,
		}

		backupManager := backup.NewBackupManager(jobBackupConfig)

		if deleteDryRun {
			// Dry run - just show what would be cleaned up
			backups, err := backupManager.ListBackups()
			if err != nil {
				fmt.Printf("Warning: Failed to list backups for job %s: %v\n", job.Name, err)
				continue
			}

			fmt.Printf("  Retention: %d days, %d recent, %d monthly\n",
				job.Retention.KeepDays, job.Retention.KeepCount, job.Retention.KeepMonthly)
			fmt.Printf("  Found %d backups\n", len(backups))

		} else {
			// Actual cleanup
			if err := backupManager.CleanupBackups(); err != nil {
				fmt.Printf("Warning: Failed to cleanup backups for job %s: %v\n", job.Name, err)
			}
		}
	}

	if deleteDryRun {
		fmt.Println("\n📋 Dry run completed - no backups were deleted")
	} else {
		fmt.Println("\n✅ Force cleanup completed")
	}
}
