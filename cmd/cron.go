package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mitexleo/backtide/internal/commands"
	"github.com/spf13/cobra"
)

var (
	cronUser     string
	cronSchedule string
	cronConfig   string
)

// cronCmd represents the cron command
var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage cron jobs for scheduled backups",
	Long: `Manage cron jobs for automated backup scheduling.

This command helps create and manage cron jobs for automated
backup scheduling as an alternative to systemd.`,
}

// cronInstallCmd represents the cron install command
var cronInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install cron job for automated backups",
	Long: `Install a cron job for automated backups.

This command will:
1. Get the absolute path to the backtide binary
2. Create a cron job entry
3. Install it in the user's crontab

The cron job will run the backup command according to the specified schedule.`,
	Run: runCronInstall,
}

// cronUninstallCmd represents the cron uninstall command
var cronUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall cron job",
	Long: `Uninstall the backtide cron job.

This command will remove any backtide-related entries from the user's crontab.`,
	Run: runCronUninstall,
}

// cronStatusCmd represents the cron status command
var cronStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cron job status",
	Long:  `Show the current status of backtide cron jobs.`,
	Run:   runCronStatus,
}

func init() {
	cronCmd.AddCommand(cronInstallCmd)
	cronCmd.AddCommand(cronUninstallCmd)
	cronCmd.AddCommand(cronStatusCmd)

	cronInstallCmd.Flags().StringVar(&cronUser, "user", "", "user to install cron job for (default: current user)")
	cronInstallCmd.Flags().StringVar(&cronSchedule, "schedule", "0 2 * * *", "cron schedule expression (default: daily at 2 AM)")
	cronInstallCmd.Flags().StringVar(&cronConfig, "config", "", "config file path (default: auto-detected)")

	// Register with command registry
	commands.RegisterCommand("cron", cronCmd)
}

func runCronInstall(cmd *cobra.Command, args []string) {
	fmt.Println("Installing cron job...")

	// Get binary path
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting binary path: %v\n", err)
		os.Exit(1)
	}

	// Get config path
	if cronConfig == "" {
		cronConfig = getConfigPath()
	}

	// Validate config exists
	if _, err := os.Stat(cronConfig); os.IsNotExist(err) {
		fmt.Printf("Error: Config file not found: %s\n", cronConfig)
		fmt.Println("Please create a configuration file first or specify with --config")
		os.Exit(1)
	}

	// Build the cron command
	cronCommand := fmt.Sprintf("%s backup --config %s", binaryPath, cronConfig)

	// Add log redirection for better logging
	cronCommand += " >> /var/log/backtide.log 2>&1"

	// Create cron entry
	cronEntry := fmt.Sprintf("%s %s\n", cronSchedule, cronCommand)

	// Determine which user's crontab to modify
	if cronUser == "" {
		cronUser = os.Getenv("USER")
		if cronUser == "" {
			cronUser = os.Getenv("LOGNAME")
		}
	}

	fmt.Printf("Installing cron job for user: %s\n", cronUser)
	fmt.Printf("Schedule: %s\n", cronSchedule)
	fmt.Printf("Command: %s\n", cronCommand)

	if dryRun {
		fmt.Println("DRY RUN: Would add the following cron entry:")
		fmt.Println(cronEntry)
		return
	}

	// Get current crontab
	var currentCrontab string
	if cronUser == "root" || os.Geteuid() == 0 {
		// For root, we can use crontab -l directly
		cmd := exec.Command("crontab", "-l")
		output, err := cmd.Output()
		if err != nil && err.Error() != "exit status 1" {
			// exit status 1 means no crontab, which is fine
			fmt.Printf("Error reading current crontab: %v\n", err)
			os.Exit(1)
		}
		currentCrontab = string(output)
	} else {
		// For non-root users, we need to use sudo if installing for different user
		if cronUser != os.Getenv("USER") {
			fmt.Printf("Error: Cannot install cron job for user '%s' without root privileges\n", cronUser)
			os.Exit(1)
		}
		cmd := exec.Command("crontab", "-l")
		output, err := cmd.Output()
		if err != nil && err.Error() != "exit status 1" {
			fmt.Printf("Error reading current crontab: %v\n", err)
			os.Exit(1)
		}
		currentCrontab = string(output)
	}

	// Remove any existing backtide entries
	lines := strings.Split(currentCrontab, "\n")
	var newCrontabLines []string
	for _, line := range lines {
		if !strings.Contains(line, "backtide") && strings.TrimSpace(line) != "" {
			newCrontabLines = append(newCrontabLines, line)
		}
	}

	// Add the new entry
	newCrontabLines = append(newCrontabLines, cronEntry)
	newCrontab := strings.Join(newCrontabLines, "\n") + "\n"

	// Install new crontab
	if cronUser == "root" || os.Geteuid() == 0 {
		// For root, we can write directly
		cmd := exec.Command("crontab", "-")
		cmd.Stdin = strings.NewReader(newCrontab)
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Error installing crontab: %v\n", string(output))
			os.Exit(1)
		}
	} else {
		// For current user
		cmd := exec.Command("crontab", "-")
		cmd.Stdin = strings.NewReader(newCrontab)
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Error installing crontab: %v\n", string(output))
			os.Exit(1)
		}
	}

	// Create log directory if it doesn't exist
	logDir := "/var/log"
	if err := os.MkdirAll(logDir, 0755); err != nil && !os.IsExist(err) {
		fmt.Printf("Warning: Could not create log directory: %v\n", err)
	}

	fmt.Println("Cron job installed successfully!")
	fmt.Printf("Logs will be written to: %s\n", "/var/log/backtide.log")
	fmt.Println("To verify: crontab -l")
}

