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
	config config.BucketConfig
}

// NewS3FSManager creates a new S3FS manager instance
func NewS3FSManager(cfg config.BucketConfig) *S3FSManager {
	return &S3FSManager{
		config: cfg,
	}
}

// InstallS3FS installs s3fs-fuse if not already installed
func (sm *S3FSManager) InstallS3FS() error {
	// Check if s3fs is already installed
	if sm.isS3FSInstalled() {
		fmt.Println("❌ s3fs is not installed")
		return nil
	}

	fmt.Println("Installing s3fs...")
	fmt.Println("⚠️  This operation requires sudo privileges.")

	// Try different package managers
	packageManagers := []string{"apt-get", "yum", "dnf", "zypper", "apk"}
	var installCmd *exec.Cmd
	var needsSudo bool = true

	for _, pm := range packageManagers {
		if sm.isPackageManagerAvailable(pm) {
			switch pm {
			case "apt-get":
				// Try without sudo first
				if sm.isPackageManagerAvailable("apt") {
					installCmd = exec.Command("apt", "update")
					if installCmd.Run() == nil {
						installCmd = exec.Command("apt", "install", "-y", "s3fs")
						needsSudo = false
						break
					}
				}
				// Fallback to apt-get with sudo
				installCmd = exec.Command("sudo", "apt-get", "update")
				if err := installCmd.Run(); err != nil {
					return fmt.Errorf("failed to update package lists: %w", err)
				}
				installCmd = exec.Command("sudo", "apt-get", "install", "-y", "s3fs")
			case "yum", "dnf":
				installCmd = exec.Command("sudo", pm, "install", "-y", "s3fs")
			case "zypper":
				installCmd = exec.Command("sudo", "zypper", "install", "-y", "s3fs")
			case "apk":
				installCmd = exec.Command("sudo", "apk", "add", "s3fs")
			}
			break
		}
	}

	if installCmd == nil {
		return fmt.Errorf("no supported package manager found. Please install s3fs manually")
	}

	if needsSudo {
		fmt.Println("🔐 Running with sudo to install s3fs...")
		fmt.Println("   You may be prompted for your password")
	}

	if err := installCmd.Run(); err != nil {
		if needsSudo {
			return fmt.Errorf("failed to install s3fs with sudo: %w\n💡 Try running: sudo apt-get install s3fs", err)
		}
		return fmt.Errorf("failed to install s3fs: %w", err)
	}

	fmt.Println("✅ s3fs installed successfully")
	return nil
}

// SetupS3FS creates necessary directories and configuration
func (sm *S3FSManager) SetupS3FS() error {
	// Create mount point directory
	if err := os.MkdirAll(sm.config.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point directory: %w", err)
	}

	// Create credentials file in user-specific location, per bucket
	// Try to get the original user's home directory, not root's when using sudo
	homeDir := os.Getenv("SUDO_USER")
	if homeDir == "" {
		// Fall back to current user if not using sudo
		homeDir = os.Getenv("HOME")
	}
	if homeDir == "" {
		// Final fallback to UserHomeDir
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
	}

	credsDir := filepath.Join(homeDir, ".config", "backtide", "s3-credentials")
	if err := os.MkdirAll(credsDir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	// Create unique credential file per bucket using bucket ID
	credsFile := filepath.Join(credsDir, fmt.Sprintf("passwd-s3fs-%s", sm.config.ID))
	credsContent := fmt.Sprintf("%s:%s", sm.config.AccessKey, sm.config.SecretKey)
	if err := os.WriteFile(credsFile, []byte(credsContent), 0600); err != nil {
		return fmt.Errorf("failed to create credentials file: %w", err)
	}

	fmt.Printf("S3FS setup completed. Mount point: %s\n", sm.config.MountPoint)
	return nil
}

// InstallS3FSWithPrompt installs s3fs with user confirmation
func (sm *S3FSManager) InstallS3FSWithPrompt() error {
	// Check if s3fs is already installed
	if sm.isS3FSInstalled() {
		fmt.Println("s3fs is already installed")
		return nil
	}

	fmt.Println("s3fs is required for S3 bucket operations but is not installed.")
	fmt.Print("Do you want to install it now? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		return fmt.Errorf("s3fs installation cancelled by user")
	}

	fmt.Println("Installing s3fs...")
	return sm.InstallS3FS()
}

// MountS3FS mounts the S3 bucket
func (sm *S3FSManager) MountS3FS() error {
	// Check if already mounted
	if sm.isMounted() {
		fmt.Printf("S3 bucket is already mounted at %s\n", sm.config.MountPoint)
		return nil
	}

	// Get credentials file path for this specific bucket
	// Try to get the original user's home directory, not root's when using sudo
	homeDir := os.Getenv("SUDO_USER")
	if homeDir == "" {
		// Fall back to current user if not using sudo
		homeDir = os.Getenv("HOME")
	}
	if homeDir == "" {
		// Final fallback to UserHomeDir
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
	}
	credsFile := filepath.Join(homeDir, ".config", "backtide", "s3-credentials", fmt.Sprintf("passwd-s3fs-%s", sm.config.ID))

	// Build mount command
	args := []string{
		sm.config.Bucket,
		sm.config.MountPoint,
		"-o", fmt.Sprintf("passwd_file=%s", credsFile),
		"-o", "allow_other",
		"-o", "umask=000",
	}

	// Use custom endpoint if specified, otherwise use region-based endpoint
	if sm.config.Endpoint != "" {
		args = append(args, "-o", fmt.Sprintf("url=%s", sm.config.Endpoint))
	} else if sm.config.Region != "" {
		// Use region-specific endpoint for AWS
		args = append(args, "-o", fmt.Sprintf("url=https://s3.%s.amazonaws.com", sm.config.Region))
	} else {
		// Default to global AWS endpoint
		args = append(args, "-o", "url=https://s3.amazonaws.com")
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
	// Get credentials file path for fstab for this specific bucket
	// Try to get the original user's home directory, not root's when using sudo
	homeDir := os.Getenv("SUDO_USER")
	if homeDir == "" {
		// Fall back to current user if not using sudo
		homeDir = os.Getenv("HOME")
	}
	if homeDir == "" {
		// Final fallback to UserHomeDir
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
	}
	credsFile := filepath.Join(homeDir, ".config", "backtide", "s3-credentials", fmt.Sprintf("passwd-s3fs-%s", sm.config.ID))

	// Build fstab options
	options := []string{
		"_netdev",
		"allow_other",
		fmt.Sprintf("passwd_file=%s", credsFile),
	}

	// Add endpoint URL
	if sm.config.Endpoint != "" {
		options = append(options, fmt.Sprintf("url=%s", sm.config.Endpoint))
	} else if sm.config.Region != "" {
		options = append(options, fmt.Sprintf("url=https://s3.%s.amazonaws.com", sm.config.Region))
	} else {
		options = append(options, "url=https://s3.amazonaws.com")
	}

	// Add path style if specified
	if sm.config.UsePathStyle {
		options = append(options, "use_path_request_style")
	}

	fstabEntry := fmt.Sprintf(
		"s3fs#%s %s fuse %s 0 0",
		sm.config.Bucket,
		sm.config.MountPoint,
		strings.Join(options, ","),
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

// IsS3FSInstalled checks if s3fs is installed (exported version)
func (sm *S3FSManager) IsS3FSInstalled() bool {
	return sm.isS3FSInstalled()
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
