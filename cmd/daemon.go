package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/commands"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

// daemonCmd represents the daemon command
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run Backtide as a scheduling daemon",
	Long: `Run Backtide as a continuously running daemon that manages backup job scheduling.

This daemon:
- Runs continuously as a background process
- Manages scheduling for ALL backup jobs internally
- Acts as "our own cron" - no external scheduling dependencies
- Automatically runs jobs according to their configured schedules
- Handles dynamic job configuration changes

The daemon reads the configuration file and runs each backup job
according to its individual schedule.`,
	Run: runDaemon,
}

func init() {
	// Register with command registry
	commands.RegisterCommand("daemon", daemonCmd)
}

func runDaemon(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Starting Backtide Scheduling Daemon...")
	fmt.Println("📋 Internal cron: Managing ALL backup job schedules")
	fmt.Println("💡 Use Ctrl+C to stop the daemon")
	fmt.Println()

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Load initial configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Create and start job scheduler
	scheduler := NewJobScheduler(cfg)
	if err := scheduler.Start(); err != nil {
		fmt.Printf("❌ Error starting scheduler: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Daemon started successfully!")
	fmt.Printf("📊 Monitoring %d backup jobs\n", len(cfg.Jobs))
	fmt.Println()

	// Wait for shutdown signal
	<-signalChan

	fmt.Println("\n🛑 Shutting down daemon...")
	scheduler.Stop()
	fmt.Println("✅ Daemon stopped gracefully")
}

// JobScheduler manages the scheduling and execution of ALL backup jobs
type JobScheduler struct {
	config   *config.BackupConfig
	stopChan chan struct{}
	ticker   *time.Ticker
	lastRun  map[string]time.Time
}

// NewJobScheduler creates a new job scheduler
func NewJobScheduler(cfg *config.BackupConfig) *JobScheduler {
	return &JobScheduler{
		config:   cfg,
		stopChan: make(chan struct{}),
		ticker:   time.NewTicker(1 * time.Minute), // Check every minute
		lastRun:  make(map[string]time.Time),
	}
}

// Start begins the scheduling loop
func (js *JobScheduler) Start() error {
	fmt.Println("⏰ Starting internal job scheduler...")

	// Start the scheduling loop in a goroutine
	go js.schedulingLoop()

	return nil
}

// Stop gracefully stops the scheduler
func (js *JobScheduler) Stop() {
	close(js.stopChan)
	js.ticker.Stop()
}

// schedulingLoop is the main scheduling logic
func (js *JobScheduler) schedulingLoop() {
	for {
		select {
		case <-js.stopChan:
			return
		case <-js.ticker.C:
			js.checkAndRunJobs()
		}
	}
}

// checkAndRunJobs checks if any jobs are due to run and executes them
func (js *JobScheduler) checkAndRunJobs() {
	// Reload configuration to pick up any changes
	configPath := getConfigPath()
	if cfg, err := config.LoadConfig(configPath); err == nil {
		js.config = cfg
	}

	now := time.Now()

	for _, job := range js.config.Jobs {
		if !job.Enabled || !job.Schedule.Enabled {
			continue
		}

		// Check if this job is due to run
		if js.isJobDue(job, now) {
			fmt.Printf("🔄 Running scheduled backup: %s\n", job.Name)
			go js.runBackupJob(job) // Run in goroutine to not block other jobs
			js.lastRun[job.Name] = now
		}
	}
}

// isJobDue checks if a job should run based on its schedule and last run time
func (js *JobScheduler) isJobDue(job config.BackupJob, now time.Time) bool {
	lastRun, exists := js.lastRun[job.Name]

	// If never run before, schedule it
	if !exists {
		return true
	}

	// Parse the schedule interval
	duration, err := parseScheduleInterval(job.Schedule.Interval)
	if err != nil {
		fmt.Printf("⚠️  Could not parse schedule for job %s: %v, defaulting to daily\n", job.Name, err)
		duration = 24 * time.Hour
	}

	// Check if enough time has passed since last run
	return now.Sub(lastRun) >= duration
}

// parseScheduleInterval parses human-readable schedule intervals
func parseScheduleInterval(interval string) (time.Duration, error) {
	// First try to parse as Go duration (e.g., "24h", "1h30m")
	if duration, err := time.ParseDuration(interval); err == nil {
		return duration, nil
	}

	// Handle human-readable intervals
	switch strings.ToLower(interval) {
	case "daily", "1d", "24h":
		return 24 * time.Hour, nil
	case "hourly", "1h":
		return time.Hour, nil
	case "weekly", "7d", "168h":
		return 7 * 24 * time.Hour, nil
	case "monthly", "30d", "720h":
		return 30 * 24 * time.Hour, nil
	case "15m", "15min":
		return 15 * time.Minute, nil
	case "30m", "30min":
		return 30 * time.Minute, nil
	default:
		return 0, fmt.Errorf("unknown schedule interval: %s", interval)
	}
}

// runBackupJob executes a specific backup job
func (js *JobScheduler) runBackupJob(job config.BackupJob) {
	fmt.Printf("   📦 Starting backup: %s\n", job.Name)

	// Run actual backup using the backup runner with background context
	backupRunner := backup.NewBackupRunner(*js.config)
	metadata, err := backupRunner.RunJob(context.Background(), job.Name)
	if err != nil {
		fmt.Printf("   ❌ Backup failed for job %s: %v\n", job.Name, err)
		return
	}

	fmt.Printf("   ✅ Completed backup: %s (ID: %s)\n", job.Name, metadata.ID)
	fmt.Printf("   📊 Backup size: %d bytes\n", metadata.TotalSize)

	// Log the execution
	fmt.Printf("   📝 Job %s completed at %s\n", job.Name, time.Now().Format("15:04:05"))
}
