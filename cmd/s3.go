package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitexleo/backtide/internal/s3fs"

	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	s3Force bool
)

// s3Cmd represents the s3 command
var s3Cmd = &cobra.Command{
	Use:   "s3",
	Short: "Manage S3 bucket configurations",
	Long: `Manage S3 bucket configurations separately from backup jobs.

This command allows you to:
- List all configured S3 buckets
- Add new bucket configurations
- Remove existing bucket configurations
- Test bucket connectivity

Buckets can be reused by multiple backup jobs.`,
}

// s3ListCmd represents the s3 list command
var s3ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all bucket configurations",
	Long: `List all S3 bucket configurations from the current configuration file.

This command shows:
- Bucket ID and name
- Provider type and bucket details
- Mount points and endpoints
- Usage count (how many jobs use this bucket)`,
	Run: runS3List,
}

// s3AddCmd represents the s3 add command
var s3AddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new bucket configuration",
	Long: `Add a new S3 bucket configuration to the global configuration.

This configuration can be reused by multiple backup jobs.
The configuration will be added to the bucket settings.`,
	Run: runS3Add,
}

// s3RemoveCmd represents the s3 remove command
var s3RemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a bucket configuration",
	Long: `Remove a bucket configuration from the global settings.

This will remove the bucket configuration but will not affect
jobs that reference this bucket. Use with caution.`,
	Run: runS3Remove,
}

// s3TestCmd represents the s3 test command
var s3TestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test bucket connectivity",
	Long: `Test connectivity to a configured S3 bucket.

This command will:
- Attempt to mount the S3 bucket
- Create a test file
- Verify read/write permissions
- Clean up test files`,
	Run: runS3Test,
}

func init() {
	rootCmd.AddCommand(s3Cmd)
	s3Cmd.AddCommand(s3ListCmd)
	s3Cmd.AddCommand(s3AddCmd)
	s3Cmd.AddCommand(s3RemoveCmd)
	s3Cmd.AddCommand(s3TestCmd)

	s3RemoveCmd.Flags().BoolVarP(&s3Force, "force", "f", false, "force removal without confirmation")
}

func runS3List(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// For S3 commands, allow empty config and create minimal one
		cfg = config.DefaultConfig()
	}

	fmt.Println("=== S3 Bucket Configurations ===")

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
		printBucketConfig(bucket, usageCount[bucket.ID])
	}

	fmt.Printf("\n📊 Summary: %d bucket configurations\n", len(cfg.Buckets))
}

func runS3Add(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// For S3 commands, allow empty config and create minimal one
		cfg = config.DefaultConfig()
	}

	fmt.Println("=== Add S3 Bucket Configuration ===")

	// Check and install s3fs if needed
	fmt.Println("🔧 Checking for s3fs dependency...")
	checkS3FSManager := s3fs.NewS3FSManager(config.BucketConfig{})
	if !checkS3FSManager.IsS3FSInstalled() {
		fmt.Println("📦 s3fs not found. Installing...")
		if err := checkS3FSManager.InstallS3FS(); err != nil {
			fmt.Printf("❌ Failed to install s3fs: %v\n", err)
			fmt.Println("💡 Please install s3fs manually:")
			fmt.Println("   Ubuntu/Debian: sudo apt-get install s3fs")
			fmt.Println("   CentOS/RHEL: sudo yum install s3fs-fuse")
			fmt.Println("   Fedora: sudo dnf install s3fs-fuse")
			fmt.Println("   openSUSE: sudo zypper install s3fs-fuse")
			fmt.Println("   Alpine: sudo apk add s3fs-fuse")
			return
		}
		fmt.Println("✅ s3fs installed successfully")
	} else {
		fmt.Println("✅ s3fs is already installed")
	}

	// Configure new bucket
	newBucket := configureBucketForAdd()

	// Check for duplicate bucket names
	for _, existingBucket := range cfg.Buckets {
		if existingBucket.Bucket == newBucket.Bucket {
			fmt.Printf("⚠️  A bucket configuration for '%s' already exists.\n", newBucket.Bucket)
			fmt.Print("Do you want to continue anyway? (y/N): ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				fmt.Println("Operation cancelled.")
				return
			}
			break
		}
	}

	cfg.Buckets = append(cfg.Buckets, newBucket)

	// Save configuration
	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	// Create mount point directory
	fmt.Printf("\n📁 Creating mount point: %s\n", newBucket.MountPoint)
	if err := os.MkdirAll(newBucket.MountPoint, 0755); err != nil {
		fmt.Printf("⚠️  Warning: Could not create mount point directory: %v\n", err)
		fmt.Println("   You may need to create it manually or run with sudo")
	} else {
		fmt.Printf("✅ Mount point created: %s\n", newBucket.MountPoint)
	}

	// Add to fstab for persistence
	fmt.Println("📝 Adding to /etc/fstab for automatic mounting...")
	s3fsManager := s3fs.NewS3FSManager(newBucket)
	if err := s3fsManager.AddToFstab(); err != nil {
		fmt.Printf("⚠️  Warning: Could not add to /etc/fstab: %v\n", err)
		fmt.Println("   You may need to run with sudo or add it manually")
	} else {
		fmt.Println("✅ Added to /etc/fstab for automatic mounting")
	}

	fmt.Printf("\n✅ S3 bucket configuration added successfully!\n")
	fmt.Printf("Name: %s\n", newBucket.Name)
	fmt.Printf("Bucket: %s\n", newBucket.Bucket)
	fmt.Printf("Provider: %s\n", newBucket.Provider)
	fmt.Printf("Mount point: %s\n", newBucket.MountPoint)
}

