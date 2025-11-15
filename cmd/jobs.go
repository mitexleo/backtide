package cmd

import (
	"fmt"
	"os"

	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	jobsShowAll bool
)

// jobsCmd represents the jobs command
var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage backup jobs",
	Long: `Manage backup jobs configuration.

This command allows you to:
- List all configured backup jobs
- Show detailed information about jobs
- Enable or disable jobs

Examples:
  backtide jobs list
  backtide jobs show daily-backup
  backtide jobs enable weekly-backup
  backtide jobs disable test-job`,
}

// jobsListCmd represents the jobs list command
var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backup jobs",
	Long: `List all configured backup jobs with their status and basic information.

This command shows:
- Job name and description
- Enabled/disabled status
- Schedule information
- Storage configuration
- Number of directories`,
	Run: runJobsList,
}

// jobsShowCmd represents the jobs show command
var jobsShowCmd = &cobra.Command{
	Use:   "show [job-name]",
	Short: "Show detailed information about a backup job",
	Long: `Show detailed information about a specific backup job.

This command displays:
- Complete job configuration
- Directory paths and settings
- Retention policy
- Storage configuration
- Schedule details`,
	Args: cobra.ExactArgs(1),
	Run:  runJobsShow,
}

// jobsEnableCmd represents the jobs enable command
var jobsEnableCmd = &cobra.Command{
	Use:   "enable [job-name]",
	Short: "Enable a backup job",
	Long: `Enable a backup job to allow it to run during backup operations.

This will set the job's enabled flag to true, allowing it to be executed
when running 'backtide backup --all' or when specifically called.`,
	Args: cobra.ExactArgs(1),
	Run:  runJobsEnable,
}

// jobsDisableCmd represents the jobs disable command
var jobsDisableCmd = &cobra.Command{
	Use:   "disable [job-name]",
	Short: "Disable a backup job",
	Long: `Disable a backup job to prevent it from running during backup operations.

This will set the job's enabled flag to false, preventing it from being
executed even when running 'backtide backup --all'.`,
	Args: cobra.ExactArgs(1),
	Run:  runJobsDisable,
}

func init() {
	rootCmd.AddCommand(jobsCmd)
	jobsCmd.AddCommand(jobsListCmd)
	jobsCmd.AddCommand(jobsShowCmd)
	jobsCmd.AddCommand(jobsEnableCmd)
	jobsCmd.AddCommand(jobsDisableCmd)

	jobsListCmd.Flags().BoolVar(&jobsShowAll, "all", false, "show all jobs including disabled ones")
}

func runJobsList(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Backup Jobs ===")

	if len(cfg.Jobs) == 0 {
		fmt.Println("No backup jobs configured.")
		fmt.Println("Use 'backtide init' to create backup jobs.")
		return
	}

	for i, job := range cfg.Jobs {
		if !job.Enabled && !jobsShowAll {
			continue
		}

		fmt.Printf("\n%d. %s\n", i+1, job.Name)

		status := "❌ disabled"
		if job.Enabled {
			status = "✅ enabled"
		}
		fmt.Printf("   Status: %s\n", status)

		if job.Description != "" {
			fmt.Printf("   Description: %s\n", job.Description)
		}

		// Schedule information
		if job.Schedule.Enabled {
			fmt.Printf("   Schedule: %s (%s)\n", job.Schedule.Type, job.Schedule.Interval)
		} else {
			fmt.Printf("   Schedule: manual only\n")
		}

		// Directories
		fmt.Printf("   Directories: %d\n", len(job.Directories))
		for _, dir := range job.Directories {
			compression := ""
			if dir.Compression {
				compression = " (compressed)"
			}
			fmt.Printf("     - %s -> %s%s\n", dir.Path, dir.Name, compression)
		}

		// Storage configuration
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

		// Bucket reference
		if job.BucketID != "" {
			bucketName := "unknown"
			for _, bucket := range cfg.Buckets {
				if bucket.ID == job.BucketID {
					bucketName = bucket.Name
					break
				}
			}
			fmt.Printf("   S3 Bucket: %s (%s)\n", bucketName, job.BucketID)
		}

		// Retention policy
		fmt.Printf("   Retention: %d days, %d recent, %d monthly\n",
			job.Retention.KeepDays, job.Retention.KeepCount, job.Retention.KeepMonthly)

		// Docker configuration
		if job.SkipDocker {
			fmt.Printf("   Docker: containers will NOT be stopped\n")
		} else {
			fmt.Printf("   Docker: containers will be stopped during backup\n")
		}

		// S3 configuration
		if job.SkipS3 {
			fmt.Printf("   S3: operations will be skipped\n")
		}
	}

	enabledCount := 0
	for _, job := range cfg.Jobs {
		if job.Enabled {
			enabledCount++
		}
	}

	fmt.Printf("\n📊 Summary: %d total jobs, %d enabled\n", len(cfg.Jobs), enabledCount)
}

