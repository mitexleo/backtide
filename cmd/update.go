package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Backtide to the latest version",
	Long: `Update Backtide to the latest version automatically.

This command will:
1. Check for the latest release on GitHub
2. Download the appropriate binary for your platform
3. Replace the current binary with the updated version
4. Preserve your configuration and data

Examples:
  backtide update        # Update to latest version
  backtide update --dry-run  # Show what would be updated without making changes`,
	Run: runUpdate,
}

var (
	updateDryRun bool
	updateForce  bool
)

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "show what would be updated without making changes")
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "force update even if already on latest version")
}

func runUpdate(cmd *cobra.Command, args []string) {
	fmt.Println("🔍 Checking for updates...")

	// Get current version
	currentVersion := version
	if currentVersion == "dev" {
		fmt.Println("⚠️  You're running a development build. Update command may not work correctly.")
		if !updateForce {
			fmt.Println("Use --force to update anyway.")
			return
		}
	}

	// Get latest release info
	latestRelease, err := getLatestRelease()
	if err != nil {
		// Check if error is due to no releases available
		if strings.Contains(err.Error(), "could not find download URL") ||
			strings.Contains(err.Error(), "could not find version") ||
			strings.Contains(err.Error(), "no releases available") {
			fmt.Println("❌ No releases available for update.")
			fmt.Println("   Visit https://github.com/mitexleo/backtide/releases")
			fmt.Println("   Or build from source: git clone https://github.com/mitexleo/backtide")
		} else {
			fmt.Printf("❌ Failed to check for updates: %v\n", err)
		}
		return
	}

	fmt.Printf("📦 Current version: %s\n", currentVersion)
	fmt.Printf("🚀 Latest version: %s\n", latestRelease.Version)

	if currentVersion == latestRelease.Version && !updateForce {
		fmt.Println("✅ You're already on the latest version!")
		return
	}

	if updateDryRun {
		fmt.Printf("📋 Dry run: Would update from %s to %s\n", currentVersion, latestRelease.Version)
		fmt.Printf("📋 Would download: %s\n", latestRelease.DownloadURL)
		return
	}

	fmt.Printf("⬇️  Downloading Backtide %s...\n", latestRelease.Version)

	// Download the new binary
	tempFile, err := downloadBinary(latestRelease.DownloadURL)
	if err != nil {
		fmt.Printf("❌ Download failed: %v\n", err)
		return
	}
	defer os.Remove(tempFile)

	// Verify the downloaded binary works
	if err := verifyBinary(tempFile, latestRelease.Version); err != nil {
		fmt.Printf("❌ Downloaded binary verification failed: %v\n", err)
		return
	}

	// Get current executable path
	currentExec, err := os.Executable()
	if err != nil {
		fmt.Printf("❌ Could not determine current executable path: %v\n", err)
		return
	}

	// Replace the current binary
	if err := replaceBinary(currentExec, tempFile); err != nil {
		fmt.Printf("❌ Update failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Successfully updated Backtide from %s to %s!\n", currentVersion, latestRelease.Version)
	fmt.Println("💡 Restart any running Backtide processes to use the new version.")
}

// ReleaseInfo holds information about a GitHub release
type ReleaseInfo struct {
	Version      string
	DownloadURL  string
	ReleaseNotes string
}

// GitHubRelease represents the GitHub API release response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// getLatestRelease fetches the latest release information from GitHub
func getLatestRelease() (*ReleaseInfo, error) {
	// GitHub API URL for latest release
	apiURL := "https://api.github.com/repos/mitexleo/backtide/releases/latest"

	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases available")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the JSON response using proper JSON parsing
	version, downloadURL, err := parseReleaseJSON(body)
	if err != nil {
		return nil, err
	}

	return &ReleaseInfo{
		Version:     version,
		DownloadURL: downloadURL,
	}, nil
}

// parseReleaseJSON extracts version and download URL from GitHub API response
func parseReleaseJSON(data []byte) (string, string, error) {
	var release GitHubRelease
	if err := json.Unmarshal(data, &release); err != nil {
		return "", "", fmt.Errorf("failed to parse GitHub API response: %v", err)
	}

	if release.TagName == "" {
		return "", "", fmt.Errorf("no releases available")
	}

	// Remove 'v' prefix from version
	version := strings.TrimPrefix(release.TagName, "v")

	// Determine correct binary name for current platform
	binaryName := getBinaryNameForPlatform()

	// Find download URL for the correct binary
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	// Fallback to main binary if platform-specific not found
	if downloadURL == "" {
		for _, asset := range release.Assets {
			if asset.Name == "backtide" {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
	}

	if downloadURL == "" {
		return "", "", fmt.Errorf("no releases available")
	}

	return version, downloadURL, nil
}

// getBinaryNameForPlatform returns the appropriate binary name for the current platform
func getBinaryNameForPlatform() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	switch os {
	case "linux":
		if arch == "amd64" {
			return "backtide-linux-amd64"
		}
		return "backtide"
	case "darwin":
		return "backtide-darwin-amd64"
	case "windows":
		return "backtide-windows-amd64.exe"
	default:
		return "backtide"
	}
}

// downloadBinary downloads the binary to a temporary file
func downloadBinary(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "backtide-update-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Download to temporary file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	// Make executable
	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// verifyBinary checks if the downloaded binary works correctly
func verifyBinary(filePath, expectedVersion string) error {
	// Try to run the binary and check its version
	cmd := execCommand(filePath, "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("downloaded binary is not executable: %v", err)
	}

	// Check if version matches expected
	if !strings.Contains(string(output), expectedVersion) {
		return fmt.Errorf("version mismatch: expected %s, got %s", expectedVersion, string(output))
	}

	return nil
}

// replaceBinary replaces the current binary with the new one
func replaceBinary(currentPath, newPath string) error {
	// Get directory of current binary (keep for future use if needed)
	_ = filepath.Dir(currentPath)

	// Create backup of current binary
	backupPath := currentPath + ".backup"
	if err := copyFile(currentPath, backupPath); err != nil {
		return fmt.Errorf("could not create backup: %v", err)
	}

	// Replace the binary
	if err := copyFile(newPath, currentPath); err != nil {
		// Restore from backup if replacement fails
		copyFile(backupPath, currentPath)
		os.Remove(backupPath)
		return fmt.Errorf("could not replace binary: %v", err)
	}

	// Clean up backup
	os.Remove(backupPath)
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	// Preserve executable permissions
	if err := os.Chmod(dst, 0755); err != nil {
		return err
	}

	return nil
}

// execCommand is a wrapper for exec.Command for testing
var execCommand = func(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}
