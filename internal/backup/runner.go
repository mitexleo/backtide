package backup

import (
	"fmt"
	"time"

	"github.com/mitexleo/backtide/internal/config"
	"github.com/mitexleo/backtide/internal/docker"
	"github.com/mitexleo/backtide/internal/s3fs"
)

// BackupRunner handles execution of backup jobs
type BackupRunner struct {
	config     config.BackupConfig
	backupPath string
}

// NewBackupRunner creates a new backup runner instance
func NewBackupRunner(cfg config.BackupConfig) *BackupRunner {
	return &BackupRunner{
		config:     cfg,
		backupPath: cfg.BackupPath,
	}
}

// RunJob executes a specific backup job
func (br *BackupRunner) RunJob(jobName string) (*config.BackupMetadata, error) {
	job, err := br.findJob(jobName)
	if err != nil {
		return nil, err
	}

	if !job.Enabled {
		return nil, fmt.Errorf("job %s is disabled", jobName)
	}

	fmt.Printf("Starting backup job: %s\n", job.Name)
	fmt.Printf("Description: %s\n", job.Description)

	// Initialize managers
	dockerManager := docker.NewDockerManager("/var/lib/backtide/containers.json")
	s3Manager := s3fs.NewS3FSManager(job.S3Config)

	var stoppedContainers []config.DockerContainerInfo

	// Step 1: Stop Docker containers if enabled
	if !job.SkipDocker {
		fmt.Println("\nStep 1: Managing Docker containers...")
		if err := dockerManager.CheckDockerAvailable(); err != nil {
			fmt.Printf("Warning: Docker is not available: %v\n", err)
		} else {
			stoppedContainers, err = dockerManager.StopContainers()
			if err != nil {
				fmt.Printf("Error stopping containers: %v\n", err)
				// Continue with backup, but warn user
			} else {
				fmt.Printf("Stopped %d containers\n", len(stoppedContainers))
			}
		}
	}

	// Step 2: Setup and mount S3 if enabled
	if !job.SkipS3 {
		fmt.Println("\nStep 2: Setting up S3FS...")
		// Install s3fs if needed
		if err := s3Manager.InstallS3FS(); err != nil {
			fmt.Printf("Error installing s3fs: %v\n", err)
			// Continue with local backup
		}

		// Setup s3fs
		if err := s3Manager.SetupS3FS(); err != nil {
			fmt.Printf("Error setting up s3fs: %v\n", err)
		}

		// Mount S3 bucket
		if err := s3Manager.MountS3FS(); err != nil {
			fmt.Printf("Error mounting S3 bucket: %v\n", err)
		}
	}

	// Step 3: Create backup with job-specific directories
	fmt.Println("\nStep 3: Creating backup...")

	// Create job-specific backup config
	jobBackupConfig := br.config
	jobBackupConfig.Directories = job.Directories
	jobBackupConfig.S3Config = job.S3Config
	jobBackupConfig.RetentionPolicy = job.Retention

	jobBackupManager := NewBackupManager(jobBackupConfig)

	// Create the backup
	metadata, err := jobBackupManager.CreateBackup()
	if err != nil {
		fmt.Printf("Error creating backup: %v\n", err)
		// Try to restore containers before exiting
		if len(stoppedContainers) > 0 {
			fmt.Println("Attempting to restore Docker containers...")
			if err := dockerManager.RestoreContainers(); err != nil {
				fmt.Printf("Error restoring containers: %v\n", err)
			}
		}
		return nil, err
	}

	// Add job name to metadata
	metadata.ID = fmt.Sprintf("%s-%s", job.Name, metadata.ID)

	fmt.Printf("Backup created successfully: %s\n", metadata.ID)
	fmt.Printf("Total size: %d bytes\n", metadata.TotalSize)
	fmt.Printf("Directories backed up: %d\n", len(metadata.Directories))

	// Step 4: Restore Docker containers if they were stopped
	if len(stoppedContainers) > 0 {
		fmt.Println("\nStep 4: Restoring Docker containers...")
		if err := dockerManager.RestoreContainers(); err != nil {
			fmt.Printf("Error restoring containers: %v\n", err)
			// Don't exit, just warn
		}
	}

	// Step 5: Cleanup old backups for this job
	fmt.Println("\nStep 5: Cleaning up old backups...")
	if err := jobBackupManager.CleanupOldBackups(); err != nil {
		fmt.Printf("Warning: Failed to cleanup old backups: %v\n", err)
	}

	fmt.Printf("\nBackup job '%s' completed successfully!\n", job.Name)
	return metadata, nil
}

// RunAllJobs executes all enabled backup jobs
func (br *BackupRunner) RunAllJobs() ([]*config.BackupMetadata, error) {
	var allMetadata []*config.BackupMetadata
	var errors []string

	for _, job := range br.config.Jobs {
		if job.Enabled {
			metadata, err := br.RunJob(job.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("job %s: %v", job.Name, err))
				continue
			}
			allMetadata = append(allMetadata, metadata)
		}
	}

	if len(errors) > 0 {
		return allMetadata, fmt.Errorf("some jobs failed: %v", errors)
	}

	return allMetadata, nil
}

// RunJobByIndex executes a backup job by its index in the configuration
func (br *BackupRunner) RunJobByIndex(index int) (*config.BackupMetadata, error) {
	if index < 0 || index >= len(br.config.Jobs) {
		return nil, fmt.Errorf("invalid job index: %d", index)
	}

	job := br.config.Jobs[index]
	return br.RunJob(job.Name)
}

// ListJobs returns information about all configured jobs
func (br *BackupRunner) ListJobs() []config.BackupJob {
	return br.config.Jobs
}

// GetJob returns a specific job by name
func (br *BackupRunner) GetJob(jobName string) (*config.BackupJob, error) {
	return br.findJob(jobName)
}

// GetEnabledJobs returns only enabled jobs
func (br *BackupRunner) GetEnabledJobs() []config.BackupJob {
	var enabledJobs []config.BackupJob
	for _, job := range br.config.Jobs {
		if job.Enabled {
			enabledJobs = append(enabledJobs, job)
		}
	}
	return enabledJobs
}

// findJob finds a job by name
func (br *BackupRunner) findJob(jobName string) (*config.BackupJob, error) {
	for _, job := range br.config.Jobs {
		if job.Name == jobName {
			return &job, nil
		}
	}
	return nil, fmt.Errorf("job not found: %s", jobName)
}

// UpdateJobState updates the state of a job
func (br *BackupRunner) UpdateJobState(jobName, status string) error {
	// TODO: Implement state file reading/writing
	// For now, just log the update
	fmt.Printf("Job %s: %s at %s\n", jobName, status, time.Now().Format(time.RFC3339))

	return nil
}

// GetNextScheduledRun calculates when the next backup should run for a job
func (br *BackupRunner) GetNextScheduledRun(job config.BackupJob) (time.Time, error) {
	if !job.Schedule.Enabled {
		return time.Time{}, fmt.Errorf("job %s is not scheduled", job.Name)
	}

	// Simple implementation - for cron/systemd, this would be more complex
	// For now, return a placeholder
	switch job.Schedule.Type {
	case "daily":
		return time.Now().Add(24 * time.Hour), nil
	case "weekly":
		return time.Now().Add(7 * 24 * time.Hour), nil
	case "monthly":
		return time.Now().Add(30 * 24 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported schedule type: %s", job.Schedule.Type)
	}
}
