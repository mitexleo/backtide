package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/mitexleo/backtide/internal/docker"
	"github.com/mitexleo/backtide/internal/s3fs"
	"github.com/mitexleo/backtide/internal/utils"
	"github.com/spf13/cobra"
)

var (
	backupSkipDocker bool
	backupSkipS3     bool
	backupJobName    string
	backupAllJobs    bool
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup [job-name]",
	Short: "Create a backup using configured backup jobs",
	Long: `Create backups using configured backup jobs.

If no job name is specified, runs the default backup job.
Use --all to run all enabled backup jobs.

Each backup job will:
1. Stop all running Docker containers (optional)
2. Mount S3 bucket using s3fs (optional)
3. Create compressed backups of specified directories
4. Restart Docker containers (if stopped)
5. Clean up old backups according to retention policy`,
	Args: cobra.MaximumNArgs(1),
	Run:  runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().BoolVar(&backupSkipDocker, "skip-docker", false, "skip Docker container management")
	backupCmd.Flags().BoolVar(&backupSkipS3, "skip-s3", false, "skip S3 operations")
	backupCmd.Flags().StringVarP(&backupJobName, "job", "j", "", "specific backup job to run")
	backupCmd.Flags().BoolVarP(&backupAllJobs, "all", "a", false, "run all enabled backup jobs")
}

