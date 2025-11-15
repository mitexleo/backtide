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
	Short: "Manage systemd services for multiple backup jobs",
	Long: `Manage systemd services and timers for multiple backup jobs.

This command generates systemd services and timers for all enabled
backup jobs with scheduled backups, allowing each job to run on its
own schedule.`,
}

// systemdJobsInstallCmd represents the systemd-jobs install command
var systemdJobsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install systemd services for all scheduled backup jobs",
	Long: `Install systemd services and timers for all enabled backup jobs.

This command will:
1. Generate systemd service files for each enabled backup job
2. Generate systemd timer files for jobs with schedules
3. Enable and start the timers
4. Reload systemd daemon`,
	Run: runSystemdJobsInstall,
}

// systemdJobsUninstallCmd represents the systemd-jobs uninstall command
var systemdJobsUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall systemd services for backup jobs",
	Long: `Uninstall systemd services and timers for backup jobs.

This command will:
1. Stop and disable all backtide job timers
2. Remove all systemd service and timer files
3. Reload systemd daemon`,
	Run: runSystemdJobsUninstall,
}

// systemdJobsStatusCmd represents the systemd-jobs status command
var systemdJobsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show systemd services status for backup jobs",
	Long:  `Show the current status of all backtide systemd services and timers.`,
	Run:   runSystemdJobsStatus,
}

func init() {
	rootCmd.AddCommand(systemdJobsCmd)
	systemdJobsCmd.AddCommand(systemdJobsInstallCmd)
	systemdJobsCmd.AddCommand(systemdJobsUninstallCmd)
	systemdJobsCmd.AddCommand(systemdJobsStatusCmd)

	systemdJobsInstallCmd.Flags().StringVar(&systemdJobsServiceName, "service-name", "backtide", "base name for systemd services")
	systemdJobsInstallCmd.Flags().StringVar(&systemdJobsUser, "user", "root", "user to run the services as")
	systemdJobsInstallCmd.Flags().StringVar(&systemdJobsBasePath, "base-path", "/etc/backtide", "base path for job configurations")
}

func runSystemdJobsInstall(cmd *cobra.Command, args []string) {
	fmt.Println("Installing systemd services for backup jobs...")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		os.Exit(1)
	}

	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
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

	// Create base path for job configurations
	if err := os.MkdirAll(systemdJobsBasePath, 0755); err != nil {
		fmt.Printf("Error creating base path: %v\n", err)
		os.Exit(1)
	}

	// Get absolute path to backtide binary
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting binary path: %v\n", err)
		os.Exit(1)
	}

	// Generate services for all enabled jobs
	installedCount := 0
	scheduledCount := 0

	for _, job := range cfg.Jobs {
		if !job.Enabled {
			continue
		}

		// Create job-specific config file
		jobConfigPath := filepath.Join(systemdJobsBasePath, fmt.Sprintf("%s.yaml", job.Name))
		jobConfig := config.BackupConfig{
			Jobs:       []config.BackupJob{job},
			BackupPath: cfg.BackupPath,
			TempPath:   cfg.TempPath,
		}

		if err := config.SaveConfig(&jobConfig, jobConfigPath); err != nil {
			fmt.Printf("Error creating job config for %s: %v\n", job.Name, err)
			continue
		}

		// Create service file
		serviceName := fmt.Sprintf("%s-%s", systemdJobsServiceName, job.Name)
		serviceFile := filepath.Join(systemdDir, serviceName+".service")
		serviceContent := generateJobServiceFile(binaryPath, jobConfigPath, systemdJobsUser, job.Name)
		if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
			fmt.Printf("Error creating service file for %s: %v\n", job.Name, err)
			continue
		}

		// Create timer file if job has a schedule
		if job.Schedule.Enabled && job.Schedule.Type == "systemd" {
			timerFile := filepath.Join(systemdDir, serviceName+".timer")
			timerContent := generateJobTimerFile(serviceName, job.Schedule.Interval)
			if err := os.WriteFile(timerFile, []byte(timerContent), 0644); err != nil {
				fmt.Printf("Error creating timer file for %s: %v\n", job.Name, err)
				continue
			}

			scheduledCount++
		}

		installedCount++
		fmt.Printf("✅ Created service for job: %s\n", job.Name)
	}

	if installedCount == 0 {
		fmt.Println("No enabled backup jobs found to install")
		return
	}

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		fmt.Printf("Error reloading systemd: %v\n", err)
		os.Exit(1)
	}

	// Enable and start timers for scheduled jobs
	for _, job := range cfg.Jobs {
		if job.Enabled && job.Schedule.Enabled && job.Schedule.Type == "systemd" {
			serviceName := fmt.Sprintf("%s-%s", systemdJobsServiceName, job.Name)

			// Enable timer
			if err := exec.Command("systemctl", "enable", serviceName+".timer").Run(); err != nil {
				fmt.Printf("Error enabling timer for %s: %v\n", job.Name, err)
				continue
			}

			// Start timer
			if err := exec.Command("systemctl", "start", serviceName+".timer").Run(); err != nil {
				fmt.Printf("Error starting timer for %s: %v\n", job.Name, err)
				continue
			}

			fmt.Printf("✅ Started timer for job: %s\n", job.Name)
		}
	}

	fmt.Printf("\nSystemd services installed successfully!\n")
	fmt.Printf("Services created: %d\n", installedCount)
	fmt.Printf("Scheduled jobs: %d\n", scheduledCount)
	fmt.Printf("Config base path: %s\n", systemdJobsBasePath)
	fmt.Println("\nTo check status: backtide systemd-jobs status")
	fmt.Println("To view logs: journalctl -u backtide-<job-name>.service")
}

