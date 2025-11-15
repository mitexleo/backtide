package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	backupJobName string
	backupAll     bool
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Run backup operations",
	Long: `Run backup operations for configured jobs.

This command can:
- Run a specific backup job by name
- Run all enabled backup jobs
- Show backup progress and results

Examples:
  backtide backup --job daily-backup
  backtide backup --all
  backtide backup (runs all enabled jobs)`,
	Run: runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&backupJobName, "job", "j", "", "run specific backup job by name")
	backupCmd.Flags().BoolVarP(&backupAll, "all", "a", false, "run all enabled backup jobs")
}

func runBackup(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Check if we have any jobs configured
	if len(cfg.Jobs) == 0 {
		fmt.Println("No backup jobs configured.")
		fmt.Println("Use 'backtide init' to create backup jobs.")
		return
	}

	backupRunner := backup.NewBackupRunner(*cfg)

	// Determine which jobs to run
	if backupJobName != "" {
		// Run specific job
		fmt.Printf("Running backup job: %s\n", backupJobName)
		metadata, err := backupRunner.RunJob(backupJobName)
		if err != nil {
			fmt.Printf("Error running backup job: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Backup completed successfully: %s\n", metadata.ID)
	} else if backupAll || len(cfg.Jobs) == 1 {
		// Run all enabled jobs
		fmt.Println("Running all enabled backup jobs...")
		metadatas, err := backupRunner.RunAllJobs()
		if err != nil {
			fmt.Printf("Error running backup jobs: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ All backup jobs completed successfully (%d jobs)\n", len(metadatas))
	} else {
		// Show available jobs and let user choose
		fmt.Println("Available backup jobs:")
		for i, job := range cfg.Jobs {
			status := "❌ disabled"
			if job.Enabled {
				status = "✅ enabled"
			}
			fmt.Printf("%d. %s - %s\n", i+1, job.Name, status)
			if job.Description != "" {
				fmt.Printf("   Description: %s\n", job.Description)
			}
			fmt.Printf("   Directories: %d\n", len(job.Directories))
			fmt.Printf("   Storage: ")
			if job.Storage.Local && job.Storage.S3 {
				fmt.Printf("Local + S3\n")
			} else if job.Storage.Local {
				fmt.Printf("Local only\n")
			} else if job.Storage.S3 {
				fmt.Printf("S3 only\n")
			} else {
				fmt.Printf("None configured\n")
			}
			fmt.Println()
		}

		fmt.Print("Select job to run (number) or 'all' for all enabled jobs: ")
		var choice string
		fmt.Scanln(&choice)

		if choice == "all" {
			fmt.Println("Running all enabled backup jobs...")
			metadatas, err := backupRunner.RunAllJobs()
			if err != nil {
				fmt.Printf("Error running backup jobs: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ All backup jobs completed successfully (%d jobs)\n", len(metadatas))
		} else {
			var jobIndex int
			if _, err := fmt.Sscanf(choice, "%d", &jobIndex); err == nil && jobIndex >= 1 && jobIndex <= len(cfg.Jobs) {
				job := cfg.Jobs[jobIndex-1]
				if !job.Enabled {
					fmt.Printf("Job '%s' is disabled. Enable it in the configuration first.\n", job.Name)
					return
				}
				fmt.Printf("Running backup job: %s\n", job.Name)
				metadata, err := backupRunner.RunJob(job.Name)
				if err != nil {
					fmt.Printf("Error running backup job: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("✅ Backup completed successfully: %s\n", metadata.ID)
			} else {
				fmt.Println("Invalid selection.")
			}
		}
	}
}

// getConfigPath returns the configuration file path
func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}

	// Try to find config file in common locations
	if found := config.FindConfigFile(); found != "" {
		return found
	}

	// Create default config if none exists
	defaultPath := filepath.Join(os.Getenv("HOME"), ".backtide.toml")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		fmt.Printf("No configuration file found. Creating default config at %s\n", defaultPath)
		if err := config.CreateDefaultConfig(defaultPath); err != nil {
			fmt.Printf("Error creating default config: %v\n", err)
			os.Exit(1)
		}
	}

	return defaultPath
}
