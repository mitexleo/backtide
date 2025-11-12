package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/mitexleo/backtide/internal/docker"
	"github.com/mitexleo/backtide/internal/s3fs"
	"github.com/spf13/cobra"
)

var (
	restoreTarget string
	restoreForce  bool
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore [backup-id]",
	Short: "Restore a backup to original locations",
	Long: `Restore a previously created backup to its original locations.

This command will:
1. Stop all running Docker containers (optional)
2. Mount S3 bucket if backup is stored there
3. Restore directories with original permissions
4. Restart Docker containers (if stopped)

If no backup ID is specified, the latest backup will be restored.`,
	Args: cobra.MaximumNArgs(1),
	Run:  runRestore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVarP(&restoreTarget, "target", "t", "", "specific backup ID to restore")
	restoreCmd.Flags().BoolVarP(&restoreForce, "force", "f", false, "force restore without confirmation")
}

func runRestore(cmd *cobra.Command, args []string) {
	fmt.Println("Starting restore operation...")

	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize managers
	dockerManager := docker.NewDockerManager("/var/lib/backtide/containers.json")
	s3Manager := s3fs.NewS3FSManager(cfg.S3Config)
	backupManager := backup.NewBackupManager(*cfg)

	// Determine which backup to restore
	backupID := restoreTarget
	if backupID == "" && len(args) > 0 {
		backupID = args[0]
	}

	if backupID == "" {
		// Find the latest backup
		backups, err := backupManager.ListBackups()
		if err != nil {
			fmt.Printf("Error listing backups: %v\n", err)
			os.Exit(1)
		}

		if len(backups) == 0 {
			fmt.Println("No backups found to restore")
			os.Exit(1)
		}

		// Find the most recent backup
		var latestBackup config.BackupMetadata
		for _, backup := range backups {
			if backup.Timestamp.After(latestBackup.Timestamp) {
				latestBackup = backup
			}
		}

		backupID = latestBackup.ID
		fmt.Printf("No backup ID specified, using latest: %s\n", backupID)
	}

	// Confirm restore operation
	if !restoreForce && !dryRun {
		fmt.Printf("\nWARNING: This will restore backup %s to original locations.\n", backupID)
		fmt.Println("This will overwrite existing files in the target directories.")
		fmt.Print("Are you sure you want to continue? (yes/no): ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			fmt.Println("Restore cancelled")
			return
		}
	}

	var stoppedContainers []config.DockerContainerInfo

	// Step 1: Stop Docker containers
	fmt.Println("\nStep 1: Managing Docker containers...")
	if dryRun {
		fmt.Println("DRY RUN: Would stop all running Docker containers")
	} else {
		if err := dockerManager.CheckDockerAvailable(); err != nil {
			fmt.Printf("Warning: Docker is not available: %v\n", err)
		} else {
			stoppedContainers, err = dockerManager.StopContainers()
			if err != nil {
				fmt.Printf("Error stopping containers: %v\n", err)
				// Continue with restore, but warn user
			} else {
				fmt.Printf("Stopped %d containers\n", len(stoppedContainers))
			}
		}
	}

	// Step 2: Ensure S3 is mounted if needed
	fmt.Println("\nStep 2: Setting up S3FS...")
	if dryRun {
		fmt.Println("DRY RUN: Would ensure S3 bucket is mounted")
	} else {
		// Check if backup exists locally first
		backupPath := filepath.Join(cfg.BackupPath, backupID)
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			// Backup not found locally, try to mount S3
			fmt.Println("Backup not found locally, attempting to mount S3...")

			// Install s3fs if needed
			if err := s3Manager.InstallS3FS(); err != nil {
				fmt.Printf("Error installing s3fs: %v\n", err)
				fmt.Printf("Backup %s not found locally and cannot access S3\n", backupID)
				os.Exit(1)
			}

			// Setup s3fs
			if err := s3Manager.SetupS3FS(); err != nil {
				fmt.Printf("Error setting up s3fs: %v\n", err)
				fmt.Printf("Backup %s not found locally and cannot access S3\n", backupID)
				os.Exit(1)
			}

			// Mount S3 bucket
			if err := s3Manager.MountS3FS(); err != nil {
				fmt.Printf("Error mounting S3 bucket: %v\n", err)
				fmt.Printf("Backup %s not found locally and cannot access S3\n", backupID)
				os.Exit(1)
			}
		}
	}

	// Step 3: Restore backup
	fmt.Println("\nStep 3: Restoring backup...")
	if dryRun {
		fmt.Printf("DRY RUN: Would restore backup %s\n", backupID)
	} else {
		if err := backupManager.RestoreBackup(backupID); err != nil {
			fmt.Printf("Error restoring backup: %v\n", err)
			// Try to restore containers before exiting
			if len(stoppedContainers) > 0 {
				fmt.Println("Attempting to restore Docker containers...")
				if err := dockerManager.RestoreContainers(); err != nil {
					fmt.Printf("Error restoring containers: %v\n", err)
				}
			}
			os.Exit(1)
		}
	}

	// Step 4: Restore Docker containers
	if len(stoppedContainers) > 0 {
		fmt.Println("\nStep 4: Restoring Docker containers...")
		if dryRun {
			fmt.Println("DRY RUN: Would restart previously stopped Docker containers")
		} else {
			if err := dockerManager.RestoreContainers(); err != nil {
				fmt.Printf("Error restoring containers: %v\n", err)
				// Don't exit, just warn
			}
		}
	}

	fmt.Println("\nRestore operation completed successfully!")
}
