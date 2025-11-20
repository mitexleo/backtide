package config

import (
	"time"
)

// BucketConfig represents a standalone S3 bucket configuration
type BucketConfig struct {
	ID           string `toml:"id"`
	Name         string `toml:"name"`
	Bucket       string `toml:"bucket"`
	Region       string `toml:"region"`
	AccessKey    string `toml:"access_key"`
	SecretKey    string `toml:"secret_key"`
	Endpoint     string `toml:"endpoint"`
	MountPoint   string `toml:"mount_point"`
	UsePathStyle bool   `toml:"use_path_style"`
	Provider     string `toml:"provider"`
	Description  string `toml:"description"`
}

// BackupConfig represents the configuration for backup operations
type BackupConfig struct {
	Jobs       []BackupJob      `toml:"jobs"`
	Buckets    []BucketConfig   `toml:"buckets"`
	BackupPath string           `toml:"backup_path"`
	TempPath   string           `toml:"temp_path"`
	AutoUpdate AutoUpdateConfig `toml:"auto_update"`
}

// BackupJob represents a complete backup configuration with scheduling
type BackupJob struct {
	ID          string            `toml:"id"`
	Name        string            `toml:"name"`
	Description string            `toml:"description"`
	Enabled     bool              `toml:"enabled"`
	Schedule    ScheduleConfig    `toml:"schedule"`
	Directories []DirectoryConfig `toml:"directories"`
	BucketID    string            `toml:"bucket_id"`
	Retention   RetentionPolicy   `toml:"retention"`
	SkipDocker  bool              `toml:"skip_docker"`
	SkipS3      bool              `toml:"skip_s3"`
	Storage     StorageConfig     `toml:"storage"`
}

// ScheduleConfig represents backup scheduling configuration
type ScheduleConfig struct {
	Type     string `toml:"type"`
	Interval string `toml:"interval"`
	Enabled  bool   `toml:"enabled"`
}

// DirectoryConfig represents configuration for a single directory to backup
type DirectoryConfig struct {
	Path        string `toml:"path"`
	Name        string `toml:"name"`
	Compression bool   `toml:"compression"`
}

// StorageConfig defines where backups should be stored
type StorageConfig struct {
	Local bool `toml:"local"`
	S3    bool `toml:"s3"`
}

// RetentionPolicy defines how long to keep backups
type RetentionPolicy struct {
	KeepDays    int `toml:"keep_days"`
	KeepCount   int `toml:"keep_count"`
	KeepMonthly int `toml:"keep_monthly"`
}

// BackupMetadata stores information about each backup
type BackupMetadata struct {
	ID          string            `toml:"id"`
	Timestamp   time.Time         `toml:"timestamp"`
	Directories []BackupDirectory `toml:"directories"`
	TotalSize   int64             `toml:"total_size"`
	Checksum    string            `toml:"checksum"`
	Compressed  bool              `toml:"compressed"`
}

// BackupDirectory contains metadata for each backed up directory
type BackupDirectory struct {
	Path        string              `toml:"path"`
	Name        string              `toml:"name"`
	Size        int64               `toml:"size"`
	FileCount   int                 `toml:"file_count"`
	Permissions map[string]FilePerm `toml:"permissions"`
	Checksum    string              `toml:"checksum"`
	Compressed  bool                `toml:"compressed"`
}

// FilePerm stores file permission information
type FilePerm struct {
	Mode    string `toml:"mode"`
	UID     int    `toml:"uid"`
	GID     int    `toml:"gid"`
	Size    int64  `toml:"size"`
	ModTime string `toml:"mod_time"`
}

// DockerContainerInfo stores information about stopped containers
type DockerContainerInfo struct {
	ID      string    `toml:"id"`
	Name    string    `toml:"name"`
	Image   string    `toml:"image"`
	Status  string    `toml:"status"`
	Stopped time.Time `toml:"stopped"`
}

// BackupState tracks the current state of backup operations
type BackupState struct {
	CurrentBackupID   string                `toml:"current_backup_id"`
	StoppedContainers []DockerContainerInfo `toml:"stopped_containers"`
	LastBackupTime    time.Time             `toml:"last_backup_time"`
	IsRunning         bool                  `toml:"is_running"`
}

// JobState tracks the state of individual backup jobs
type JobState struct {
	JobName       string    `toml:"job_name"`
	LastRun       time.Time `toml:"last_run"`
	LastStatus    string    `toml:"last_status"`
	NextScheduled time.Time `toml:"next_scheduled"`
	RunCount      int       `toml:"run_count"`
}

// AutoUpdateConfig defines automatic update checking settings
type AutoUpdateConfig struct {
	Enabled       bool          `toml:"enabled"`
	CheckInterval time.Duration `toml:"check_interval"`
}
