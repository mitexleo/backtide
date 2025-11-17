package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ServiceManager provides abstraction for systemd service operations
type ServiceManager struct {
	ServiceName string
	BinaryPath  string
	ConfigPath  string
	User        string
}

// NewServiceManager creates a new systemd service manager
func NewServiceManager(serviceName, binaryPath, configPath, user string) *ServiceManager {
	return &ServiceManager{
		ServiceName: serviceName,
		BinaryPath:  binaryPath,
		ConfigPath:  configPath,
		User:        user,
	}
}

// ServiceInfo represents information about a systemd service
type ServiceInfo struct {
	Name        string
	IsInstalled bool
	IsRunning   bool
	IsEnabled   bool
	BinaryPath  string
}

// ServiceStatus represents the current status of a systemd service
type ServiceStatus struct {
	ServiceName string
	IsActive    bool
	IsEnabled   bool
	IsRunning   bool
	LoadState   string
	ActiveState string
	SubState    string
}

// IsServiceInstalled checks if the systemd service is installed
func (sm *ServiceManager) IsServiceInstalled() (bool, error) {
	cmd := exec.Command("systemctl", "list-unit-files", sm.ServiceName+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check service installation: %v", err)
	}

	return strings.Contains(string(output), sm.ServiceName+".service"), nil
}

// GetServiceStatus retrieves detailed status of the systemd service
func (sm *ServiceManager) GetServiceStatus() (*ServiceStatus, error) {
	cmd := exec.Command("systemctl", "show", sm.ServiceName+".service", "--property=LoadState,ActiveState,SubState")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get service status: %v", err)
	}

	status := &ServiceStatus{ServiceName: sm.ServiceName}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		switch parts[0] {
		case "LoadState":
			status.LoadState = parts[1]
			status.IsEnabled = parts[1] == "loaded"
		case "ActiveState":
			status.ActiveState = parts[1]
			status.IsActive = parts[1] == "active"
		case "SubState":
			status.SubState = parts[1]
			status.IsRunning = parts[1] == "running"
		}
	}

	return status, nil
}

// GenerateServiceFile generates the systemd service file content
func (sm *ServiceManager) GenerateServiceFile() string {
	return `[Unit]
Description=Backtide Backup Service
Documentation=https://github.com/mitexleo/backtide
After=network.target docker.service
Requires=docker.service

[Service]
Type=oneshot
User=` + sm.User + `
ExecStart=` + sm.BinaryPath + ` backup --config ` + sm.ConfigPath + `
StandardOutput=journal
StandardError=journal
TimeoutStopSec=300

[Install]
WantedBy=multi-user.target
`
}

// GenerateTimerFile generates the systemd timer file content
func (sm *ServiceManager) GenerateTimerFile(schedule string) string {
	var onCalendar string

	switch strings.ToLower(schedule) {
	case "daily":
		onCalendar = "daily"
	case "weekly":
		onCalendar = "weekly"
	case "monthly":
		onCalendar = "monthly"
	case "hourly":
		onCalendar = "hourly"
	default:
		// Assume it's a cron-like expression or systemd calendar event
		onCalendar = schedule
	}

	return `[Unit]
Description=Backtide Backup Timer
Documentation=https://github.com/mitexleo/backtide
Requires=` + sm.ServiceName + `.service

[Timer]
OnCalendar=` + onCalendar + `
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
`
}

// ReloadDaemon reloads the systemd daemon
func (sm *ServiceManager) ReloadDaemon() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %s, error: %v", string(output), err)
	}
	return nil
}

// EnableService enables the systemd service
func (sm *ServiceManager) EnableService() error {
	cmd := exec.Command("systemctl", "enable", sm.ServiceName+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable service: %s, error: %v", string(output), err)
	}
	return nil
}

// EnableTimer enables the systemd timer
func (sm *ServiceManager) EnableTimer() error {
	cmd := exec.Command("systemctl", "enable", sm.ServiceName+".timer")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable timer: %s, error: %v", string(output), err)
	}
	return nil
}

// StartTimer starts the systemd timer
func (sm *ServiceManager) StartTimer() error {
	cmd := exec.Command("systemctl", "start", sm.ServiceName+".timer")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start timer: %s, error: %v", string(output), err)
	}
	return nil
}

// StopTimer stops the systemd timer
func (sm *ServiceManager) StopTimer() error {
	cmd := exec.Command("systemctl", "stop", sm.ServiceName+".timer")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop timer: %s, error: %v", string(output), err)
	}
	return nil
}

// DisableTimer disables the systemd timer
func (sm *ServiceManager) DisableTimer() error {
	cmd := exec.Command("systemctl", "disable", sm.ServiceName+".timer")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to disable timer: %s, error: %v", string(output), err)
	}
	return nil
}

// GetServiceFilePath returns the full path to the service file
func (sm *ServiceManager) GetServiceFilePath() string {
	return filepath.Join("/etc/systemd/system", sm.ServiceName+".service")
}

// GetTimerFilePath returns the full path to the timer file
func (sm *ServiceManager) GetTimerFilePath() string {
	return filepath.Join("/etc/systemd/system", sm.ServiceName+".timer")
}

// UpdateServiceFiles updates the systemd service files with current binary path
func (sm *ServiceManager) UpdateServiceFiles(schedule string) error {
	// Create service file
	serviceFile := sm.GetServiceFilePath()
	serviceContent := sm.GenerateServiceFile()
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to update service file: %v", err)
	}

	// Create timer file
	timerFile := sm.GetTimerFilePath()
	timerContent := sm.GenerateTimerFile(schedule)
	if err := os.WriteFile(timerFile, []byte(timerContent), 0644); err != nil {
		return fmt.Errorf("failed to update timer file: %v", err)
	}

	// Reload systemd daemon
	if err := sm.ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd after update: %v", err)
	}

	return nil
}
