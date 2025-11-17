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
	initForce           bool
	initExamples        bool
	initSkipInteractive bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Backtide system configuration",
	Long: `Initialize Backtide system configuration.

This command creates the system-wide configuration in /etc/backtide/
and optionally creates your first backup job.

Examples:
  backtide init                    # Interactive setup
  backtide init --skip-interactive # Create config only, no job setup
  backtide init --examples         # Create example configuration
  backtide init --force            # Overwrite existing configuration`,
	Run: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing configuration")
	initCmd.Flags().BoolVar(&initExamples, "examples", false, "create example configuration")
	initCmd.Flags().BoolVar(&initSkipInteractive, "skip-interactive", false, "skip interactive job setup")
}

func runInit(cmd *cobra.Command, args []string) {
	fmt.Println("Initializing backtide...")

	// Determine config file path - always use system location for init
	configPath := "/etc/backtide/config.toml"
	if cfgFile != "" {
		configPath = cfgFile
	}

	// Check if config file already exists
	var existingConfig *config.BackupConfig
	if _, err := os.Stat(configPath); err == nil {
		if !initForce {
			fmt.Printf("Configuration file already exists: %s\n", configPath)
			fmt.Println("Use --force to overwrite existing configuration")
			fmt.Println("Or use 'backtide jobs add' to add jobs to existing configuration")
			os.Exit(1)
		}
		// Load existing config to preserve settings
		existingConfig, _ = config.LoadConfig(configPath)
	}

	// Create configuration directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating configuration directory: %v\n", err)
		fmt.Println("💡 You may need to run with sudo for system configuration")
		fmt.Println("   Try: sudo backtide init")
		os.Exit(1)
	}

	// Create default configuration
	var defaultConfig *config.BackupConfig
	if existingConfig != nil {
		// Use existing config as base - create a copy to avoid modifying the original
		defaultConfig = &config.BackupConfig{
			Jobs:       append([]config.BackupJob{}, existingConfig.Jobs...),
			Buckets:    append([]config.BucketConfig{}, existingConfig.Buckets...),
			BackupPath: existingConfig.BackupPath,
			TempPath:   existingConfig.TempPath,
		}
	} else if initExamples {
		defaultConfig = createExampleConfig()
	} else {
		defaultConfig = config.DefaultConfig()
	}

	// Save configuration to system location FIRST
	fmt.Printf("💾 Saving configuration to: %s\n", configPath)
	if err := config.SaveConfig(defaultConfig, configPath); err != nil {
		fmt.Printf("❌ Error saving configuration: %v\n", err)
		fmt.Println("💡 You may need to run with sudo for system configuration")
		fmt.Println("   Try: sudo backtide init")
		os.Exit(1)
	}

	// Create necessary system directories
	fmt.Println("📁 Creating system directories...")
	dirs := []string{
		"/etc/backtide",
		"/etc/backtide/s3-credentials",
		"/var/lib/backtide",
		"/var/log/backtide",
		"/mnt/backup",
		"/mnt/s3backup",
		"/tmp/backtide",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("  Warning: Could not create %s: %v\n", dir, err)
		} else {
			fmt.Printf("  Created: %s\n", dir)
		}
	}

	// Interactive job configuration setup (only if not skipped)
	if !dryRun && !initSkipInteractive {
		fmt.Println("\n=== Backup Job Setup ===")
		fmt.Println("Would you like to create your first backup job now?")
		fmt.Println("1. Yes, create a backup job with scheduling")
		fmt.Println("2. No, just create the configuration")
		fmt.Print("Choose option (1-2): ")

		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "1" {
			fmt.Println("\nLet's create your first backup job with scheduling and retention.")
			fmt.Println()

			// Create a complete backup job
			job := configureBackupJobInteractive(configPath, defaultConfig)
			// Add to existing jobs
			defaultConfig.Jobs = append(defaultConfig.Jobs, job)

			// Save configuration with new job
			fmt.Printf("💾 Saving configuration with new job to: %s\n", configPath)
			if err := config.SaveConfig(defaultConfig, configPath); err != nil {
				fmt.Printf("❌ Error saving configuration: %v\n", err)
				fmt.Println("💡 You may need to run with sudo for system configuration")
				fmt.Println("   Try: sudo backtide init")
				os.Exit(1)
			}

			fmt.Printf("\n🎉 Backup job '%s' configured successfully!\n", job.Name)
		} else {
			fmt.Println("\n✅ Configuration created without backup job.")
			fmt.Println("   Use 'backtide jobs add' to add backup jobs later.")
		}
	}

	fmt.Printf("\n✅ Configuration created successfully: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file with your specific settings")
	fmt.Println("2. Set S3 credentials and bucket information")
	fmt.Println("3. Configure directories you want to backup")
	fmt.Println("4. Test the backup: backtide backup --dry-run")
	fmt.Println("5. Set up automated backups: backtide systemd install")
	fmt.Println("\nExample commands:")
	fmt.Println("  backtide backup                    # Run backup")
	fmt.Println("  backtide list                      # List backups")
	fmt.Println("  backtide jobs add                  # Add backup job")
	fmt.Println("  backtide s3 add                    # Add S3 bucket")
	fmt.Println("  backtide systemd install           # Set up systemd service")
	fmt.Println("  backtide cron install              # Set up cron job")
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
		bucketID := configureBucketForJob(configPath, currentConfig)
		job.BucketID = bucketID
	case "2":
		job.Storage.S3 = false
		job.Storage.Local = true
		job.SkipS3 = true
		fmt.Println("✅ Backups will be stored locally only")
	case "3":
		job.Storage.S3 = true
		job.Storage.Local = true
		fmt.Println("✅ Backups will be stored in both S3 and locally")
		bucketID := configureBucketForJob(configPath, currentConfig)
		job.BucketID = bucketID
	default:
		// Default to S3 only for safety
		job.Storage.S3 = true
		job.Storage.Local = false
		fmt.Println("❌ Invalid choice, defaulting to S3 only")
		bucketID := configureBucketForJob(configPath, currentConfig)
		job.BucketID = bucketID
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

