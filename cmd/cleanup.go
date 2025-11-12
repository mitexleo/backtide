package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	cleanupDryRun bool
	cleanupForce  bool
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up old backups according to retention policy",
	Long: `Clean up old backups based on the configured retention policy.

This command will:
- Remove backups older than the configured number of days
- Keep only the specified number of most recent backups
- Apply monthly retention if configured

The cleanup operation follows the retention policy defined in the configuration file.`,
	Run: runCleanup,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "show what would be deleted without actually deleting")
	cleanupCmd.Flags().BoolVarP(&cleanupForce, "force", "f", false, "skip confirmation prompt")
}

func runCleanup(cmd *cobra.Command, args []string) {
	fmt.Println("Starting cleanup operation...")

	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize backup manager
	backupManager := backup.NewBackupManager(*cfg)

	// List current backups
	backups, err := backupManager.ListBackups()
	if err != nil {
		fmt.Printf("Error listing backups: %v\n", err)
		os.Exit(1)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found to clean up")
		return
	}

	// Show retention policy
	fmt.Printf("\nRetention Policy:\n")
	fmt.Printf("  Keep days: %d\n", cfg.RetentionPolicy.KeepDays)
	fmt.Printf("  Keep count: %d\n", cfg.RetentionPolicy.KeepCount)
	fmt.Printf("  Keep monthly: %d\n", cfg.RetentionPolicy.KeepMonthly)
	fmt.Printf("  Current backups: %d\n", len(backups))

	// Determine which backups would be deleted
	backupsToDelete := calculateBackupsToDelete(backups, cfg.RetentionPolicy)

	if len(backupsToDelete) == 0 {
		fmt.Println("\nNo backups need to be deleted according to the retention policy")
		return
	}

	// Show what would be deleted
	fmt.Printf("\nBackups that would be deleted (%d):\n", len(backupsToDelete))
	for _, backup := range backupsToDelete {
		fmt.Printf("  - %s (%s)\n", backup.ID, backup.Timestamp.Format("2006-01-02 15:04:05"))
	}

	// Confirm deletion if not in dry-run mode and not forced
	if !cleanupDryRun && !cleanupForce && !dryRun {
		fmt.Printf("\nWARNING: This will permanently delete %d backup(s).\n", len(backupsToDelete))
		fmt.Print("Are you sure you want to continue? (yes/no): ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			fmt.Println("Cleanup cancelled")
			return
		}
	}

	// Perform cleanup
	if cleanupDryRun || dryRun {
		fmt.Println("\nDRY RUN: No backups were actually deleted")
		return
	}

	fmt.Println("\nDeleting old backups...")
	if err := backupManager.CleanupOldBackups(); err != nil {
		fmt.Printf("Error during cleanup: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Cleanup completed successfully! %d backup(s) removed\n", len(backupsToDelete))
}

// calculateBackupsToDelete determines which backups should be deleted based on retention policy
func calculateBackupsToDelete(backups []config.BackupMetadata, policy config.RetentionPolicy) []config.BackupMetadata {
	var toDelete []config.BackupMetadata
	now := time.Now()

	// Apply day-based retention
	if policy.KeepDays > 0 {
		for _, backup := range backups {
			age := now.Sub(backup.Timestamp)
			if age.Hours() > float64(policy.KeepDays*24) {
				toDelete = append(toDelete, backup)
			}
		}
	}

	// Apply count-based retention
	if policy.KeepCount > 0 && len(backups) > policy.KeepCount {
		// Sort backups by timestamp (newest first)
		sortedBackups := make([]config.BackupMetadata, len(backups))
		copy(sortedBackups, backups)

		for i := range sortedBackups {
			for j := i + 1; j < len(sortedBackups); j++ {
				if sortedBackups[i].Timestamp.Before(sortedBackups[j].Timestamp) {
					sortedBackups[i], sortedBackups[j] = sortedBackups[j], sortedBackups[i]
				}
			}
		}

		// Mark old backups for deletion
		for i := policy.KeepCount; i < len(sortedBackups); i++ {
			// Check if not already marked for deletion
			found := false
			for _, existing := range toDelete {
				if existing.ID == sortedBackups[i].ID {
					found = true
					break
				}
			}
			if !found {
				toDelete = append(toDelete, sortedBackups[i])
			}
		}
	}

	// Apply monthly retention (simplified implementation)
	if policy.KeepMonthly > 0 {
		// Group backups by month
		monthlyBackups := make(map[string][]config.BackupMetadata)
		for _, backup := range backups {
			monthKey := backup.Timestamp.Format("2006-01")
			monthlyBackups[monthKey] = append(monthlyBackups[monthKey], backup)
		}

		// For each month, keep only the most recent backup
		for _, monthBackups := range monthlyBackups {
			if len(monthBackups) > 1 {
				// Find the most recent backup in this month
				var latestBackup config.BackupMetadata
				for _, backup := range monthBackups {
					if backup.Timestamp.After(latestBackup.Timestamp) {
						latestBackup = backup
					}
				}

				// Mark all but the latest for deletion
				for _, backup := range monthBackups {
					if backup.ID != latestBackup.ID {
						// Check if not already marked for deletion
						found := false
						for _, existing := range toDelete {
							if existing.ID == backup.ID {
								found = true
								break
							}
						}
						if !found {
							toDelete = append(toDelete, backup)
						}
					}
				}
			}
		}
	}

	return toDelete
}
