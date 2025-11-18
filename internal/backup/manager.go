package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitexleo/backtide/internal/config"
)

// BackupManager handles backup operations
type BackupManager struct {
	config     config.BackupConfig
	backupPath string
}

// NewBackupManager creates a new backup manager instance
func NewBackupManager(cfg config.BackupConfig) *BackupManager {
	return &BackupManager{
		config:     cfg,
		backupPath: cfg.BackupPath,
	}
}

// CreateBackup creates a backup of specified directories
func (bm *BackupManager) CreateBackup(ctx context.Context) (*config.BackupMetadata, error) {
	backupID := generateBackupID()
	backupDir := filepath.Join(bm.backupPath, backupID)

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	var backupDirs []config.BackupDirectory
	totalSize := int64(0)
	fileCount := 0

	fmt.Printf("Creating backup: %s\n", backupID)
	fmt.Printf("Backup directory: %s\n", backupDir)

	// Process each directory in the first job (for now, single job support)
	if len(bm.config.Jobs) == 0 {
		return nil, fmt.Errorf("no backup jobs configured")
	}

	job := bm.config.Jobs[0]

	for _, dirConfig := range job.Directories {
		fmt.Printf("Backing up directory: %s -> %s\n", dirConfig.Path, dirConfig.Name)

		// Check if source directory exists
		if _, err := os.Stat(dirConfig.Path); os.IsNotExist(err) {
			fmt.Printf("⚠️  Warning: Source directory does not exist: %s\n", dirConfig.Path)
			continue
		}

		// Create backup file
		backupFileName := fmt.Sprintf("%s.tar.gz", dirConfig.Name)
		if dirConfig.Compression {
			backupFileName = fmt.Sprintf("%s.tar.gz", dirConfig.Name)
		} else {
			backupFileName = fmt.Sprintf("%s.tar", dirConfig.Name)
		}
		backupFilePath := filepath.Join(backupDir, backupFileName)

		// Check for cancellation
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("backup cancelled: %w", err)
		}

		// Create backup file
		backupFile, err := os.Create(backupFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create backup file: %w", err)
		}
		defer backupFile.Close()

		var writer io.Writer = backupFile
		if dirConfig.Compression {
			gzipWriter := gzip.NewWriter(backupFile)
			defer gzipWriter.Close()
			writer = gzipWriter
		}

		tarWriter := tar.NewWriter(writer)
		defer tarWriter.Close()

		// Backup the directory
		dirSize, dirFileCount, err := bm.backupDirectory(ctx, tarWriter, dirConfig.Path, dirConfig.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to backup directory %s: %w", dirConfig.Path, err)
		}

		// Calculate checksum
		checksum, err := bm.calculateChecksum(backupFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate checksum: %w", err)
		}

		backupDirInfo := config.BackupDirectory{
			Path:        dirConfig.Path,
			Name:        dirConfig.Name,
			Size:        dirSize,
			FileCount:   dirFileCount,
			Permissions: make(map[string]config.FilePerm),
			Checksum:    checksum,
			Compressed:  dirConfig.Compression,
		}

		backupDirs = append(backupDirs, backupDirInfo)
		totalSize += dirSize
		fileCount += dirFileCount

		fmt.Printf("✅ Backed up %s: %d files, %d bytes\n", dirConfig.Name, dirFileCount, dirSize)
	}

	// Create metadata
	metadata := &config.BackupMetadata{
		ID:          backupID,
		Timestamp:   time.Now(),
		Directories: backupDirs,
		TotalSize:   totalSize,
		Checksum:    bm.calculateOverallChecksum(backupDirs),
		Compressed:  job.Directories[0].Compression, // Assume all same compression for now
	}

	// Save metadata
	if err := bm.saveMetadata(backupDir, metadata); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	fmt.Printf("✅ Backup completed: %s\n", backupID)
	fmt.Printf("📊 Summary: %d directories, %d total files, %d total bytes\n",
		len(backupDirs), fileCount, totalSize)

	return metadata, nil
}

// backupDirectory recursively backs up a directory to tar
func (bm *BackupManager) backupDirectory(ctx context.Context, tarWriter *tar.Writer, sourceDir, backupName string) (int64, int, error) {
	var totalSize int64
	var fileCount int

	err := filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		// Check for cancellation
		if ctx.Err() != nil {
			return fmt.Errorf("backup cancelled")
		}

		if err != nil {
			return err
		}

		// Skip the directory itself
		if filePath == sourceDir {
			return nil
		}

		// Create relative path for tar header
		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}
		tarPath := filepath.Join(backupName, relPath)

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = tarPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If it's a regular file, write its content
		if info.Mode().IsRegular() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}

			totalSize += info.Size()
			fileCount++
		}

		return nil
	})

	return totalSize, fileCount, err
}

