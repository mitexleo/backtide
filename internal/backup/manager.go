package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
func (bm *BackupManager) CreateBackup() (*config.BackupMetadata, error) {
	backupID := generateBackupID()
	backupDir := filepath.Join(bm.backupPath, backupID)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	var backupDirs []config.BackupDirectory
	var totalSize int64
	checksums := sha256.New()

	for _, dirCfg := range bm.config.Directories {
		backupDirInfo, err := bm.backupDirectory(dirCfg, backupDir)
		if err != nil {
			return nil, fmt.Errorf("failed to backup directory %s: %w", dirCfg.Path, err)
		}

		backupDirs = append(backupDirs, *backupDirInfo)
		totalSize += backupDirInfo.Size
		checksums.Write([]byte(backupDirInfo.Checksum))
	}

	backupChecksum := hex.EncodeToString(checksums.Sum(nil))

	metadata := &config.BackupMetadata{
		ID:          backupID,
		Timestamp:   time.Now(),
		Directories: backupDirs,
		TotalSize:   totalSize,
		Checksum:    backupChecksum,
		Compressed:  bm.config.Directories[0].Compression, // Assume all have same compression setting
	}

	// Save metadata
	if err := bm.saveMetadata(backupDir, metadata); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	fmt.Printf("Backup created successfully: %s (Size: %d bytes)\n", backupID, totalSize)
	return metadata, nil
}

// RestoreBackup restores a backup to original locations
func (bm *BackupManager) RestoreBackup(backupID string) error {
	backupDir := filepath.Join(bm.backupPath, backupID)
	metadata, err := bm.loadMetadata(backupDir)
	if err != nil {
		return fmt.Errorf("failed to load backup metadata: %w", err)
	}

	for _, backupDirInfo := range metadata.Directories {
		if err := bm.restoreDirectory(backupDir, backupDirInfo); err != nil {
			return fmt.Errorf("failed to restore directory %s: %w", backupDirInfo.Path, err)
		}
	}

	fmt.Printf("Backup %s restored successfully\n", backupID)
	return nil
}

// ListBackups returns a list of all available backups
func (bm *BackupManager) ListBackups() ([]config.BackupMetadata, error) {
	entries, err := os.ReadDir(bm.backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []config.BackupMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []config.BackupMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		backupDir := filepath.Join(bm.backupPath, entry.Name())
		metadata, err := bm.loadMetadata(backupDir)
		if err != nil {
			fmt.Printf("Warning: Failed to load metadata for backup %s: %v\n", entry.Name(), err)
			continue
		}

		backups = append(backups, *metadata)
	}

	return backups, nil
}

// CleanupOldBackups removes backups according to retention policy
func (bm *BackupManager) CleanupOldBackups() error {
	backups, err := bm.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	now := time.Now()
	var toDelete []string

	// Apply retention policies
	for _, backup := range backups {
		age := now.Sub(backup.Timestamp)
		shouldDelete := false

		// Keep by days
		if bm.config.RetentionPolicy.KeepDays > 0 && age.Hours() > float64(bm.config.RetentionPolicy.KeepDays*24) {
			shouldDelete = true
		}

		// Keep by count (we'll handle this after sorting)
		if !shouldDelete && bm.config.RetentionPolicy.KeepCount > 0 {
			// We'll handle count-based retention separately
			continue
		}

		if shouldDelete {
			toDelete = append(toDelete, backup.ID)
		}
	}

	// Apply count-based retention
	if bm.config.RetentionPolicy.KeepCount > 0 && len(backups) > bm.config.RetentionPolicy.KeepCount {
		// Sort backups by timestamp (newest first)
		sortedBackups := make([]config.BackupMetadata, len(backups))
		copy(sortedBackups, backups)

		for i := range sortedBackups {
			for j := i + 1; j < len(sortedBackups); j++ {
				if sortedBackups[i].Timestamp.Before(sortedBackups[j].Timestamp) {
					sortedBackups[i], sortedBackups[j] = sortedBackups[j], sortedBackups[i]
				}
			}
		}

		// Mark old backups for deletion
		for i := bm.config.RetentionPolicy.KeepCount; i < len(sortedBackups); i++ {
			toDelete = append(toDelete, sortedBackups[i].ID)
		}
	}

	// Remove duplicates
	toDelete = unique(toDelete)

	// Delete marked backups
	for _, backupID := range toDelete {
		backupDir := filepath.Join(bm.backupPath, backupID)
		if err := os.RemoveAll(backupDir); err != nil {
			fmt.Printf("Warning: Failed to delete backup %s: %v\n", backupID, err)
			continue
		}
		fmt.Printf("Deleted old backup: %s\n", backupID)
	}

	fmt.Printf("Cleanup completed. Removed %d old backups\n", len(toDelete))
	return nil
}