func runS3Remove(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Please specify which bucket configuration to remove.")
		fmt.Println("Usage: backtide s3 remove <bucket-id>")
		fmt.Println("Use 'backtide s3 list' to see available buckets.")
		os.Exit(1)
	}

	bucketID := args[0]
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Remove S3 Bucket Configuration ===")

	// Find the bucket to remove
	var bucketToRemove *config.BucketConfig
	var bucketIndex int = -1
	for i, bucket := range cfg.Buckets {
		if bucket.ID == bucketID || bucket.Name == bucketID {
			bucketToRemove = &cfg.Buckets[i]
			bucketIndex = i
			break
		}
	}

	if bucketToRemove == nil {
		fmt.Printf("Error: No bucket found with ID or name '%s'\n", bucketID)
		fmt.Println("Use 'backtide s3 list' to see available buckets.")
		os.Exit(1)
	}

	// Check if any jobs depend on this bucket
	dependentJobs := []string{}
	for _, job := range cfg.Jobs {
		if job.BucketID == bucketToRemove.ID {
			dependentJobs = append(dependentJobs, job.Name)
		}
	}

	fmt.Printf("Bucket configuration to remove:\n")
	printBucketConfig(*bucketToRemove, len(dependentJobs))

	if len(dependentJobs) > 0 {
		fmt.Printf("\n⚠️  Warning: The following jobs depend on this bucket:\n")
		for _, jobName := range dependentJobs {
			fmt.Printf("   - %s\n", jobName)
		}
		fmt.Println("These jobs will need to be updated with different bucket configurations.")
	}

	if !s3Force {
		fmt.Print("\nAre you sure you want to remove this bucket configuration? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Operation cancelled.")
			return
		}
	}

	// Save bucket name before removal for success message
	bucketName := bucketToRemove.Name

	// Remove the bucket
	cfg.Buckets = append(cfg.Buckets[:bucketIndex], cfg.Buckets[bucketIndex+1:]...)

	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	// Clean up credentials file
	fmt.Println("\n🧹 Cleaning up credentials...")
	if err := cleanupBucketCredentials(*bucketToRemove); err != nil {
		fmt.Printf("⚠️  Warning: Could not clean up credentials: %v\n", err)
		fmt.Println("   You may need to manually remove the credentials file")
	} else {
		fmt.Println("✅ Credentials cleaned up successfully")
	}

	// Remove from fstab
	fmt.Println("📝 Removing from /etc/fstab...")
	s3fsManager := s3fs.NewS3FSManager(*bucketToRemove)
	if err := s3fsManager.RemoveFromFstab(); err != nil {
		fmt.Printf("⚠️  Warning: Could not remove from /etc/fstab: %v\n", err)
		fmt.Println("   You may need to remove it manually or run with sudo")
	} else {
		fmt.Println("✅ Removed from /etc/fstab")
	}

	// Remove mount point directory if empty
	fmt.Println("📁 Removing mount point directory...")
	if err := removeMountPointIfEmpty(bucketToRemove.MountPoint); err != nil {
		fmt.Printf("⚠️  Warning: Could not remove mount point: %v\n", err)
		fmt.Println("   You may need to remove it manually")
	} else {
		fmt.Println("✅ Mount point directory removed")
	}

	fmt.Printf("✅ S3 bucket configuration '%s' removed successfully!\n", bucketName)
	if len(dependentJobs) > 0 {
		fmt.Println("Remember to update dependent jobs with different bucket configurations.")
	}
}