func runCronUninstall(cmd *cobra.Command, args []string) {
	fmt.Println("Uninstalling cron job...")

	// Determine which user's crontab to modify
	if cronUser == "" {
		cronUser = os.Getenv("USER")
		if cronUser == "" {
			cronUser = os.Getenv("LOGNAME")
		}
	}

	fmt.Printf("Removing backtide cron jobs for user: %s\n", cronUser)

	if dryRun {
		fmt.Println("DRY RUN: Would remove all backtide entries from crontab")
		return
	}

	// Get current crontab
	var currentCrontab string
	if cronUser == "root" || os.Geteuid() == 0 {
		cmd := exec.Command("crontab", "-l")
		output, err := cmd.Output()
		if err != nil && err.Error() != "exit status 1" {
			fmt.Printf("Error reading current crontab: %v\n", err)
			os.Exit(1)
		}
		currentCrontab = string(output)
	} else {
		if cronUser != os.Getenv("USER") {
			fmt.Printf("Error: Cannot modify cron job for user '%s' without root privileges\n", cronUser)
			os.Exit(1)
		}
		cmd := exec.Command("crontab", "-l")
		output, err := cmd.Output()
		if err != nil && err.Error() != "exit status 1" {
			fmt.Printf("Error reading current crontab: %v\n", err)
			os.Exit(1)
		}
		currentCrontab = string(output)
	}

	// Remove backtide entries
	lines := strings.Split(currentCrontab, "\n")
	var newCrontabLines []string
	removedCount := 0
	for _, line := range lines {
		if strings.Contains(line, "backtide") {
			removedCount++
			continue
		}
		if strings.TrimSpace(line) != "" {
			newCrontabLines = append(newCrontabLines, line)
		}
	}

	newCrontab := strings.Join(newCrontabLines, "\n")
	if newCrontab != "" {
		newCrontab += "\n"
	}

	// Install updated crontab
	if cronUser == "root" || os.Geteuid() == 0 {
		cmd := exec.Command("crontab", "-")
		cmd.Stdin = strings.NewReader(newCrontab)
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Error updating crontab: %v\n", string(output))
			os.Exit(1)
		}
	} else {
		cmd := exec.Command("crontab", "-")
		cmd.Stdin = strings.NewReader(newCrontab)
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Error updating crontab: %v\n", string(output))
			os.Exit(1)
		}
	}

	fmt.Printf("Cron job uninstalled successfully! Removed %d entries\n", removedCount)
}

func runCronStatus(cmd *cobra.Command, args []string) {
	fmt.Println("Checking cron job status...")

	// Determine which user's crontab to check
	if cronUser == "" {
		cronUser = os.Getenv("USER")
		if cronUser == "" {
			cronUser = os.Getenv("LOGNAME")
		}
	}

	fmt.Printf("Cron jobs for user: %s\n", cronUser)

	// Get current crontab
	var cmdOutput []byte
	var err error
	if cronUser == "root" || os.Geteuid() == 0 {
		cmd := exec.Command("crontab", "-l")
		cmdOutput, err = cmd.Output()
	} else {
		if cronUser != os.Getenv("USER") {
			fmt.Printf("Error: Cannot read cron jobs for user '%s' without root privileges\n", cronUser)
			os.Exit(1)
		}
		cmd := exec.Command("crontab", "-l")
		cmdOutput, err = cmd.Output()
	}

	if err != nil {
		if err.Error() == "exit status 1" {
			fmt.Println("No crontab found for this user")
			return
		}
		fmt.Printf("Error reading crontab: %v\n", err)
		os.Exit(1)
	}

	currentCrontab := string(cmdOutput)
	lines := strings.Split(currentCrontab, "\n")

	// Find backtide entries
	var backtideEntries []string
	for _, line := range lines {
		if strings.Contains(line, "backtide") {
			backtideEntries = append(backtideEntries, line)
		}
	}

	if len(backtideEntries) == 0 {
		fmt.Println("No backtide cron jobs found")
		return
	}

	fmt.Printf("Found %d backtide cron job(s):\n", len(backtideEntries))
	for i, entry := range backtideEntries {
		fmt.Printf("  %d. %s\n", i+1, strings.TrimSpace(entry))
	}

	// Check if cron service is running
	fmt.Println("\nCron service status:")
	if output, err := exec.Command("systemctl", "is-active", "cron").Output(); err == nil {
		fmt.Printf("  cron service: %s", string(output))
	} else if output, err := exec.Command("systemctl", "is-active", "crond").Output(); err == nil {
		fmt.Printf("  crond service: %s", string(output))
	} else {
		fmt.Println("  cron service: unknown (neither cron nor crond service found)")
	}
}
