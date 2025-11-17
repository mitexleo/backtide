package cmd

import (
	"fmt"
	"os"

	"github.com/mitexleo/backtide/internal/commands"
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
	Short: "Manage the Backtide scheduling daemon",
	Long: `Manage the Backtide scheduling daemon with systemd.

This command sets up and manages the Backtide daemon that runs
continuously and handles ALL backup job scheduling internally.`,
}

// systemdInstallCmd represents the systemd install command
var systemdInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Enable the scheduling daemon",
	Long: `Enable the Backtide scheduling daemon.

This command sets up the Backtide daemon that runs continuously
and manages ALL backup job schedules internally.`,
	Run: runSystemdInstall,
}

// systemdUninstallCmd represents the systemd uninstall command
var systemdUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Disable the scheduling daemon",
	Long:  `Disable the Backtide scheduling daemon and remove systemd service.`,
	Run:   runSystemdUninstall,
}

// systemdStatusCmd represents the systemd status command
var systemdStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Long:  `Check the status of the Backtide scheduling daemon.`,
	Run:   runSystemdStatus,
}

func init() {
	systemdCmd.AddCommand(systemdInstallCmd)
	systemdCmd.AddCommand(systemdUninstallCmd)
	systemdCmd.AddCommand(systemdStatusCmd)

	// Register with command registry
	commands.RegisterCommand("systemd", systemdCmd)
}

func runSystemdInstall(cmd *cobra.Command, args []string) {
	fmt.Println("Enabling Backtide scheduling daemon...")

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

	// Create service file for continuous daemon
	if err := manager.UpdateServiceFile(); err != nil {
		fmt.Printf("Error setting up scheduling daemon: %v\n", err)
		os.Exit(1)
	}

	// Enable and start service
	if err := manager.EnableService(); err != nil {
		fmt.Printf("Error enabling scheduling daemon: %v\n", err)
		os.Exit(1)
	}

	if err := manager.StartService(); err != nil {
		fmt.Printf("Error starting scheduling daemon: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Scheduling daemon enabled!\n")
	fmt.Println("Daemon will manage ALL backup job schedules internally")
	fmt.Println("\nTo check status: backtide systemd status")
	fmt.Println("To view logs: journalctl -u backtide.service -f")
}

func runSystemdUninstall(cmd *cobra.Command, args []string) {
	fmt.Println("Disabling scheduling daemon...")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		fmt.Println("Try: sudo backtide systemd uninstall")
		os.Exit(1)
	}

	// Create systemd service manager
	manager := systemd.NewServiceManager("backtide", "", "", "")

	// Stop and disable service
	if err := manager.StopService(); err != nil {
		fmt.Printf("Warning: Failed to stop scheduling daemon: %v\n", err)
	}

	if err := manager.DisableService(); err != nil {
		fmt.Printf("Warning: Failed to disable scheduling daemon: %v\n", err)
	}

	// Remove service file
	serviceFile := manager.GetServiceFilePath()

	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Error removing service file: %v\n", err)
	}

	// Reload systemd
	if err := manager.ReloadDaemon(); err != nil {
		fmt.Printf("Error reloading systemd: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Scheduling daemon disabled!")
}

func runSystemdStatus(cmd *cobra.Command, args []string) {
	fmt.Println("Checking daemon status...")

	// Create systemd service manager
	manager := systemd.NewServiceManager("backtide", "", "", "")

	// Get service status
	status, err := manager.GetServiceStatus()
	if err != nil {
		fmt.Printf("Error checking daemon status: %v\n", err)
		return
	}

	// Load configuration to get job count
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	var jobCount int
	if err != nil {
		jobCount = 0
	} else {
		jobCount = len(cfg.Jobs)
	}

	if status.IsEnabled {
		fmt.Printf("✅ Scheduling daemon: ENABLED\n")
		if status.IsRunning {
			fmt.Printf("🟢 Status: ACTIVE (running)\n")
		} else if status.IsActive {
			fmt.Printf("🟡 Status: ACTIVE (not running)\n")
		} else {
			fmt.Printf("🔴 Status: INACTIVE\n")
		}
		fmt.Printf("📅 Internal scheduling: ALL backup jobs\n")
		fmt.Printf("📊 Active jobs: %d\n", jobCount)
	} else {
		fmt.Printf("❌ Scheduling daemon: DISABLED\n")
		fmt.Printf("💡 Run 'sudo backtide systemd install' to enable\n")
	}
}

// generateServiceFile is kept for backward compatibility but now uses the systemd manager internally
func generateServiceFile(binaryPath, configPath, user string) string {
	manager := systemd.NewServiceManager("backtide", "", configPath, user)
	return manager.GenerateServiceFile()
}
