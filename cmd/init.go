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

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize backtide configuration",
	Long: `Initialize backtide with a default configuration file.

This command will:
1. Create a default configuration file
2. Set up necessary directories
3. Provide guidance for next steps

The configuration file will be created in the default location
(~/.backtide.yaml) unless specified otherwise.`,
	Run: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing configuration file")
	initCmd.Flags().BoolVar(&initExamples, "examples", false, "include example configurations")
	initCmd.Flags().BoolVar(&initSkipInteractive, "skip-interactive", false, "skip interactive configuration setup")
}

func runInit(cmd *cobra.Command, args []string) {
	fmt.Println("Initializing backtide...")

	// Determine config file path
	configPath := cfgFile
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		configPath = filepath.Join(home, ".backtide.yaml")
	}

	// Check if config file already exists
	var existingConfig *config.BackupConfig
	if _, err := os.Stat(configPath); err == nil {
		if !initForce {
			fmt.Printf("Configuration file already exists: %s\n", configPath)
			fmt.Println("Use --force to overwrite existing configuration")
			os.Exit(1)
		}
		// Load existing config to preserve settings
		existingConfig, _ = config.LoadConfig(configPath)
	}

	// Create configuration directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating configuration directory: %v\n", err)
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

	// Interactive job configuration setup
	if !dryRun && !initSkipInteractive {
		fmt.Println("\n=== Backup Job Setup ===")
		fmt.Println("Let's create your first backup job with scheduling and retention.")
		fmt.Println()

		// Create a complete backup job
		job := configureBackupJobInteractive(configPath)
		// Add to existing jobs (will be empty if no existing config)
		defaultConfig.Jobs = append(defaultConfig.Jobs, job)

	}

	// Save configuration
	if err := config.SaveConfig(defaultConfig, configPath); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	// Create necessary directories
	dirs := []string{
		"/var/lib/backtide",
		"/var/log/backtide",
		"/mnt/backup",
		"/mnt/s3backup",
		"/tmp/backtide",
	}

	fmt.Println("\nCreating necessary directories...")
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("  Warning: Could not create %s: %v\n", dir, err)
		} else {
			fmt.Printf("  Created: %s\n", dir)
		}
	}

	fmt.Printf("\nConfiguration created successfully: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file with your specific settings")
	fmt.Println("2. Set S3 credentials and bucket information")
	fmt.Println("3. Configure directories you want to backup")
	fmt.Println("4. Test the backup: backtide backup --dry-run")
	fmt.Println("5. Set up automated backups: backtide systemd install")
	fmt.Println("\nExample commands:")
	fmt.Println("  backtide backup                    # Run backup")
	fmt.Println("  backtide list                      # List backups")
	fmt.Println("  backtide systemd install           # Set up systemd service")
	fmt.Println("  backtide cron install              # Set up cron job")
}