func runS3Test(cmd *cobra.Command, args []string) {
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// For S3 commands, allow empty config and create minimal one
		cfg = config.DefaultConfig()
	}

	fmt.Println("=== Test S3 Bucket Connectivity ===")

	if len(cfg.Buckets) == 0 {
		fmt.Println("No bucket configurations found to test.")
		fmt.Println("Use 'backtide s3 add' to add a configuration first.")
		return
	}

	// Check and install s3fs if needed
	fmt.Println("🔧 Checking for s3fs dependency...")
	checkS3FSManager := s3fs.NewS3FSManager(config.BucketConfig{})
	if !checkS3FSManager.IsS3FSInstalled() {
		fmt.Println("📦 s3fs not found. Installing...")
		if err := checkS3FSManager.InstallS3FS(); err != nil {
			fmt.Printf("❌ Failed to install s3fs: %v\n", err)
			fmt.Println("💡 Please install s3fs manually:")
			fmt.Println("   Ubuntu/Debian: sudo apt-get install s3fs")
			fmt.Println("   CentOS/RHEL: sudo yum install s3fs-fuse")
			fmt.Println("   Fedora: sudo dnf install s3fs-fuse")
			fmt.Println("   openSUSE: sudo zypper install s3fs-fuse")
			fmt.Println("   Alpine: sudo apk add s3fs-fuse")
			return
		}
		fmt.Println("✅ s3fs installed successfully")
	} else {
		fmt.Println("✅ s3fs is already installed")
	}

	// If no specific bucket specified, show available options
	if len(args) == 0 {
		fmt.Println("Available buckets:")
		for i, bucket := range cfg.Buckets {
			fmt.Printf("%d. %s (%s)\n", i+1, bucket.Name, bucket.Bucket)
		}
		fmt.Print("\nSelect bucket to test (number): ")

		reader := bufio.NewReader(os.Stdin)
		choiceStr, _ := reader.ReadString('\n')
		choiceStr = strings.TrimSpace(choiceStr)

		var choice int
		fmt.Sscanf(choiceStr, "%d", &choice)

		if choice < 1 || choice > len(cfg.Buckets) {
			fmt.Println("Invalid selection.")
			return
		}

		bucket := cfg.Buckets[choice-1]
		testBucket(bucket)
		return
	}

	// Test specific bucket
	bucketID := args[0]
	for _, bucket := range cfg.Buckets {
		if bucket.ID == bucketID || bucket.Name == bucketID {
			testBucket(bucket)
			return
		}
	}

	fmt.Printf("Error: No bucket found with ID or name '%s'\n", bucketID)
	fmt.Println("Use 'backtide s3 list' to see available buckets.")
}

func printBucketConfig(bucket config.BucketConfig, usageCount int) {
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
	fmt.Printf("   Credentials File: %s\n", getCredentialsFilePath(bucket.ID))
	fmt.Printf("   Used by: %d job(s)\n", usageCount)
}

