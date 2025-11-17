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
	cleanupJobName string
	cleanupAll     bool
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up old backups based on retention policies",
	Long: `Clean up old backups according to configured retention policies.

This command will:
- Remove backups older than the configured keep_days
- Keep only the most recent backups up to keep_count
- Preserve monthly backups up to keep_monthly

Examples:
  backtide cleanup --job daily-backup
  backtide cleanup --all
  backtide cleanup (cleans up all jobs)`,
	Run: runCleanup,
}

func init() {
	cleanupCmd.Flags().StringVarP(&cleanupJobName, "job", "j", "", "clean up backups for specific job")
	cleanupCmd.Flags().BoolVarP(&cleanupAll, "all", "a", false, "clean up backups for all jobs")

	// Register with command registry
	commands.RegisterCommand("cleanup", cleanupCmd)
}

func runCleanup(cmd *cobra.Command, args []string) {
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

	backupRunner := backup.NewBackupRunner(*cfg)

	// Determine which jobs to clean up
	if cleanupJobName != "" {
		// Clean up specific job
		fmt.Printf("Cleaning up backups for job: %s\n", cleanupJobName)
		if err := backupRunner.RunJobCleanup(cleanupJobName); err != nil {
			fmt.Printf("Error cleaning up backups: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Cleanup completed for job: %s\n", cleanupJobName)
	} else if cleanupAll {
		// Clean up all jobs
		fmt.Println("Cleaning up backups for all jobs...")
		var cleanedJobs int
		var errors []string

		for _, job := range cfg.Jobs {
			if job.Enabled {
				if err := backupRunner.RunJobCleanup(job.Name); err != nil {
					errors = append(errors, fmt.Sprintf("job %s: %v", job.Name, err))
				} else {
					cleanedJobs++
				}
			}
		}

		if len(errors) > 0 {
			fmt.Printf("⚠️  Cleanup completed with %d errors:\n", len(errors))
			for _, err := range errors {
				fmt.Printf("   - %s\n", err)
			}
		}

		fmt.Printf("✅ Cleanup completed for %d jobs\n", cleanedJobs)
	} else {
		// Show available jobs and let user choose
		fmt.Println("Available backup jobs for cleanup:")
		for i, job := range cfg.Jobs {
			status := "❌ disabled"
			if job.Enabled {
				status = "✅ enabled"
			}
			fmt.Printf("%d. %s - %s\n", i+1, job.Name, status)
			if job.Description != "" {
				fmt.Printf("   Description: %s\n", job.Description)
			}
			fmt.Printf("   Retention: %d days, %d recent, %d monthly\n",
				job.Retention.KeepDays, job.Retention.KeepCount, job.Retention.KeepMonthly)
			fmt.Println()
		}

		fmt.Print("Select job to clean up (number) or 'all' for all enabled jobs: ")
		var choice string
		fmt.Scanln(&choice)

		if choice == "all" {
			fmt.Println("Cleaning up backups for all enabled jobs...")
			var cleanedJobs int
			var errors []string

			for _, job := range cfg.Jobs {
				if job.Enabled {
					if err := backupRunner.RunJobCleanup(job.Name); err != nil {
						errors = append(errors, fmt.Sprintf("job %s: %v", job.Name, err))
					} else {
						cleanedJobs++
					}
				}
			}

			if len(errors) > 0 {
				fmt.Printf("⚠️  Cleanup completed with %d errors:\n", len(errors))
				for _, err := range errors {
					fmt.Printf("   - %s\n", err)
				}
			}

			fmt.Printf("✅ Cleanup completed for %d jobs\n", cleanedJobs)
		} else {
			var jobIndex int
			if _, err := fmt.Sscanf(choice, "%d", &jobIndex); err == nil && jobIndex >= 1 && jobIndex <= len(cfg.Jobs) {
				job := cfg.Jobs[jobIndex-1]
				if !job.Enabled {
					fmt.Printf("Job '%s' is disabled. Enable it in the configuration first.\n", job.Name)
					return
				}
				fmt.Printf("Cleaning up backups for job: %s\n", job.Name)
				if err := backupRunner.RunJobCleanup(job.Name); err != nil {
					fmt.Printf("Error cleaning up backups: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("✅ Cleanup completed for job: %s\n", job.Name)
			} else {
				fmt.Println("Invalid selection.")
			}
		}
	}
}
