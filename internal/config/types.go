package config

import (
	"time"
)

// BackupConfig represents the configuration for backup operations
type BackupConfig struct {
	Directories     []DirectoryConfig `json:"directories" yaml:"directories"`
	S3Config        S3Config          `json:"s3" yaml:"s3"`
	RetentionPolicy RetentionPolicy   `json:"retention" yaml:"retention"`
	BackupPath      string            `json:"backup_path" yaml:"backup_path"`
	TempPath        string            `json:"temp_path" yaml:"temp_path"`
}

// DirectoryConfig represents configuration for a single directory to backup
type DirectoryConfig struct {
	Path        string `json:"path" yaml:"path"`
	Name        string `json:"name" yaml:"name"`
	Compression bool   `json:"compression" yaml:"compression"`
}

// S3Config represents S3 bucket configuration
type S3Config struct {
	Bucket       string `json:"bucket" yaml:"bucket"`
	Region       string `json:"region" yaml:"region"`
	AccessKey    string `json:"access_key" yaml:"access_key"`
	SecretKey    string `json:"secret_key" yaml:"secret_key"`
	Endpoint     string `json:"endpoint" yaml:"endpoint"`
	MountPoint   string `json:"mount_point" yaml:"mount_point"`
	UsePathStyle bool   `json:"use_path_style" yaml:"use_path_style"`
}

// RetentionPolicy defines how long to keep backups
type RetentionPolicy struct {
	KeepDays    int `json:"keep_days" yaml:"keep_days"`
	KeepCount   int `json:"keep_count" yaml:"keep_count"`
	KeepMonthly int `json:"keep_monthly" yaml:"keep_monthly"`
}

// BackupMetadata stores information about each backup
type BackupMetadata struct {
	ID          string            `json:"id" yaml:"id"`
	Timestamp   time.Time         `json:"timestamp" yaml:"timestamp"`
	Directories []BackupDirectory `json:"directories" yaml:"directories"`
	TotalSize   int64             `json:"total_size" yaml:"total_size"`
	Checksum    string            `json:"checksum" yaml:"checksum"`
	Compressed  bool              `json:"compressed" yaml:"compressed"`
}

// BackupDirectory contains metadata for each backed up directory
type BackupDirectory struct {
	Path        string              `json:"path" yaml:"path"`
	Name        string              `json:"name" yaml:"name"`
	Size        int64               `json:"size" yaml:"size"`
	FileCount   int                 `json:"file_count" yaml:"file_count"`
	Permissions map[string]FilePerm `json:"permissions" yaml:"permissions"`
	Checksum    string              `json:"checksum" yaml:"checksum"`
	Compressed  bool                `json:"compressed" yaml:"compressed"`
}

// FilePerm stores file permission information
type FilePerm struct {
	Mode    string `json:"mode" yaml:"mode"`
	UID     int    `json:"uid" yaml:"uid"`
	GID     int    `json:"gid" yaml:"gid"`
	Size    int64  `json:"size" yaml:"size"`
	ModTime string `json:"mod_time" yaml:"mod_time"`
}

// DockerContainerInfo stores information about stopped containers
type DockerContainerInfo struct {
	ID      string    `json:"id" yaml:"id"`
	Name    string    `json:"name" yaml:"name"`
	Image   string    `json:"image" yaml:"image"`
	Status  string    `json:"status" yaml:"status"`
	Stopped time.Time `json:"stopped" yaml:"stopped"`
}

// BackupState tracks the current state of backup operations
type BackupState struct {
	CurrentBackupID   string                `json:"current_backup_id" yaml:"current_backup_id"`
	StoppedContainers []DockerContainerInfo `json:"stopped_containers" yaml:"stopped_containers"`
	LastBackupTime    time.Time             `json:"last_backup_time" yaml:"last_backup_time"`
	IsRunning         bool                  `json:"is_running" yaml:"is_running"`
}
