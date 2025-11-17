package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mitexleo/backtide/internal/config"
	"github.com/mitexleo/backtide/internal/systemd"
	"github.com/spf13/cobra"
)

var (
	systemdServiceName string
	systemdUser        string
	systemdSchedule    string
)

// systemdCmd represents the systemd command
var systemdCmd = &cobra.Command{
	Use:   "systemd",
	Short: "Manage automated backup scheduling",
	Long: `Manage automated backup scheduling with systemd.

This command automatically sets up and manages systemd services
for running backups on a daily schedule.`,
}

// systemdInstallCmd represents the systemd install command
var systemdInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Enable automated daily backups",
	Long: `Enable automated daily backups with systemd.

This command automatically sets up systemd to run backups daily.`,
	Run: runSystemdInstall,
}

// systemdUninstallCmd represents the systemd uninstall command
var systemdUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Disable automated backups",
	Long: `Disable automated backups and remove systemd scheduling.`,
	Run: runSystemdUninstall,
}

// systemdStatusCmd represents the systemd status command
var systemdStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check backup schedule status",
	Long:  `Check the status of automated backup scheduling.`,
	Run:   runSystemdStatus,
}

func init() {
	rootCmd.AddCommand(systemdCmd)
	systemdCmd.AddCommand(systemdInstallCmd)
	systemdCmd.AddCommand(systemdUninstallCmd)
	systemdCmd.AddCommand(systemdStatusCmd)
}

func runSystemdInstall(cmd *cobra.Command, args []string) {
	fmt.Println("Enabling automated daily backups...")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		fmt.Println("Try: sudo backtide systemd install")
		os.Exit(1)
	}

	// Create systemd service manager with default values
	manager := systemd.NewServiceManager("backtide", "", "", "root")

	// Create systemd service directory if it doesn't exist
	systemdDir := "/etc/systemd/system"
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		fmt.Printf("Error creating systemd directory: %v\n", err)
		os.Exit(1)
	}

	// Create service files with daily schedule
	if err := manager.UpdateServiceFiles("daily"); err != nil {
		fmt.Printf("Error setting up automated backups: %v\n", err)
		os.Exit(1)
	}

	// Enable and start timer
	if err := manager.EnableTimer(); err != nil {
		fmt.Printf("Error enabling automated backups: %v\n", err)
		os.Exit(1)
	}

	if err := manager.StartTimer(); err != nil {
		fmt.Printf("Error starting automated backups: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Automated backups enabled!\n")
	fmt.Println("Backups will run daily at a random time")
	fmt.Println("\nTo check status: backtide systemd status")
	fmt.Println("To view logs: journalctl -u backtide.service")
}

func runSystemdUninstall(cmd *cobra.Command, args []string) {
	fmt.Println("Uninstalling systemd service...")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		os.Exit(1)
	}

	// Create systemd service manager (binary path doesn't matter for uninstall)
	manager := systemd.NewServiceManager(systemdServiceName, "", "", "")

	// Stop and disable timer
	if err := manager.StopTimer(); err != nil {
		fmt.Printf("Warning: Failed to stop timer: %v\n", err)
	}

	if err := manager.DisableTimer(); err != nil {
		fmt.Printf("Warning: Failed to disable timer: %v\n", err)
	}

	// Remove service and timer files
	serviceFile := manager.GetServiceFilePath()
	timerFile := manager.GetTimerFilePath()

	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Error removing service file: %v\n", err)
	}

	if err := os.Remove(timerFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Error removing timer file: %v\n", err)
	}

	// Reload systemd
	if err := manager.ReloadDaemon(); err != nil {
		fmt.Printf("Error reloading systemd: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Systemd service uninstalled successfully!")
}

func runSystemdStatus(cmd *cobra.Command, args []string) {
	fmt.Println("Checking systemd service status...")

	// Check timer status
	cmdTimer := exec.Command("systemctl", "status", systemdServiceName+".timer")
	if output, err := cmdTimer.CombinedOutput(); err != nil {
		fmt.Printf("Timer status: %s\n", string(output))
	} else {
		fmt.Printf("Timer status: %s\n", string(output))
	}

	fmt.Println()

	// Check service status (last run)
	cmdService := exec.Command("systemctl", "status", systemdServiceName+".service")
	if output, err := cmdService.CombinedOutput(); err != nil {
		fmt.Printf("Service status: %s\n", string(output))
	} else {
		fmt.Printf("Service status: %s\n", string(output))
	}

	fmt.Println("\nRecent logs:")
	cmdLogs := exec.Command("journalctl", "-u", systemdServiceName+".service", "--since", "1 hour ago", "-n", "10")
	if output, err := cmdLogs.CombinedOutput(); err != nil {
		fmt.Printf("Error getting logs: %v\n", err)
	} else {
		fmt.Printf("%s\n", string(output))
	}
}

// generateServiceFile is kept for backward compatibility but now uses the systemd manager internally
func generateServiceFile(binaryPath, configPath, user string) string {
	manager := systemd.NewServiceManager("backtide", "", configPath, user)
	return manager.GenerateServiceFile()
}

// generateTimerFile is kept for backward compatibility but now uses the systemd manager internally
func generateTimerFile(serviceName, schedule string) string {
	manager := systemd.NewServiceManager(serviceName, "", "", "")
	return manager.GenerateTimerFile(schedule)
}
