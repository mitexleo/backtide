package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
- Add new backup jobs
- Show detailed information about jobs
- Enable or disable jobs

Examples:
  backtide jobs list
  backtide jobs add
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

// jobsAddCmd represents the jobs add command
var jobsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new backup job",
	Long: `Add a new backup job with interactive configuration.

This command will guide you through creating a complete backup job
with scheduling, retention policies, and storage configuration.`,
	Run: runJobsAdd,
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
	jobsCmd.AddCommand(jobsListCmd)
	jobsCmd.AddCommand(jobsShowCmd)
	jobsCmd.AddCommand(jobsEnableCmd)
	jobsCmd.AddCommand(jobsDisableCmd)
	jobsCmd.AddCommand(jobsAddCmd)

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
		fmt.Println("Use 'backtide jobs add' to create backup jobs.")
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

func runJobsAdd(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Add Backup Job ===")
	fmt.Println("Let's create a new backup job with scheduling and retention.")
	fmt.Println()

	// Create a complete backup job
	job := configureBackupJobInteractive(configPath, cfg)
	cfg.Jobs = append(cfg.Jobs, job)

	// Save configuration with new job
	fmt.Printf("💾 Saving configuration with new job to: %s\n", configPath)
	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("❌ Error saving configuration: %v\n", err)
		fmt.Println("💡 You may need to run with sudo for system configuration")
		fmt.Println("   Try: sudo backtide jobs add")
		os.Exit(1)
	}

	fmt.Printf("\n🎉 Backup job '%s' added successfully!\n", job.Name)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Test the backup: backtide backup --dry-run")
	fmt.Println("2. Set up automated backups: backtide systemd install")
	fmt.Println("3. Run the backup: backtide backup")
}

