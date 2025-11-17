package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/commands"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	listJobs    bool
	listBuckets bool
	listBackups bool
	listAll     bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List various components of the backup system",
	Long: `List information about backup jobs, buckets, and backups.

This command can show:
- Configured backup jobs and their status
- S3 bucket configurations
- Available backups with metadata

Examples:
  backtide list --jobs
  backtide list --buckets
  backtide list --backups
  backtide list --all`,
	Run: runList,
}

func init() {
	listCmd.Flags().BoolVar(&listJobs, "jobs", false, "list backup jobs")
	listCmd.Flags().BoolVar(&listBuckets, "buckets", false, "list S3 bucket configurations")
	listCmd.Flags().BoolVar(&listBackups, "backups", false, "list available backups")
	listCmd.Flags().BoolVar(&listAll, "all", false, "list all information")

	// Register with command registry
	commands.RegisterCommand("list", listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Default to showing all if no specific flags are set
	if !listJobs && !listBuckets && !listBackups {
		listAll = true
	}

	if listAll || listJobs {
		listBackupJobs(cfg)
	}

	if listAll || listBuckets {
		listS3Buckets(cfg)
	}

	if listAll || listBackups {
		listAvailableBackups(cfg)
	}
}

func listBackupJobs(cfg *config.BackupConfig) {
	fmt.Println("=== Backup Jobs ===")

	if len(cfg.Jobs) == 0 {
		fmt.Println("No backup jobs configured.")
		fmt.Println("Use 'backtide jobs add' to create backup jobs.")
		return
	}

	for i, job := range cfg.Jobs {
		fmt.Printf("\n%d. %s\n", i+1, job.Name)

		status := "❌ disabled"
		if job.Enabled {
			status = "✅ enabled"
		}
		fmt.Printf("   Status: %s\n", status)

		if job.Description != "" {
			fmt.Printf("   Description: %s\n", job.Description)
		}

		fmt.Printf("   ID: %s\n", job.ID)

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

	fmt.Printf("\n📊 Total jobs: %d\n", len(cfg.Jobs))
}

func listS3Buckets(cfg *config.BackupConfig) {
	fmt.Println("\n=== S3 Bucket Configurations ===")

	if len(cfg.Buckets) == 0 {
		fmt.Println("No bucket configurations found.")
		fmt.Println("Use 'backtide s3 add' to add a bucket configuration.")
		return
	}

	// Calculate usage count for each bucket
	usageCount := make(map[string]int)
	for _, job := range cfg.Jobs {
		if job.BucketID != "" {
			usageCount[job.BucketID]++
		}
	}

	for _, bucket := range cfg.Buckets {
		fmt.Printf("\n📦 %s\n", bucket.Name)
		if bucket.Description != "" {
			fmt.Printf("   Description: %s\n", bucket.Description)
		}
		fmt.Printf("   ID: %s\n", bucket.ID)
		fmt.Printf("   Provider: %s\n", bucket.Provider)
		fmt.Printf("   Bucket: %s\n", bucket.Bucket)
		fmt.Printf("   Region: %s\n", bucket.Region)
		fmt.Printf("   Endpoint: %s\n", func() string {
			if bucket.Endpoint == "" {
				return "AWS default"
			}
			return bucket.Endpoint
		}())
		fmt.Printf("   Mount Point: %s\n", bucket.MountPoint)
		fmt.Printf("   Path Style: %v\n", bucket.UsePathStyle)
		fmt.Printf("   Access Key: %s\n", maskString(bucket.AccessKey))
		fmt.Printf("   Secret Key: %s\n", maskString(bucket.SecretKey))
		fmt.Printf("   Used by: %d job(s)\n", usageCount[bucket.ID])
	}

	fmt.Printf("\n📊 Total buckets: %d\n", len(cfg.Buckets))
}

func listAvailableBackups(cfg *config.BackupConfig) {
	fmt.Println("\n=== Available Backups ===")

	backupRunner := backup.NewBackupRunner(*cfg)
	backups, err := backupRunner.ListBackups()
	if err != nil {
		fmt.Printf("Error listing backups: %v\n", err)
		return
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return
	}

	// Sort backups by timestamp (newest first)
	for i := range backups {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].Timestamp.Before(backups[j].Timestamp) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	for i, backup := range backups {
		fmt.Printf("\n%d. %s\n", i+1, backup.ID)
		fmt.Printf("   Timestamp: %s\n", backup.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Age: %s\n", time.Since(backup.Timestamp).Round(time.Hour))
		fmt.Printf("   Total Size: %d bytes\n", backup.TotalSize)
		fmt.Printf("   Compressed: %v\n", backup.Compressed)
		fmt.Printf("   Checksum: %s\n", backup.Checksum)

		if len(backup.Directories) > 0 {
			fmt.Printf("   Directories: %d\n", len(backup.Directories))
			for _, dir := range backup.Directories {
				fmt.Printf("     - %s: %d files, %d bytes\n", dir.Name, dir.FileCount, dir.Size)
			}
		}
	}

	fmt.Printf("\n📊 Total backups: %d\n", len(backups))
}

func maskString(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
