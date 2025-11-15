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
	"strconv"
	"syscall"
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
	// Determine backup directory based on storage configuration
	var backupDir string
	var useLocalStorage bool

	// Check if the current job uses local storage
	if len(bm.config.Jobs) > 0 {
		useLocalStorage = bm.config.Jobs[0].Storage.Local
	}

	if useLocalStorage && bm.backupPath != "" {
		backupDir = filepath.Join(bm.backupPath, backupID)
	} else {
		// S3-only mode - use temporary directory that will be copied to S3
		backupDir = filepath.Join(bm.config.TempPath, backupID)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// In S3-only mode, ensure we clean up the temp directory after backup
	if !useLocalStorage {
		defer func() {
			if err := os.RemoveAll(backupDir); err != nil {
				fmt.Printf("Warning: Failed to clean up temp directory %s: %v\n", backupDir, err)
			}
		}()
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

	// Save metadata based on storage configuration
	if useLocalStorage {
		// Save metadata to local storage
		if err := bm.saveMetadata(backupDir, metadata); err != nil {
			return nil, fmt.Errorf("failed to save backup metadata locally: %w", err)
		}
		fmt.Printf("✅ Backup metadata stored locally\n")
	}

	// Save metadata to S3 if configured and S3 storage is enabled
	var useS3Storage bool
	if len(bm.config.Jobs) > 0 {
		useS3Storage = bm.config.Jobs[0].Storage.S3
	}

	if useS3Storage && bm.config.S3Config.Bucket != "" {
		s3MetadataPath := filepath.Join(bm.config.S3Config.MountPoint, "backtide-metadata", backupID, "metadata.json")
		if err := bm.saveMetadataToPath(s3MetadataPath, metadata); err != nil {
			fmt.Printf("Warning: Failed to save metadata to S3: %v\n", err)
			if !useLocalStorage {
				// In S3-only mode, this is critical - we can't proceed without S3 metadata
				return nil, fmt.Errorf("S3-only mode requires metadata storage in S3: %w", err)
			} else {
				fmt.Println("Metadata is only stored locally. Consider mounting S3 for disaster recovery.")
			}
		} else {
			fmt.Printf("✅ Backup metadata stored in S3\n")
		}
	} else if !useLocalStorage && useS3Storage {
		return nil, fmt.Errorf("S3 storage is enabled but S3 configuration is missing")
	}

	fmt.Printf("Backup created successfully: %s (Size: %d bytes)\n", backupID, totalSize)
	return metadata, nil
}

// RestoreBackup restores a backup to original locations
func (bm *BackupManager) RestoreBackup(backupID string) error {
	// Check storage configuration for this job
	var useLocalStorage, useS3Storage bool
	if len(bm.config.Jobs) > 0 {
		useLocalStorage = bm.config.Jobs[0].Storage.Local
		useS3Storage = bm.config.Jobs[0].Storage.S3
	}

	// Determine backup directory location based on storage configuration
	var backupDir string
	if useLocalStorage && bm.backupPath != "" {
		backupDir = filepath.Join(bm.backupPath, backupID)
	} else if useS3Storage {
		// S3 storage mode - look for backup in S3
		backupDir = filepath.Join(bm.config.S3Config.MountPoint, "backups", backupID)
	} else {
		return fmt.Errorf("no valid storage location found for backup %s", backupID)
	}
	// Try to load metadata from appropriate location
	var metadata *config.BackupMetadata
	var err error
	if useS3Storage && !useLocalStorage {
		// S3-only mode: load metadata from S3
		s3MetadataPath := filepath.Join(bm.config.S3Config.MountPoint, "backtide-metadata", backupID, "metadata.json")
		metadata, err = bm.loadMetadata(s3MetadataPath)
		if err != nil {
			return fmt.Errorf("failed to load backup metadata from S3: %w", err)
		}
	} else if useLocalStorage {
		// Local storage mode: load metadata from local
		metadata, err = bm.loadMetadata(backupDir)
		if err != nil && useS3Storage {
			// If local metadata not found but S3 is enabled, try S3
			s3MetadataPath := filepath.Join(bm.config.S3Config.MountPoint, "backtide-metadata", backupID, "metadata.json")
			metadata, err = bm.loadMetadata(s3MetadataPath)
			if err != nil {
				return fmt.Errorf("failed to load backup metadata from S3: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to load backup metadata: %w", err)
		}
	} else {
		return fmt.Errorf("no valid storage location found for metadata")
	}
	if err != nil {
		return fmt.Errorf("failed to load backup metadata: %w", err)
	}

	// For S3 storage mode, we need to copy backup files from S3 to temp location
	var tempRestoreDir string
	if useS3Storage && !useLocalStorage {
		tempRestoreDir = filepath.Join(bm.config.TempPath, "restore", backupID)
		if err := os.MkdirAll(tempRestoreDir, 0755); err != nil {
			return fmt.Errorf("failed to create temp restore directory: %w", err)
		}
		defer os.RemoveAll(tempRestoreDir)
	}

	for _, backupDirInfo := range metadata.Directories {
		var restoreSourceDir string
		if useS3Storage && !useLocalStorage {
			// S3-only mode: copy backup file from S3 to temp location
			s3BackupFile := filepath.Join(bm.config.S3Config.MountPoint, "backups", backupID, fmt.Sprintf("%s.tar.gz", backupDirInfo.Name))
			tempBackupFile := filepath.Join(tempRestoreDir, fmt.Sprintf("%s.tar.gz", backupDirInfo.Name))

			if err := copyFile(s3BackupFile, tempBackupFile); err != nil {
				return fmt.Errorf("failed to copy backup file from S3: %w", err)
			}
			restoreSourceDir = tempRestoreDir
		} else {
			restoreSourceDir = backupDir
		}

		if err := bm.restoreDirectory(restoreSourceDir, backupDirInfo); err != nil {
			return fmt.Errorf("failed to restore directory %s: %w", backupDirInfo.Path, err)
		}
	}

	fmt.Printf("Backup %s restored successfully\n", backupID)
	return nil
}

// ListBackups returns a list of all available backups
func (bm *BackupManager) ListBackups() ([]config.BackupMetadata, error) {
	// Check storage configuration for this job
	var useLocalStorage, useS3Storage bool
	if len(bm.config.Jobs) > 0 {
		useLocalStorage = bm.config.Jobs[0].Storage.Local
		useS3Storage = bm.config.Jobs[0].Storage.S3
	}

	// If only S3 storage is enabled, list from S3
	if useS3Storage && !useLocalStorage {
		return bm.listBackupsFromS3()
	}

	// If only local storage is enabled, list from local
	if useLocalStorage && !useS3Storage {
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

	// If both storage locations are enabled, merge backups from both
	var localBackups, s3Backups []config.BackupMetadata

	if useLocalStorage {
		entries, err := os.ReadDir(bm.backupPath)
		if err == nil {
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

				localBackups = append(localBackups, *metadata)
			}
		}
	}

	if useS3Storage {
		s3Backups, _ = bm.listBackupsFromS3()
	}

	// Merge backups, removing duplicates by ID
	allBackups := make(map[string]config.BackupMetadata)
	for _, backup := range localBackups {
		allBackups[backup.ID] = backup
	}
	for _, backup := range s3Backups {
		allBackups[backup.ID] = backup
	}

	// Convert map back to slice
	var backups []config.BackupMetadata
	for _, backup := range allBackups {
		backups = append(backups, backup)
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
		// Check storage configuration for this job
		var useLocalStorage, useS3Storage bool
		if len(bm.config.Jobs) > 0 {
			useLocalStorage = bm.config.Jobs[0].Storage.Local
			useS3Storage = bm.config.Jobs[0].Storage.S3
		}

		// Delete from local storage if enabled
		if useLocalStorage && bm.backupPath != "" {
			backupDir := filepath.Join(bm.backupPath, backupID)
			if err := os.RemoveAll(backupDir); err != nil {
				fmt.Printf("Warning: Failed to delete backup %s from local storage: %v\n", backupID, err)
			} else {
				fmt.Printf("✅ Deleted backup from local storage: %s\n", backupID)
			}
		}

		// Delete from S3 storage if enabled
		if useS3Storage {
			if err := bm.deleteBackupFromS3(backupID); err != nil {
				fmt.Printf("Warning: Failed to delete backup %s from S3: %v\n", backupID, err)
			} else {
				fmt.Printf("✅ Deleted backup from S3: %s\n", backupID)
			}
		}
	}

	fmt.Printf("Cleanup completed. Removed %d old backups\n", len(toDelete))
	return nil
}

// backupDirectory handles backing up a single directory
func (bm *BackupManager) backupDirectory(dirCfg config.DirectoryConfig, backupDir string) (*config.BackupDirectory, error) {
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s.tar.gz", dirCfg.Name))

	// Check storage configuration for this job
	var useS3Storage bool
	if len(bm.config.Jobs) > 0 {
		useS3Storage = bm.config.Jobs[0].Storage.S3
	}

	// Copy backup file to S3 if S3 storage is enabled
	if useS3Storage {
		defer func() {
			// Copy backup file to S3
			backupID := generateBackupID()
			s3BackupFile := filepath.Join(bm.config.S3Config.MountPoint, "backups", backupID, fmt.Sprintf("%s.tar.gz", dirCfg.Name))
			if err := copyFile(backupFile, s3BackupFile); err != nil {
				fmt.Printf("Warning: Failed to copy backup file to S3: %v\n", err)
			} else {
				fmt.Printf("✅ Backup file copied to S3: %s\n", s3BackupFile)
			}
		}()
	}

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

	// Store the original permissions for the root directory
	rootInfo, err := os.Stat(dirCfg.Path)
	if err == nil {
		permissions["."] = getFilePerm(dirCfg.Path, rootInfo)
	}

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

		// Store permissions with proper UID/GID extraction
		permissions[relPath] = getFilePerm(filePath, info)

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

			// Restore permissions and ownership
			if perm, exists := backupDirInfo.Permissions[header.Name]; exists {
				if err := restoreFilePermissions(targetPath, perm); err != nil {
					return fmt.Errorf("failed to restore permissions for %s: %w", targetPath, err)
				}
			}
		}
	}

	return nil
}

