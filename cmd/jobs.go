package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	jobsListDetailed bool
	jobsEnableAll    bool
	jobsDisableAll   bool
)

// jobsCmd represents the jobs command
var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage backup jobs",
	Long: `Manage and view backup jobs configuration.

This command allows you to:
- List all configured backup jobs
- Enable or disable specific jobs
- View job details and schedules
- Check job status and next run times`,
	Run: runJobs,
}

// jobsListCmd represents the jobs list command
var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backup jobs",
	Long:  `List all configured backup jobs with their status and schedules.`,
	Run:   runJobsList,
}

// jobsEnableCmd represents the jobs enable command
var jobsEnableCmd = &cobra.Command{
	Use:   "enable [job-name]",
	Short: "Enable a backup job",
	Long:  `Enable a specific backup job to run according to its schedule.`,
	Args:  cobra.ExactArgs(1),
	Run:   runJobsEnable,
}

// jobsDisableCmd represents the jobs disable command
var jobsDisableCmd = &cobra.Command{
	Use:   "disable [job-name]",
	Short: "Disable a backup job",
	Long:  `Disable a specific backup job to prevent it from running.`,
	Args:  cobra.ExactArgs(1),
	Run:   runJobsDisable,
}

// jobsStatusCmd represents the jobs status command
var jobsStatusCmd = &cobra.Command{
	Use:   "status [job-name]",
	Short: "Show job status and next run time",
	Long:  `Show detailed status information for a backup job including next scheduled run.`,
	Args:  cobra.MaximumNArgs(1),
	Run:   runJobsStatus,
}

func init() {
	rootCmd.AddCommand(jobsCmd)
	jobsCmd.AddCommand(jobsListCmd)
	jobsCmd.AddCommand(jobsEnableCmd)
	jobsCmd.AddCommand(jobsDisableCmd)
	jobsCmd.AddCommand(jobsStatusCmd)

	jobsListCmd.Flags().BoolVarP(&jobsListDetailed, "detailed", "d", false, "show detailed job information")
	jobsCmd.Flags().BoolVar(&jobsEnableAll, "enable-all", false, "enable all backup jobs")
	jobsCmd.Flags().BoolVar(&jobsDisableAll, "disable-all", false, "disable all backup jobs")
}

func runJobs(cmd *cobra.Command, args []string) {
	// Default to list if no subcommand specified
	runJobsList(cmd, args)
}

func runJobsList(cmd *cobra.Command, args []string) {
	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize backup runner
	backupRunner := backup.NewBackupRunner(*cfg)

	// Get all jobs
	jobs := backupRunner.ListJobs()

	if len(jobs) == 0 {
		fmt.Println("No backup jobs configured.")
		fmt.Println("Run 'backtide init' to create your first backup job.")
		return
	}

	fmt.Printf("Found %d backup job(s):\n\n", len(jobs))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if jobsListDetailed {
		fmt.Fprintln(w, "NAME\tENABLED\tSCHEDULE\tDESCRIPTION\tDIRECTORIES\tRETENTION")
		fmt.Fprintln(w, "----\t-------\t--------\t-----------\t-----------\t---------")

		for _, job := range jobs {
			scheduleInfo := "manual"
			if job.Schedule.Enabled {
				scheduleInfo = job.Schedule.Interval
				if job.Schedule.Type != "" {
					scheduleInfo = fmt.Sprintf("%s (%s)", scheduleInfo, job.Schedule.Type)
				}
			}

			enabledStatus := "❌"
			if job.Enabled {
				enabledStatus = "✅"
			}

			dirCount := fmt.Sprintf("%d dirs", len(job.Directories))
			retention := fmt.Sprintf("%d days", job.Retention.KeepDays)

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				job.Name,
				enabledStatus,
				scheduleInfo,
				job.Description,
				dirCount,
				retention,
			)
		}
	} else {
		fmt.Fprintln(w, "NAME\tENABLED\tSCHEDULE\tDESCRIPTION")
		fmt.Fprintln(w, "----\t-------\t--------\t-----------")

		for _, job := range jobs {
			scheduleInfo := "manual"
			if job.Schedule.Enabled {
				scheduleInfo = job.Schedule.Interval
			}

			enabledStatus := "❌"
			if job.Enabled {
				enabledStatus = "✅"
			}

			// Truncate description if too long
			description := job.Description
			if len(description) > 40 {
				description = description[:37] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				job.Name,
				enabledStatus,
				scheduleInfo,
				description,
			)
		}
	}

	w.Flush()

	// Show summary
	enabledCount := 0
	scheduledCount := 0
	for _, job := range jobs {
		if job.Enabled {
			enabledCount++
		}
		if job.Schedule.Enabled {
			scheduledCount++
		}
	}

	fmt.Printf("\nSummary: %d enabled, %d scheduled, %d total jobs\n", enabledCount, scheduledCount, len(jobs))
}

