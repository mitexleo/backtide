package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// DefaultConfig returns a default configuration
func DefaultConfig() *BackupConfig {
	return &BackupConfig{
		BackupPath: "", // Empty = no local storage, use S3 only
		TempPath:   "/tmp/backtide",
		Jobs:       []BackupJob{},
		Buckets:    []BucketConfig{},
		AutoUpdate: AutoUpdateConfig{
			Enabled:       false,          // Disabled by default for safety
			CheckInterval: 24 * time.Hour, // Check once per day
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*BackupConfig, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path cannot be empty")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()

	// Parse as TOML
	if err := toml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file as TOML: %w", err)
	}

	// Validate the configuration
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *BackupConfig, configPath string) error {
	if configPath == "" {
		return fmt.Errorf("config path cannot be empty")
	}

	// Ensure the directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration
func ValidateConfig(config *BackupConfig) error {
	// Allow empty config for S3 management operations
	if len(config.Jobs) == 0 {
		return nil
	}

	// Validate bucket configurations
	bucketIDs := make(map[string]bool)
	bucketNames := make(map[string]bool)
	for i, bucket := range config.Buckets {
		if bucket.ID == "" {
			return fmt.Errorf("bucket ID cannot be empty for bucket %d", i)
		}
		if bucketIDs[bucket.ID] {
			return fmt.Errorf("duplicate bucket ID: %s", bucket.ID)
		}
		bucketIDs[bucket.ID] = true

		if bucket.Name == "" {
			return fmt.Errorf("bucket name cannot be empty for bucket %s", bucket.ID)
		}
		if bucketNames[bucket.Name] {
			return fmt.Errorf("duplicate bucket name: %s", bucket.Name)
		}
		bucketNames[bucket.Name] = true

		if bucket.Bucket == "" {
			return fmt.Errorf("S3 bucket name cannot be empty for bucket %s", bucket.ID)
		}
		if bucket.AccessKey == "" {
			return fmt.Errorf("S3 access key cannot be empty for bucket %s", bucket.ID)
		}
		if bucket.SecretKey == "" {
			return fmt.Errorf("S3 secret key cannot be empty for bucket %s", bucket.ID)
		}
		if bucket.MountPoint == "" {
			return fmt.Errorf("S3 mount point cannot be empty for bucket %s", bucket.ID)
		}
	}

	// Validate jobs if using job-based config
	if len(config.Jobs) > 0 {
		for i, job := range config.Jobs {
			if job.Name == "" {
				return fmt.Errorf("job name cannot be empty for job %d", i)
			}

			// Allow jobs without directories for initial configuration
			// Directories can be added later through configuration editing
			if len(job.Directories) == 0 {
				// Only warn, don't fail validation for empty directories
				// This allows initial configuration to be created without directories
			}

			for j, dir := range job.Directories {
				if dir.Path == "" {
					return fmt.Errorf("directory path cannot be empty for directory %d in job %s", j, job.Name)
				}
				if dir.Name == "" {
					return fmt.Errorf("directory name cannot be empty for directory %d in job %s", j, job.Name)
				}
			}

			// Validate S3 storage configuration
			if !job.SkipS3 && job.Storage.S3 {
				if job.BucketID == "" {
					return fmt.Errorf("bucket ID cannot be empty for job %s when using S3 storage", job.Name)
				}

				// Validate bucket ID reference
				bucketExists := false
				for _, bucket := range config.Buckets {
					if bucket.ID == job.BucketID {
						bucketExists = true
						break
					}
				}
				if !bucketExists {
					return fmt.Errorf("job %s references non-existent bucket ID: %s", job.Name, job.BucketID)
				}
			} else if job.Storage.S3 && job.BucketID == "" {
				// Job has S3 storage enabled but no bucket configured
				return fmt.Errorf("job %s has S3 storage enabled but no bucket ID configured", job.Name)
			}
		}
	}

	// Check if any job uses local storage
	usesLocalStorage := false
	for _, job := range config.Jobs {
		if job.Storage.Local {
			usesLocalStorage = true
			break
		}
	}

	// Only require backup path if local storage is used and jobs exist
	if usesLocalStorage && config.BackupPath == "" && len(config.Jobs) > 0 {
		return fmt.Errorf("backup path cannot be empty when using local storage")
	}

	if config.TempPath == "" {
		return fmt.Errorf("temp path cannot be empty")
	}

	return nil
}

// EnsureSystemDirectories creates necessary system directories for Backtide
func EnsureSystemDirectories() error {
	// Create /etc/backtide directory for configuration
	if err := os.MkdirAll("/etc/backtide", 0755); err != nil {
		return fmt.Errorf("failed to create configuration directory: %w", err)
	}

	// Create /etc/backtide/s3-credentials directory for credentials
	credsDir := filepath.Join("/etc", "backtide", "s3-credentials")
	if err := os.MkdirAll(credsDir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	return nil
}

// FindConfigFile searches for configuration file in common locations
func FindConfigFile() string {
	// System-wide configuration locations (preferred)
	locations := []string{
		"/etc/backtide/config.toml",
		"/etc/backtide/backtide.toml",
	}

	for _, location := range locations {
		// Expand ~ to home directory
		if location[0] == '~' {
			if home, err := os.UserHomeDir(); err == nil {
				location = filepath.Join(home, location[1:])
			}
		}

		if _, err := os.Stat(location); err == nil {
			return location
		}
	}

	// If no system configuration found, check for development locations
	devLocations := []string{
		"/usr/local/etc/backtide/config.toml",
		"~/.config/backtide/config.toml",
		"~/.backtide.toml",
		"./backtide.toml",
		"./config.toml",
	}

	for _, location := range devLocations {
		// Expand ~ to home directory
		if location[0] == '~' {
			if home, err := os.UserHomeDir(); err == nil {
				location = filepath.Join(home, location[1:])
			}
		}

		if _, err := os.Stat(location); err == nil {
			fmt.Printf("⚠️  Using development configuration: %s\n", location)
			fmt.Println("💡 For production, use: /etc/backtide/config.toml")
			return location
		}
	}

	return ""
}

// SaveBackupMetadata saves backup metadata to a file
func SaveBackupMetadata(metadata *BackupMetadata, filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	data, err := toml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// LoadBackupMetadata loads backup metadata from a file
func LoadBackupMetadata(filePath string) (*BackupMetadata, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata BackupMetadata
	if err := toml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata file: %w", err)
	}

	return &metadata, nil
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig(configPath string) error {
	defaultConfig := DefaultConfig()

	// Create a default backup job
	defaultJob := BackupJob{
		ID:          "job-default",
		Name:        "default-backup",
		Description: "Default backup job for Docker volumes and application data",
		Enabled:     true,
		Schedule: ScheduleConfig{
			Type:     "manual",
			Interval: "",
			Enabled:  false,
		},
		Directories: []DirectoryConfig{
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
		Retention: RetentionPolicy{
			KeepDays:  30,
			KeepCount: 10,
		},
		Storage: StorageConfig{
			Local: true,
			S3:    false,
		},
		SkipDocker: false,
		SkipS3:     false,
	}

	defaultConfig.Jobs = []BackupJob{defaultJob}
	return SaveConfig(defaultConfig, configPath)
}
