package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mitexleo/backtide/internal/config"
	"github.com/spf13/cobra"
)

var (
	systemdJobsServiceName string
	systemdJobsUser        string
	systemdJobsBasePath    string
)

// systemdJobsCmd represents the systemd-jobs command
var systemdJobsCmd = &cobra.Command{
	Use:   "systemd-jobs",
	Short: "Manage systemd service for all backup jobs",
	Long: `Manage systemd service and timer for all backup jobs.

This command generates a single systemd service that runs all enabled
backup jobs according to their individual schedules.`,
}

// systemdJobsInstallCmd represents the systemd-jobs install command
var systemdJobsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install systemd service for all backup jobs",
	Long: `Install systemd service and timer for all backup jobs.

This command will:
1. Generate a single systemd service file that reads the config file directly
2. Generate systemd timer for scheduled execution
3. Enable and start the timer
4. Reload systemd daemon`,
	Run: runSystemdJobsInstall,
}

// systemdJobsUninstallCmd represents the systemd-jobs uninstall command
var systemdJobsUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall systemd service for backup jobs",
	Long: `Uninstall systemd service and timer for backup jobs.

This command will:
1. Stop and disable the backtide timer
2. Remove systemd service and timer files
3. Reload systemd daemon`,
	Run: runSystemdJobsUninstall,
}

// systemdJobsStatusCmd represents the systemd-jobs status command
var systemdJobsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show systemd service status for backup jobs",
	Long:  `Show the current status of the backtide systemd service and timer.`,
	Run:   runSystemdJobsStatus,
}

// systemdJobsRestartCmd represents the systemd-jobs restart command
var systemdJobsRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart systemd service for backup jobs",
	Long: `Restart the backtide systemd service and timer.

This command will:
1. Stop the backtide timer
2. Restart the backtide service
3. Start the backtide timer
4. Show the new status`,
	Run: runSystemdJobsRestart,
}

func init() {
	systemdJobsCmd.AddCommand(systemdJobsInstallCmd)
	systemdJobsCmd.AddCommand(systemdJobsUninstallCmd)
	systemdJobsCmd.AddCommand(systemdJobsStatusCmd)
	systemdJobsCmd.AddCommand(systemdJobsRestartCmd)

	systemdJobsInstallCmd.Flags().StringVar(&systemdJobsServiceName, "service-name", "backtide", "base name for systemd services")
	systemdJobsInstallCmd.Flags().StringVar(&systemdJobsUser, "user", "root", "user to run the services as")
	systemdJobsInstallCmd.Flags().StringVar(&systemdJobsBasePath, "base-path", "/etc/backtide", "base path for job configurations")
}

func runSystemdJobsInstall(cmd *cobra.Command, args []string) {
	fmt.Println("Installing systemd service for backup jobs...")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		os.Exit(1)
	}

	// Load configuration
	configPath := getConfigPath()
	_, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Create systemd service directory if it doesn't exist
	systemdDir := "/etc/systemd/system"
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		fmt.Printf("Error creating systemd directory: %v\n", err)
		os.Exit(1)
	}

	// Create base path for systemd service files
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		fmt.Printf("Error creating systemd directory: %v\n", err)
		os.Exit(1)
	}

	// Get absolute path to backtide binary
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting binary path: %v\n", err)
		os.Exit(1)
	}

	// Create single service file for all jobs
	serviceName := systemdJobsServiceName
	serviceFile := filepath.Join(systemdDir, serviceName+".service")
	serviceContent := generateJobServiceFile(binaryPath, configPath, systemdJobsUser)
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		fmt.Printf("Error creating service file: %v\n", err)
		os.Exit(1)
	}

	// Create timer file for scheduled execution
	timerFile := filepath.Join(systemdDir, serviceName+".timer")
	timerContent := generateJobTimerFile(serviceName, "daily")
	if err := os.WriteFile(timerFile, []byte(timerContent), 0644); err != nil {
		fmt.Printf("Error creating timer file: %v\n", err)
		os.Exit(1)
	}

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		fmt.Printf("Error reloading systemd: %v\n", err)
		os.Exit(1)
	}

	// Enable and start timer
	if err := exec.Command("systemctl", "enable", serviceName+".timer").Run(); err != nil {
		fmt.Printf("Error enabling timer: %v\n", err)
		os.Exit(1)
	}

	if err := exec.Command("systemctl", "start", serviceName+".timer").Run(); err != nil {
		fmt.Printf("Error starting timer: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nSystemd service installed successfully!\n")
	fmt.Printf("Service: %s.service\n", serviceName)
	fmt.Printf("Timer: %s.timer\n", serviceName)
	fmt.Printf("Config: %s\n", configPath)
	fmt.Println("\nTo check status: systemctl status backtide.timer")
	fmt.Println("To view logs: journalctl -u backtide.service")
}