func configureBucketForAdd() config.BucketConfig {
	reader := bufio.NewReader(os.Stdin)
	bucket := config.BucketConfig{
		MountPoint: "/mnt/s3backup",
	}

	// Generate a unique ID
	bucket.ID = generateBucketID()

	fmt.Print("Bucket name (display name): ")
	name, _ := reader.ReadString('\n')
	bucket.Name = strings.TrimSpace(name)

	fmt.Print("Description (optional): ")
	desc, _ := reader.ReadString('\n')
	bucket.Description = strings.TrimSpace(desc)

	fmt.Println("\nS3 Provider Options:")
	fmt.Println("1. AWS S3")
	fmt.Println("2. Backblaze B2")
	fmt.Println("3. Wasabi")
	fmt.Println("4. DigitalOcean Spaces")
	fmt.Println("5. MinIO")
	fmt.Println("6. Other S3-compatible provider")
	fmt.Print("Choose provider (1-6): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var providerName string
	var defaultEndpoint string
	var recommendedPathStyle bool

	switch choice {
	case "1":
		providerName = "AWS S3"
		recommendedPathStyle = false
		fmt.Print("AWS Region (e.g., us-east-1): ")
		region, _ := reader.ReadString('\n')
		bucket.Region = strings.TrimSpace(region)

	case "2":
		providerName = "Backblaze B2"
		defaultEndpoint = "https://s3.us-west-002.backblazeb2.com"
		recommendedPathStyle = true
		bucket.Region = ""

	case "3":
		providerName = "Wasabi"
		defaultEndpoint = "https://s3.wasabisys.com"
		recommendedPathStyle = false
		fmt.Print("Wasabi Region (e.g., us-east-1): ")
		region, _ := reader.ReadString('\n')
		bucket.Region = strings.TrimSpace(region)
	case "4":
		providerName = "DigitalOcean Spaces"
		defaultEndpoint = "https://nyc3.digitaloceanspaces.com"
		recommendedPathStyle = false
		fmt.Print("DO Region (e.g., nyc3): ")
		region, _ := reader.ReadString('\n')
		bucket.Region = strings.TrimSpace(region)
	case "5":
		providerName = "MinIO"
		defaultEndpoint = "http://localhost:9000"
		recommendedPathStyle = true
		bucket.Region = ""

	case "6":
		providerName = "Other S3-compatible"
		recommendedPathStyle = false
		fmt.Print("Endpoint URL (e.g., https://s3.example.com): ")
		endpoint, _ := reader.ReadString('\n')
		defaultEndpoint = strings.TrimSpace(endpoint)
	default:
		fmt.Println("Invalid choice, using AWS S3 defaults")
		providerName = "AWS S3"
		recommendedPathStyle = false
	}

	bucket.Provider = providerName
	fmt.Printf("\nConfiguring %s...\n", providerName)

	// Bucket name
	fmt.Print("S3 Bucket name: ")
	s3Bucket, _ := reader.ReadString('\n')
	bucket.Bucket = strings.TrimSpace(s3Bucket)

	// Endpoint
	if defaultEndpoint != "" {
		fmt.Printf("Endpoint [%s]: ", defaultEndpoint)
		endpoint, _ := reader.ReadString('\n')
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			bucket.Endpoint = defaultEndpoint
		} else {
			bucket.Endpoint = endpoint
		}
	} else {
		fmt.Print("Endpoint (leave empty for AWS default): ")
		endpoint, _ := reader.ReadString('\n')
		bucket.Endpoint = strings.TrimSpace(endpoint)
	}

	// Path style
	fmt.Printf("Use path-style endpoints? (recommended: %v) (y/N): ", recommendedPathStyle)
	pathStyle, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(pathStyle)) == "y" {
		bucket.UsePathStyle = true
	} else {
		bucket.UsePathStyle = false
	}

	// Access key
	fmt.Print("Access Key: ")
	accessKey, _ := reader.ReadString('\n')
	bucket.AccessKey = strings.TrimSpace(accessKey)

	// Secret key
	fmt.Print("Secret Key: ")
	secretKey, _ := reader.ReadString('\n')
	bucket.SecretKey = strings.TrimSpace(secretKey)

	// Mount point
	fmt.Printf("Mount point [%s]: ", bucket.MountPoint)
	mountPoint, _ := reader.ReadString('\n')
	mountPoint = strings.TrimSpace(mountPoint)
	if mountPoint != "" {
		bucket.MountPoint = mountPoint
	}

	fmt.Printf("✅ S3 bucket configuration for %s completed!\n", providerName)

	return bucket
}

func generateBucketID() string {
	// Simple ID generation - in production you might want something more robust
	return fmt.Sprintf("bucket-%d", time.Now().Unix())
}

// getCredentialsFilePath returns the path to the credentials file for a bucket
func getCredentialsFilePath(bucketID string) string {
	// Try to get the original user's home directory, not root's when using sudo
	homeDir := os.Getenv("SUDO_USER")
	if homeDir == "" {
		// Fall back to current user if not using sudo
		homeDir = os.Getenv("HOME")
	}
	if homeDir == "" {
		// Final fallback to UserHomeDir
		if dir, err := os.UserHomeDir(); err == nil {
			homeDir = dir
		} else {
			return "unknown"
		}
	}
	return filepath.Join(homeDir, ".config", "backtide", "s3-credentials", fmt.Sprintf("passwd-s3fs-%s", bucketID))
}

// cleanupBucketCredentials removes the credentials file for a bucket
func cleanupBucketCredentials(bucket config.BucketConfig) error {
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

	credsFile := filepath.Join(homeDir, ".config", "backtide", "s3-credentials", fmt.Sprintf("passwd-s3fs-%s", bucket.ID))

	// Check if file exists before trying to remove
	if _, err := os.Stat(credsFile); err == nil {
		if err := os.Remove(credsFile); err != nil {
			return fmt.Errorf("failed to remove credentials file: %w", err)
		}
	}

	return nil
}

