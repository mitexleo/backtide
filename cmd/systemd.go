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
	Short: "Manage systemd service for scheduled backups",
	Long: `Manage systemd service and timer for scheduled backups.

This command helps create and manage systemd services and timers
for automated backup scheduling.`,
}

// systemdInstallCmd represents the systemd install command
var systemdInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install systemd service and timer",
	Long: `Install systemd service and timer for automated backups.

This command will:
1. Create systemd service file
2. Create systemd timer file
3. Enable and start the timer
4. Reload systemd daemon`,
	Run: runSystemdInstall,
}

// systemdUninstallCmd represents the systemd uninstall command
var systemdUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall systemd service and timer",
	Long: `Uninstall systemd service and timer.

This command will:
1. Stop and disable the timer
2. Remove systemd service and timer files
3. Reload systemd daemon`,
	Run: runSystemdUninstall,
}

// systemdStatusCmd represents the systemd status command
var systemdStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show systemd service status",
	Long:  `Show the current status of the backtide systemd service and timer.`,
	Run:   runSystemdStatus,
}

func init() {
	systemdCmd.AddCommand(systemdInstallCmd)
	systemdCmd.AddCommand(systemdUninstallCmd)
	systemdCmd.AddCommand(systemdStatusCmd)

	systemdInstallCmd.Flags().StringVar(&systemdServiceName, "service-name", "backtide", "systemd service name")
	systemdInstallCmd.Flags().StringVar(&systemdUser, "user", "root", "user to run the service as")
	systemdInstallCmd.Flags().StringVar(&systemdSchedule, "schedule", "daily", "backup schedule (daily, weekly, monthly, or cron expression)")
}

func runSystemdInstall(cmd *cobra.Command, args []string) {
	fmt.Println("Installing systemd service...")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		os.Exit(1)
	}

	// Load configuration to get config path
	configPath := getConfigPath()
	_, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Get absolute path to backtide binary
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting binary path: %v\n", err)
		os.Exit(1)
	}

	// Create systemd service manager
	manager := systemd.NewServiceManager(systemdServiceName, binaryPath, configPath, systemdUser)

	// Create systemd service directory if it doesn't exist
	systemdDir := "/etc/systemd/system"
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		fmt.Printf("Error creating systemd directory: %v\n", err)
		os.Exit(1)
	}

	// Update service files with current binary path
	if err := manager.UpdateServiceFiles(systemdSchedule); err != nil {
		fmt.Printf("Error creating systemd service files: %v\n", err)
		os.Exit(1)
	}

	// Enable and start timer
	if err := manager.EnableTimer(); err != nil {
		fmt.Printf("Error enabling timer: %v\n", err)
		os.Exit(1)
	}

	if err := manager.StartTimer(); err != nil {
		fmt.Printf("Error starting timer: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Systemd service installed successfully!\n")
	fmt.Printf("Service: %s.service\n", systemdServiceName)
	fmt.Printf("Timer: %s.timer\n", systemdServiceName)
	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("Schedule: %s\n", systemdSchedule)
	fmt.Println("\nTo check status: systemctl status backtide.timer")
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
	manager := systemd.NewServiceManager("backtide", binaryPath, configPath, user)
	return manager.GenerateServiceFile()
}

// generateTimerFile is kept for backward compatibility but now uses the systemd manager internally
func generateTimerFile(serviceName, schedule string) string {
	manager := systemd.NewServiceManager(serviceName, "", "", "")
	return manager.GenerateTimerFile(schedule)
}
