package backup

import (
	"fmt"
	"os"
	"path/filepath"

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

	// Find the bucket configuration for this job
	var bucketConfig *config.BucketConfig
	for _, bucket := range br.config.Buckets {
		if bucket.ID == job.BucketID {
			bucketConfig = &bucket
			break
		}
	}

	if bucketConfig == nil && job.Storage.S3 {
		return nil, fmt.Errorf("bucket configuration not found for job %s", job.Name)
	}

	// Use S3 mount point as backup path if S3 storage is enabled
	backupPath := br.backupPath
	if job.Storage.S3 && bucketConfig != nil {
		backupPath = bucketConfig.MountPoint
		fmt.Printf("Using S3 mount point for backup: %s\n", backupPath)
	}

	// Initialize managers
	// Use user-writable directory for Docker state
	dockerStateDir := filepath.Join(os.Getenv("HOME"), ".backtide")
	if err := os.MkdirAll(dockerStateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backtide directory: %w", err)
	}
	dockerStateFile := filepath.Join(dockerStateDir, "containers.json")
	dockerManager := docker.NewDockerManager(dockerStateFile)
	var s3Manager *s3fs.S3FSManager
	if bucketConfig != nil {
		s3Manager = s3fs.NewS3FSManager(*bucketConfig)
	}

	var stoppedContainers []config.DockerContainerInfo

	// Step 1: Stop Docker containers if enabled
	if !job.SkipDocker {
		fmt.Println("\nStep 1: Managing Docker containers...")
		if err := dockerManager.CheckDockerAvailable(); err != nil {
			fmt.Printf("Warning: Docker is not available: %v\n", err)
		} else {
			stopped, err := dockerManager.StopContainers()
			if err != nil {
				return nil, fmt.Errorf("failed to stop Docker containers: %w", err)
			}
			stoppedContainers = stopped
			fmt.Printf("✅ Stopped %d Docker containers\n", len(stoppedContainers))
		}
	}

	// Step 2: Setup S3FS if S3 storage is enabled
	if !job.SkipS3 && job.Storage.S3 && s3Manager != nil {
		fmt.Println("\nStep 2: Setting up S3 storage...")
		if err := s3Manager.InstallS3FS(); err != nil {
			return nil, fmt.Errorf("failed to install S3FS: %w", err)
		}
		if err := s3Manager.SetupS3FS(); err != nil {
			return nil, fmt.Errorf("failed to setup S3FS: %w", err)
		}
		if err := s3Manager.MountS3FS(); err != nil {
			return nil, fmt.Errorf("failed to mount S3 bucket: %w", err)
		}
		fmt.Println("✅ S3 storage setup completed")
	}

	// Step 3: Create backup configuration for this job
	jobBackupConfig := config.BackupConfig{
		Jobs:       []config.BackupJob{*job},
		Buckets:    br.config.Buckets,
		BackupPath: backupPath,
		TempPath:   br.config.TempPath,
	}

	// Step 4: Run backup
	fmt.Println("\nStep 3: Creating backup...")
	backupManager := NewBackupManager(jobBackupConfig)
	metadata, err := backupManager.CreateBackup()
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Step 5: Restart Docker containers if they were stopped
	if !job.SkipDocker && len(stoppedContainers) > 0 {
		fmt.Println("\nStep 4: Restarting Docker containers...")
		if err := dockerManager.RestoreContainers(); err != nil {
			fmt.Printf("Warning: Failed to restart some Docker containers: %v\n", err)
		} else {
			fmt.Println("✅ Docker containers restarted")
		}
	}

	// Step 6: Cleanup old backups
	fmt.Println("\nStep 5: Cleaning up old backups...")
	if err := backupManager.CleanupBackups(); err != nil {
		fmt.Printf("Warning: Failed to cleanup old backups: %v\n", err)
	} else {
		fmt.Println("✅ Old backups cleaned up")
	}

	fmt.Printf("\n✅ Backup job completed successfully: %s\n", job.Name)
	return metadata, nil
}

// RunAllJobs executes all enabled backup jobs
func (br *BackupRunner) RunAllJobs() ([]config.BackupMetadata, error) {
	var allMetadata []config.BackupMetadata

	for _, job := range br.config.Jobs {
		if job.Enabled {
			metadata, err := br.RunJob(job.Name)
			if err != nil {
				fmt.Printf("Failed to run job %s: %v\n", job.Name, err)
				continue
			}
			allMetadata = append(allMetadata, *metadata)
		}
	}

	return allMetadata, nil
}

// RunJobCleanup cleans up old backups for a specific job
func (br *BackupRunner) RunJobCleanup(jobName string) error {
	job, err := br.findJob(jobName)
	if err != nil {
		return err
	}

	if !job.Enabled {
		return fmt.Errorf("job %s is disabled", jobName)
	}

	fmt.Printf("Cleaning up old backups for job: %s\n", job.Name)
	fmt.Printf("Retention policy: %d days, %d recent, %d monthly\n",
		job.Retention.KeepDays, job.Retention.KeepCount, job.Retention.KeepMonthly)

	// Find the bucket configuration for this job
	var bucketConfig *config.BucketConfig
	for _, bucket := range br.config.Buckets {
		if bucket.ID == job.BucketID {
			bucketConfig = &bucket
			break
		}
	}

	// Use S3 mount point as backup path if S3 storage is enabled
	backupPath := br.backupPath
	if job.Storage.S3 && bucketConfig != nil {
		backupPath = bucketConfig.MountPoint
		fmt.Printf("Using S3 mount point for cleanup: %s\n", backupPath)
	}

	// Create job-specific backup config
	jobBackupConfig := config.BackupConfig{
		Jobs:       []config.BackupJob{*job},
		Buckets:    br.config.Buckets,
		BackupPath: backupPath,
		TempPath:   br.config.TempPath,
	}

	backupManager := NewBackupManager(jobBackupConfig)
	if err := backupManager.CleanupBackups(); err != nil {
		return fmt.Errorf("failed to cleanup backups: %w", err)
	}

	fmt.Printf("✅ Cleanup completed for job: %s\n", job.Name)
	return nil
}

// ListBackups returns a list of all available backups
func (br *BackupRunner) ListBackups() ([]config.BackupMetadata, error) {
	// For now, return an empty list
	// This will be implemented properly in future versions
	fmt.Println("Listing backups functionality will be implemented in future versions")
	return []config.BackupMetadata{}, nil
}

// findJob finds a job by name
func (br *BackupRunner) findJob(jobName string) (*config.BackupJob, error) {
	for i, job := range br.config.Jobs {
		if job.Name == jobName {
			return &br.config.Jobs[i], nil
		}
	}
	return nil, fmt.Errorf("job not found: %s", jobName)
}