// backupDirectory handles backing up a single directory
func (bm *BackupManager) backupDirectory(dirCfg config.DirectoryConfig, backupDir string) (*config.BackupDirectory, error) {
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s.tar.gz", dirCfg.Name))

	// Create backup file
	file, err := os.Create(backupFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	var writer io.Writer = file
	var gzipWriter *gzip.Writer

	if dirCfg.Compression {
		gzipWriter = gzip.NewWriter(file)
		writer = gzipWriter
		defer gzipWriter.Close()
	}

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Walk through directory and add files to archive
	var totalSize int64
	var fileCount int
	permissions := make(map[string]config.FilePerm)
	checksum := sha256.New()

	err = filepath.Walk(dirCfg.Path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if filePath == dirCfg.Path {
			return nil
		}

		// Get relative path for tar header
		relPath, err := filepath.Rel(dirCfg.Path, filePath)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

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

			// Calculate checksum
			if _, err := file.Seek(0, 0); err == nil {
				if _, err := io.Copy(checksum, file); err != nil {
					return err
				}
			}
		}

		// Store permissions
		permissions[relPath] = config.FilePerm{
			Mode:    info.Mode().String(),
			UID:     getUID(filePath),
			GID:     getGID(filePath),
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		}

		totalSize += info.Size()
		fileCount++

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	backupChecksum := hex.EncodeToString(checksum.Sum(nil))

	return &config.BackupDirectory{
		Path:        dirCfg.Path,
		Name:        dirCfg.Name,
		Size:        totalSize,
		FileCount:   fileCount,
		Permissions: permissions,
		Checksum:    backupChecksum,
		Compressed:  dirCfg.Compression,
	}, nil
}

// restoreDirectory restores a single directory from backup
func (bm *BackupManager) restoreDirectory(backupDir string, backupDirInfo config.BackupDirectory) error {
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s.tar.gz", backupDirInfo.Name))

	// Open backup file
	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file
	var gzipReader *gzip.Reader

	if backupDirInfo.Compressed {
		gzipReader, err = gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		reader = gzipReader
		defer gzipReader.Close()
	}

	tarReader := tar.NewReader(reader)

	// Ensure target directory exists
	if err := os.MkdirAll(backupDirInfo.Path, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		targetPath := filepath.Join(backupDirInfo.Path, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Create parent directories if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directories: %w", err)
			}

			// Create file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()

			// Restore permissions
			if perm, exists := backupDirInfo.Permissions[header.Name]; exists {
				if err := os.Chmod(targetPath, parseFileMode(perm.Mode)); err != nil {
					return fmt.Errorf("failed to restore permissions: %w", err)
				}
				if err := os.Chown(targetPath, perm.UID, perm.GID); err != nil {
					// Chown might fail if not running as root, just log warning
					fmt.Printf("Warning: Failed to change ownership of %s: %v\n", targetPath, err)
				}
			}
		}
	}

	return nil
}

// saveMetadata saves backup metadata to JSON file
func (bm *BackupManager) saveMetadata(backupDir string, metadata *config.BackupMetadata) error {
	metadataFile := filepath.Join(backupDir, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataFile, data, 0644)
}

// loadMetadata loads backup metadata from JSON file
func (bm *BackupManager) loadMetadata(backupDir string) (*config.BackupMetadata, error) {
	metadataFile := filepath.Join(backupDir, "metadata.json")
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, err
	}

	var metadata config.BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// Helper functions
func generateBackupID() string {
	return fmt.Sprintf("backup-%s", time.Now().Format("2006-01-02-15-04-05"))
}

func getUID(path string) int {
	// In a real implementation, you would use syscall.Stat to get UID
	// For now, return 0 (root) as placeholder
	return 0
}

func getGID(path string) int {
	// In a real implementation, you would use syscall.Stat to get GID
	// For now, return 0 (root) as placeholder
	return 0
}

func parseFileMode(modeStr string) os.FileMode {
	// Parse file mode string to os.FileMode
	// This is a simplified implementation
	if strings.HasPrefix(modeStr, "drwx") {
		return 0755
	}
	if strings.HasPrefix(modeStr, "-rwx") {
		return 0755
	}
	if strings.HasPrefix(modeStr, "-rw-") {
		return 0644
	}
	return 0644
}

func unique(strings []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range strings {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