// calculateChecksum calculates SHA256 checksum of a file
func (bm *BackupManager) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// calculateOverallChecksum calculates a combined checksum for all backup directories
func (bm *BackupManager) calculateOverallChecksum(dirs []config.BackupDirectory) string {
	hash := sha256.New()
	for _, dir := range dirs {
		hash.Write([]byte(dir.Checksum))
		hash.Write([]byte(dir.Path))
		hash.Write([]byte(dir.Name))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// RestoreBackup restores a backup to original locations
func (bm *BackupManager) RestoreBackup(backupID string) error {
	return bm.restoreBackupInternal(backupID, "")
}

// RestoreBackupToPath restores a backup to a custom target path
func (bm *BackupManager) RestoreBackupToPath(backupID string, targetPath string) error {
	if targetPath == "" {
		return fmt.Errorf("target path cannot be empty")
	}
	return bm.restoreBackupInternal(backupID, targetPath)
}

// restoreBackupInternal handles the core restoration logic
func (bm *BackupManager) restoreBackupInternal(backupID string, targetPath string) error {
	backupDir := filepath.Join(bm.backupPath, backupID)

	// Check if backup exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	// Load metadata
	metadata, err := bm.loadMetadata(backupDir)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	fmt.Printf("Restoring backup: %s\n", backupID)
	fmt.Printf("Backup date: %s\n", metadata.Timestamp.Format(time.RFC3339))

	if targetPath != "" {
		fmt.Printf("Target path: %s\n", targetPath)
		// Validate target path
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	for _, dir := range metadata.Directories {
		// Determine target directory
		actualTargetPath := dir.Path
		if targetPath != "" {
			// Use custom target path + directory name
			actualTargetPath = filepath.Join(targetPath, dir.Name)
		}

		fmt.Printf("Restoring directory: %s -> %s\n", dir.Name, actualTargetPath)

		// Create target directory
		if err := os.MkdirAll(actualTargetPath, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}

		// Find backup file
		backupFileName := fmt.Sprintf("%s.tar", dir.Name)
		if dir.Compressed {
			backupFileName = fmt.Sprintf("%s.tar.gz", dir.Name)
		}
		backupFilePath := filepath.Join(backupDir, backupFileName)

		if _, err := os.Stat(backupFilePath); os.IsNotExist(err) {
			return fmt.Errorf("backup file not found: %s", backupFilePath)
		}

		// Restore from tar
		if err := bm.restoreFromTar(backupFilePath, actualTargetPath, dir.Compressed); err != nil {
			return fmt.Errorf("failed to restore %s: %w", dir.Name, err)
		}

		fmt.Printf("✅ Restored %s: %d files, %d bytes\n", dir.Name, dir.FileCount, dir.Size)
	}

	fmt.Printf("✅ Restore completed: %s\n", backupID)
	return nil
}

// restoreFromTar extracts files from tar archive
func (bm *BackupManager) restoreFromTar(tarPath, targetDir string, compressed bool) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file
	if compressed {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip the root backup name directory and extract relative paths
		parts := strings.Split(header.Name, string(filepath.Separator))
		if len(parts) > 1 {
			relPath := filepath.Join(parts[1:]...)
			targetPath := filepath.Join(targetDir, relPath)

			// Create directory if needed
			if header.Typeflag == tar.TypeDir {
				if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
					return err
				}
				continue
			}

			// Skip sockets and other special files that can't be restored
			if header.Typeflag == tar.TypeFifo || header.Typeflag == tar.TypeChar ||
				header.Typeflag == tar.TypeBlock || header.Typeflag == tar.TypeSymlink {
				fmt.Printf("⚠️  Skipping special file: %s\n", header.Name)
				continue
			}

			// Create parent directories
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			// Create file
			outFile, err := os.Create(targetPath)
			if err != nil {
				// If we can't create the file (permission issues), skip with warning
				fmt.Printf("⚠️  Warning: Failed to create file %s: %v\n", targetPath, err)
				continue
			}

			// Copy file content
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				// If copy fails, remove the partial file
				os.Remove(targetPath)
				fmt.Printf("⚠️  Warning: Failed to copy content to %s: %v\n", targetPath, err)
				continue
			}

			// Set file permissions
			if err := outFile.Chmod(os.FileMode(header.Mode)); err != nil {
				outFile.Close()
				fmt.Printf("⚠️  Warning: Failed to set permissions on %s: %v\n", targetPath, err)
				// Continue anyway - better to have the file with wrong permissions than not at all
			}

			outFile.Close()
		}
	}

	return nil
}

