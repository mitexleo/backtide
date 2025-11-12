package s3fs

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitexleo/backtide/internal/config"
)

// S3FSManager handles S3FS mount operations
type S3FSManager struct {
	config config.S3Config
}

// NewS3FSManager creates a new S3FS manager instance
func NewS3FSManager(cfg config.S3Config) *S3FSManager {
	return &S3FSManager{
		config: cfg,
	}
}

// InstallS3FS installs s3fs-fuse if not already installed
func (sm *S3FSManager) InstallS3FS() error {
	// Check if s3fs is already installed
	if sm.isS3FSInstalled() {
		fmt.Println("s3fs-fuse is already installed")
		return nil
	}

	fmt.Println("Installing s3fs-fuse...")

	// Try different package managers
	packageManagers := []string{"apt-get", "yum", "dnf", "zypper", "apk"}
	var installCmd *exec.Cmd

	for _, pm := range packageManagers {
		if sm.isPackageManagerAvailable(pm) {
			switch pm {
			case "apt-get":
				installCmd = exec.Command("apt-get", "update")
				if err := installCmd.Run(); err != nil {
					return fmt.Errorf("failed to update package lists: %w", err)
				}
				installCmd = exec.Command("apt-get", "install", "-y", "s3fs")
			case "yum", "dnf":
				installCmd = exec.Command(pm, "install", "-y", "s3fs-fuse")
			case "zypper":
				installCmd = exec.Command("zypper", "install", "-y", "s3fs")
			case "apk":
				installCmd = exec.Command("apk", "add", "s3fs-fuse")
			}
			break
		}
	}

	if installCmd == nil {
		return fmt.Errorf("no supported package manager found. Please install s3fs-fuse manually")
	}

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install s3fs-fuse: %w", err)
	}

	fmt.Println("s3fs-fuse installed successfully")
	return nil
}

// SetupS3FS creates necessary directories and configuration
func (sm *S3FSManager) SetupS3FS() error {
	// Create mount point directory
	if err := os.MkdirAll(sm.config.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point directory: %w", err)
	}

	// Create credentials file
	credsDir := filepath.Dir("/etc/passwd-s3fs")
	if err := os.MkdirAll(credsDir, 0755); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	credsContent := fmt.Sprintf("%s:%s", sm.config.AccessKey, sm.config.SecretKey)
	if err := os.WriteFile("/etc/passwd-s3fs", []byte(credsContent), 0600); err != nil {
		return fmt.Errorf("failed to create credentials file: %w", err)
	}

	fmt.Printf("S3FS setup completed. Mount point: %s\n", sm.config.MountPoint)
	return nil
}

// MountS3FS mounts the S3 bucket
func (sm *S3FSManager) MountS3FS() error {
	// Check if already mounted
	if sm.isMounted() {
		fmt.Printf("S3 bucket is already mounted at %s\n", sm.config.MountPoint)
		return nil
	}

	// Build mount command
	args := []string{
		sm.config.Bucket,
		sm.config.MountPoint,
		"-o", "passwd_file=/etc/passwd-s3fs",
		"-o", "use_path_request_style",
		"-o", "url=https://s3.amazonaws.com",
		"-o", "allow_other",
		"-o", "umask=000",
	}

	// Add region if specified
	if sm.config.Region != "" {
		args = append(args, "-o", fmt.Sprintf("endpoint=%s", sm.config.Region))
	}

	// Add custom endpoint if specified
	if sm.config.Endpoint != "" {
		args = append(args, "-o", fmt.Sprintf("url=%s", sm.config.Endpoint))
	}

	// Add path style if specified
	if sm.config.UsePathStyle {
		args = append(args, "-o", "use_path_request_style")
	}

	cmd := exec.Command("s3fs", args...)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount S3 bucket: %s, error: %w", string(output), err)
	}

	fmt.Printf("Successfully mounted S3 bucket %s at %s\n", sm.config.Bucket, sm.config.MountPoint)
	return nil
}

// UnmountS3FS unmounts the S3 bucket
func (sm *S3FSManager) UnmountS3FS() error {
	if !sm.isMounted() {
		fmt.Println("S3 bucket is not mounted")
		return nil
	}

	cmd := exec.Command("fusermount", "-u", sm.config.MountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unmount S3 bucket: %s, error: %w", string(output), err)
	}

	fmt.Printf("Successfully unmounted S3 bucket from %s\n", sm.config.MountPoint)
	return nil
}

// AddToFstab adds S3FS mount to /etc/fstab for persistence
func (sm *S3FSManager) AddToFstab() error {
	fstabEntry := fmt.Sprintf(
		"s3fs#%s %s fuse _netdev,allow_other,use_path_request_style,passwd_file=/etc/passwd-s3fs,url=https://s3.amazonaws.com 0 0",
		sm.config.Bucket,
		sm.config.MountPoint,
	)

	// Read current fstab
	data, err := os.ReadFile("/etc/fstab")
	if err != nil {
		return fmt.Errorf("failed to read /etc/fstab: %w", err)
	}

	// Check if entry already exists
	if strings.Contains(string(data), fstabEntry) {
		fmt.Println("S3FS entry already exists in /etc/fstab")
		return nil
	}

	// Append entry to fstab
	f, err := os.OpenFile("/etc/fstab", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open /etc/fstab: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(fstabEntry + "\n"); err != nil {
		return fmt.Errorf("failed to write to /etc/fstab: %w", err)
	}

	fmt.Println("Successfully added S3FS entry to /etc/fstab")
	return nil
}

// RemoveFromFstab removes S3FS mount from /etc/fstab
func (sm *S3FSManager) RemoveFromFstab() error {
	data, err := os.ReadFile("/etc/fstab")
	if err != nil {
		return fmt.Errorf("failed to read /etc/fstab: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	entryPattern := fmt.Sprintf("s3fs#%s %s fuse", sm.config.Bucket, sm.config.MountPoint)

	for _, line := range lines {
		if !strings.Contains(line, entryPattern) {
			newLines = append(newLines, line)
		}
	}

	if err := os.WriteFile("/etc/fstab", []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write /etc/fstab: %w", err)
	}

	fmt.Println("Successfully removed S3FS entry from /etc/fstab")
	return nil
}

// isS3FSInstalled checks if s3fs is installed
func (sm *S3FSManager) isS3FSInstalled() bool {
	cmd := exec.Command("which", "s3fs")
	return cmd.Run() == nil
}

// isPackageManagerAvailable checks if a package manager is available
func (sm *S3FSManager) isPackageManagerAvailable(manager string) bool {
	cmd := exec.Command("which", manager)
	return cmd.Run() == nil
}

// isMounted checks if the S3 bucket is currently mounted
func (sm *S3FSManager) isMounted() bool {
	cmd := exec.Command("mount")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), sm.config.MountPoint) && strings.Contains(scanner.Text(), "s3fs") {
			return true
		}
	}

	return false
}

// GetMountPoint returns the configured mount point
func (sm *S3FSManager) GetMountPoint() string {
	return sm.config.MountPoint
}