func configureBackupJobInteractive(configPath string, currentConfig *config.BackupConfig) config.BackupJob {
	reader := bufio.NewReader(os.Stdin)
	job := config.BackupJob{
		ID:         generateJobID(),
		Enabled:    true,
		SkipDocker: false,
		SkipS3:     false,
		Storage: config.StorageConfig{
			Local: false,
			S3:    false,
		},
	}

	// Job name and description
	fmt.Print("Backup job name (e.g., 'daily-docker-backup'): ")
	name, _ := reader.ReadString('\n')
	job.Name = strings.TrimSpace(name)
	if job.Name == "" {
		job.Name = "default-backup"
	}
	fmt.Printf("Job ID: %s\n", job.ID)

	fmt.Print("Job description: ")
	desc, _ := reader.ReadString('\n')
	job.Description = strings.TrimSpace(desc)

	// Schedule configuration
	fmt.Println("\n=== Backup Schedule ===")
	fmt.Println("When should this backup run automatically?")
	fmt.Println("1. Daily (at 2 AM)")
	fmt.Println("2. Weekly (Sunday at 2 AM)")
	fmt.Println("3. Monthly (1st at 2 AM)")
	fmt.Println("4. Custom cron schedule")
	fmt.Println("5. Manual only (no automatic scheduling)")
	fmt.Print("Choose schedule (1-5): ")

	scheduleChoice, _ := reader.ReadString('\n')
	scheduleChoice = strings.TrimSpace(scheduleChoice)

	switch scheduleChoice {
	case "1":
		job.Schedule = config.ScheduleConfig{
			Type:     "systemd",
			Interval: "daily",
			Enabled:  true,
		}
		fmt.Println("✅ Set to run daily at 2 AM")
	case "2":
		job.Schedule = config.ScheduleConfig{
			Type:     "systemd",
			Interval: "weekly",
			Enabled:  true,
		}
		fmt.Println("✅ Set to run weekly on Sunday at 2 AM")
	case "3":
		job.Schedule = config.ScheduleConfig{
			Type:     "systemd",
			Interval: "monthly",
			Enabled:  true,
		}
		fmt.Println("✅ Set to run monthly on the 1st at 2 AM")
	case "4":
		fmt.Print("Enter cron expression (e.g., '0 2 * * *' for daily at 2 AM): ")
		cronExpr, _ := reader.ReadString('\n')
		cronExpr = strings.TrimSpace(cronExpr)
		if cronExpr != "" {
			job.Schedule = config.ScheduleConfig{
				Type:     "cron",
				Interval: cronExpr,
				Enabled:  true,
			}
			fmt.Printf("✅ Set to run with cron: %s\n", cronExpr)
		} else {
			job.Schedule.Enabled = false
			fmt.Println("❌ No schedule set (manual only)")
		}
	case "5":
		job.Schedule.Enabled = false
		fmt.Println("✅ Set to manual mode (no automatic scheduling)")
	default:
		job.Schedule.Enabled = false
		fmt.Println("❌ Invalid choice, set to manual mode")
	}

	// Retention policy
	fmt.Println("\n=== Retention Policy ===")
	fmt.Println("How long should we keep backups?")
	fmt.Print("Keep backups for how many days? [30]: ")
	daysInput, _ := reader.ReadString('\n')
	daysInput = strings.TrimSpace(daysInput)
	keepDays := 30
	if daysInput != "" {
		if days, err := strconv.Atoi(daysInput); err == nil && days > 0 {
			keepDays = days
		}
	}

	fmt.Print("Keep how many recent backups? [10]: ")
	countInput, _ := reader.ReadString('\n')
	countInput = strings.TrimSpace(countInput)
	keepCount := 10
	if countInput != "" {
		if count, err := strconv.Atoi(countInput); err == nil && count > 0 {
			keepCount = count
		}
	}

	fmt.Print("Keep how many monthly backups? [6]: ")
	monthlyInput, _ := reader.ReadString('\n')
	monthlyInput = strings.TrimSpace(monthlyInput)
	keepMonthly := 6
	if monthlyInput != "" {
		if monthly, err := strconv.Atoi(monthlyInput); err == nil && monthly > 0 {
			keepMonthly = monthly
		}
	}

	job.Retention = config.RetentionPolicy{
		KeepDays:    keepDays,
		KeepCount:   keepCount,
		KeepMonthly: keepMonthly,
	}
	fmt.Printf("✅ Retention: %d days, %d recent, %d monthly\n", keepDays, keepCount, keepMonthly)

	// Storage location configuration
	fmt.Println("\n=== Storage Location Configuration ===")
	fmt.Println("Where should backups be stored?")
	fmt.Println("1. S3 only (recommended - prevents local disk exhaustion)")
	fmt.Println("2. Local only (no S3)")
	fmt.Println("3. Both S3 and local (redundant storage)")
	fmt.Print("Choose storage location (1-3): ")

	storageChoice, _ := reader.ReadString('\n')
	storageChoice = strings.TrimSpace(storageChoice)

	switch storageChoice {
	case "1":
		job.Storage.S3 = true
		job.Storage.Local = false
		fmt.Println("✅ Backups will be stored in S3 only")
		if len(currentConfig.Buckets) > 0 {
			bucketID := configureBucketForJob(configPath, currentConfig)
			job.BucketID = bucketID
		} else {
			fmt.Println("⚠️  No S3 buckets configured. You can add one later with 'backtide s3 add'")
		}
	case "2":
		job.Storage.S3 = false
		job.Storage.Local = true
		job.SkipS3 = true
		fmt.Println("✅ Backups will be stored locally only")
	case "3":
		job.Storage.S3 = true
		job.Storage.Local = true
		fmt.Println("✅ Backups will be stored in both S3 and locally")
		if len(currentConfig.Buckets) > 0 {
			bucketID := configureBucketForJob(configPath, currentConfig)
			job.BucketID = bucketID
		} else {
			fmt.Println("⚠️  No S3 buckets configured. You can add one later with 'backtide s3 add'")
		}
	default:
		// Default to S3 only for safety
		job.Storage.S3 = true
		job.Storage.Local = false
		fmt.Println("❌ Invalid choice, defaulting to S3 only")
		if len(currentConfig.Buckets) > 0 {
			bucketID := configureBucketForJob(configPath, currentConfig)
			job.BucketID = bucketID
		} else {
			fmt.Println("⚠️  No S3 buckets configured. You can add one later with 'backtide s3 add'")
		}
	}

	// Docker configuration
	fmt.Println("\n=== Docker Configuration ===")
	fmt.Print("Stop Docker containers during backup? (Y/n): ")
	stopDocker, _ := reader.ReadString('\n')
	stopDocker = strings.TrimSpace(stopDocker)

	if stopDocker == "" || strings.ToLower(stopDocker) == "y" {
		job.SkipDocker = false
		fmt.Println("✅ Docker containers will be stopped during backup")
	} else {
		job.SkipDocker = true
		fmt.Println("✅ Docker containers will NOT be stopped")
	}

	// Directory configuration
	fmt.Println("\n=== Directory Configuration ===")
	job.Directories = configureDirectoriesInteractive()

	return job
}

func configureDirectoriesInteractive() []config.DirectoryConfig {
	reader := bufio.NewReader(os.Stdin)
	var directories []config.DirectoryConfig

	fmt.Println("Configure directories to backup:")
	fmt.Println("Enter directory paths one by one (empty line to finish)")

	for {
		fmt.Print("Directory path (e.g., /var/lib/docker/volumes): ")
		path, _ := reader.ReadString('\n')
		path = strings.TrimSpace(path)

		if path == "" {
			break
		}

		// Check if directory exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("⚠️  Warning: Directory does not exist: %s\n", path)
			fmt.Print("Continue anyway? (y/N): ")
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			if confirm != "y" && confirm != "yes" {
				continue
			}
		}

		fmt.Print("Backup name (e.g., 'docker-volumes'): ")
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name == "" {
			name = filepath.Base(path)
		}

		fmt.Print("Enable compression? (Y/n): ")
		compression, _ := reader.ReadString('\n')
		compression = strings.TrimSpace(compression)
		enableCompression := true
		if compression == "n" || compression == "no" {
			enableCompression = false
		}

		directory := config.DirectoryConfig{
			Path:        path,
			Name:        name,
			Compression: enableCompression,
		}

		directories = append(directories, directory)
		fmt.Printf("✅ Added: %s -> %s (compression: %v)\n", path, name, enableCompression)
	}

	if len(directories) == 0 {
		fmt.Println("⚠️  No directories configured. You can add them later in the configuration file.")
	}

	return directories
}