func getExistingBuckets(configPath string) []config.BucketConfig {
	var existingBuckets []config.BucketConfig

	// Load current configuration
	currentConfig, err := config.LoadConfig(configPath)
	if err != nil {
		return existingBuckets
	}

	// Return all configured buckets
	return currentConfig.Buckets
}

func getExistingBucketsFromConfig(currentConfig *config.BackupConfig) []config.BucketConfig {
	if currentConfig == nil {
		return []config.BucketConfig{}
	}
	return currentConfig.Buckets
}

func configureBucketForJob(configPath string, currentConfig *config.BackupConfig) string {
	reader := bufio.NewReader(os.Stdin)

	// Check for existing buckets
	existingBuckets := getExistingBucketsFromConfig(currentConfig)
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

// configureBasicBucketForInit creates a basic bucket configuration without credentials
func configureBasicBucketForInit() config.BucketConfig {
	reader := bufio.NewReader(os.Stdin)
	bucket := config.BucketConfig{
		MountPoint: "/mnt/s3backup",
	}

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

	fmt.Println("\nS3 Provider Options:")
	fmt.Println("1. AWS S3")
	fmt.Println("2. Backblaze B2")
	fmt.Println("3. Wasabi")
	fmt.Println("4. DigitalOcean Spaces")
	fmt.Println("5. MinIO")
	fmt.Println("6. Other S3-compatible provider")
	fmt.Print("Choose provider (1-6): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var providerName string
	var defaultEndpoint string
	var recommendedPathStyle bool

	switch choice {
	case "1":
		providerName = "AWS S3"
		recommendedPathStyle = false
		fmt.Print("AWS Region (e.g., us-east-1): ")
		region, _ := reader.ReadString('\n')
		bucket.Region = strings.TrimSpace(region)

	case "2":
		providerName = "Backblaze B2"
		defaultEndpoint = "https://s3.us-west-002.backblazeb2.com"
		recommendedPathStyle = true
		bucket.Region = ""

	case "3":
		providerName = "Wasabi"
		defaultEndpoint = "https://s3.wasabisys.com"
		recommendedPathStyle = false
		fmt.Print("Wasabi Region (e.g., us-east-1): ")
		region, _ := reader.ReadString('\n')
		bucket.Region = strings.TrimSpace(region)
	case "4":
		providerName = "DigitalOcean Spaces"
		defaultEndpoint = "https://nyc3.digitaloceanspaces.com"
		recommendedPathStyle = false
		fmt.Print("DO Region (e.g., nyc3): ")
		region, _ := reader.ReadString('\n')
		bucket.Region = strings.TrimSpace(region)
	case "5":
		providerName = "MinIO"
		defaultEndpoint = "http://localhost:9000"
		recommendedPathStyle = true
		bucket.Region = ""

	case "6":
		providerName = "Other S3-compatible"
		recommendedPathStyle = false
		fmt.Print("Endpoint URL (e.g., https://s3.example.com): ")
		endpoint, _ := reader.ReadString('\n')
		defaultEndpoint = strings.TrimSpace(endpoint)
	default:
		fmt.Println("Invalid choice, using AWS S3 defaults")
		providerName = "AWS S3"
		recommendedPathStyle = false
	}

	bucket.Provider = providerName
	fmt.Printf("\nConfiguring %s...\n", providerName)

	// Bucket name
	fmt.Print("S3 Bucket name: ")
	s3Bucket, _ := reader.ReadString('\n')
	bucket.Bucket = strings.TrimSpace(s3Bucket)
	if bucket.Bucket == "" {
		bucket.Bucket = "my-backup-bucket"
	}

	// Endpoint
	if defaultEndpoint != "" {
		fmt.Printf("Endpoint [%s]: ", defaultEndpoint)
		endpoint, _ := reader.ReadString('\n')
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			bucket.Endpoint = defaultEndpoint
		} else {
			bucket.Endpoint = endpoint
		}
	} else {
		fmt.Print("Endpoint (leave empty for AWS default): ")
		endpoint, _ := reader.ReadString('\n')
		bucket.Endpoint = strings.TrimSpace(endpoint)
	}

	// Path style
	fmt.Printf("Use path-style endpoints? (recommended: %v) (y/N): ", recommendedPathStyle)
	pathStyle, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(pathStyle)) == "y" {
		bucket.UsePathStyle = true
	} else {
		bucket.UsePathStyle = false
	}

	// Mount point
	fmt.Printf("Mount point [%s]: ", bucket.MountPoint)
	mountPoint, _ := reader.ReadString('\n')
	mountPoint = strings.TrimSpace(mountPoint)
	if mountPoint != "" {
		bucket.MountPoint = mountPoint
	}

	// Skip credentials for now - they can be added later
	bucket.AccessKey = "YOUR_ACCESS_KEY_HERE"
	bucket.SecretKey = "YOUR_SECRET_KEY_HERE"

	fmt.Printf("✅ S3 bucket configuration for %s completed!\n", providerName)
	fmt.Println("💡 Note: You'll need to update the bucket credentials later using 'backtide s3 edit'")

	return bucket
}

func generateJobID() string {
	return fmt.Sprintf("job-%s", time.Now().Format("20060102-150405"))
}

func createExampleConfig() *config.BackupConfig {
	cfg := config.DefaultConfig()

	// Add example backup job
	exampleJob := config.BackupJob{
		ID:          "job-example",
		Name:        "example-backup",
		Description: "Example backup job for demonstration",
		Enabled:     true,
		Schedule: config.ScheduleConfig{
			Type:     "systemd",
			Interval: "daily",
			Enabled:  true,
		},
		Directories: []config.DirectoryConfig{
			{
				Path:        "/var/lib/docker/volumes",
				Name:        "docker-volumes",
				Compression: true,
			},
			{
				Path:        "/opt/app/data",
				Name:        "app-data",
				Compression: true,
			},
		},
		Retention: config.RetentionPolicy{
			KeepDays:    30,
			KeepCount:   10,
			KeepMonthly: 6,
		},
		Storage: config.StorageConfig{
			Local: true,
			S3:    false,
		},
		SkipDocker: false,
		SkipS3:     false,
	}

	cfg.Jobs = []config.BackupJob{exampleJob}
	cfg.BackupPath = "/mnt/backup"

	return cfg
}