// configureS3Interactive interactively configures S3 settings
func configureBackupJobInteractive(configPath string) config.BackupJob {
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
		bucketID := configureBucketForJob(configPath)
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
		bucketID := configureBucketForJob(configPath)
		job.BucketID = bucketID
	default:
		// Default to S3 only for safety
		job.Storage.S3 = true
		job.Storage.Local = false
		fmt.Println("❌ Invalid choice, defaulting to S3 only")
		bucketID := configureBucketForJob(configPath)
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

	fmt.Printf("\n🎉 Backup job '%s' configured successfully!\n", job.Name)
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

func configureBucketForJob(configPath string) string {
	reader := bufio.NewReader(os.Stdin)

	// Check for existing buckets
	existingBuckets := getExistingBuckets(configPath)
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

	// Load config to add new bucket
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Configure new bucket
	newBucket := configureBucketForAdd()
	cfg.Buckets = append(cfg.Buckets, newBucket)

	// Save configuration with new bucket
	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ New bucket configuration '%s' added!\n", newBucket.Name)

	return newBucket.ID
}

// configureDirectoriesInteractive interactively configures directories to backup
func configureDirectoriesInteractive() []config.DirectoryConfig {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Common directories to backup:")
	fmt.Println("1. Docker volumes (/var/lib/docker/volumes)")
	fmt.Println("2. User home directory")
	fmt.Println("3. System configuration (/etc)")
	fmt.Println("4. Application data (/opt)")
	fmt.Println("5. Custom directory")
	fmt.Println("6. Skip directory configuration for now")
	fmt.Print("Choose directories to backup (comma-separated, e.g., 1,3,5): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "6" {
		fmt.Println("Skipping directory configuration.")
		return []config.DirectoryConfig{}
	}

	choices := strings.Split(choice, ",")
	var directories []config.DirectoryConfig

	for _, c := range choices {
		c = strings.TrimSpace(c)
		switch c {
		case "1":
			// Docker volumes
			directories = append(directories, config.DirectoryConfig{
				Path:        "/var/lib/docker/volumes",
				Name:        "docker-volumes",
				Compression: true,
			})
			fmt.Println("✅ Added Docker volumes (/var/lib/docker/volumes)")

		case "2":
			// User home
			home, err := os.UserHomeDir()
			if err == nil {
				dirName := filepath.Base(home) + "-home"
				directories = append(directories, config.DirectoryConfig{
					Path:        home,
					Name:        dirName,
					Compression: true,
				})
				fmt.Printf("✅ Added home directory (%s)\n", home)
			}

		case "3":
			// System config
			directories = append(directories, config.DirectoryConfig{
				Path:        "/etc",
				Name:        "system-config",
				Compression: true,
			})
			fmt.Println("✅ Added system configuration (/etc)")

		case "4":
			// Application data
			directories = append(directories, config.DirectoryConfig{
				Path:        "/opt",
				Name:        "application-data",
				Compression: true,
			})
			fmt.Println("✅ Added application data (/opt)")

		case "5":
			// Custom directory
			fmt.Print("Enter directory path to backup: ")
			path, _ := reader.ReadString('\n')
			path = strings.TrimSpace(path)

			if path != "" {
				fmt.Print("Enter backup name for this directory: ")
				name, _ := reader.ReadString('\n')
				name = strings.TrimSpace(name)
				if name == "" {
					name = filepath.Base(path)
				}

				fmt.Print("Enable compression? (Y/n): ")
				compress, _ := reader.ReadString('\n')
				compress = strings.TrimSpace(compress)
				compression := compress == "" || strings.ToLower(compress) == "y"

				directories = append(directories, config.DirectoryConfig{
					Path:        path,
					Name:        name,
					Compression: compression,
				})
				fmt.Printf("✅ Added custom directory: %s\n", path)
			}
		}
	}

	if len(directories) > 0 {
		fmt.Printf("\n✅ Configured %d directories for backup\n", len(directories))
	} else {
		fmt.Println("No directories configured for backup.")
		fmt.Println("You can add them later by editing the configuration file.")
	}

	return directories
}

// generateJobID creates a unique identifier for backup jobs
func generateJobID() string {
	return fmt.Sprintf("job-%s", time.Now().Format("20060102-150405"))
}

func createExampleConfig() *config.BackupConfig {
	cfg := config.DefaultConfig()

	// Example S3 configuration - Choose ONE provider and configure accordingly:

	// Example bucket configuration
	exampleBucket := config.BucketConfig{
		ID:           "bucket-example-1",
		Name:         "AWS S3 Example",
		Bucket:       "my-backup-bucket",
		Region:       "us-east-1",
		AccessKey:    "YOUR_ACCESS_KEY_HERE",
		SecretKey:    "YOUR_SECRET_KEY_HERE",
		Endpoint:     "", // Leave empty for AWS
		MountPoint:   "/mnt/s3backup",
		UsePathStyle: false,
		Provider:     "AWS S3",
		Description:  "Example AWS S3 bucket configuration",
	}
	cfg.Buckets = append(cfg.Buckets, exampleBucket)

	// Backblaze B2 example (recommended)
	// Uncomment and configure for Backblaze B2
	// b2Bucket := config.BucketConfig{
	//     ID:           "bucket-b2-1",
	//     Name:         "Backblaze B2 Example",
	//     Bucket:       "my-b2-bucket",
	//     Region:       "",
	//     AccessKey:    "YOUR_B2_ACCESS_KEY",
	//     SecretKey:    "YOUR_B2_SECRET_KEY",
	//     Endpoint:     "https://s3.us-west-002.backblazeb2.com",
	//     MountPoint:   "/mnt/s3backup-b2",
	//     UsePathStyle: true,
	//     Provider:     "Backblaze B2",
	//     Description:  "Example Backblaze B2 bucket configuration",
	// }
	// cfg.Buckets = append(cfg.Buckets, b2Bucket)
	// cfg.S3Config.Region = ""  // Not used for B2
	// cfg.S3Config.AccessKey = "YOUR_APPLICATION_KEY_ID"
	// cfg.S3Config.SecretKey = "YOUR_APPLICATION_KEY"
	// cfg.S3Config.MountPoint = "/mnt/s3backup"
	// cfg.S3Config.Endpoint = "https://s3.us-west-002.backblazeb2.com"  // Your B2 endpoint
	// cfg.S3Config.UsePathStyle = true  // REQUIRED for B2

	// Wasabi
	// cfg.S3Config.Bucket = "my-backup-bucket"
	// cfg.S3Config.Region = "us-east-1"  // Your Wasabi region
	// cfg.S3Config.AccessKey = "YOUR_ACCESS_KEY_HERE"
	// cfg.S3Config.SecretKey = "YOUR_SECRET_KEY_HERE"
	// cfg.S3Config.MountPoint = "/mnt/s3backup"
	// cfg.S3Config.Endpoint = "https://s3.wasabisys.com"  // Wasabi endpoint
	// cfg.S3Config.UsePathStyle = false

	// DigitalOcean Spaces
	// cfg.S3Config.Bucket = "my-backup-bucket"
	// cfg.S3Config.Region = "nyc3"  // Your DO region
	// cfg.S3Config.AccessKey = "YOUR_SPACES_KEY"
	// cfg.S3Config.SecretKey = "YOUR_SPACES_SECRET"
	// cfg.S3Config.MountPoint = "/mnt/s3backup"
	// cfg.S3Config.Endpoint = "https://nyc3.digitaloceanspaces.com"  // Your DO endpoint
	// cfg.S3Config.UsePathStyle = false

	// Example configuration file
	cfg = config.DefaultConfig()

	// Example backup job
	defaultJob := config.BackupJob{
		ID:          "job-default",
		Name:        "default-backup",
		Description: "Default backup job for Docker volumes and application data",
		Enabled:     true,
		Schedule: config.ScheduleConfig{
			Type:     "manual",
			Interval: "",
			Enabled:  false,
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
		BucketID: "",
		Retention: config.RetentionPolicy{
			KeepDays:    30,
			KeepCount:   10,
			KeepMonthly: 6,
		},
		SkipDocker: false,
		SkipS3:     false,
		Storage: config.StorageConfig{
			Local: true,
			S3:    false,
		},
	}

	cfg.Jobs = []config.BackupJob{defaultJob}
	return cfg
}