// removeMountPointIfEmpty removes the mount point directory if it's empty
func removeMountPointIfEmpty(mountPoint string) error {
	// Check if directory exists
	if _, err := os.Stat(mountPoint); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to do
	}

	// Check if directory is empty
	dir, err := os.Open(mountPoint)
	if err != nil {
		return err
	}
	defer dir.Close()

	_, err = dir.Readdirnames(1)
	if err == io.EOF {
		// Directory is empty, safe to remove
		if err := os.Remove(mountPoint); err != nil {
			return err
		}
	}
	// If directory is not empty, leave it alone
	return nil
}

func testBucket(bucket config.BucketConfig) {
	fmt.Printf("Testing connectivity to: %s\n", bucket.Bucket)
	fmt.Printf("Provider: %s\n", bucket.Provider)
	fmt.Printf("Endpoint: %s\n", func() string {
		if bucket.Endpoint == "" {
			return "AWS default"
		}
		return bucket.Endpoint
	}())
	fmt.Printf("Mount Point: %s\n", bucket.MountPoint)

	fmt.Println("\n🔧 Testing S3 bucket connectivity...")

	// Create S3FS manager
	s3fsManager := s3fs.NewS3FSManager(bucket)

	// Check if s3fs is installed
	fmt.Println("1. Checking if s3fs-fuse is installed...")
	if !s3fsManager.IsS3FSInstalled() {
		fmt.Println("❌ s3fs is not installed")
		fmt.Println("💡 Install it with:")
		fmt.Println("   Ubuntu/Debian: sudo apt-get install s3fs")
		fmt.Println("   CentOS/RHEL: sudo yum install s3fs-fuse")
		fmt.Println("   Fedora: sudo dnf install s3fs-fuse")
		fmt.Println("   openSUSE: sudo zypper install s3fs-fuse")
		fmt.Println("   Alpine: sudo apk add s3fs-fuse")
		return
	}
	fmt.Println("✅ s3fs-fuse is installed")

	// Setup S3FS (create mount point and credentials)
	fmt.Println("2. Setting up S3FS configuration...")
	if err := s3fsManager.SetupS3FS(); err != nil {
		fmt.Printf("❌ Setup failed: %v\n", err)
		return
	}
	fmt.Println("✅ S3FS setup completed")

	// Mount the bucket
	fmt.Println("3. Mounting S3 bucket...")
	if err := s3fsManager.MountS3FS(); err != nil {
		fmt.Printf("❌ Mount failed: %v\n", err)
		fmt.Println("💡 Check your credentials and network connectivity")
		return
	}
	fmt.Println("✅ S3 bucket mounted successfully")

	// Test file operations
	fmt.Println("4. Testing file operations...")
	testFilePath := filepath.Join(bucket.MountPoint, "backtide-test-file.txt")
	testContent := fmt.Sprintf("Backtide connectivity test - %s", time.Now().Format(time.RFC3339))

	// Write test file
	if err := os.WriteFile(testFilePath, []byte(testContent), 0644); err != nil {
		fmt.Printf("❌ Write test failed: %v\n", err)
		s3fsManager.UnmountS3FS()
		return
	}
	fmt.Println("✅ Write test passed")

	// Read test file
	readContent, err := os.ReadFile(testFilePath)
	if err != nil {
		fmt.Printf("❌ Read test failed: %v\n", err)
		s3fsManager.UnmountS3FS()
		return
	}

	if string(readContent) != testContent {
		fmt.Printf("❌ Read verification failed: expected '%s', got '%s'\n", testContent, string(readContent))
		s3fsManager.UnmountS3FS()
		return
	}
	fmt.Println("✅ Read test passed")

	// Delete test file
	if err := os.Remove(testFilePath); err != nil {
		fmt.Printf("❌ Cleanup failed: %v\n", err)
		s3fsManager.UnmountS3FS()
		return
	}
	fmt.Println("✅ Cleanup test passed")

	// Unmount
	fmt.Println("5. Unmounting test bucket...")
	if err := s3fsManager.UnmountS3FS(); err != nil {
		fmt.Printf("⚠️  Warning: Could not unmount bucket: %v\n", err)
		fmt.Println("   You may need to unmount manually with: fusermount -u " + bucket.MountPoint)
	} else {
		fmt.Println("✅ Bucket unmounted successfully")
	}

	// Clean up test credentials
	fmt.Println("6. Cleaning up test credentials...")
	if err := cleanupBucketCredentials(bucket); err != nil {
		fmt.Printf("⚠️  Warning: Could not clean up test credentials: %v\n", err)
	} else {
		fmt.Println("✅ Test credentials cleaned up")
	}

	fmt.Println("\n🎉 All tests passed! S3 bucket connectivity is working correctly.")
	fmt.Printf("📊 Summary: %s bucket '%s' is accessible and functional\n", bucket.Provider, bucket.Bucket)
}