// ListBackups lists available backups
func (bm *BackupManager) ListBackups() ([]config.BackupMetadata, error) {
	return bm.listBackupsFromPath(bm.backupPath)
}

// ListBackupsFromPath lists backups from a specific path (config-independent)
func (bm *BackupManager) ListBackupsFromPath(path string) ([]config.BackupMetadata, error) {
	return bm.listBackupsFromPath(path)
}

// listBackupsFromPath is the internal implementation for listing backups
func (bm *BackupManager) listBackupsFromPath(path string) ([]config.BackupMetadata, error) {
	var backups []config.BackupMetadata

	// Check if backup directory exists or if backup path is empty
	if path == "" {
		return backups, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return backups, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "backup-") {
			backupDir := filepath.Join(path, entry.Name())
			metadata, err := bm.loadMetadata(backupDir)
			if err != nil {
				fmt.Printf("Warning: Failed to load metadata for %s: %v\n", entry.Name(), err)
				continue
			}
			backups = append(backups, *metadata)
		}
	}

	return backups, nil
}

// CleanupBackups removes old backups based on retention policy
func (bm *BackupManager) CleanupBackups() error {
	if len(bm.config.Jobs) == 0 {
		return fmt.Errorf("no backup jobs configured")
	}

	job := bm.config.Jobs[0]
	retention := job.Retention

	backups, err := bm.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	fmt.Printf("Cleaning up backups based on retention: %d days, %d recent, %d monthly\n",
		retention.KeepDays, retention.KeepCount, retention.KeepMonthly)

	// Sort backups by timestamp (newest first)
	for i := 0; i < len(backups); i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].Timestamp.Before(backups[j].Timestamp) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	removedCount := 0
	cutoffTime := time.Now().AddDate(0, 0, -retention.KeepDays)

	for i, backup := range backups {
		shouldRemove := false

		// Remove if older than retention days
		if backup.Timestamp.Before(cutoffTime) {
			shouldRemove = true
		}

		// Remove if beyond recent count (keep the most recent ones)
		if i >= retention.KeepCount {
			shouldRemove = true
		}

		// TODO: Implement monthly retention logic

		if shouldRemove {
			backupDir := filepath.Join(bm.backupPath, backup.ID)
			if err := os.RemoveAll(backupDir); err != nil {
				fmt.Printf("Warning: Failed to remove backup %s: %v\n", backup.ID, err)
			} else {
				fmt.Printf("Removed old backup: %s (%s)\n", backup.ID, backup.Timestamp.Format("2006-01-02"))
				removedCount++
			}
		}
	}

	fmt.Printf("✅ Cleanup completed: removed %d old backups\n", removedCount)
	return nil
}

// GetBackupInfo returns information about a specific backup
func (bm *BackupManager) GetBackupInfo(backupID string) (*config.BackupMetadata, error) {
	backupDir := filepath.Join(bm.backupPath, backupID)
	return bm.loadMetadata(backupDir)
}

// generateBackupID generates a unique backup ID
func generateBackupID() string {
	return fmt.Sprintf("backup-%d", time.Now().Unix())
}

// saveMetadata saves backup metadata to a file
func (bm *BackupManager) saveMetadata(backupDir string, metadata *config.BackupMetadata) error {
	metadataPath := filepath.Join(backupDir, "metadata.toml")
	return config.SaveBackupMetadata(metadata, metadataPath)
}

// loadMetadata loads backup metadata from a file
func (bm *BackupManager) loadMetadata(backupDir string) (*config.BackupMetadata, error) {
	metadataPath := filepath.Join(backupDir, "metadata.toml")
	return config.LoadBackupMetadata(metadataPath)
}

// saveMetadataToPath saves metadata to a specific path
func (bm *BackupManager) saveMetadataToPath(path string, metadata *config.BackupMetadata) error {
	return config.SaveBackupMetadata(metadata, path)
}