func runJobsEnable(cmd *cobra.Command, args []string) {
	jobName := args[0]

	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Find and enable the job
	found := false
	for i := range cfg.Jobs {
		if cfg.Jobs[i].Name == jobName {
			cfg.Jobs[i].Enabled = true
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("Error: Job '%s' not found\n", jobName)
		os.Exit(1)
	}

	// Save configuration
	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Backup job '%s' enabled\n", jobName)
}

func runJobsDisable(cmd *cobra.Command, args []string) {
	jobName := args[0]

	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Find and disable the job
	found := false
	for i := range cfg.Jobs {
		if cfg.Jobs[i].Name == jobName {
			cfg.Jobs[i].Enabled = false
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("Error: Job '%s' not found\n", jobName)
		os.Exit(1)
	}

	// Save configuration
	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Backup job '%s' disabled\n", jobName)
}

func runJobsStatus(cmd *cobra.Command, args []string) {
	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize backup runner
	backupRunner := backup.NewBackupRunner(*cfg)

	var jobs []config.BackupJob
	if len(args) > 0 {
		// Show status for specific job
		jobName := args[0]
		job, err := backupRunner.GetJob(jobName)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		jobs = []config.BackupJob{*job}
	} else {
		// Show status for all jobs
		jobs = backupRunner.ListJobs()
	}

	if len(jobs) == 0 {
		fmt.Println("No backup jobs configured.")
		return
	}

	for _, job := range jobs {
		fmt.Printf("\n=== Job: %s ===\n", job.Name)
		fmt.Printf("Description: %s\n", job.Description)

		// Status
		status := "❌ DISABLED"
		if job.Enabled {
			status = "✅ ENABLED"
		}
		fmt.Printf("Status: %s\n", status)

		// Schedule
		if job.Schedule.Enabled {
			fmt.Printf("Schedule: %s (%s)\n", job.Schedule.Interval, job.Schedule.Type)

			// Calculate next run (simplified)
			nextRun, err := backupRunner.GetNextScheduledRun(job)
			if err == nil && !nextRun.IsZero() {
				fmt.Printf("Next run: %s (in %s)\n",
					nextRun.Format("2006-01-02 15:04:05"),
					time.Until(nextRun).Round(time.Minute),
				)
			}
		} else {
			fmt.Printf("Schedule: Manual only\n")
		}

		// Configuration
		fmt.Printf("Directories: %d\n", len(job.Directories))
		for _, dir := range job.Directories {
			compression := "compressed"
			if !dir.Compression {
				compression = "uncompressed"
			}
			fmt.Printf("  - %s → %s (%s)\n", dir.Path, dir.Name, compression)
		}

		fmt.Printf("Docker: ")
		if job.SkipDocker {
			fmt.Printf("containers will NOT be stopped\n")
		} else {
			fmt.Printf("containers will be stopped during backup\n")
		}

		fmt.Printf("S3 Storage: ")
		if job.SkipS3 {
			fmt.Printf("disabled\n")
		} else {
			fmt.Printf("enabled (%s)\n", job.S3Config.Bucket)
		}

		fmt.Printf("Retention: %d days, keep %d recent, %d monthly\n",
			job.Retention.KeepDays, job.Retention.KeepCount, job.Retention.KeepMonthly)
	}
}