func runSystemdJobsUninstall(cmd *cobra.Command, args []string) {
	fmt.Println("Uninstalling systemd service for backup jobs...")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		os.Exit(1)
	}

	// Stop and disable timer
	serviceName := systemdJobsServiceName
	if err := exec.Command("systemctl", "stop", serviceName+".timer").Run(); err != nil {
		fmt.Printf("Warning: Failed to stop timer: %v\n", err)
	}

	if err := exec.Command("systemctl", "disable", serviceName+".timer").Run(); err != nil {
		fmt.Printf("Warning: Failed to disable timer: %v\n", err)
	}

	// Remove service and timer files
	systemdDir := "/etc/systemd/system"
	serviceFile := filepath.Join(systemdDir, serviceName+".service")
	timerFile := filepath.Join(systemdDir, serviceName+".timer")

	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Error removing service file: %v\n", err)
	}

	if err := os.Remove(timerFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Error removing timer file: %v\n", err)
	}

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		fmt.Printf("Error reloading systemd: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Systemd service uninstalled successfully!")
}

func runSystemdJobsRestart(cmd *cobra.Command, args []string) {
	fmt.Println("Restarting systemd service for backup jobs...")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		os.Exit(1)
	}

	serviceName := systemdJobsServiceName

	// Stop timer
	fmt.Println("Stopping timer...")
	if err := exec.Command("systemctl", "stop", serviceName+".timer").Run(); err != nil {
		fmt.Printf("Warning: Failed to stop timer: %v\n", err)
	}

	// Restart service
	fmt.Println("Restarting service...")
	if err := exec.Command("systemctl", "restart", serviceName+".service").Run(); err != nil {
		fmt.Printf("Error restarting service: %v\n", err)
		os.Exit(1)
	}

	// Start timer
	fmt.Println("Starting timer...")
	if err := exec.Command("systemctl", "start", serviceName+".timer").Run(); err != nil {
		fmt.Printf("Error starting timer: %v\n", err)
		os.Exit(1)
	}

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		fmt.Printf("Error reloading systemd: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Systemd service restarted successfully!")
	fmt.Println("\nNew status:")
	runSystemdJobsStatus(cmd, args)
}

func runSystemdJobsStatus(cmd *cobra.Command, args []string) {
	fmt.Println("Checking systemd service status for backup jobs...")

	serviceName := systemdJobsServiceName

	// Check timer status
	cmdTimer := exec.Command("systemctl", "status", serviceName+".timer")
	if output, err := cmdTimer.CombinedOutput(); err != nil {
		fmt.Printf("Timer status: %s\n", string(output))
	} else {
		fmt.Printf("Timer status: %s\n", string(output))
	}

	fmt.Println()

	// Check service status (last run)
	cmdService := exec.Command("systemctl", "status", serviceName+".service")
	if output, err := cmdService.CombinedOutput(); err != nil {
		fmt.Printf("Service status: %s\n", string(output))
	} else {
		fmt.Printf("Service status: %s\n", string(output))
	}

	fmt.Println("\nRecent logs:")
	cmdLogs := exec.Command("journalctl", "-u", serviceName+".service", "--since", "1 hour ago", "-n", "10")
	if output, err := cmdLogs.CombinedOutput(); err != nil {
		fmt.Printf("Error getting logs: %v\n", err)
	} else {
		fmt.Printf("%s\n", string(output))
	}
}

func generateJobServiceFile(binaryPath, configPath, user string) string {
	tmpl := `[Unit]
Description=Backtide Backup Service - All Jobs
Documentation=https://github.com/mitexleo/backtide
After=network.target docker.service
Requires=docker.service

[Service]
Type=oneshot
User={{.User}}
ExecStart={{.BinaryPath}} backup --config {{.ConfigPath}} --all
StandardOutput=journal
StandardError=journal
TimeoutStopSec=300
Restart=no

[Install]
WantedBy=multi-user.target
`

	data := struct {
		BinaryPath string
		ConfigPath string
		User       string
	}{
		BinaryPath: binaryPath,
		ConfigPath: configPath,
		User:       user,
	}

	var buf strings.Builder
	t := template.Must(template.New("service").Parse(tmpl))
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}

	return buf.String()
}

func generateJobTimerFile(serviceName, schedule string) string {
	tmpl := `[Unit]
Description=Backtide Backup Timer - All Jobs
Documentation=https://github.com/mitexleo/backtide
Requires={{.ServiceName}}.service

[Timer]
OnCalendar={{.Schedule}}
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
`

	data := struct {
		ServiceName string
		Schedule    string
	}{
		ServiceName: serviceName,
		Schedule:    schedule,
	}

	var buf strings.Builder
	t := template.Must(template.New("timer").Parse(tmpl))
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}

	return buf.String()
}