func runSystemdJobsUninstall(cmd *cobra.Command, args []string) {
	fmt.Println("Uninstalling systemd services for backup jobs...")

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires root privileges")
		os.Exit(1)
	}

	// Load configuration to get job names
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Stop and disable all timers
	for _, job := range cfg.Jobs {
		serviceName := fmt.Sprintf("%s-%s", systemdJobsServiceName, job.Name)

		// Stop timer
		if err := exec.Command("systemctl", "stop", serviceName+".timer").Run(); err != nil {
			fmt.Printf("Warning: Failed to stop timer for %s: %v\n", job.Name, err)
		}

		// Disable timer
		if err := exec.Command("systemctl", "disable", serviceName+".timer").Run(); err != nil {
			fmt.Printf("Warning: Failed to disable timer for %s: %v\n", job.Name, err)
		}

		// Remove service and timer files
		systemdDir := "/etc/systemd/system"
		serviceFile := filepath.Join(systemdDir, serviceName+".service")
		timerFile := filepath.Join(systemdDir, serviceName+".timer")

		if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error removing service file for %s: %v\n", job.Name, err)
		}

		if err := os.Remove(timerFile); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error removing timer file for %s: %v\n", job.Name, err)
		}

		// Remove job config file
		jobConfigPath := filepath.Join(systemdJobsBasePath, fmt.Sprintf("%s.yaml", job.Name))
		if err := os.Remove(jobConfigPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error removing config file for %s: %v\n", job.Name, err)
		}

		fmt.Printf("✅ Removed service for job: %s\n", job.Name)
	}

	// Remove base directory if empty
	if err := os.Remove(systemdJobsBasePath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: Could not remove base directory: %v\n", err)
	}

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		fmt.Printf("Error reloading systemd: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Systemd services uninstalled successfully!")
}

func runSystemdJobsStatus(cmd *cobra.Command, args []string) {
	fmt.Println("Checking systemd services status for backup jobs...")

	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Jobs) == 0 {
		fmt.Println("No backup jobs configured")
		return
	}

	for _, job := range cfg.Jobs {
		if !job.Enabled {
			continue
		}

		serviceName := fmt.Sprintf("%s-%s", systemdJobsServiceName, job.Name)
		fmt.Printf("\n=== Job: %s ===\n", job.Name)

		// Check timer status if scheduled
		if job.Schedule.Enabled && job.Schedule.Type == "systemd" {
			cmdTimer := exec.Command("systemctl", "status", serviceName+".timer")
			if output, err := cmdTimer.CombinedOutput(); err != nil {
				fmt.Printf("Timer: %s\n", string(output))
			} else {
				fmt.Printf("Timer: %s\n", string(output))
			}
		}

		// Check service status (last run)
		cmdService := exec.Command("systemctl", "status", serviceName+".service")
		if output, err := cmdService.CombinedOutput(); err != nil {
			fmt.Printf("Service: %s\n", string(output))
		} else {
			fmt.Printf("Service: %s\n", string(output))
		}

		// Show recent logs
		fmt.Println("Recent logs:")
		cmdLogs := exec.Command("journalctl", "-u", serviceName+".service", "--since", "1 hour ago", "-n", "5")
		if output, err := cmdLogs.CombinedOutput(); err != nil {
			fmt.Printf("  No logs found\n")
		} else {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("  %s\n", line)
				}
			}
		}
	}
}

func generateJobServiceFile(binaryPath, configPath, user, jobName string) string {
	tmpl := `[Unit]
Description=Backtide Backup Service - {{.JobName}}
Documentation=https://github.com/mitexleo/backtide
After=network.target docker.service
Requires=docker.service

[Service]
Type=oneshot
User={{.User}}
ExecStart={{.BinaryPath}} backup --config {{.ConfigPath}} --job {{.JobName}}
StandardOutput=journal
StandardError=journal
TimeoutStopSec=300

[Install]
WantedBy=multi-user.target
`

	data := struct {
		BinaryPath string
		ConfigPath string
		User       string
		JobName    string
	}{
		BinaryPath: binaryPath,
		ConfigPath: configPath,
		User:       user,
		JobName:    jobName,
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
Description=Backtide Backup Timer - {{.ServiceName}}
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