func generateJobID() string {
	return fmt.Sprintf("job-%s", time.Now().Format("20060102-150405"))
}

func configureBucketForJob(configPath string, currentConfig *config.BackupConfig) string {
	reader := bufio.NewReader(os.Stdin)

	// Check for existing buckets
	existingBuckets := currentConfig.Buckets
	if len(existingBuckets) > 0 {
		fmt.Println("\n=== Existing Bucket Configurations ===")
		fmt.Println("Choose from existing buckets or create new:")
		fmt.Println("0. Create new bucket configuration")

		for i, bucket := range existingBuckets {
			fmt.Printf("%d. %s - %s (%s)\n",
				i+1, bucket.Name, bucket.Bucket, bucket.Provider)
		}

		fmt.Print("Choose bucket (0-", len(existingBuckets), "): ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choiceIndex, err := strconv.Atoi(choice); err == nil && choiceIndex > 0 && choiceIndex <= len(existingBuckets) {
			// User selected existing bucket
			selectedBucket := existingBuckets[choiceIndex-1]
			fmt.Printf("✅ Using existing bucket: %s (%s)\n", selectedBucket.Name, selectedBucket.Bucket)

			return selectedBucket.ID
		}
		// User chose to create new bucket (choice 0 or invalid)
		fmt.Println("Creating new bucket configuration...")
	}

	// No existing buckets or user wants to create new one
	fmt.Println("No existing buckets found or creating new bucket...")

	// Configure new bucket (basic setup without credentials)
	newBucket := configureBasicBucketForInit()
	currentConfig.Buckets = append(currentConfig.Buckets, newBucket)

	fmt.Printf("✅ New bucket configuration '%s' added!\n", newBucket.Name)
	fmt.Println("💡 Note: You'll need to update the bucket credentials later using 'backtide s3 edit'")

	return newBucket.ID
}

func configureBasicBucketForInit() config.BucketConfig {
	reader := bufio.NewReader(os.Stdin)
	bucket := config.BucketConfig{}

	// Generate a unique ID
	bucket.ID = generateBucketID()

	fmt.Print("Bucket name (display name): ")
	name, _ := reader.ReadString('\n')
	bucket.Name = strings.TrimSpace(name)
	if bucket.Name == "" {
		bucket.Name = "default-bucket"
	}

	fmt.Print("Description (optional): ")
	desc, _ := reader.ReadString('\n')
	bucket.Description = strings.TrimSpace(desc)

	// Provider name
	fmt.Print("Provider name (e.g., AWS S3, Backblaze B2, MinIO): ")
	provider, _ := reader.ReadString('\n')
	bucket.Provider = strings.TrimSpace(provider)

	// Bucket name
	fmt.Print("S3 Bucket name: ")
	s3Bucket, _ := reader.ReadString('\n')
	bucket.Bucket = strings.TrimSpace(s3Bucket)
	if bucket.Bucket == "" {
		bucket.Bucket = "my-backup-bucket"
	}

	// Region
	fmt.Print("Region (leave empty if not applicable): ")
	region, _ := reader.ReadString('\n')
	bucket.Region = strings.TrimSpace(region)

	// Path style - set smart defaults based on provider
	providerLower := strings.ToLower(bucket.Provider)
	defaultPathStyle := strings.Contains(providerLower, "backblaze") ||
		strings.Contains(providerLower, "b2") ||
		strings.Contains(providerLower, "minio")

	if defaultPathStyle {
		fmt.Print("Use path-style endpoints? (recommended: Y) (Y/n): ")
	} else {
		fmt.Print("Use path-style endpoints? (recommended: n) (y/N): ")
	}
	pathStyleInput, _ := reader.ReadString('\n')

	// Endpoint
	fmt.Print("Endpoint URL (leave empty for AWS default): ")
	endpointInput, _ := reader.ReadString('\n')
	bucket.Endpoint = strings.TrimSpace(endpointInput)

	// Set path style with smart default
	pathStyleInput = strings.TrimSpace(strings.ToLower(pathStyleInput))
	if pathStyleInput == "y" || pathStyleInput == "yes" {
		bucket.UsePathStyle = true
	} else if pathStyleInput == "n" || pathStyleInput == "no" {
		bucket.UsePathStyle = false
	} else {
		// Use default based on provider
		bucket.UsePathStyle = defaultPathStyle
	}

	// Mount point
	fmt.Print("Mount point (e.g., /mnt/s3backup): ")
	mountPoint, _ := reader.ReadString('\n')
	bucket.MountPoint = strings.TrimSpace(mountPoint)

	// Skip credentials for now - they can be added later
	bucket.AccessKey = "YOUR_ACCESS_KEY_HERE"
	bucket.SecretKey = "YOUR_SECRET_KEY_HERE"

	fmt.Printf("✅ S3 bucket configuration for %s completed!\n", bucket.Provider)
	fmt.Println("💡 Note: You'll need to update the bucket credentials later using 'backtide s3 edit'")

	return bucket
}
