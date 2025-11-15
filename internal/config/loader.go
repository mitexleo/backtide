package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultConfig returns a default configuration
func DefaultConfig() *BackupConfig {
	return &BackupConfig{
		BackupPath: "", // Empty = no local storage, use S3 only
		TempPath:   "/tmp/backtide",
		S3Config: S3Config{
			MountPoint:   "/mnt/s3backup",
			Endpoint:     "",    // Set to your S3-compatible endpoint (e.g., https://s3.us-west-002.backblazeb2.com)
			UsePathStyle: false, // Set to true for path-style endpoints
		},
		RetentionPolicy: RetentionPolicy{
			KeepDays:  30,
			KeepCount: 10,
		},
		Jobs: []BackupJob{},
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

	// Try to parse as YAML first, then JSON
	if err := yaml.Unmarshal(data, config); err != nil {
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file as YAML or JSON: %w", err)
		}
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

	data, err := yaml.Marshal(config)
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
	// Check if using legacy config or new job-based config
	if len(config.Jobs) == 0 && len(config.Directories) == 0 {
		return fmt.Errorf("no backup jobs or directories configured")
	}

	// Validate jobs if using job-based config
	if len(config.Jobs) > 0 {
		for i, job := range config.Jobs {
			if job.Name == "" {
				return fmt.Errorf("job name cannot be empty for job %d", i)
			}

			if len(job.Directories) == 0 {
				return fmt.Errorf("at least one directory must be specified for job %s", job.Name)
			}

			for j, dir := range job.Directories {
				if dir.Path == "" {
					return fmt.Errorf("directory path cannot be empty for directory %d in job %s", j, job.Name)
				}
				if dir.Name == "" {
					return fmt.Errorf("directory name cannot be empty for directory %d in job %s", j, job.Name)
				}
			}

			if !job.SkipS3 {
				if job.S3Config.Bucket == "" {
					return fmt.Errorf("S3 bucket name cannot be empty for job %s", job.Name)
				}
				if job.S3Config.AccessKey == "" {
					return fmt.Errorf("S3 access key cannot be empty for job %s", job.Name)
				}
				if job.S3Config.SecretKey == "" {
					return fmt.Errorf("S3 secret key cannot be empty for job %s", job.Name)
				}
			}
		}
	} else {
		// Legacy config validation
		if len(config.Directories) == 0 {
			return fmt.Errorf("at least one directory must be specified")
		}

		for i, dir := range config.Directories {
			if dir.Path == "" {
				return fmt.Errorf("directory path cannot be empty for directory %d", i)
			}
			if dir.Name == "" {
				return fmt.Errorf("directory name cannot be empty for directory %d", i)
			}
		}

		if config.S3Config.Bucket == "" {
			return fmt.Errorf("S3 bucket name cannot be empty")
		}

		if config.S3Config.AccessKey == "" {
			return fmt.Errorf("S3 access key cannot be empty")
		}

		if config.S3Config.SecretKey == "" {
			return fmt.Errorf("S3 secret key cannot be empty")
		}
	}

	if config.S3Config.Endpoint == "" {
		fmt.Println("Warning: No S3 endpoint specified. Using AWS default endpoints.")
		fmt.Println("For S3-compatible providers (Backblaze B2, Wasabi, etc.), set the endpoint in config.")
	}

	if config.S3Config.MountPoint == "" {
		return fmt.Errorf("S3 mount point cannot be empty")
	}

	if config.BackupPath == "" {
		return fmt.Errorf("backup path cannot be empty")
	}

	if config.TempPath == "" {
		return fmt.Errorf("temp path cannot be empty")
	}

	return nil
}

// FindConfigFile searches for configuration file in common locations
func FindConfigFile() string {
	// Common configuration file locations
	locations := []string{
		"/etc/backtide/config.yaml",
		"/etc/backtide/config.yml",
		"/etc/backtide/config.json",
		"/usr/local/etc/backtide/config.yaml",
		"~/.config/backtide/config.yaml",
		"~/.backtide.yaml",
		"./backtide.yaml",
		"./backtide.yml",
		"./backtide.json",
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

	return ""
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
			Local: false,
			S3:    true,
		},
		SkipDocker: false,
		SkipS3:     false,
	}

	defaultConfig.Jobs = []BackupJob{defaultJob}
	return SaveConfig(defaultConfig, configPath)
}