// saveMetadata saves backup metadata to JSON file
func (bm *BackupManager) saveMetadata(backupDir string, metadata *config.BackupMetadata) error {
	metadataFile := filepath.Join(backupDir, "metadata.json")
	return bm.saveMetadataToPath(metadataFile, metadata)
}

// saveMetadataToPath saves backup metadata to a specific path
func (bm *BackupManager) saveMetadataToPath(path string, metadata *config.BackupMetadata) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
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

// getFilePerm extracts detailed file permissions including UID/GID
func getFilePerm(path string, info os.FileInfo) config.FilePerm {
	perm := config.FilePerm{
		Mode:    info.Mode().String(),
		Size:    info.Size(),
		ModTime: info.ModTime().Format(time.RFC3339),
	}

	// Get UID and GID using syscall
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		perm.UID = int(stat.Uid)
		perm.GID = int(stat.Gid)
	}

	return perm
}

// restoreFilePermissions restores file permissions and ownership
func restoreFilePermissions(path string, perm config.FilePerm) error {
	// Parse file mode
	mode, err := parseFileMode(perm.Mode)
	if err != nil {
		return fmt.Errorf("failed to parse mode %s: %w", perm.Mode, err)
	}

	// Set file permissions
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("failed to chmod %s: %w", path, err)
	}

	// Set file ownership (requires root privileges)
	if os.Geteuid() == 0 {
		if err := os.Chown(path, perm.UID, perm.GID); err != nil {
			// Log warning but don't fail - ownership might not be critical
			fmt.Printf("Warning: Failed to chown %s to %d:%d: %v\n", path, perm.UID, perm.GID, err)
		}
	} else {
		fmt.Printf("Info: Skipping chown for %s (not running as root)\n", path)
	}

	return nil
}