func runJobsShow(cmd *cobra.Command, args []string) {
	jobName := args[0]
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	var job *config.BackupJob
	for i, j := range cfg.Jobs {
		if j.Name == jobName {
			job = &cfg.Jobs[i]
			break
		}
	}

	if job == nil {
		fmt.Printf("Error: Job '%s' not found\n", jobName)
		fmt.Println("Use 'backtide jobs list' to see available jobs.")
		os.Exit(1)
	}

	fmt.Printf("=== Job Details: %s ===\n\n", job.Name)

	status := "❌ disabled"
	if job.Enabled {
		status = "✅ enabled"
	}
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("ID: %s\n", job.ID)

	if job.Description != "" {
		fmt.Printf("Description: %s\n", job.Description)
	}

	fmt.Println("\n--- Schedule ---")
	if job.Schedule.Enabled {
		fmt.Printf("Type: %s\n", job.Schedule.Type)
		fmt.Printf("Interval: %s\n", job.Schedule.Interval)
	} else {
		fmt.Println("Manual only (no automatic scheduling)")
	}

	fmt.Println("\n--- Directories ---")
	if len(job.Directories) == 0 {
		fmt.Println("No directories configured")
	} else {
		for i, dir := range job.Directories {
			compression := "no"
			if dir.Compression {
				compression = "yes"
			}
			fmt.Printf("%d. %s\n", i+1, dir.Name)
			fmt.Printf("   Path: %s\n", dir.Path)
			fmt.Printf("   Compression: %s\n", compression)
			fmt.Println()
		}
	}

	fmt.Println("--- Storage ---")
	if job.Storage.Local && job.Storage.S3 {
		fmt.Println("Type: Local + S3 (redundant storage)")
	} else if job.Storage.Local {
		fmt.Println("Type: Local only")
	} else if job.Storage.S3 {
		fmt.Println("Type: S3 only")
	} else {
		fmt.Println("Type: None configured")
	}

	if job.BucketID != "" {
		bucketName := "unknown"
		var bucketConfig *config.BucketConfig
		for _, bucket := range cfg.Buckets {
			if bucket.ID == job.BucketID {
				bucketName = bucket.Name
				bucketConfig = &bucket
				break
			}
		}
		fmt.Printf("S3 Bucket: %s (%s)\n", bucketName, job.BucketID)
		if bucketConfig != nil {
			fmt.Printf("  - Provider: %s\n", bucketConfig.Provider)
			fmt.Printf("  - Bucket: %s\n", bucketConfig.Bucket)
			fmt.Printf("  - Region: %s\n", bucketConfig.Region)
			fmt.Printf("  - Endpoint: %s\n", func() string {
				if bucketConfig.Endpoint == "" {
					return "AWS default"
				}
				return bucketConfig.Endpoint
			}())
			fmt.Printf("  - Mount Point: %s\n", bucketConfig.MountPoint)
		}
	}

	fmt.Println("\n--- Retention Policy ---")
	fmt.Printf("Keep days: %d\n", job.Retention.KeepDays)
	fmt.Printf("Keep count: %d\n", job.Retention.KeepCount)
	fmt.Printf("Keep monthly: %d\n", job.Retention.KeepMonthly)

	fmt.Println("\n--- Configuration ---")
	if job.SkipDocker {
		fmt.Println("Docker: Containers will NOT be stopped during backup")
	} else {
		fmt.Println("Docker: Containers will be stopped during backup")
	}

	if job.SkipS3 {
		fmt.Println("S3: Operations will be skipped")
	} else {
		fmt.Println("S3: Operations will be performed")
	}
}

func runJobsEnable(cmd *cobra.Command, args []string) {
	jobName := args[0]
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	var job *config.BackupJob
	for i, j := range cfg.Jobs {
		if j.Name == jobName {
			job = &cfg.Jobs[i]
			break
		}
	}

	if job == nil {
		fmt.Printf("Error: Job '%s' not found\n", jobName)
		fmt.Println("Use 'backtide jobs list' to see available jobs.")
		os.Exit(1)
	}

	if job.Enabled {
		fmt.Printf("Job '%s' is already enabled\n", jobName)
		return
	}

	job.Enabled = true

	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Job '%s' enabled successfully\n", jobName)
}

func runJobsDisable(cmd *cobra.Command, args []string) {
	jobName := args[0]
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	var job *config.BackupJob
	for i, j := range cfg.Jobs {
		if j.Name == jobName {
			job = &cfg.Jobs[i]
			break
		}
	}

	if job == nil {
		fmt.Printf("Error: Job '%s' not found\n", jobName)
		fmt.Println("Use 'backtide jobs list' to see available jobs.")
		os.Exit(1)
	}

	if !job.Enabled {
		fmt.Printf("Job '%s' is already disabled\n", jobName)
		return
	}

	job.Enabled = false

	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Job '%s' disabled successfully\n", jobName)
}
