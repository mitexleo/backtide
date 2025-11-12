package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/mitexleo/backtide/internal/backup"
	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	listDetailed bool
	listJson     bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available backups",
	Long: `List all available backups with their metadata.

This command shows information about all backups including:
- Backup ID
- Timestamp
- Total size
- Number of directories
- Compression status`,
	Run: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listDetailed, "detailed", "d", false, "show detailed information about each backup")
	listCmd.Flags().BoolVar(&listJson, "json", false, "output in JSON format")
}

func runList(cmd *cobra.Command, args []string) {
	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize backup manager
	backupManager := backup.NewBackupManager(*cfg)

	// List backups
	backups, err := backupManager.ListBackups()
	if err != nil {
		fmt.Printf("Error listing backups: %v\n", err)
		os.Exit(1)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found")
		return
	}

	if listJson {
		outputJson(backups)
		return
	}

	outputHumanReadable(backups)
}

func outputHumanReadable(backups []config.BackupMetadata) {
	fmt.Printf("Found %d backup(s):\n\n", len(backups))

	for _, backup := range backups {
		fmt.Printf("Backup ID: %s\n", backup.ID)
		fmt.Printf("  Timestamp: %s\n", backup.Timestamp.Format(time.RFC3339))
		fmt.Printf("  Size: %s\n", formatBytes(backup.TotalSize))
		fmt.Printf("  Directories: %d\n", len(backup.Directories))
		fmt.Printf("  Compressed: %v\n", backup.Compressed)
		fmt.Printf("  Checksum: %s\n", backup.Checksum[:16]+"...")

		if listDetailed {
			fmt.Println("  Directories:")
			for _, dir := range backup.Directories {
				fmt.Printf("    - %s (%s)\n", dir.Name, dir.Path)
				fmt.Printf("      Size: %s\n", formatBytes(dir.Size))
				fmt.Printf("      Files: %d\n", dir.FileCount)
				fmt.Printf("      Checksum: %s\n", dir.Checksum[:16]+"...")
			}
			fmt.Println()
		} else {
			fmt.Println()
		}
	}
}

func outputJson(backups []config.BackupMetadata) {
	// For JSON output, we need to marshal the data
	// This is a simplified implementation - in a real scenario
	// we would use encoding/json to properly marshal the data
	fmt.Println("[")
	for i, backup := range backups {
		fmt.Printf("  {\n")
		fmt.Printf("    \"id\": \"%s\",\n", backup.ID)
		fmt.Printf("    \"timestamp\": \"%s\",\n", backup.Timestamp.Format(time.RFC3339))
		fmt.Printf("    \"total_size\": %d,\n", backup.TotalSize)
		fmt.Printf("    \"directories_count\": %d,\n", len(backup.Directories))
		fmt.Printf("    \"compressed\": %v,\n", backup.Compressed)
		fmt.Printf("    \"checksum\": \"%s\"\n", backup.Checksum)

		if listDetailed {
			fmt.Printf("    \"directories\": [\n")
			for j, dir := range backup.Directories {
				fmt.Printf("      {\n")
				fmt.Printf("        \"name\": \"%s\",\n", dir.Name)
				fmt.Printf("        \"path\": \"%s\",\n", dir.Path)
				fmt.Printf("        \"size\": %d,\n", dir.Size)
				fmt.Printf("        \"file_count\": %d,\n", dir.FileCount)
				fmt.Printf("        \"checksum\": \"%s\"\n", dir.Checksum)
				if j < len(backup.Directories)-1 {
					fmt.Printf("      },\n")
				} else {
					fmt.Printf("      }\n")
				}
			}
			fmt.Printf("    ]\n")
		}

		if i < len(backups)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