func runBackup(cmd *cobra.Command, args []string) {
	fmt.Println("Starting backup operation...")

	// Determine which job to run
	var jobName string
	if len(args) > 0 {
		jobName = args[0]
	} else if backupJobName != "" {
		jobName = backupJobName
	}

	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize backup runner
	backupRunner := backup.NewBackupRunner(*cfg)

	// Check if running as root for certain operations
	if !backupSkipS3 {
		if err := utils.CheckRootPrivileges(); err != nil {
			fmt.Printf("Warning: %v. S3 operations will be skipped.\n", err)
			backupSkipS3 = true
		}
	}

	// Determine backup mode
	if backupAllJobs {
		// Run all enabled jobs
		fmt.Println("Running all enabled backup jobs...")
		if dryRun {
			fmt.Println("DRY RUN: Would run all enabled backup jobs")
			enabledJobs := backupRunner.GetEnabledJobs()
			for _, job := range enabledJobs {
				fmt.Printf("  - %s: %s\n", job.Name, job.Description)
			}
		} else {
			metadata, err := backupRunner.RunAllJobs()
			if err != nil {
				fmt.Printf("Error running backup jobs: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Successfully completed %d backup jobs\n", len(metadata))
		}
	} else if jobName != "" {
		// Run specific job
		fmt.Printf("Running backup job: %s\n", jobName)
		if dryRun {
			fmt.Printf("DRY RUN: Would run backup job '%s'\n", jobName)
		} else {
			_, err := backupRunner.RunJob(jobName)
			if err != nil {
				fmt.Printf("Error running backup job: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		// Run default/legacy backup
		runLegacyBackup(cfg, backupRunner)
	}
}

func runLegacyBackup(cfg *config.BackupConfig, backupRunner *backup.BackupRunner) {
	// Check if using legacy config
	if len(cfg.Jobs) == 0 && len(cfg.Directories) > 0 {
		fmt.Println("Using legacy configuration format...")
		// Fall back to original backup logic
		runLegacyBackupLogic(cfg)
		return
	}

	// Find default job
	var defaultJob *config.BackupJob
	jobs := backupRunner.ListJobs()
	for i, job := range jobs {
		if job.Name == "default-backup" || i == 0 {
			defaultJob = &jobs[i]
			break
		}
	}

	if defaultJob == nil {
		fmt.Println("Error: No backup jobs configured and no default job found")
		fmt.Println("Run 'backtide init' to configure backup jobs")
		os.Exit(1)
	}

	fmt.Printf("Running default backup job: %s\n", defaultJob.Name)
	if dryRun {
		fmt.Printf("DRY RUN: Would run backup job '%s'\n", defaultJob.Name)
	} else {
		_, err := backupRunner.RunJob(defaultJob.Name)
		if err != nil {
			fmt.Printf("Error running backup job: %v\n", err)
			os.Exit(1)
		}
	}
}

func runLegacyBackupLogic(cfg *config.BackupConfig) {

	// Initialize managers for legacy backup
	dockerManager := docker.NewDockerManager("/var/lib/backtide/containers.json")
	s3Manager := s3fs.NewS3FSManager(cfg.S3Config)
	backupManager := backup.NewBackupManager(*cfg)

	var stoppedContainers []config.DockerContainerInfo

	// Step 1: Stop Docker containers if enabled
	if !backupSkipDocker {
		fmt.Println("\nStep 1: Managing Docker containers...")
		if err := dockerManager.CheckDockerAvailable(); err != nil {
			fmt.Printf("Warning: Docker is not available: %v\n", err)
		} else {
			if dryRun {
				fmt.Println("DRY RUN: Would stop all running Docker containers")
			} else {
				stoppedContainers, err = dockerManager.StopContainers()
				if err != nil {
					fmt.Printf("Error stopping containers: %v\n", err)
					// Continue with backup, but warn user
				} else {
					fmt.Printf("Stopped %d containers\n", len(stoppedContainers))
				}
			}
		}
	}

	// Step 2: Setup and mount S3 if enabled
	if !backupSkipS3 {
		fmt.Println("\nStep 2: Setting up S3FS...")
		if dryRun {
			fmt.Println("DRY RUN: Would install and setup s3fs, mount S3 bucket")
		} else {
			// Install s3fs if needed
			if err := s3Manager.InstallS3FS(); err != nil {
				fmt.Printf("Error installing s3fs: %v\n", err)
				// Continue with local backup
				backupSkipS3 = true
			}

			// Setup s3fs
			if err := s3Manager.SetupS3FS(); err != nil {
				fmt.Printf("Error setting up s3fs: %v\n", err)
				backupSkipS3 = true
			}

			// Mount S3 bucket
			if err := s3Manager.MountS3FS(); err != nil {
				fmt.Printf("Error mounting S3 bucket: %v\n", err)
				backupSkipS3 = true
			}

			// Add to fstab for persistence
			if err := s3Manager.AddToFstab(); err != nil {
				fmt.Printf("Warning: Failed to add to fstab: %v\n", err)
			}
		}
	}

	// Step 3: Create backup
	fmt.Println("\nStep 3: Creating backup...")
	if dryRun {
		fmt.Println("DRY RUN: Would create backup of configured directories")
		for _, dir := range cfg.Directories {
			fmt.Printf("  - %s -> %s\n", dir.Path, dir.Name)
		}
	} else {
		// Ensure backup directory exists
		if err := utils.CreateDirectory(cfg.BackupPath); err != nil {
			fmt.Printf("Error creating backup directory: %v\n", err)
			os.Exit(1)
		}

		// Create the backup
		metadata, err := backupManager.CreateBackup()
		if err != nil {
			fmt.Printf("Error creating backup: %v\n", err)
			// Try to restore containers before exiting
			if len(stoppedContainers) > 0 {
				fmt.Println("Attempting to restore Docker containers...")
				if err := dockerManager.RestoreContainers(); err != nil {
					fmt.Printf("Error restoring containers: %v\n", err)
				}
			}
			os.Exit(1)
		}

		fmt.Printf("Backup created successfully: %s\n", metadata.ID)
		fmt.Printf("Total size: %d bytes\n", metadata.TotalSize)
		fmt.Printf("Directories backed up: %d\n", len(metadata.Directories))
	}

	// Step 4: Restore Docker containers if they were stopped
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

	// Step 5: Cleanup old backups
	fmt.Println("\nStep 5: Cleaning up old backups...")
	if dryRun {
		fmt.Println("DRY RUN: Would cleanup old backups according to retention policy")
	} else {
		if err := backupManager.CleanupOldBackups(); err != nil {
			fmt.Printf("Warning: Failed to cleanup old backups: %v\n", err)
		}
	}

	// Step 6: Unmount S3 if it was mounted and we're done
	if !backupSkipS3 {
		fmt.Println("\nStep 6: Cleaning up S3FS...")
		if dryRun {
			fmt.Println("DRY RUN: Would unmount S3 bucket")
		} else {
			// Note: We typically leave S3 mounted for future backups
			// Only unmount if explicitly requested or for specific scenarios
			if force {
				if err := s3Manager.UnmountS3FS(); err != nil {
					fmt.Printf("Warning: Failed to unmount S3: %v\n", err)
				}
			} else {
				fmt.Println("S3 bucket remains mounted for future backups")
			}
		}
	}

	fmt.Println("\nBackup operation completed successfully!")
}

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}

	// Try to find config file in common locations
	if found := config.FindConfigFile(); found != "" {
		return found
	}

	// Create default config if none exists
	defaultPath := filepath.Join(os.Getenv("HOME"), ".backtide.yaml")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		fmt.Printf("No configuration file found. Creating default config at %s\n", defaultPath)
		if err := config.CreateDefaultConfig(defaultPath); err != nil {
			fmt.Printf("Error creating default config: %v\n", err)
			os.Exit(1)
		}
	}

	return defaultPath
}