func parseFileMode(modeStr string) (os.FileMode, error) {
	// Handle octal mode strings (e.g., "0755")
	if len(modeStr) == 4 && modeStr[0] >= '0' && modeStr[0] <= '7' {
		mode, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			return 0, err
		}
		return os.FileMode(mode), nil
	}

	// Handle symbolic mode strings (e.g., "drwxr-xr-x")
	// Extract the permission bits (last 9 characters)
	if len(modeStr) > 9 {
		modeStr = modeStr[len(modeStr)-9:]
	}

	var mode os.FileMode
	for i, c := range modeStr {
		bitPos := uint(8 - i)
		switch c {
		case 'r':
			mode |= 1 << bitPos
		case 'w':
			mode |= 1 << (bitPos - 1)
		case 'x':
			mode |= 1 << (bitPos - 2)
		case 's', 'S', 't', 'T':
			// Handle special bits - for now, just preserve execute bits
			if i == 2 || i == 5 || i == 8 {
				mode |= 1 << (bitPos - 2)
			}
		case '-':
			// Skip, already 0
		default:
			return 0, fmt.Errorf("invalid file mode character: %c", c)
		}
	}

	return mode, nil
}

// listBackupsFromS3 lists backups from S3 metadata
func (bm *BackupManager) listBackupsFromS3() ([]config.BackupMetadata, error) {
	if bm.config.S3Config.Bucket == "" {
		return nil, fmt.Errorf("S3 configuration required for S3-only mode")
	}

	metadataDir := filepath.Join(bm.config.S3Config.MountPoint, "backtide-metadata")
	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []config.BackupMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read S3 metadata directory: %w", err)
	}

	var backups []config.BackupMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metadataPath := filepath.Join(metadataDir, entry.Name(), "metadata.json")
		metadata, err := bm.loadMetadata(metadataPath)
		if err != nil {
			fmt.Printf("Warning: Failed to load metadata for backup %s: %v\n", entry.Name(), err)
			continue
		}

		backups = append(backups, *metadata)
	}

	return backups, nil
}

// deleteBackupFromS3 deletes a backup from S3 storage
func (bm *BackupManager) deleteBackupFromS3(backupID string) error {
	if bm.config.S3Config.Bucket == "" {
		return fmt.Errorf("S3 configuration required for S3-only mode")
	}

	// Delete metadata from S3
	metadataPath := filepath.Join(bm.config.S3Config.MountPoint, "backtide-metadata", backupID)
	if err := os.RemoveAll(metadataPath); err != nil {
		return fmt.Errorf("failed to delete metadata from S3: %w", err)
	}

	// Delete backup files from S3
	backupPath := filepath.Join(bm.config.S3Config.MountPoint, "backups", backupID)
	if err := os.RemoveAll(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup files from S3: %w", err)
	}

	return nil
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
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
