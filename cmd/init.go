package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitexleo/backtide/internal/config"
	"github.com/mitexleo/backtide/internal/s3fs"
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
	if _, err := os.Stat(configPath); err == nil && !initForce {
		fmt.Printf("Configuration file already exists: %s\n", configPath)
		fmt.Println("Use --force to overwrite existing configuration")
		os.Exit(1)
	}

	// Create configuration directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating configuration directory: %v\n", err)
		os.Exit(1)
	}

	// Create default configuration
	var defaultConfig *config.BackupConfig
	if initExamples {
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
		job := configureBackupJobInteractive()
		defaultConfig.Jobs = []config.BackupJob{job}
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
func configureBackupJobInteractive() config.BackupJob {
	reader := bufio.NewReader(os.Stdin)
	job := config.BackupJob{
		Enabled:    true,
		SkipDocker: false,
		SkipS3:     false,
	}

	// Job name and description
	fmt.Print("Backup job name (e.g., 'daily-docker-backup'): ")
	name, _ := reader.ReadString('\n')
	job.Name = strings.TrimSpace(name)
	if job.Name == "" {
		job.Name = "default-backup"
	}

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

	// S3 configuration
	fmt.Println("\n=== S3 Storage Configuration ===")
	fmt.Print("Use S3 for backup storage? (Y/n): ")
	useS3, _ := reader.ReadString('\n')
	useS3 = strings.TrimSpace(useS3)

	if useS3 == "" || strings.ToLower(useS3) == "y" {
		job.S3Config = configureS3Interactive()
	} else {
		job.SkipS3 = true
		fmt.Println("✅ S3 storage disabled")
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

func configureS3Interactive() config.S3Config {
	reader := bufio.NewReader(os.Stdin)
	s3Config := config.S3Config{
		MountPoint: "/mnt/s3backup",
	}

	fmt.Println("S3 Provider Options:")
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
	var defaultPathStyle bool

	switch choice {
	case "1":
		providerName = "AWS S3"
		defaultPathStyle = false
		fmt.Print("AWS Region (e.g., us-east-1): ")
		region, _ := reader.ReadString('\n')
		s3Config.Region = strings.TrimSpace(region)
	case "2":
		providerName = "Backblaze B2"
		defaultEndpoint = "https://s3.us-west-002.backblazeb2.com"
		defaultPathStyle = true
		s3Config.Region = ""
	case "3":
		providerName = "Wasabi"
		defaultEndpoint = "https://s3.wasabisys.com"
		defaultPathStyle = false
		fmt.Print("Wasabi Region (e.g., us-east-1): ")
		region, _ := reader.ReadString('\n')
		s3Config.Region = strings.TrimSpace(region)
	case "4":
		providerName = "DigitalOcean Spaces"
		defaultEndpoint = "https://nyc3.digitaloceanspaces.com"
		defaultPathStyle = false
		fmt.Print("DO Region (e.g., nyc3): ")
		region, _ := reader.ReadString('\n')
		s3Config.Region = strings.TrimSpace(region)
	case "5":
		providerName = "MinIO"
		defaultEndpoint = "http://localhost:9000"
		defaultPathStyle = true
		s3Config.Region = ""
	case "6":
		providerName = "Other S3-compatible"
		defaultPathStyle = false
		fmt.Print("Endpoint URL (e.g., https://s3.example.com): ")
		endpoint, _ := reader.ReadString('\n')
		defaultEndpoint = strings.TrimSpace(endpoint)
		fmt.Print("Use path-style endpoints? (y/N): ")
		pathStyle, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(pathStyle)) == "y" {
			defaultPathStyle = true
		}
	default:
		fmt.Println("Invalid choice, using AWS S3 defaults")
		providerName = "AWS S3"
		defaultPathStyle = false
	}

	fmt.Printf("\nConfiguring %s...\n", providerName)

	// Bucket name
	fmt.Print("Bucket name: ")
	bucket, _ := reader.ReadString('\n')
	s3Config.Bucket = strings.TrimSpace(bucket)

	// Endpoint
	if defaultEndpoint != "" {
		fmt.Printf("Endpoint [%s]: ", defaultEndpoint)
		endpoint, _ := reader.ReadString('\n')
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			s3Config.Endpoint = defaultEndpoint
		} else {
			s3Config.Endpoint = endpoint
		}
	} else {
		fmt.Print("Endpoint (leave empty for AWS default): ")
		endpoint, _ := reader.ReadString('\n')
		s3Config.Endpoint = strings.TrimSpace(endpoint)
	}

	// Path style
	s3Config.UsePathStyle = defaultPathStyle
	if defaultPathStyle {
		fmt.Println("Using path-style endpoints (required for this provider)")
	}

	// Access key
	fmt.Print("Access Key: ")
	accessKey, _ := reader.ReadString('\n')
	s3Config.AccessKey = strings.TrimSpace(accessKey)

	// Secret key
	fmt.Print("Secret Key: ")
	secretKey, _ := reader.ReadString('\n')
	s3Config.SecretKey = strings.TrimSpace(secretKey)

	// Mount point
	fmt.Printf("Mount point [%s]: ", s3Config.MountPoint)
	mountPoint, _ := reader.ReadString('\n')
	mountPoint = strings.TrimSpace(mountPoint)
	if mountPoint != "" {
		s3Config.MountPoint = mountPoint
	}

	fmt.Printf("\n✅ S3 configuration for %s completed!\n", providerName)

	// Ask if user wants to add to fstab for persistence
	if os.Geteuid() == 0 {
		fmt.Print("\nAdd S3 mount to /etc/fstab for persistence? (Y/n): ")
		addFstab, _ := reader.ReadString('\n')
		addFstab = strings.TrimSpace(addFstab)

		if addFstab == "" || strings.ToLower(addFstab) == "y" {
			s3Manager := s3fs.NewS3FSManager(s3Config)
			if err := s3Manager.AddToFstab(); err != nil {
				fmt.Printf("Warning: Failed to add to fstab: %v\n", err)
				fmt.Println("You may need to add the mount manually to /etc/fstab")
			} else {
				fmt.Println("✅ S3FS entry added to /etc/fstab")
			}
		}
	} else {
		fmt.Println("Note: Run as root to automatically add S3 mount to /etc/fstab")
	}

	return s3Config
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

func createExampleConfig() *config.BackupConfig {
	cfg := config.DefaultConfig()

	// Example S3 configuration - Choose ONE provider and configure accordingly:

	// AWS S3 (default)
	cfg.S3Config.Bucket = "my-backup-bucket"
	cfg.S3Config.Region = "us-east-1"
	cfg.S3Config.AccessKey = "YOUR_ACCESS_KEY_HERE"
	cfg.S3Config.SecretKey = "YOUR_SECRET_KEY_HERE"
	cfg.S3Config.MountPoint = "/mnt/s3backup"
	cfg.S3Config.Endpoint = "" // Leave empty for AWS
	cfg.S3Config.UsePathStyle = false

	// Backblaze B2 (recommended)
	// cfg.S3Config.Bucket = "my-backup-bucket"
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

	// Example directories to backup
	cfg.Directories = []config.DirectoryConfig{}
	cfg.Directories = append(cfg.Directories, config.DirectoryConfig{
		Path:        "/var/lib/docker/volumes",
		Name:        "docker-volumes",
		Compression: true,
	})
	cfg.Directories = append(cfg.Directories, config.DirectoryConfig{
		Path:        "/opt/myapp/data",
		Name:        "app-data",
		Compression: true,
	})
	cfg.Directories = append(cfg.Directories, config.DirectoryConfig{
		Path:        "/home/user/documents",
		Name:        "user-documents",
		Compression: true,
	})
	cfg.Directories = append(cfg.Directories, config.DirectoryConfig{
		Path:        "/etc",
		Name:        "system-config",
		Compression: true,
	})

	// Example retention policy
	cfg.RetentionPolicy.KeepDays = 30
	cfg.RetentionPolicy.KeepCount = 10
	cfg.RetentionPolicy.KeepMonthly = 6

	// Backup paths
	cfg.BackupPath = "/mnt/backup"
	cfg.TempPath = "/tmp/backtide"

	return cfg
}
